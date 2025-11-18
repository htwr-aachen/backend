package handlers

import (
	"github.com/htwr-aachen/backend/pkg/config"
	"github.com/htwr-aachen/backend/pkg/qa/db"
)

type Handler struct {
	db     *db.DB
	config *config.QAAPI
}

func New(config *config.QAAPI, db *db.DB) (*Handler, error) {
	return &Handler{
		db:     db,
		config: config,
	}, nil
}
