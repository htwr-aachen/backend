package dto

import (
	"time"

	"github.com/htwr-aachen/backend/pkg/qa/schema"
)

type NewQuestion struct {
	Title       string `json:"title" validate:"required,gt=5,lt=150"`
	Description string `json:"description,omitempty" validate:"lt=4096"`
}

type Question struct {
	ID                    uint32    `json:"id"`
	Title                 string    `json:"title"`
	Description           string    `json:"description,omitempty"`
	CreatedAt             time.Time `json:"created_at"`
	Answer                *Answer   `json:"answer,omitempty"`
	Priority              int32     `json:"priority"`
	DeletionRequestsCount int32     `json:"deletion_requests_count,omitempty"`
}

func NewQuestionFromSchema(q *schema.Question) Question {
	description := ""
	if q.Description.Valid {
		description = q.Description.String
	}

	var answer *Answer
	if len(q.Answers) > 0 {
		dtoAnswer := NewAnswerFromSchema(&q.Answers[0])
		answer = &dtoAnswer
	}

	return Question{
		ID:                    q.Id,
		Title:                 q.Title,
		Description:           description,
		CreatedAt:             q.CreatedAt,
		Answer:                answer,
		DeletionRequestsCount: q.DeletionRequestsCount,
	}
}

func QuestionsFromSchemas(qs []schema.Question) []Question {
	if len(qs) <= 0 {
		return []Question{}
	}

	dtos := make([]Question, len(qs))
	for i, q := range qs {
		dtos[i] = NewQuestionFromSchema(&q)
	}

	return dtos
}

type GetQuestionsQueryDTO struct {
	Answered     bool
	LastPriority int
	LastId       uint32
	Limit        int
}

type GetQuestionsQueryResDTO struct {
	Questions []Question `json:"questions"`
}
