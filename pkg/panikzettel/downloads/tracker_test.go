package downloads

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/htwr-aachen/backend/pkg/config"
	"github.com/htwr-aachen/backend/pkg/panikzettel/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeStore is an in-memory stand-in for the postgres backed store.
type fakeStore struct {
	mu      sync.Mutex
	stats   map[string]models.DownloadStat
	flushes int
	err     error
}

func newFakeStore() *fakeStore {
	return &fakeStore{stats: make(map[string]models.DownloadStat)}
}

func (s *fakeStore) IncrementDownloads(_ context.Context, deltas []models.DownloadDelta) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.err != nil {
		return s.err
	}

	s.flushes++
	for _, delta := range deltas {
		k := delta.Filename + "/" + delta.Semester

		stat, ok := s.stats[k]
		if !ok {
			s.stats[k] = models.DownloadStat{
				Filename:        delta.Filename,
				Semester:        delta.Semester,
				Downloads:       delta.Count,
				FirstDownloadAt: delta.FirstDownloadAt,
				LastDownloadAt:  delta.LastDownloadAt,
			}
			continue
		}

		stat.Downloads += delta.Count
		if delta.LastDownloadAt.After(stat.LastDownloadAt) {
			stat.LastDownloadAt = delta.LastDownloadAt
		}
		s.stats[k] = stat
	}

	return nil
}

func (s *fakeStore) ListDownloads(_ context.Context, semester string) ([]models.DownloadStat, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.err != nil {
		return nil, s.err
	}

	stats := make([]models.DownloadStat, 0, len(s.stats))
	for _, stat := range s.stats {
		if stat.Semester == semester {
			stats = append(stats, stat)
		}
	}
	sortStats(stats)

	return stats, nil
}

func (s *fakeStore) setError(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.err = err
}

func (s *fakeStore) count(filename string) int64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.stats[filename+"/"+models.CurrentSemester()].Downloads
}

// newTestTracker creates a tracker that only flushes when explicitly asked to,
// so tests stay deterministic.
func newTestTracker(store Store) *Tracker {
	return NewTracker(store, &config.PanikzettelDownloads{
		Enabled:       true,
		FlushInterval: time.Hour,
		FlushTimeout:  10 * time.Second,
	})
}

func TestTrackerFlushesRecordedDownloads(t *testing.T) {
	store := newFakeStore()
	tracker := newTestTracker(store)

	tracker.Record("math101.pdf")
	tracker.Record("math101.pdf")
	tracker.Record("phys201.pdf")

	tracker.Close()

	assert.Equal(t, int64(2), store.count("math101.pdf"))
	assert.Equal(t, int64(1), store.count("phys201.pdf"))
}

func TestTrackerIgnoresEmptyFilename(t *testing.T) {
	store := newFakeStore()
	tracker := newTestTracker(store)

	tracker.Record("")
	tracker.Close()

	assert.Equal(t, 0, store.flushes)
}

func TestNilTrackerIsNoop(t *testing.T) {
	var tracker *Tracker

	assert.NotPanics(t, func() {
		tracker.Record("math101.pdf")
		tracker.Close()
	})

	stats, err := tracker.Stats(context.Background())
	require.NoError(t, err)
	assert.Empty(t, stats)
}

func TestTrackerStatsIncludeUnflushedCounts(t *testing.T) {
	store := newFakeStore()
	tracker := newTestTracker(store)
	ctx := context.Background()

	tracker.Record("math101.pdf")
	tracker.flush()

	// Buffered, not yet flushed.
	tracker.Record("math101.pdf")
	tracker.Record("phys201.pdf")

	stats, err := tracker.Stats(ctx)
	require.NoError(t, err)
	require.Len(t, stats, 2)

	// Most downloaded first.
	assert.Equal(t, "math101.pdf", stats[0].Filename)
	assert.Equal(t, int64(2), stats[0].Downloads)
	assert.Equal(t, "phys201.pdf", stats[1].Filename)
	assert.Equal(t, int64(1), stats[1].Downloads)

	counts, err := tracker.Counts(ctx)
	require.NoError(t, err)
	assert.Equal(t, map[string]int64{"math101.pdf": 2, "phys201.pdf": 1}, counts)

	tracker.Close()
}

func TestTrackerStatsOnlyCoverTheRunningSemester(t *testing.T) {
	store := newFakeStore()
	tracker := newTestTracker(store)
	ctx := context.Background()

	previous := models.SemesterAt(time.Now().AddDate(0, -6, 0))
	require.NoError(t, store.IncrementDownloads(ctx, []models.DownloadDelta{
		{Filename: "math101.pdf", Semester: previous, Count: 42, FirstDownloadAt: time.Now(), LastDownloadAt: time.Now()},
	}))

	tracker.Record("math101.pdf")
	tracker.flush()

	stats, err := tracker.Stats(ctx)
	require.NoError(t, err)
	require.Len(t, stats, 1)
	assert.Equal(t, models.CurrentSemester(), stats[0].Semester)
	assert.Equal(t, int64(1), stats[0].Downloads)

	counts, err := tracker.Counts(ctx)
	require.NoError(t, err)
	assert.Equal(t, map[string]int64{"math101.pdf": 1}, counts)

	tracker.Close()
}

func TestTrackerRetainsCountsOnFlushError(t *testing.T) {
	store := newFakeStore()
	tracker := newTestTracker(store)

	store.setError(errors.New("database is down"))

	tracker.Record("math101.pdf")
	tracker.Record("math101.pdf")
	tracker.flush()

	// Nothing may be lost, and the next download must add to the retained count.
	tracker.Record("math101.pdf")

	store.setError(nil)
	tracker.flush()

	assert.Equal(t, int64(3), store.count("math101.pdf"))

	tracker.Close()
}

func TestTrackerFlushesOnInterval(t *testing.T) {
	store := newFakeStore()
	tracker := NewTracker(store, &config.PanikzettelDownloads{
		Enabled:       true,
		FlushInterval: 10 * time.Millisecond,
		FlushTimeout:  time.Second,
	})
	defer tracker.Close()

	tracker.Record("math101.pdf")

	assert.Eventually(t, func() bool {
		return store.count("math101.pdf") == 1
	}, time.Second, 5*time.Millisecond)
}

func TestTrackerRecordIsConcurrencySafe(t *testing.T) {
	store := newFakeStore()
	tracker := newTestTracker(store)

	const goroutines, perGoroutine = 16, 100

	var wg sync.WaitGroup
	wg.Add(goroutines)
	for range goroutines {
		go func() {
			defer wg.Done()
			for range perGoroutine {
				tracker.Record("math101.pdf")
			}
		}()
	}
	wg.Wait()

	tracker.Close()

	assert.Equal(t, int64(goroutines*perGoroutine), store.count("math101.pdf"))
}
