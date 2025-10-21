package db

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/htwr-aachen/backend/pkg/qa/schema"
	"github.com/jackc/pgx/v5"
	"github.com/rs/zerolog/log"
)

type rowQuerier interface {
	QueryRow(ctx context.Context, query string, args ...any) pgx.Row
}

func getAnswer(db rowQuerier, ctx context.Context, id uint32) (*schema.Answer, error) {
	query := `
	SELECT
		a.id,
		a.question_id,
		a.answer,
		a.answer_rendered,
		a.approved,
		a.known_since,
		a.created_at,
		a.render_version,
		(adr.id IS NOT NULL AND adr.status = 'pending') AS deletion_requested,
		adr.created_at AS deletion_requested_since
	FROM
		answers a
	LEFT JOIN
		deletion_requests adr ON a.id = adr.entity_id AND adr.entity_type = 'answer' AND adr.status = 'pending'
	WHERE
		a.id = $1
`

	var a schema.Answer
	row := db.QueryRow(ctx, query, id)
	err := row.Scan(
		&a.Id,
		&a.QuestionId,
		&a.Answer,
		&a.AnswerRendered,
		&a.Approved,
		&a.KnownSince,
		&a.CreatedAt,
		&a.RenderVersion,
		&a.DeletionRequested,
		&a.DeletionRequestedSince,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, sql.ErrNoRows
		}
		return nil, fmt.Errorf("scanning answer: %w", err)
	}

	log.Trace().Object("answer", a).Msg("fetched answer from db")

	return &a, nil
}

func (db *DB) GetAnswer(ctx context.Context, id uint32) (*schema.Answer, error) {
	return getAnswer(db.db, ctx, id)
}
