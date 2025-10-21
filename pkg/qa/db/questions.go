package db

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/htwr-aachen/backend/pkg/qa/schema"
	"github.com/lib/pq"
)

func getQuestion(rowQuerier rowQuerier, ctx context.Context, questionId uint32) (*schema.Question, error) {
	query := `
		SELECT
			q.id,
			q.approved,
			q.title,
			q.description,
			q.description_rendered,
			q.render_version,
			q.created_at,
			(qdr.id IS NOT NULL AND qdr.status = 'pending') AS deletion_requested,
			qdr.created_at AS deletion_requested_since
		FROM
			questions q
		LEFT JOIN
			deletion_requests qdr ON q.id = qdr.entity_id AND qdr.entity_type = 'question' AND qdr.status = 'pending'
		WHERE
			q.id = $1
	`

	var q schema.Question
	row := rowQuerier.QueryRow(ctx, query, questionId)
	err := row.Scan(
		&q.Id,
		&q.Approved,
		&q.Title,
		&q.Description,
		&q.DescriptionRendered,
		&q.RenderVersion,
		&q.CreatedAt,
		&q.DeletionRequested,
		&q.DeletionRequestedSince,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, sql.ErrNoRows
		}
		return nil, fmt.Errorf("scanning question: %w", err)
	}

	return &q, nil
}

func (db *DB) GetQuestion(ctx context.Context, questionId uint32) (*schema.Question, error) {
	return getQuestion(db.db, ctx, questionId)
}

func (db *DB) getQuestions(ctx context.Context, questionIDs []uint32) ([]schema.Question, error) {
	query := `
		SELECT
			q.id,
			q.approved,
			q.title,
			q.description,
			q.description_rendered,
			q.render_version,
			q.created_at,
			(qdr.id IS NOT NULL AND qdr.status = 'pending') AS deletion_requested,
			qdr.created_at AS deletion_requested_since
		FROM
			questions q
		LEFT JOIN
			deletion_requests qdr ON q.id = qdr.entity_id AND qdr.entity_type = 'question' AND qdr.status = 'pending'
		WHERE
			q.id = ANY($1)
	`

	rows, err := db.db.Query(ctx, query, pq.Array(questionIDs))
	if err != nil {
		return nil, fmt.Errorf("querying questions by IDs: %w", err)
	}
	defer rows.Close()

	var questions []schema.Question
	for rows.Next() {
		var q schema.Question
		err := rows.Scan(
			&q.Id,
			&q.Approved,
			&q.Title,
			&q.Description,
			&q.DescriptionRendered,
			&q.RenderVersion,
			&q.CreatedAt,
			&q.DeletionRequested,
			&q.DeletionRequestedSince,
		)
		if err != nil {
			return nil, fmt.Errorf("scanning question: %w", err)
		}
		q.Answers = []schema.Answer{}
		questions = append(questions, q)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating questions: %w", err)
	}

	return questions, nil
}

func (db *DB) QuestionExists(ctx context.Context, id uint32) (bool, error) {
	var exists bool
	err := db.db.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM questions WHERE approved IS TRUE AND id = $1)", id).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("query error checking question existence (%v): %w", id, err)
	}
	return exists, nil
}
