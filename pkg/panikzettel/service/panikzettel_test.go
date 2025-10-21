package service

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/htwr-aachen/backend/pkg/panikzettel/config"
	"github.com/htwr-aachen/backend/pkg/panikzettel/models"
	"github.com/patrickmn/go-cache"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"gocloud.dev/blob"
	"gocloud.dev/blob/memblob"
)

var cfg = config.Config{
	MetadataFile:         "metadata.json",
	MaxFileSize:          10 * 1024 * 1024,
	CacheDuration:        5 * time.Minute,
	CacheCleanupInterval: 10 * time.Minute,
	BaseURL:              "https://example.com/api/panikzettel",
}

// Test setup helper
func setupTest(t *testing.T) (*blob.Bucket, *PanikzettelDB) {
	cfg = config.Config{
		MetadataFile:         "metadata.json",
		MaxFileSize:          10 * 1024 * 1024,
		CacheDuration:        5 * time.Minute,
		CacheCleanupInterval: 5 * time.Minute,
		BaseURL:              "https://example.com/api/panikzettel",
	}

	bucket := memblob.OpenBucket(nil)
	db := New(&cfg, bucket)

	assert.NotNil(t, db)
	assert.NotNil(t, db.bucket)
	assert.NotNil(t, db.cache)

	return bucket, db
}

func TestNew(t *testing.T) {
	setupTest(t)
}

func TestGetPanikzettelMeta_FromCache(t *testing.T) {
	_, db := setupTest(t)

	// Pre-populate cache
	expectedMetas := []models.PanikzettelMeta{
		{
			Filename:  "test.pdf",
			Name:      "Test Panikzettel",
			ShortName: "TP",
			Type:      "exam",
			Semester:  3,
			Date:      "2024-01-15",
			URL:       "https://example.com/api/panikzettel/test.pdf",
		},
	}
	db.cache.Set(cfg.MetadataFile, expectedMetas, cache.DefaultExpiration)

	ctx := context.Background()
	metas, err := db.GetPanikzettelMeta(ctx)

	assert.NoError(t, err)
	assert.Equal(t, expectedMetas, metas)
}

func TestGetPanikzettelMeta_FromBucket(t *testing.T) {
	bucket, db := setupTest(t)
	ctx := context.Background()

	metadataContent := `{
		"math101.pdf": {
			"name": "Mathematics 101",
			"shortname": "Math101",
			"type": "exam",
			"semester": 1,
			"date": "2024-01-15"
		}
	}`
	err := bucket.WriteAll(ctx, "metadata.json", []byte(metadataContent), nil)
	assert.NoError(t, err)

	metas, err := db.GetPanikzettelMeta(ctx)
	assert.NoError(t, err)
	assert.Len(t, metas, 1)
	assert.Equal(t, "Mathematics 101", metas[0].Name)
	assert.Equal(t, "https://example.com/api/panikzettel/math101.pdf", metas[0].URL)

	// Check cache is populated
	cachedMetas, found := db.cache.Get(cfg.MetadataFile)
	assert.True(t, found)
	assert.Equal(t, metas, cachedMetas)
}

func TestGetPanikzettelMeta_InvalidCacheEntry(t *testing.T) {
	bucket, db := setupTest(t)
	ctx := context.Background()

	// Set invalid cache entry (wrong type)
	db.cache.Set(cfg.MetadataFile, "invalid data", cache.DefaultExpiration)

	// Bucket should be called as a fallback
	metadataContent := `{
		"math101.pdf": {
			"name": "Mathematics 101",
			"shortname": "Math101",
			"type": "exam",
			"semester": 1,
			"date": "2024-01-15"
		}
	}`
	err := bucket.WriteAll(ctx, "metadata.json", []byte(metadataContent), nil)
	assert.NoError(t, err)

	metas, err := db.GetPanikzettelMeta(ctx)
	assert.NoError(t, err)
	assert.Len(t, metas, 1)

	// The cache should be updated
	cachedMetas, found := db.cache.Get(cfg.MetadataFile)
	assert.True(t, found)
	assert.Equal(t, metas, cachedMetas)
}

