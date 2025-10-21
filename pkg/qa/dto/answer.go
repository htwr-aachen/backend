package dto

import (
	"time"

	"github.com/htwr-aachen/backend/pkg/qa/schema"
)

type NewAnswer struct {
	Answer     string    `json:"answer" validate:"required,gt=2,lt=4096"`
	KnownSince time.Time `json:"known_since" validate:"required,lt"`
}

type Answer struct {
	Id                    uint32    `json:"id"`
	Answer                string    `json:"answer"`
	KnownSince            time.Time `json:"known_since"`
	CreatedAt             time.Time `json:"created_at"`
	Priority              int32     `json:"priority"`
	DeletionRequestsCount int32     `json:"deletion_requests_count"`
}

func NewAnswerFromSchema(a *schema.Answer) Answer {
	return Answer{
		Id:                    a.Id,
		Answer:                a.Answer,
		KnownSince:            a.KnownSince,
		CreatedAt:             a.CreatedAt,
		Priority:              a.Priority,
		DeletionRequestsCount: a.DeletionRequestsCount,
	}
}
