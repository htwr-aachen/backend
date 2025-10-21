package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/htwr-aachen/backend/internal/httputils"
	"github.com/htwr-aachen/backend/pkg/qa/dto"
	"github.com/htwr-aachen/backend/pkg/qa/schema"
	"github.com/rs/zerolog/log"
)

const QUERY_BEFORE_ID string = "before_id"
const QUERY_LIMIT string = "limit"
const QUERY_ANSWERED string = "answered"

func (h *Handler) GetQuestions(w http.ResponseWriter, r *http.Request) {
	var rDTO dto.GetQuestionsQueryDTO
	var err error

	rDTO.LastPriority, rDTO.LastId, rDTO.Limit, err = httputils.GetPaginationParams(r, false)
	if err != nil {
		log.Err(err).Msg("parsing pagination params")
		http.Error(w, "invalid offset or limit query given", http.StatusBadRequest)
		return
	}

	if rDTO.LastId == 0 {
		log.Debug().Msg("get questions without offset param")
		rDTO.LastId = h.config.PaginationOffsetDefault
	}

	if rDTO.Limit == 0 {
		log.Debug().Msg("get questions without limit param")
		rDTO.Limit = h.config.PaginationLimitDefault
	}

	// set max limit cap
	rDTO.Limit = min(h.config.PaginationLimitMax, rDTO.Limit)

	rDTO.Answered, err = strconv.ParseBool(r.URL.Query().Get(QUERY_ANSWERED))
	if err != nil {
		log.Err(err).Msg("invalid answered given")
		http.Error(w, "invalid answered query given", http.StatusBadRequest)
		return
	}

	var questions []schema.Question
	if rDTO.Answered {
		questions, err = h.db.ListAnsweredApprovedQuestions(r.Context(), rDTO.LastPriority, rDTO.LastId, rDTO.Limit)
		log.Trace().Uint32("before_id", rDTO.LastId).Int("limit", rDTO.Limit).Int("response_length", len(questions)).Msg("fetched answered approved questions")
	} else {
		questions, err = h.db.ListUnansweredApprovedQuestions(r.Context(), rDTO.LastPriority, rDTO.LastId, rDTO.Limit)
		log.Trace().Uint32("before_id", rDTO.LastId).Int("limit", rDTO.Limit).Int("response_length", len(questions)).Msg("fetched unanswered approved questions")
	}

	if err != nil {
		log.Err(err).Msg("querying questions")
		http.Error(w, "something went wrong", http.StatusInternalServerError)
		return
	}

	wDTO := dto.GetQuestionsQueryResDTO{
		Questions: dto.QuestionsFromSchemas(questions),
	}

	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(wDTO)
	if err != nil {
		log.Err(err).Msg("serializing questions")
		http.Error(w, "could not serialize dto", http.StatusInternalServerError)
	}
}
