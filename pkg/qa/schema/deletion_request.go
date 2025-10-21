package schema

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/rs/zerolog"
)

const (
	DELETION_REQUEST_TYPE_QUESTION = "question"
	DELETION_REQUEST_TYPE_ANSWER   = "answer"
)

const (
	DELETION_REQUEST_STATUS_PENDING  = "pending"
	DELETION_REQUEST_STATUS_APPROVED = "approved"
	DELETION_REQUEST_STATUS_REJECTED = "rejected"
)

type DeletionRequest struct {
	Id         uint32
	EntityId   uint32
	EntityType string
	QuestionId uint32
	Reason     string
	Status     string
	ReviewedAt sql.NullTime
	CreatedAt  time.Time
}

func (d *DeletionRequest) String() string {
	return fmt.Sprintf("DeletionRequest{Id: %d, EntityId: %d, EntityType: %q, QuestionId: %d, Reason: %s, Status: %q, ReviewedAt: %s, CreatedAt: %s}",
		d.Id,
		d.EntityId,
		d.EntityType,
		d.QuestionId,
		truncateString(d.Reason, 50),
		d.Status,
		nullTimeToString(d.ReviewedAt),
		d.CreatedAt.Format(time.RFC3339),
	)
}

func (d DeletionRequest) MarshalZerologObject(e *zerolog.Event) {
	e.Uint32("deletion_request_id", d.Id).
		Uint32("deletion_request_entity_id", d.EntityId).
		Str("deletion_request_entity_type", d.EntityType).
		Uint32("deletion_request_question_id", d.QuestionId).
		Str("deletion_request_reason", d.Reason).
		Str("deletion_request_status", d.Status).
		Time("deletion_request_created_at", d.CreatedAt)

	if d.ReviewedAt.Valid {
		e.Time("deletion_request_reviewed_at", d.ReviewedAt.Time)
	}
}
