package db

import (
	"context"
	"fmt"

	"github.com/htwr-aachen/backend/pkg/qa/schema"
	"github.com/lib/pq"
	"github.com/rs/zerolog/log"
)

// query := `
// 	SELECT
// 		q.id,
// 		q.approved,
// 		q.title,
// 		q.description,
// 		q.created_at,
// 		jsonb_agg(
// 			jsonb_build_object(
// 				'id', a.id,
// 				'question_id', a.question_id,
// 				'approved', a.approved,
// 				'answer', a.answer,
// 				'known_since', a.known_since,
// 				'created_at', a.created_at
// 			) ORDER BY a.created_at ASC
// 		) AS approved_answers_json
// 	FROM questions q
// 	JOIN answers a ON q.id = a.question_id
// 	WHERE a.approved = $3
// 	AND q.id < $1
// 	AND q.id IN (
// 		SELECT DISTINCT question_id
// 		FROM answers
// 		WHERE approved = $3
// 		AND question_id < $1
// 		ORDER BY question_id DESC
// 		LIMIT $2
// 	)
// 	GROUP BY q.id, q.title, q.description, q.created_at
// 	ORDER BY q.id DESC;
// `

func (db *DB) ListApprovedAnswers(ctx context.Context, questionId uint32, beforeId uint32, limit int) (map[uint32][]schema.Answer, error) {
	return db.ListAnswers(ctx, []uint32{questionId}, schema.FILTER_UNAPPROVED, beforeId, limit)
}
func (db *DB) ListUnapprovedAnswers(ctx context.Context, questionId uint32, beforeId uint32, limit int) (map[uint32][]schema.Answer, error) {
	return db.ListAnswers(ctx, []uint32{questionId}, schema.FILTER_UNAPPROVED, beforeId, limit)
}

func (db *DB) ListAnswers(ctx context.Context, questionIDs []uint32, approvedFilter schema.ApprovedFilter, beforeId uint32, limit int) (map[uint32][]schema.Answer, error) {

	if len(questionIDs) == 0 {
		return make(map[uint32][]schema.Answer), nil
	}

	query := `
			SELECT
				a.id,
				a.question_id,
				a.answer,
				a.approved,
				a.known_since,
				a.created_at
			FROM answers a
			WHERE
				a.question_id = ANY($1)
				AND ($2::boolean IS NULL OR a.approved = $2)
				AND a.id < $3
			ORDER BY created_at ASC
			LIMIT $4
		`

	// we have to do it as such, to allow for null value
	var approvedValue *bool
	var approvedFilterStr string
	switch approvedFilter {
	case schema.FILTER_APPROVED:
		approvedValue = new(bool)
		*approvedValue = true
		approvedFilterStr = "approved"
	case schema.FILTER_UNAPPROVED:
		approvedValue = new(bool)
		*approvedValue = false
		approvedFilterStr = "unapproved"
	case schema.FILTER_BOTH:
		approvedFilterStr = "both"
	}

	rows, err := db.db.Query(ctx, query, pq.Array(questionIDs), approvedValue, beforeId, limit)
	if err != nil {
		return nil, fmt.Errorf("querying answers for question IDs: %w", err)
	}
	defer rows.Close()
	answersByQuestion := make(map[uint32][]schema.Answer)
	for rows.Next() {
		var a schema.Answer
		if err := rows.Scan(&a.Id, &a.QuestionId, &a.Answer, &a.Approved, &a.KnownSince, &a.CreatedAt); err != nil {
			return nil, fmt.Errorf("scanning answer: %w", err)
		}

		log.Trace().Object("answer", a).Str("approved", approvedFilterStr).Uint32("before_id", beforeId).Int("limit", limit).Msg("fetched answer from db")

		answersByQuestion[a.QuestionId] = append(answersByQuestion[a.QuestionId], a)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating answer rows: %w", err)
	}

	return answersByQuestion, nil
}
