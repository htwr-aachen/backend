package db

import (
	"context"
	"fmt"

	"github.com/htwr-aachen/backend/pkg/qa/schema"
	"github.com/jackc/pgx/v5"
	"github.com/rs/zerolog/log"
)

// ApproveAnswer approves an answer to a question to be shown to others
func (db *DB) ApproveAnswer(ctx context.Context, id uint32, by string) (*schema.Answer, error) {
	tx, err := db.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}

	updateQuery := `
		UPDATE answers a
		SET approved = true,
			approved_by = $2
		WHERE a.id = $1
	`

	_, err = tx.Exec(ctx, updateQuery, id, by)
	if err != nil {
		return nil, fmt.Errorf("updating approved answer: %w", err)
	}

	answer, err := getAnswer(tx, ctx, id)
	if err != nil {
		err2 := tx.Rollback(context.Background())
		log.Error().Err(err2).Msg("rolling back approved answer")
		return nil, fmt.Errorf("fetching approved answer: %w", err)
	}

	if err = tx.Commit(context.Background()); err != nil {
		return nil, fmt.Errorf("committing approve transaction: %w", err)
	}

	log.Debug().Object("question", answer).Msg("approved answer")

	return answer, nil
}
