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

// day is a single persisted daily bucket of the fake store.
type day struct {
	filename string
	day      time.Time
}

// fakeStore is an in-memory stand-in for the postgres backed store.
type fakeStore struct {
	mu      sync.Mutex
	days    map[day]models.DownloadDelta
	flushes int
	err     error
}

func newFakeStore() *fakeStore {
	return &fakeStore{days: make(map[day]models.DownloadDelta)}
}

func (s *fakeStore) IncrementDownloads(_ context.Context, deltas []models.DownloadDelta) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.err != nil {
		return s.err
	}

	s.flushes++
	for _, delta := range deltas {
		k := day{filename: delta.Filename, day: delta.Day}

		stored, ok := s.days[k]
		if !ok {
			s.days[k] = delta
			continue
		}

		stored.Count += delta.Count
		if delta.FirstDownloadAt.Before(stored.FirstDownloadAt) {
			stored.FirstDownloadAt = delta.FirstDownloadAt
		}
		if delta.LastDownloadAt.After(stored.LastDownloadAt) {
			stored.LastDownloadAt = delta.LastDownloadAt
		}
		s.days[k] = stored
	}

	return nil
}

func (s *fakeStore) ListDownloads(_ context.Context, windows models.DownloadWindows) ([]models.DownloadStat, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.err != nil {
		return nil, s.err
	}

	byFilename := make(map[string]*models.DownloadStat, len(s.days))
	for k, stored := range s.days {
		if k.day.Before(windows.Start()) {
			continue
		}

		stat, ok := byFilename[k.filename]
		if !ok {
			stat = &models.DownloadStat{
				Filename:        k.filename,
				FirstDownloadAt: stored.FirstDownloadAt,
				LastDownloadAt:  stored.LastDownloadAt,
			}
			byFilename[k.filename] = stat
		}

		windows.Add(&stat.Downloads, k.day, stored.Count)
		if stored.FirstDownloadAt.Before(stat.FirstDownloadAt) {
			stat.FirstDownloadAt = stored.FirstDownloadAt
		}
		if stored.LastDownloadAt.After(stat.LastDownloadAt) {
			stat.LastDownloadAt = stored.LastDownloadAt
		}
	}

	stats := make([]models.DownloadStat, 0, len(byFilename))
	for _, stat := range byFilename {
		stats = append(stats, *stat)
	}
	sortStats(stats)

	return stats, nil
}

func (s *fakeStore) setError(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.err = err
}

// count is the total the store holds for a panikzettel, across all days.
func (s *fakeStore) count(filename string) int64 {
	s.mu.Lock()
	defer s.mu.Unlock()

	var total int64
	for k, stored := range s.days {
		if k.filename == filename {
			total += stored.Count
		}
	}

	return total
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
	assert.Equal(t, int64(2), stats[0].Downloads.Semester)
	assert.Equal(t, "phys201.pdf", stats[1].Filename)
	assert.Equal(t, int64(1), stats[1].Downloads.Semester)

	counts, err := tracker.Counts(ctx)
	require.NoError(t, err)
	assert.Equal(t, map[string]models.DownloadCounts{
		"math101.pdf": {Semester: 2, Last30Days: 2, Last14Days: 2, Last7Days: 2},
		"phys201.pdf": {Semester: 1, Last30Days: 1, Last14Days: 1, Last7Days: 1},
	}, counts)

	tracker.Close()
}

func TestTrackerStatsCountEachWindow(t *testing.T) {
	store := newFakeStore()
	tracker := newTestTracker(store)
	ctx := context.Background()

	now := time.Now()
	semesterStart := models.SemesterStart(now)

	// One download per window, each on a day only the wider windows cover. Days
	// outside the running semester are left out, they are not reported at all.
	days := []time.Time{models.Day(now), models.Day(now.AddDate(0, 0, -10)), models.Day(now.AddDate(0, 0, -20))}
	deltas := make([]models.DownloadDelta, 0, len(days))
	for _, day := range days {
		if day.Before(semesterStart) {
			continue
		}
		deltas = append(deltas, models.DownloadDelta{
			Filename: "math101.pdf", Day: day, Count: 1,
			FirstDownloadAt: day, LastDownloadAt: day,
		})
	}
	require.NoError(t, store.IncrementDownloads(ctx, deltas))

	stats, err := tracker.Stats(ctx)
	require.NoError(t, err)
	require.Len(t, stats, 1)

	expected := models.DownloadCounts{}
	windows := models.DownloadWindowsAt(now)
	for _, delta := range deltas {
		windows.Add(&expected, delta.Day, delta.Count)
	}

	assert.Equal(t, models.CurrentSemester(), stats[0].Semester)
	assert.Equal(t, expected, stats[0].Downloads)
	assert.Equal(t, int64(1), stats[0].Downloads.Last7Days)

	tracker.Close()
}

func TestTrackerStatsIgnoreDownloadsBeforeTheRunningSemester(t *testing.T) {
	store := newFakeStore()
	tracker := newTestTracker(store)
	ctx := context.Background()

	before := models.SemesterStart(time.Now()).AddDate(0, 0, -1)
	require.NoError(t, store.IncrementDownloads(ctx, []models.DownloadDelta{
		{Filename: "math101.pdf", Day: before, Count: 42, FirstDownloadAt: before, LastDownloadAt: before},
	}))

	tracker.Record("math101.pdf")
	tracker.flush()

	stats, err := tracker.Stats(ctx)
	require.NoError(t, err)
	require.Len(t, stats, 1)
	assert.Equal(t, models.CurrentSemester(), stats[0].Semester)
	assert.Equal(t, int64(1), stats[0].Downloads.Semester)

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
