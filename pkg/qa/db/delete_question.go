package db

import (
	"context"
	"fmt"

	"github.com/htwr-aachen/backend/pkg/qa/schema"
	"github.com/jackc/pgx/v5"
)

// DeleteQuestion removes a question from the db
func (db *DB) DeleteQuestion(ctx context.Context, id uint32) error {

	tx, err := db.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}

	query := `
		DELETE FROM questions q
		WHERE q.id = $1
	`
	_, err = tx.Exec(context.Background(), query, id)
	if err != nil {
		return fmt.Errorf("deleting question: %w", err)
	}

	if err = tx.Commit(context.Background()); err != nil {
		return fmt.Errorf("committing approve transaction: %w", err)
	}
	return nil
}

// RequestAnswerDeletion insert a new question_deletion_request for question {id}
func (db *DB) RequestQuestionDeletion(ctx context.Context, questionId uint32, reason string) (*schema.DeletionRequest, error) {
	return db.insertDeletionRequest(ctx, questionId, schema.DELETION_REQUEST_TYPE_QUESTION, reason)
}
