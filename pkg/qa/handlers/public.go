package handlers

import (
	"github.com/htwr-aachen/backend/pkg/qa/config"
	"github.com/htwr-aachen/backend/pkg/qa/db"
)

type Handler struct {
	db     *db.DB
	config *config.APIConfig
}

func New(config *config.APIConfig, db *db.DB) (*Handler, error) {
	return &Handler{
		db:     db,
		config: config,
	}, nil
}
