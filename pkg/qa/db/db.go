package db

import (
	"context"
	"fmt"
	"time"

	"github.com/htwr-aachen/backend/internal/database"
	"github.com/htwr-aachen/backend/pkg/qa/schema"
	"github.com/jackc/pgx/v5/pgxpool"
)

type DB struct {
	db *pgxpool.Pool
}

func New(ctx context.Context) (*DB, error) {
	db, ok := database.FromContext(ctx)
	if !ok {
		return nil, fmt.Errorf("could not get db from context most likely did not initialize correctly")
	}

	timeoutCtx, cancel := context.WithTimeout(context.Background(), time.Minute)
	err := db.Ping(timeoutCtx)
	cancel()

	if err != nil {
		return nil, fmt.Errorf("qa db ping failed: %w", err)
	}

	return &DB{
		db: db,
	}, nil
}

func validateApprovedFilter(filter schema.ApprovedFilter) error {
	if filter >= schema.APPROVED_FILTER_LEN {
		return fmt.Errorf("invalid filter %v >= APPROVED_FILTER_LEN", filter)
	}
	return nil
}
