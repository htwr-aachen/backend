package db

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/htwr-aachen/backend/pkg/qa/schema"
	"github.com/jackc/pgx/v5"
	"github.com/rs/zerolog/log"
)

func (db *DB) insertDeletionRequest(ctx context.Context, entityId uint32, entityType string, reason string) (*schema.DeletionRequest, error) {
	tx, err := db.db.BeginTx(ctx, pgx.TxOptions{})

	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}

	inserted := schema.DeletionRequest{
		EntityId:   entityId,
		EntityType: entityType,
		Reason:     reason,
	}

	query := `
	INSERT INTO deletion_requests (entity_id, entity_type, reason)
	VALUES ($1, $2, $3)
	RETURNING (id, status, created_at);
`

	err = tx.QueryRow(ctx, query, inserted.EntityId, inserted.EntityType, inserted.Reason).Scan(&inserted.Id, &inserted.Status, &inserted.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("executing deletion request insertion: %w", err)
	}

	if err = tx.Commit(context.Background()); err != nil {
		return nil, fmt.Errorf("committing question deletion request: %w", err)
	}

	log.Trace().Object("deletion_request", inserted).Msg("inserted deletion_request into db")

	return &inserted, nil

}

// GetDeletionRequests retrieves a specific deletion request by ID
func (db *DB) GetDeletionRequests(ctx context.Context, deletionRequestId uint32) (*schema.DeletionRequest, error) {
	query := `
		SELECT id, entity_type, entity_id, reason, status, reviewed_at, created_at
		FROM deletion_requests
		WHERE id = $1
	`

	var dr schema.DeletionRequest
	var reason sql.NullString
	err := db.db.QueryRow(ctx, query, deletionRequestId).Scan(
		&dr.Id,
		&dr.EntityType,
		&dr.EntityId,
		&reason,
		&dr.Status,
		&dr.ReviewedAt,
		&dr.CreatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, err
		}
		return nil, fmt.Errorf("getting deletion request: %w", err)
	}

	if reason.Valid {
		dr.Reason = reason.String
	}

	log.Trace().Object("deletion_request", dr).Msg("fetched deletion_request from db")

	return &dr, nil
}

// GetAnswerDeletionRequests retrieves all deletion requests for a specific answer
func (db *DB) GetAnswerDeletionRequests(ctx context.Context, answerId uint32) ([]schema.DeletionRequest, error) {
	query := `
		SELECT id, entity_type, entity_id, reason, status, reviewed_at, created_at
		FROM deletion_requests
		WHERE entity_type = $1 AND entity_id = $2
		ORDER BY created_at DESC
	`

	rows, err := db.db.Query(ctx, query, schema.DELETION_REQUEST_TYPE_ANSWER, answerId)
	if err != nil {
		return nil, fmt.Errorf("getting answer deletion requests: %w", err)
	}
	defer rows.Close()

	var requests []schema.DeletionRequest
	for rows.Next() {
		var dr schema.DeletionRequest
		var reason sql.NullString
		err := rows.Scan(
			&dr.Id,
			&dr.EntityType,
			&dr.EntityId,
			&reason,
			&dr.Status,
			&dr.ReviewedAt,
			&dr.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scanning deletion request: %w", err)
		}
		if reason.Valid {
			dr.Reason = reason.String
		}
		requests = append(requests, dr)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating deletion requests: %w", err)
	}

	log.Trace().Msg("fetched answer deletion_requests from db")

	return requests, nil
}

// GetQuestionDeletionRequests retrieves all deletion requests for a specific question
func (db *DB) GetQuestionDeletionRequests(ctx context.Context, questionId uint32) ([]schema.DeletionRequest, error) {
	query := `
		SELECT id, entity_type, entity_id, reason, status, reviewed_at, created_at
		FROM deletion_requests
		WHERE entity_type = $1 AND entity_id = $2
		ORDER BY created_at DESC
	`

	rows, err := db.db.Query(ctx, query, schema.DELETION_REQUEST_TYPE_QUESTION, questionId)
	if err != nil {
		return nil, fmt.Errorf("getting question deletion requests: %w", err)
	}
	defer rows.Close()

	var requests []schema.DeletionRequest
	for rows.Next() {
		var dr schema.DeletionRequest
		var reason sql.NullString
		err := rows.Scan(
			&dr.Id,
			&dr.EntityType,
			&dr.EntityId,
			&reason,
			&dr.Status,
			&dr.ReviewedAt,
			&dr.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scanning deletion request: %w", err)
		}
		if reason.Valid {
			dr.Reason = reason.String
		}
		requests = append(requests, dr)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating deletion requests: %w", err)
	}

	log.Trace().Msg("fetched question deletion_requests from db")

	return requests, nil
}

// ApproveDeletionRequest approves a deletion request and deletes the associated entity
func (db *DB) ApproveDeletionRequest(ctx context.Context, deletionRequestId uint32) error {
	tx, err := db.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("beginning transaction: %w", err)
	}

	// Get the deletion request details
	var entityType string
	var entityID uint32
	var status string

	query := `
		SELECT entity_type, entity_id, status
		FROM deletion_requests
		WHERE id = $1
	`
	err = tx.QueryRow(ctx, query, deletionRequestId).Scan(&entityType, &entityID, &status)
	if err != nil {
		if err == sql.ErrNoRows {
			return err
		}
		return fmt.Errorf("getting deletion request: %w", err)
	}

	// Check if already processed
	if status != schema.DELETION_REQUEST_STATUS_PENDING {
		return fmt.Errorf("deletion request already processed with status: %s", status)
	}

	// Update the deletion request status
	updateQuery := `
		UPDATE deletion_requests
		SET status = $1, reviewed_at = NOW()
		WHERE id = $2
	`
	_, err = tx.Exec(ctx, updateQuery, schema.DELETION_REQUEST_STATUS_APPROVED, deletionRequestId)
	if err != nil {
		return fmt.Errorf("updating deletion request: %w", err)
	}

	// Delete the entity (CASCADE will handle the deletion request)
	var deleteQuery string
	switch entityType {
	case schema.DELETION_REQUEST_TYPE_QUESTION:
		deleteQuery = "DELETE FROM questions WHERE id = $1"
	case schema.DELETION_REQUEST_TYPE_ANSWER:
		deleteQuery = "DELETE FROM answers WHERE id = $1"
	default:
		return fmt.Errorf("unknown entity type: %s", entityType)
	}

	_, err = tx.Exec(ctx, deleteQuery, entityID)
	if err != nil {
		return fmt.Errorf("deleting %s: %w", entityType, err)
	}

	if err = tx.Commit(context.Background()); err != nil {
		return fmt.Errorf("committing transaction: %w", err)
	}

	log.Debug().Uint32("deletion_request_id", deletionRequestId).Msg("approved deletion request")

	return nil
}
