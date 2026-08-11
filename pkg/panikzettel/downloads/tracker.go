// Package downloads counts how often each panikzettel is served and keeps
// those counts in postgres.
package downloads

import (
	"cmp"
	"context"
	"slices"
	"sync"
	"time"

	"github.com/htwr-aachen/backend/pkg/config"
	"github.com/htwr-aachen/backend/pkg/panikzettel/models"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/rs/zerolog/log"
)

var (
	downloadFlushes = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "htwr",
		Name:      "panikzettel_download_flushes_total",
		Help:      "Total number of panikzettel download counter flushes to the database, labeled by outcome.",
	}, []string{"outcome"})

	downloadsPending = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: "htwr",
		Name:      "panikzettel_downloads_pending",
		Help:      "Number of panikzettel with download counts not yet persisted to the database.",
	})
)

// Store persists the download counters.
type Store interface {
	IncrementDownloads(ctx context.Context, deltas []models.DownloadDelta) error
	ListDownloads(ctx context.Context, windows models.DownloadWindows) ([]models.DownloadStat, error)
}

// key counts a panikzettel separately per day, so a buffer spanning midnight is
// still attributed correctly. days are always models.Day values, which makes
// them comparable as map keys.
type key struct {
	filename string
	day      time.Time
}

type pending struct {
	count int64
	first time.Time
	last  time.Time
}

// Tracker buffers download events in memory and flushes them to the Store in
// batches, so serving a panikzettel never waits on a database round trip.
//
// All methods tolerate a nil receiver: a disabled tracker is simply a nil
// *Tracker, and recording into it is a no-op.
type Tracker struct {
	store         Store
	flushInterval time.Duration
	flushTimeout  time.Duration

	mu      sync.Mutex
	pending map[key]*pending

	stop     chan struct{}
	done     chan struct{}
	stopOnce sync.Once
}

// NewTracker starts a tracker whose buffered counts are flushed every
// cfg.FlushInterval.
func NewTracker(store Store, cfg *config.PanikzettelDownloads) *Tracker {
	t := &Tracker{
		store:         store,
		flushInterval: cfg.FlushInterval,
		flushTimeout:  cfg.FlushTimeout,
		pending:       make(map[key]*pending),
		stop:          make(chan struct{}),
		done:          make(chan struct{}),
	}

	go t.run()

	return t
}

// Record counts a single served panikzettel.
func (t *Tracker) Record(filename string) {
	if t == nil || filename == "" {
		return
	}

	now := time.Now()
	k := key{filename: filename, day: models.Day(now)}

	t.mu.Lock()
	defer t.mu.Unlock()

	if entry, ok := t.pending[k]; ok {
		entry.count++
		entry.last = now
		return
	}

	t.pending[k] = &pending{count: 1, first: now, last: now}
	downloadsPending.Set(float64(len(t.pending)))
}

// Stats returns the download counters over the reported windows, including
// everything not yet flushed, most downloaded this semester first.
func (t *Tracker) Stats(ctx context.Context) ([]models.DownloadStat, error) {
	if t == nil {
		return []models.DownloadStat{}, nil
	}

	now := time.Now()
	windows := models.DownloadWindowsAt(now)

	stored, err := t.store.ListDownloads(ctx, windows)
	if err != nil {
		return nil, err
	}

	t.mu.Lock()
	buffered := make(map[key]pending, len(t.pending))
	for k, entry := range t.pending {
		buffered[k] = *entry
	}
	t.mu.Unlock()

	semester := models.SemesterAt(now)

	byFilename := make(map[string]*models.DownloadStat, len(stored)+len(buffered))
	for i := range stored {
		stat := stored[i]
		stat.Semester = semester
		byFilename[stat.Filename] = &stat
	}

	for k, entry := range buffered {
		stat, ok := byFilename[k.filename]
		if !ok {
			// Panikzettel downloaded for the very first time are not in the db yet.
			stat = &models.DownloadStat{
				Filename:        k.filename,
				Semester:        semester,
				FirstDownloadAt: entry.first,
				LastDownloadAt:  entry.last,
			}
			byFilename[k.filename] = stat
		}

		windows.Add(&stat.Downloads, k.day, entry.count)
		if entry.last.After(stat.LastDownloadAt) {
			stat.LastDownloadAt = entry.last
		}
	}

	stats := make([]models.DownloadStat, 0, len(byFilename))
	for _, stat := range byFilename {
		stats = append(stats, *stat)
	}

	sortStats(stats)

	return stats, nil
}

// Counts returns the download counts per filename, including everything not yet
// flushed.
func (t *Tracker) Counts(ctx context.Context) (map[string]models.DownloadCounts, error) {
	stats, err := t.Stats(ctx)
	if err != nil {
		return nil, err
	}

	counts := make(map[string]models.DownloadCounts, len(stats))
	for _, stat := range stats {
		counts[stat.Filename] = stat.Downloads
	}

	return counts, nil
}

// Close stops the flush loop after persisting whatever is still buffered.
func (t *Tracker) Close() {
	if t == nil {
		return
	}

	t.stopOnce.Do(func() {
		close(t.stop)
	})
	<-t.done
}

func (t *Tracker) run() {
	defer close(t.done)

	ticker := time.NewTicker(t.flushInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			t.flush()
		case <-t.stop:
			t.flush()
			return
		}
	}
}

// flush persists the buffered counts. Counts of a failed flush are put back so
// they are retried on the next tick instead of being lost.
func (t *Tracker) flush() {
	t.mu.Lock()
	buffered := t.pending
	t.pending = make(map[key]*pending)
	downloadsPending.Set(0)
	t.mu.Unlock()

	if len(buffered) == 0 {
		return
	}

	deltas := make([]models.DownloadDelta, 0, len(buffered))
	for k, entry := range buffered {
		deltas = append(deltas, models.DownloadDelta{
			Filename:        k.filename,
			Day:             k.day,
			Count:           entry.count,
			FirstDownloadAt: entry.first,
			LastDownloadAt:  entry.last,
		})
	}

	ctx, cancel := context.WithTimeout(context.Background(), t.flushTimeout)
	defer cancel()

	if err := t.store.IncrementDownloads(ctx, deltas); err != nil {
		log.Err(err).Int("panikzettel", len(deltas)).Msg("flushing panikzettel download counts, retrying on next flush")
		downloadFlushes.With(prometheus.Labels{"outcome": "error"}).Inc()
		t.restore(buffered)
		return
	}

	downloadFlushes.With(prometheus.Labels{"outcome": "success"}).Inc()
}

// restore merges counts of a failed flush back into the pending buffer.
func (t *Tracker) restore(buffered map[key]*pending) {
	t.mu.Lock()
	defer t.mu.Unlock()

	for k, entry := range buffered {
		current, ok := t.pending[k]
		if !ok {
			t.pending[k] = entry
			continue
		}

		current.count += entry.count
		if entry.first.Before(current.first) {
			current.first = entry.first
		}
		if entry.last.After(current.last) {
			current.last = entry.last
		}
	}

	downloadsPending.Set(float64(len(t.pending)))
}

// sortStats orders the stats like the db does: most downloaded this semester
// first, ties broken by filename so the api response stays stable.
func sortStats(stats []models.DownloadStat) {
	slices.SortFunc(stats, func(a, b models.DownloadStat) int {
		if a.Downloads.Semester != b.Downloads.Semester {
			return cmp.Compare(b.Downloads.Semester, a.Downloads.Semester)
		}
		return cmp.Compare(a.Filename, b.Filename)
	})
}
