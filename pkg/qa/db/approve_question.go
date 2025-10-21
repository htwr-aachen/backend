package db

import (
	"context"
	"fmt"

	"github.com/htwr-aachen/backend/pkg/qa/schema"
	"github.com/jackc/pgx/v5"
	"github.com/rs/zerolog/log"
)

func (db *DB) ApproveQuestion(ctx context.Context, id uint32, by string) (*schema.Question, error) {
	tx, err := db.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}

	updateQuery := `
		UPDATE questions q
		SET approved = true,
			approved_by = $2
		WHERE q.id = $1
	`

	_, err = tx.Exec(ctx, updateQuery, id, by)
	if err != nil {
		return nil, fmt.Errorf("updating approved question: %w", err)
	}

	question, err := getQuestion(tx, ctx, id)
	if err != nil {
		return nil, fmt.Errorf("fetching approved question: %w", err)
	}

	if err = tx.Commit(context.Background()); err != nil {
		return nil, fmt.Errorf("committing approve transaction: %w", err)
	}

	log.Debug().Object("question", question).Msg("approved question")

	return question, nil
}
