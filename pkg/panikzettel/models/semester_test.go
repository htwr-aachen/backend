package models

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestSemesterAt(t *testing.T) {
	tests := []struct {
		name string
		at   time.Time
		want string
	}{
		{"start of summer", time.Date(2026, time.April, 1, 0, 0, 0, 0, time.UTC), "ss2026"},
		{"mid summer", time.Date(2026, time.July, 24, 12, 0, 0, 0, time.UTC), "ss2026"},
		{"end of summer", time.Date(2026, time.September, 30, 23, 59, 59, 0, time.UTC), "ss2026"},
		{"start of winter", time.Date(2026, time.October, 1, 0, 0, 0, 0, time.UTC), "ws2026"},
		{"winter after new year", time.Date(2027, time.February, 5, 0, 0, 0, 0, time.UTC), "ws2026"},
		{"end of winter", time.Date(2027, time.March, 31, 23, 59, 59, 0, time.UTC), "ws2026"},
		{"next summer", time.Date(2027, time.April, 1, 0, 0, 0, 0, time.UTC), "ss2027"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, SemesterAt(tt.at))
		})
	}
}
