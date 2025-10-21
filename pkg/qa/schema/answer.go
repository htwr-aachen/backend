package schema

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/rs/zerolog"
)

type Answer struct {
	Id                     uint32
	QuestionId             uint32
	Answer                 string
	AnswerRendered         string
	Approved               bool
	KnownSince             time.Time
	CreatedAt              time.Time
	DeletionRequested      bool
	DeletionRequestedSince sql.NullTime
	Priority               int32
	DeletionRequestsCount  int32
	RenderedItem
}

func (a *Answer) String() string {
	return fmt.Sprintf("Answer{Id: %d, QuestionId: %d, Approved: %t, Answer: %s, KnownSince: %s, CreatedAt: %s, DeletionRequested: %t, DeletionRequestedSince: %s, Priority: %d}",
		a.Id,
		a.QuestionId,
		a.Approved,
		truncateString(a.Answer, 50),
		a.KnownSince.Format(time.RFC3339),
		a.CreatedAt.Format(time.RFC3339),
		a.DeletionRequested,
		nullTimeToString(a.DeletionRequestedSince),
		a.Priority,
	)
}

func (a Answer) MarshalZerologObject(e *zerolog.Event) {
	e.Uint32("answer_id", a.Id).
		Uint32("answer_question_id", a.QuestionId).
		Bool("answer_approved", a.Approved).
		Str("answer_content", a.Answer).
		Bool("answer_deletion_requested", a.DeletionRequested).
		Int32("answer_priority", a.Priority).
		Time("answer_known_since", a.KnownSince).
		Time("answer_created_at", a.CreatedAt)

	if a.DeletionRequestedSince.Valid {
		e.Time("answer_deletion_requested_since", a.DeletionRequestedSince.Time)
	}
}
