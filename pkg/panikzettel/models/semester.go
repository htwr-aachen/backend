package models

import (
	"fmt"
	"time"
)

func SemesterAt(t time.Time) string {
	year, month := t.Year(), t.Month()

	switch {
	case month >= time.April && month <= time.September:
		return fmt.Sprintf("ss%d", year)
	case month >= time.October:
		return fmt.Sprintf("ws%d", year)
	default:
		return fmt.Sprintf("ws%d", year-1)
	}
}

func CurrentSemester() string {
	return SemesterAt(time.Now())
}

func SemesterStart(t time.Time) time.Time {
	year, month := t.Year(), t.Month()

	switch {
	case month >= time.April && month <= time.September:
		return Day(time.Date(year, time.April, 1, 0, 0, 0, 0, t.Location()))
	case month >= time.October:
		return Day(time.Date(year, time.October, 1, 0, 0, 0, 0, t.Location()))
	default:
		return Day(time.Date(year-1, time.October, 1, 0, 0, 0, 0, t.Location()))
	}
}

// Day is the calendar date of t as a plain date, matching how postgres hands
// back a DATE column, so the two can be compared directly.
func Day(t time.Time) time.Time {
	year, month, day := t.Date()
	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
}
