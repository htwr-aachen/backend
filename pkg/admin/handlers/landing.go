package handlers

import (
	"net/http"

	"github.com/htwr-aachen/backend/components/qa"
	"github.com/htwr-aachen/backend/pkg/admin/layouts"
	"github.com/rs/zerolog/log"
)

func (h *AdminHandler) Landing(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	landingComponent := qa.Landing()
	page := layouts.QAAdminLayout(landingComponent)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	err := page.Render(ctx, w)
	if err != nil {
		log.Err(err).Msg("Error rendering template")
		http.Error(w, "Error rendering template", http.StatusInternalServerError)
		return
	}
}

func (h *AdminHandler) Home(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	landingComponent := qa.Landing()
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	err := landingComponent.Render(ctx, w)
	if err != nil {
		log.Err(err).Msg("Error rendering template")
		http.Error(w, "Error rendering template", http.StatusInternalServerError)
		return
	}
}
