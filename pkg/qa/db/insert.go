package db

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/htwr-aachen/backend/pkg/qa/schema"
	"github.com/jackc/pgx/v5"
	"github.com/rs/zerolog/log"
)

// InsertAnswer inserts a new unapproved answer to a question into the db
func (db *DB) InsertAnswer(ctx context.Context, questionID uint32, answer string, known_since time.Time) (*schema.Answer, error) {

	tx, err := db.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, fmt.Errorf("begin insert transaction: %w", err)
	}

	inserted := schema.Answer{
		QuestionId: questionID,
		Answer:     answer,
		KnownSince: known_since,
		CreatedAt:  time.Now(),
	}

	query := `
		INSERT INTO answers (question_id, approved, answer, answer_rendered, render_version, known_since, created_at)
		VALUES ($1, FALSE, $2, $2, 0, $3, $4)
		RETURNING id;
	`
	var insertedId uint32
	err = tx.QueryRow(ctx, query, inserted.QuestionId, inserted.Answer, inserted.KnownSince, inserted.CreatedAt).Scan(&insertedId)
	if err != nil {
		return nil, fmt.Errorf("inserting answer: %w", err)
	}

	inserted.Id = uint32(insertedId)

	if err = tx.Commit(context.Background()); err != nil {
		return nil, fmt.Errorf("committing approve transaction: %w", err)
	}

	log.Debug().Object("answers", inserted).Msg("inserted new answer into db")

	return &inserted, nil
}

// InsertQuestion inserts a new unapproved question into the db
func (db *DB) InsertQuestion(ctx context.Context, title string, description string) (*schema.Question, error) {

	tx, err := db.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, fmt.Errorf("begin insert transaction: %w", err)
	}

	createdTime := time.Now()

	query := `
		INSERT INTO questions (approved, title, description, description_rendered, render_version, created_at)
		VALUES (FALSE, $1, $2, $2, 0, $3)
		RETURNING id;
	`
	var insertedID uint32
	err = tx.QueryRow(ctx, query, title, description, createdTime).Scan(&insertedID)
	if err != nil {
		return nil, fmt.Errorf("inserting question: %w", err)
	}

	inserted := schema.Question{
		Id:       insertedID,
		Approved: false,
		Title:    title,
		Description: sql.NullString{
			String: description,
			Valid:  description != "",
		},
		CreatedAt: createdTime,
	}

	if err = tx.Commit(context.Background()); err != nil {
		return nil, fmt.Errorf("committing approve transaction: %w", err)
	}

	log.Debug().Object("question", inserted).Msg("inserted new question into db")

	return &inserted, nil
}
