package handlers

import (
	"net/http"
	"strconv"

	"github.com/htwr-aachen/backend/components/qa"
	"github.com/rs/zerolog/log"
)

func (h *AdminHandler) Answers(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	approved := r.URL.Query().Get("approved") != "false"
	orderBy := r.URL.Query().Get("order")
	limitStr := r.URL.Query().Get("limit")
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 {
		limit = 20
	}

	questions, err := h.qadb.ListQuestionsWithAnswers(ctx, approved, orderBy, limit)
	if err != nil {
		log.Err(err).Msg("fetching questions with answers")
		http.Error(w, "Failed to list questions with answers", http.StatusInternalServerError)
		return
	}

	vms := make([]qa.QuestionViewModel, len(questions))
	for i, q := range questions {
		log.Debug().Bool("approved", q.Approved).Str("title", q.Title).Uint32("id", q.Id).Msg("fetchted question")
		vms[i] = ToQuestionViewModel(q)
	}
	log.Debug().Msg("fetched answers")

	res := qa.QuestionList(vms)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	err = res.Render(ctx, w)
	if err != nil {
		log.Err(err).Msg("Error rendering template")
		http.Error(w, "Error rendering template", http.StatusInternalServerError)
		return
	}
}
