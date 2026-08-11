package models

import (
	"fmt"
	"time"
)

type Panikzettel struct {
	Size         int64
	Content      []byte
	LastModified time.Time
	ContentType  string
}

type PanikzettelMeta struct {
	Name      string `json:"name"`
	ShortName string `json:"shortname"`
	Type      string `json:"type"`
	Filename  string `json:"filename"`
	URL       string `json:"url"`
	Semester  int    `json:"semester,omitempty"`
	Date      string `json:"date"`
	// Downloads is nil while download tracking is disabled, so the field is
	// omitted instead of reporting bogus zero counts.
	Downloads *DownloadCounts `json:"downloads,omitempty"`
}

// DownloadCounts are the downloads of a panikzettel over the reported time
// windows. The windows overlap: a download of today is counted in all of them.
type DownloadCounts struct {
	Semester   int64 `json:"semester"`
	Last30Days int64 `json:"last_30_days"`
	Last14Days int64 `json:"last_14_days"`
	Last7Days  int64 `json:"last_7_days"`
}

// DownloadStat is the download counter of a single panikzettel.
type DownloadStat struct {
	Filename        string         `json:"filename"`
	Semester        string         `json:"semester"`
	Downloads       DownloadCounts `json:"downloads"`
	FirstDownloadAt time.Time      `json:"first_download_at"`
	LastDownloadAt  time.Time      `json:"last_download_at"`
}

// DownloadDelta is a batch of downloads counted since the last flush, waiting
// to be folded into the persisted daily counters.
type DownloadDelta struct {
	Filename        string
	Day             time.Time
	Count           int64
	FirstDownloadAt time.Time
	LastDownloadAt  time.Time
}

// DownloadWindows are the first days included in each reported window.
type DownloadWindows struct {
	Semester   time.Time
	Last30Days time.Time
	Last14Days time.Time
	Last7Days  time.Time
}

func DownloadWindowsAt(t time.Time) DownloadWindows {
	today := Day(t)

	return DownloadWindows{
		Semester:   SemesterStart(t),
		Last30Days: today.AddDate(0, 0, -29),
		Last14Days: today.AddDate(0, 0, -13),
		Last7Days:  today.AddDate(0, 0, -6),
	}
}

// Start is the first day any of the windows covers.
func (w DownloadWindows) Start() time.Time {
	start := w.Semester
	for _, day := range []time.Time{w.Last30Days, w.Last14Days, w.Last7Days} {
		if day.Before(start) {
			start = day
		}
	}

	return start
}

// Add folds the downloads of a single day into the counts of every window that
// day falls into.
func (w DownloadWindows) Add(counts *DownloadCounts, day time.Time, count int64) {
	if !day.Before(w.Semester) {
		counts.Semester += count
	}
	if !day.Before(w.Last30Days) {
		counts.Last30Days += count
	}
	if !day.Before(w.Last14Days) {
		counts.Last14Days += count
	}
	if !day.Before(w.Last7Days) {
		counts.Last7Days += count
	}
}

// Errors

type PanikzettelTooLargeError struct {
	Name    string
	Size    int64
	MaxSize int64
}

func (e *PanikzettelTooLargeError) Error() string {
	return fmt.Sprintf("panikzettel %s size %d > %d exceeds max size %d", e.Name, e.Size, e.MaxSize, e.MaxSize)
}

// PanikzettelNotFoundError handles not found panikzettel errors for correct http return codes
type PanikzettelNotFoundError struct {
	Name string
}

func (e *PanikzettelNotFoundError) Error() string {
	return fmt.Sprintf("panikzettel '%s' not found", e.Name)
}

// PanikzettelEmptyNameError handles empty name errors for correct http return codes
type PanikzettelEmptyNameError struct{}

func (e *PanikzettelEmptyNameError) Error() string {
	return "panikzettel name is empty"
}

// PanikzettelReservedFilenameError handles reserved filename errors for correct http return codes
type PanikzettelReservedFilenameError struct {
	Name string
}

func (e *PanikzettelReservedFilenameError) Error() string {
	return fmt.Sprintf("panikzettel '%s' is a reserved filename", e.Name)
}