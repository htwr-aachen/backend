package handlers

import (
	"net/http"

	"github.com/htwr-aachen/backend/internal/httputils"
	"github.com/rs/zerolog/log"
)

func (h *Handler) RequestDeleteAnswer(w http.ResponseWriter, r *http.Request) {

	if r.Method != "DELETE" {
		http.Error(w, "only method DELETE allowed", http.StatusMethodNotAllowed)
		return
	}

	id, err := httputils.PathId(r, "id")
	if err != nil {
		log.Err(err).Msg("parsing answer id on deletion")
		http.Error(w, "answer id not parsable", http.StatusBadRequest)
		return
	}

	reason := r.URL.Query().Get("reason")
	_, err = h.db.RequestQuestionDeletion(r.Context(), id, reason)
	if err != nil {
		log.Err(err).Uint32("answerId", id).Msg("requesting deletion")
		http.Error(w, "something went wrong", http.StatusInternalServerError)
		return
	}

	_, err = w.Write([]byte{})
	if err != nil {
		log.Err(err).Uint32("answerId", id).Msg("deletion request reply failed")
	}
}
