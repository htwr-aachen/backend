package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/htwr-aachen/backend/internal/validation"
	"github.com/htwr-aachen/backend/pkg/qa/dto"
	"github.com/rs/zerolog/log"
)

func (h *Handler) NewQuestion(w http.ResponseWriter, r *http.Request) {

	if r.Method != "POST" {
		http.Error(w, "only method post allowed", http.StatusMethodNotAllowed)
		return
	}

	var rDTO dto.NewQuestion
	err := json.NewDecoder(r.Body).Decode(&rDTO)

	if err != nil {
		http.Error(w, "dto not parsable", http.StatusBadRequest)
		return
	}

	if err := validation.Validate.Struct(rDTO); err != nil {
		http.Error(w, fmt.Sprintf("dto validation failed %v", err.Error()), http.StatusBadRequest)
		return
	}

	rDTO.Title = validation.StrictSanitize(rDTO.Title)
	rDTO.Description = validation.PreRenderSanitization(rDTO.Description)

	inserted, err := h.db.InsertQuestion(r.Context(), rDTO.Title, rDTO.Description)

	if err != nil {
		log.Err(err).Msg("inserting question failed")
		http.Error(w, "could not add question", http.StatusInternalServerError)
		return
	}

	wDTO := dto.NewQuestionFromSchema(inserted)

	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(wDTO)
	if err != nil {
		log.Err(err).Msg("question serialization failed")
		http.Error(w, "could not serialize dto", http.StatusInternalServerError)
	}

}
