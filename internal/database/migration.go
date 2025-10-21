package database

import (
	"database/sql"
	"embed"
	"fmt"

	"github.com/pressly/goose/v3"
	"github.com/rs/zerolog/log"
)

//go:embed migrations/*.sql
var embedMigrations embed.FS

func migrate(db *sql.DB) error {
	goose.SetBaseFS(embedMigrations)

	err := goose.SetDialect(string(goose.DialectPostgres))
	if err != nil {
		log.Err(err).Msg("Setting migration dialect to postgres")
		return fmt.Errorf("setting migration dialect to postgres: %w", err)
	}

	err = goose.Up(db, "migrations")
	if err != nil {
		log.Err(err).Msg("migrating db")
		return fmt.Errorf("migrating db: %w", err)
	}

	return nil
}
