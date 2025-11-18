package database

import (
	"context"
	"fmt"

	"github.com/htwr-aachen/backend/internal/configurator"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog/log"
)

type contextKey struct{}

func CreateAndAttach(parent context.Context) (context.Context, error) {

	gcfg, ok := configurator.FromContext(parent)
	if !ok {
		log.Fatal().Stack().Msg("no configuration context")
	}

	cfg := &gcfg.Database

	pool, err := New(cfg)
	if err != nil {
		log.Info().Err(err).Msg("creating database connection pool")
		return parent, fmt.Errorf("creating database pool: %w", err)
	}

	return context.WithValue(parent, contextKey{}, pool), nil
}

func FromContext(ctx context.Context) (*pgxpool.Pool, bool) {
	pool, ok := ctx.Value(contextKey{}).(*pgxpool.Pool)
	return pool, ok
}
