package handlers

import (
	"net/http"
	"os"

	"github.com/htwr-aachen/backend/pkg/admin/assets"
	"github.com/rs/zerolog/log"
)

type AdminAssetsHandler struct {
	isDevelopment bool
	fileServer    http.Handler
}

func NewAssets() *AdminAssetsHandler {
	h := &AdminAssetsHandler{}

	h.isDevelopment = os.Getenv("GO_ENV") != "production"
	if h.isDevelopment {
		log.Info().Msg("using directory asset fs")
		h.fileServer = assets.AssetFS
	} else {
		log.Info().Msg("using embedded asset fs")
		h.fileServer = assets.EmbeddedFS
	}

	return h
}

func (h *AdminAssetsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h.isDevelopment {
		w.Header().Set("Cache-Control", "no-store")
	}

	log.Trace().Str("path", r.URL.String()).Msg("requesting asset")

	h.fileServer.ServeHTTP(w, r)
}
