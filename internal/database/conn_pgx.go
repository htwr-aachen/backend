package database

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/lib/pq"
	"github.com/rs/zerolog/log"
)

var pool *pgxpool.Pool

func New(config *DBConfig) (*pgxpool.Pool, error) {
	log.Info().Msg("Connecting to postgresql")
	var err error

	poolConfig, err := pgxpool.ParseConfig(config.DBConnStr)
	if err != nil {
		log.Err(err).Msg("parsing postgres config")
	}

	if config.dbSetConfig {
		poolConfig.MaxConns = config.DBMaxConns
		poolConfig.MaxConnLifetime = config.DBConnMaxLifetime
		poolConfig.MinConns = config.DBMinConns
		poolConfig.HealthCheckPeriod = config.DBConnHealthCheckPeriod
	}

	pool, err := pgxpool.NewWithConfig(context.Background(), poolConfig)
	if err != nil {
		if err, ok := err.(*pq.Error); !ok {
			log.Err(err).Msg("Connection to db failed")
		} else {
			log.Err(err).Str("error code", err.Code.Name()).Msg("Connection to db failed")
		}
		return nil, fmt.Errorf("db startup error: %w", err)
	}

	log.Info().Msg("DB connection established")

	ctx, cancel := context.WithTimeout(context.Background(), config.DBTimeout)
	err = pool.Ping(ctx)
	cancel()
	if err != nil {
		log.Err(err).Msg("Database connection not health")
		return nil, err
	}

	log.Info().Msg("DB applying migrations")
	err = migrate(stdlib.OpenDBFromPool(pool))
	if err != nil {
		return nil, err
	}
	log.Info().Msg("Migrations finished successfully")

	return pool, nil
}
func Get() *pgxpool.Pool {
	return pool
}

func Close() {
	if pool != nil {
		pool.Close()
	}
}