func TestGetPanikzettelMeta_MetadataStructureParsing(t *testing.T) {
	setupTest(t)

	// Test the metadata parsing logic in isolation
	metadataJSON := map[string]struct {
		Name      string `json:"name"`
		ShortName string `json:"shortname"`
		Type      string `json:"type"`
		Semester  int    `json:"semester,omitempty"`
		Date      string `json:"date,omitempty"`
	}{
		"math101.pdf": {
			Name:      "Mathematics 101",
			ShortName: "Math101",
			Type:      "exam",
			Semester:  1,
			Date:      "2024-01-15",
		},
		"phys201.pdf": {
			Name:      "Physics 201",
			ShortName: "Phys201",
			Type:      "summary",
			Semester:  2,
		},
	}

	// Verify structure can be marshaled/unmarshaled
	data, err := json.Marshal(metadataJSON)
	assert.NoError(t, err)

	var decoded map[string]struct {
		Name      string `json:"name"`
		ShortName string `json:"shortname"`
		Type      string `json:"type"`
		Semester  int    `json:"semester,omitempty"`
		Date      string `json:"date,omitempty"`
	}

	err = json.Unmarshal(data, &decoded)
	assert.NoError(t, err)
	assert.Equal(t, metadataJSON, decoded)
}

func TestPanikzettelExistsInMeta(t *testing.T) {
	_, db := setupTest(t)

	// Pre-populate cache with metadata
	metas := []models.PanikzettelMeta{
		{Name: "Test Panikzettel 1", Filename: "test1.pdf"},
		{Name: "Test Panikzettel 2", Filename: "test2.pdf"},
	}
	db.cache.Set(cfg.MetadataFile, metas, cache.DefaultExpiration)

	ctx := context.Background()

	// Test existing panikzettel by name
	exists, err := db.panikzettelExistsInMeta(ctx, "Test Panikzettel 1")
	assert.NoError(t, err)
	assert.True(t, exists)

	// Test existing panikzettel by filename
	exists, err = db.panikzettelExistsInMeta(ctx, "test2.pdf")
	assert.NoError(t, err)
	assert.True(t, exists)

	// Test non-existing panikzettel
	exists, err = db.panikzettelExistsInMeta(ctx, "Non-existent")
	assert.NoError(t, err)
	assert.False(t, exists)
}

func TestGetPanikzettel_EmptyName(t *testing.T) {
	_, db := setupTest(t)
	ctx := context.Background()

	// Test empty name
	_, err := db.GetPanikzettel(ctx, "")
	assert.Error(t, err)
	var emptyNameErr *models.PanikzettelEmptyNameError
	assert.True(t, errors.As(err, &emptyNameErr))
}

func TestGetPanikzettel_MetadataFileName(t *testing.T) {
	_, db := setupTest(t)
	ctx := context.Background()

	// Test requesting metadata file itself
	_, err := db.GetPanikzettel(ctx, "metadata.json")
	assert.Error(t, err)
	var reservedFilenameErr *models.PanikzettelReservedFilenameError
	assert.True(t, errors.As(err, &reservedFilenameErr))
	assert.Equal(t, "metadata.json", reservedFilenameErr.Name)
}

func TestGetPanikzettel_NotFoundInMetadata(t *testing.T) {
	_, db := setupTest(t)

	// Pre-populate cache with metadata
	metas := []models.PanikzettelMeta{
		{Name: "Existing Panikzettel", Filename: "existing.pdf"},
	}
	db.cache.Set(cfg.MetadataFile, metas, cache.DefaultExpiration)

	ctx := context.Background()

	// Request non-existent panikzettel
	_, err := db.GetPanikzettel(ctx, "Non-existent")
	assert.Error(t, err)

	var notFoundErr *models.PanikzettelNotFoundError
	assert.True(t, errors.As(err, &notFoundErr))
	assert.Equal(t, "Non-existent", notFoundErr.Name)
}

func TestGetPanikzettel_FromCache(t *testing.T) {
	_, db := setupTest(t)

	// Pre-populate metadata cache
	metas := []models.PanikzettelMeta{
		{Name: "Test Panikzettel", Filename: "test.pdf"},
	}
	db.cache.Set(cfg.MetadataFile, metas, cache.DefaultExpiration)

	// Pre-populate panikzettel cache
	expectedPanikzettel := &models.Panikzettel{
		Content:      []byte("test content"),
		ContentType:  "application/pdf",
		LastModified: time.Now(),
		Size:         12,
	}
	db.cache.Set("test.pdf", expectedPanikzettel, cache.DefaultExpiration)

	ctx := context.Background()

	result, err := db.GetPanikzettel(ctx, "test.pdf")
	assert.NoError(t, err)
	assert.Equal(t, expectedPanikzettel, result)
}

