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
	// Downloads counts the downloads of the running semester. It is nil while
	// download tracking is disabled, so the field is omitted instead of
	// reporting a bogus zero count.
	Downloads *int64 `json:"downloads,omitempty"`
}

// DownloadStat is the persisted download counter of a single panikzettel in a
// single semester.
type DownloadStat struct {
	Filename        string    `json:"filename"`
	Semester        string    `json:"semester"`
	Downloads       int64     `json:"downloads"`
	FirstDownloadAt time.Time `json:"first_download_at"`
	LastDownloadAt  time.Time `json:"last_download_at"`
}

// DownloadDelta is a batch of downloads counted since the last flush, waiting
// to be folded into the persisted counters.
type DownloadDelta struct {
	Filename        string
	Semester        string
	Count           int64
	FirstDownloadAt time.Time
	LastDownloadAt  time.Time
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