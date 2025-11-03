package handlers

import (
	"net/http"

	"github.com/htwr-aachen/backend/internal/httputils"
	"github.com/rs/zerolog/log"
)

func (h *Handler) RequestDeleteQuestion(w http.ResponseWriter, r *http.Request) {

	if r.Method != "DELETE" {
		http.Error(w, "only method DELETE allowed", http.StatusMethodNotAllowed)
		return
	}

	id, err := httputils.PathId(r, "id")
	if err != nil {
		log.Error().Err(err).Msg("parsing question id on deletion")
		http.Error(w, "question id not parsable", http.StatusBadRequest)
		return
	}

	reason := r.URL.Query().Get("reason")
	_, err = h.db.RequestQuestionDeletion(r.Context(), id, reason)
	if err != nil {
		log.Error().Err(err).Uint32("questionId", id).Msg("requesting deletion")
		http.Error(w, "something went wrong", http.StatusInternalServerError)
		return
	}

	_, err = w.Write([]byte{})
	if err != nil {
		log.Err(err).Uint32("answerId", id).Msg("deletion request reply failed")
	}

}
