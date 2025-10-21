package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/htwr-aachen/backend/internal/httputils"
	"github.com/htwr-aachen/backend/internal/validation"
	"github.com/htwr-aachen/backend/pkg/qa/dto"
	"github.com/rs/zerolog/log"
)

func (h *Handler) NewAnswer(w http.ResponseWriter, r *http.Request) {

	if r.Method != "POST" {
		log.Panic().Msg("request should only allow post requests")
		http.Error(w, "only method post allowed", http.StatusMethodNotAllowed)
		return
	}

	questionId, err := httputils.PathId(r, "qid")
	if err != nil {
		log.Err(err).Msg("parsing question id")
		http.Error(w, "Could not parse question id", http.StatusBadRequest)
		return
	}

	var rDTO dto.NewAnswer
	err = json.NewDecoder(r.Body).Decode(&rDTO)

	if err != nil {
		log.Err(err).Msg("parsing dto")
		http.Error(w, "dto not parsable", http.StatusBadRequest)
		return
	}

	if err := validation.Validate.Struct(rDTO); err != nil {
		http.Error(w, fmt.Sprintf("dto validation failed %v", err.Error()), http.StatusBadRequest)
		return
	}

	rDTO.Answer = validation.PreRenderSanitization(rDTO.Answer)

	questionExists, err := h.db.QuestionExists(r.Context(), questionId)
	if err != nil {
		log.Err(err).Msg("checking if question exists")
		http.Error(w, "Could not verify question exist assumptions", http.StatusNotFound)
		return
	}

	if !questionExists {
		log.Info().Str("title", rDTO.Answer).Uint32("question_id", questionId).Msg("failing question exist check")
		http.Error(w, "Could not verify question exist assumptions", http.StatusInternalServerError)
		return
	}

	inserted, err := h.db.InsertAnswer(r.Context(), questionId, rDTO.Answer, rDTO.KnownSince)

	if err != nil {
		log.Err(err).Msg("inserting answer failed")
		http.Error(w, "could not add answer", http.StatusInternalServerError)
		return
	}

	wDTO := dto.NewAnswerFromSchema(inserted)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	err = json.NewEncoder(w).Encode(wDTO)
	if err != nil {
		log.Err(err).Msg("answer serialization failed")
		http.Error(w, "could not serialize dto", http.StatusInternalServerError)
		return
	}
}
