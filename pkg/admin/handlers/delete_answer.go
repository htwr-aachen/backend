package handlers

import (
	"net/http"

	"github.com/htwr-aachen/backend/internal/httputils"
	"github.com/rs/zerolog/log"
)

func (h *AdminHandler) DeleteAnswer(w http.ResponseWriter, r *http.Request) {

	ctx := r.Context()

	id, err := httputils.PathId(r, "id")
	if err != nil {
		log.Err(err).Msg("converting answer id to uint32")
		http.Error(w, "invalid answer id", http.StatusBadRequest)
		return
	}

	err = h.qadb.DeleteAnswer(ctx, id)
	if err != nil {
		log.Err(err).Msg("deleting answer")
		http.Error(w, "could not delete", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
