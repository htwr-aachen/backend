package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/htwr-aachen/backend/pkg/panikzettel/config"
	"github.com/htwr-aachen/backend/pkg/panikzettel/models"
	"github.com/patrickmn/go-cache"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/rs/zerolog/log"
	"gocloud.dev/blob"
)

var (
	panikzettelDownloadsVec = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "htwr",
		Name:      "panikzettel_downloads_total",
		Help:      "Total number of panikzettel download attempts, labeled by outcome.",
	}, []string{"panikzettel_name", "outcome"})

	panikzettelMetadataRefreshes = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "htwr",
		Name:      "panikzettel_metadata_refreshes_total",
		Help:      "Total number of panikzettel metadata refresh attempts, labeled by outcome.",
	}, []string{"outcome"})

	panikzettelMetadataCount = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: "htwr",
		Name:      "panikzettel_metadata_count",
		Help:      "Total number of panikzettel entries in the metadata.",
	})
	panikzettelMetadataDownloads = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: "htwr",
		Name:      "panikzettel_metadata_downloads_total",
		Help:      "Total number of times the metadata file has been downloaded from the bucket.",
	})
)

type PanikzettelDB struct {
	bucket *blob.Bucket
	cache  *cache.Cache
	cfg    *config.Config
}

func New(cfg *config.Config, bucket *blob.Bucket) *PanikzettelDB {
	return &PanikzettelDB{
		cfg:    cfg,
		bucket: bucket,
		cache:  cache.New(cfg.CacheDuration, cfg.CacheCleanupInterval),
	}
}

func (db *PanikzettelDB) GetPanikzettelMeta(ctx context.Context) ([]models.PanikzettelMeta, error) {

	panikzettelMetadataDownloads.Inc()
	cached, found := db.cache.Get(db.cfg.MetadataFilename)
	if found {
		if metas, ok := cached.([]models.PanikzettelMeta); ok {
			return metas, nil
		}
		// Invalid cache entry, remove it
		db.cache.Delete(db.cfg.MetadataFilename)
		log.Warn().Msg("Invalid cache entry found for metadata, clearing cache")
	}

	log.Debug().Msg("Refreshing panikzettel cache")

	reader, err := db.bucket.NewReader(ctx, db.cfg.MetadataFilename, nil)
	if err != nil {
		panikzettelMetadataRefreshes.With(prometheus.Labels{"outcome": "error"}).Inc()
		return nil, fmt.Errorf("metadata file '%s' not accessible: %w", db.cfg.MetadataFilename, err)
	}
	defer func() {
		if closeErr := reader.Close(); closeErr != nil {
			log.Err(closeErr).Msg("Failed to close metadata reader")
		}
	}()

	var metadataResult map[string]struct {
		Name      string `json:"name"`
		ShortName string `json:"shortname"`
		Type      string `json:"type"`
		Semester  int    `json:"semester,omitempty"`
		Date      string `json:"date,omitempty"`
	}

	limitedReader := io.LimitReader(reader, db.cfg.MaxFileSize)
	decoder := json.NewDecoder(limitedReader)
	err = decoder.Decode(&metadataResult)
	if err != nil {
		panikzettelMetadataRefreshes.With(prometheus.Labels{"outcome": "error"}).Inc()
		return nil, fmt.Errorf("could not decode metadata %w", err)
	}

	// parse metadata.json file
	metas := make([]models.PanikzettelMeta, 0)

	for filename, v := range metadataResult {

		if v.Name == "" || filename == "" {
			log.Warn().Str("filename", filename).Msg("Skipping metadata entry with empty name")
			continue
		}

		meta := models.PanikzettelMeta{
			Filename:  filename,
			Name:      v.Name,
			ShortName: v.ShortName,
			Type:      v.Type,
			Semester:  v.Semester,
			Date:      v.Date,
		}
		meta.URL = fmt.Sprintf("%s/%s", db.cfg.BaseURL, filename)
		metas = append(metas, meta)
	}

	db.cache.Set(db.cfg.MetadataFilename, metas, cache.DefaultExpiration)
	log.Debug().Int("count", len(metas)).Msg("Loaded Panikzettel metadata")

	panikzettelMetadataRefreshes.With(prometheus.Labels{"outcome": "success"}).Inc()
	panikzettelMetadataCount.Set(float64(len(metas)))
	return metas, nil
}

