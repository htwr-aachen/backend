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