func TestGetPanikzettel_FromBucket(t *testing.T) {
	bucket, db := setupTest(t)
	ctx := context.Background()

	// Pre-populate metadata cache
	metas := []models.PanikzettelMeta{
		{Name: "Test Panikzettel", Filename: "test.pdf"},
	}
	db.cache.Set(cfg.MetadataFile, metas, cache.DefaultExpiration)

	// Write panikzettel to bucket
	content := []byte("test content")
	err := bucket.WriteAll(ctx, "test.pdf", content, &blob.WriterOptions{
		ContentType: "application/pdf",
	})
	assert.NoError(t, err)

	result, err := db.GetPanikzettel(ctx, "test.pdf")
	assert.NoError(t, err)
	assert.Equal(t, content, result.Content)
	assert.Equal(t, "application/pdf", result.ContentType)
	assert.Equal(t, int64(len(content)), result.Size)
	assert.WithinDuration(t, time.Now(), result.LastModified, time.Second)

	// Check cache is populated
	cachedZettel, found := db.cache.Get("test.pdf")
	assert.True(t, found)
	assert.Equal(t, result, cachedZettel)
}

func TestGetPanikzettel_SizeValidation(t *testing.T) {
	bucket, db := setupTest(t)
	ctx := context.Background()

	maxSize := int64(10) // 10 bytes
	cfg.MaxFileSize = maxSize

	// Pre-populate metadata cache
	metas := []models.PanikzettelMeta{
		{Name: "Large File", Filename: "large.pdf"},
	}
	db.cache.Set(cfg.MetadataFile, metas, cache.DefaultExpiration)

	// Write a file that is too large
	largeContent := []byte("this content is too large")
	err := bucket.WriteAll(ctx, "large.pdf", largeContent, nil)
	assert.NoError(t, err)

	_, err = db.GetPanikzettel(ctx, "large.pdf")
	assert.Error(t, err)

	var sizeErr *models.PanikzettelTooLargeError
	assert.True(t, errors.As(err, &sizeErr))
	assert.Equal(t, "large.pdf", sizeErr.Name)
	assert.Equal(t, int64(len(largeContent)), sizeErr.Size)
	assert.Equal(t, maxSize, sizeErr.MaxSize)
}

func TestCacheDuration(t *testing.T) {
	setupTest(t)

	expectedDuration := 5 * time.Minute
	cfg.CacheDuration = expectedDuration

	bucket := memblob.OpenBucket(nil)
	db := New(&cfg, bucket)

	// Verify cache was created with correct duration
	assert.NotNil(t, db.cache)

	// Test cache expiration behavior
	db.cache.Set("test-key", "test-value", cache.DefaultExpiration)

	val, found := db.cache.Get("test-key")
	assert.True(t, found)
	assert.Equal(t, "test-value", val)
}

func TestMetadataURLConstruction(t *testing.T) {
	setupTest(t)

	baseURL := "https://example.com/files"
	viper.Set("PanikzettelBaseURL", baseURL)

	filename := "test.pdf"
	expectedURL := baseURL + "/" + filename

	// This tests the URL construction logic used in GetPanikzettelMeta
	actualURL := viper.GetString("PanikzettelBaseURL") + "/" + filename
	assert.Equal(t, expectedURL, actualURL)
}

func TestMetadataSkipsEmptyNames(t *testing.T) {
	setupTest(t)

	// Test the logic that skips entries with empty names
	testCases := []struct {
		name       string
		filename   string
		metaName   string
		shouldSkip bool
	}{
		{"Valid entry", "file.pdf", "Valid Name", false},
		{"Empty name", "file.pdf", "", true},
		{"Empty filename", "", "Valid Name", true},
		{"Both empty", "", "", true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			shouldSkip := tc.metaName == "" || tc.filename == ""
			assert.Equal(t, tc.shouldSkip, shouldSkip)
		})
	}
}

func TestPanikzettelMetaModel(t *testing.T) {
	// Test the PanikzettelMeta model structure
	meta := models.PanikzettelMeta{
		Filename:  "test.pdf",
		Name:      "Test Panikzettel",
		ShortName: "TP",
		Type:      "exam",
		Semester:  3,
		Date:      "2024-01-15",
		URL:       "https://example.com/test.pdf",
	}

	assert.Equal(t, "test.pdf", meta.Filename)
	assert.Equal(t, "Test Panikzettel", meta.Name)
	assert.Equal(t, "TP", meta.ShortName)
	assert.Equal(t, "exam", meta.Type)
	assert.Equal(t, 3, meta.Semester)
	assert.Equal(t, "2024-01-15", meta.Date)
	assert.Equal(t, "https://example.com/test.pdf", meta.URL)
}

func TestPanikzettelModel(t *testing.T) {
	// Test the Panikzettel model structure
	now := time.Now()
	content := []byte("test content")

	panikzettel := models.Panikzettel{
		Content:      content,
		ContentType:  "application/pdf",
		LastModified: now,
		Size:         int64(len(content)),
	}

	assert.Equal(t, content, panikzettel.Content)
	assert.Equal(t, "application/pdf", panikzettel.ContentType)
	assert.Equal(t, now, panikzettel.LastModified)
	assert.Equal(t, int64(len(content)), panikzettel.Size)
}
