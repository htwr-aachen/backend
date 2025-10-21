package session

import (
	"context"
	"fmt"
	"time"

	"github.com/htwr-aachen/backend/internal/database"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/patrickmn/go-cache"
)

type DB struct {
	sql   *pgxpool.Pool
	cache *cache.Cache
}

func newSessionDB(ctx context.Context, config SessionConfig) (*DB, error) {

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
		sql:   db,
		cache: cache.New(config.CacheExpiration, config.CacheCleanupInterval),
	}, nil
}
