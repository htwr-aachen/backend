package handlers

import (
	"net/http"

	"github.com/htwr-aachen/backend/internal/httputils"
	"github.com/rs/zerolog/log"
)

func (h *AdminHandler) DeleteQuestion(w http.ResponseWriter, r *http.Request) {

	ctx := r.Context()

	id, err := httputils.PathId(r, "id")
	if err != nil {
		log.Err(err).Msg("converting question id to int64")
		http.Error(w, "invalid question id", http.StatusBadRequest)
		return
	}

	err = h.qadb.DeleteQuestion(ctx, id)
	if err != nil {
		log.Err(err).Msg("deleting question")
		http.Error(w, "could not delete", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