func (db *PanikzettelDB) panikzettelExistsInMeta(ctx context.Context, name string) (bool, error) {
	metas, err := db.GetPanikzettelMeta(ctx)
	if err != nil {
		return false, err
	}

	for _, meta := range metas {
		if meta.Name == name || meta.Filename == name {
			return true, nil
		}
	}
	return false, nil
}

func (db *PanikzettelDB) GetPanikzettel(ctx context.Context, name string) (*models.Panikzettel, error) {

	if name == "" {
		panikzettelDownloadsVec.With(prometheus.Labels{"panikzettel_name": name, "outcome": "error"}).Inc()
		return nil, &models.PanikzettelEmptyNameError{}
	}

	if name == db.cfg.MetadataFilename {
		panikzettelDownloadsVec.With(prometheus.Labels{"panikzettel_name": name, "outcome": "error"}).Inc()
		return nil, &models.PanikzettelReservedFilenameError{Name: name}
	}

	cached, found := db.cache.Get(name)

	if found {
		panikzettelDownloadsVec.With(prometheus.Labels{"panikzettel_name": name, "outcome": "cached"}).Inc()
		return cached.(*models.Panikzettel), nil
	}

	found, err := db.panikzettelExistsInMeta(ctx, name)

	if err != nil {
		panikzettelDownloadsVec.With(prometheus.Labels{"panikzettel_name": name, "outcome": "error"}).Inc()
		return nil, fmt.Errorf("could not check requested panikzettel due to no metadata %w", err)
	}

	if !found {
		panikzettelDownloadsVec.With(prometheus.Labels{"panikzettel_name": name, "outcome": "not_found"}).Inc()
		return nil, &models.PanikzettelNotFoundError{Name: name}
	}

	cached, found = db.cache.Get(name)

	if found {
		panikzettelDownloadsVec.With(prometheus.Labels{"panikzettel_name": name, "outcome": "cached"}).Inc()
		return cached.(*models.Panikzettel), nil
	}

	attr, err := db.bucket.Attributes(ctx, name)
	if err != nil {
		panikzettelDownloadsVec.With(prometheus.Labels{"panikzettel_name": name, "outcome": "error"}).Inc()
		return nil, fmt.Errorf("could not retrieve panikzettel attributes from bucket: %w", err)
	}

	if attr.Size > db.cfg.MaxFileSize {
		panikzettelDownloadsVec.With(prometheus.Labels{"panikzettel_name": name, "outcome": "error"}).Inc()

		return nil, &models.PanikzettelTooLargeError{
			Name:    name,
			Size:    attr.Size,
			MaxSize: db.cfg.MaxFileSize,
		}
	}

	reader, err := db.bucket.NewReader(ctx, name, nil)
	if err != nil {
		panikzettelDownloadsVec.With(prometheus.Labels{"panikzettel_name": name, "outcome": "error"}).Inc()
		return nil, fmt.Errorf("failed to create reader for panikzettel: %w", err)
	}
	defer func() {
		if closeErr := reader.Close(); closeErr != nil {
			log.Err(closeErr).Str("name", name).Msg("Failed to close panikzettel reader")
		}
	}()

	limitedReader := io.LimitReader(reader, db.cfg.MaxFileSize)
	content, err := io.ReadAll(limitedReader)
	if err != nil {
		panikzettelDownloadsVec.With(prometheus.Labels{"panikzettel_name": name, "outcome": "error"}).Inc()
		return nil, fmt.Errorf("failed to read panikzettel content: %w", err)
	}

	// Create and populate the panikzettel object
	panikzettel := &models.Panikzettel{
		Content:      content,
		ContentType:  attr.ContentType,
		LastModified: attr.ModTime,
		Size:         int64(len(content)),
	}

	db.cache.Set(name, panikzettel, cache.DefaultExpiration)
	panikzettelDownloadsVec.With(prometheus.Labels{"panikzettel_name": name, "outcome": "success"}).Inc()

	return panikzettel, nil
}
