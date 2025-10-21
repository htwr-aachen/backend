package schema

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/rs/zerolog"
)

type ApprovedFilter byte

const (
	FILTER_APPROVED     ApprovedFilter = 0
	FILTER_UNAPPROVED   ApprovedFilter = 1
	FILTER_BOTH         ApprovedFilter = 2
	APPROVED_FILTER_LEN                = iota
)

type Question struct {
	Id                     uint32
	Approved               bool
	Title                  string
	Description            sql.NullString
	DescriptionRendered    sql.NullString
	CreatedAt              time.Time
	Answers                []Answer
	DeletionRequested      bool
	DeletionRequestedSince sql.NullTime
	Priority               int32
	DeletionRequestsCount  int32
	RenderedItem
}

func (q Question) MarshalZerologObject(e *zerolog.Event) {
	e.Uint32("question_id", q.Id).
		Bool("question_approved", q.Approved).
		Str("question_title", q.Title).
		Bool("question_deletion_requested", q.DeletionRequested).
		Int32("question_priority", q.Priority).
		Time("question_created_at", q.CreatedAt).
		Int("question_answers_count", len(q.Answers)).
		Int32("question_deletion_requests_count", q.DeletionRequestsCount)

	if q.Description.Valid {
		e.Str("question_description", q.Description.String)
	}

	if q.DescriptionRendered.Valid {
		e.Str("question_description_rendered", q.DescriptionRendered.String)
	}

	if q.DeletionRequestedSince.Valid {
		e.Time("question_deletion_requested_since", q.DeletionRequestedSince.Time)
	}
}

func (q *Question) String() string {
	return fmt.Sprintf("Question{Id: %d, Approved: %t, Title: %q, Description: %s, CreatedAt: %s, Answers: %d, DeletionRequested: %t, DeletionRequestedSince: %s, Priority: %d}",
		q.Id,
		q.Approved,
		q.Title,
		nullStringToString(q.Description),
		q.CreatedAt.Format(time.RFC3339),
		len(q.Answers),
		q.DeletionRequested,
		nullTimeToString(q.DeletionRequestedSince),
		q.Priority,
	)
}

// Helper functions
func nullStringToString(ns sql.NullString) string {
	if ns.Valid {
		return fmt.Sprintf("%q", ns.String)
	}
	return "<null>"
}

func nullTimeToString(nt sql.NullTime) string {
	if nt.Valid {
		return nt.Time.Format(time.RFC3339)
	}
	return "<null>"
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return fmt.Sprintf("%q", s)
	}
	return fmt.Sprintf("%q...", s[:maxLen])
}
