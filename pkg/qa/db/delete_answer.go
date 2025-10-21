package db

import (
	"context"
	"fmt"

	"github.com/htwr-aachen/backend/pkg/qa/schema"
	"github.com/jackc/pgx/v5"
	"github.com/rs/zerolog/log"
)

// DeleteAnswer removes an answer from the db. Admin only
func (db *DB) DeleteAnswer(ctx context.Context, id uint32) error {

	tx, err := db.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}

	query := `
		DELETE FROM answers a
		WHERE a.id = $1
	`
	_, err = tx.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("deleting answer: %w", err)
	}

	if err = tx.Commit(context.Background()); err != nil {
		return fmt.Errorf("committing approve transaction: %w", err)
	}

	log.Debug().Uint32("id", id).Msg("deleted answer from db")

	return nil
}

// RequestAnswerDeletion insert a new answer_deletion_request for answer id
func (db *DB) RequestAnswerDeletion(ctx context.Context, answerId uint32, reason string) (*schema.DeletionRequest, error) {
	return db.insertDeletionRequest(ctx, answerId, schema.DELETION_REQUEST_TYPE_ANSWER, reason)
}
