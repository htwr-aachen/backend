package database

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

type contextKey struct{}

func CreateAndAttach(parent context.Context, conf *viper.Viper) (context.Context, error) {
	if conf == nil {
		log.Panic().Stack().Msg("nil conf given")
	}
	config, err := LoadDBConfig(parent, conf)
	if err != nil {
		return parent, fmt.Errorf("loading database configuration: %w", err)
	}

	pool, err := New(config)
	if err != nil {
		return parent, fmt.Errorf("creating database pool: %w", err)
	}

	return context.WithValue(parent, contextKey{}, pool), nil
}

func FromContext(ctx context.Context) (*pgxpool.Pool, bool) {
	pool, ok := ctx.Value(contextKey{}).(*pgxpool.Pool)
	return pool, ok
}
