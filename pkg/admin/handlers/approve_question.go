package handlers

import (
	"errors"
	"net/http"

	"github.com/htwr-aachen/backend/internal/httputils"
	"github.com/htwr-aachen/backend/pkg/schema"
	"github.com/rs/zerolog/log"
)

func (h *AdminHandler) ApproveQuestion(w http.ResponseWriter, r *http.Request) {
	user, err := h.sessions.RequestUser(r)
	if err != nil && errors.Is(err, schema.Unauthenticated{}) {
		log.Err(err).Msg("unauthenticated admin request")
		http.Error(w, "mandatory request authentication", http.StatusUnauthorized)
		return
	} else if err != nil {
		log.Err(err).Msg("evaluating request authentication")
		http.Error(w, "invalid request authentication", http.StatusUnauthorized)
		return
	}

	ctx := r.Context()

	id, err := httputils.PathId(r, "id")

	if err != nil {
		log.Err(err).Msg("converting question id to uint32")
		http.Error(w, "invalid question id", http.StatusBadRequest)
		return
	}

	log.Debug().Uint32("id", id).Msg("approving question")

	_, err = h.qadb.ApproveQuestion(ctx, id, user.Id)

	if err != nil {
		log.Err(err).Msg("approving question")
		http.Error(w, "could not approve", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
