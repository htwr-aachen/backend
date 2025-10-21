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