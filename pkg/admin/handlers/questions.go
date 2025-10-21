package handlers

import (
	"math"
	"net/http"

	"github.com/htwr-aachen/backend/components/qa"
	"github.com/htwr-aachen/backend/pkg/qa/schema"
	"github.com/rs/zerolog/log"
)

func (h *AdminHandler) Questions(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	approved := r.URL.Query().Get(QUERY_APPROVED) != ""

	var questions []schema.Question
	var err error

	if approved {
		log.Trace().Msg("admin fetching approved questions")
		questions, err = h.qadb.ListApprovedQuestions(ctx, math.MaxInt32-1, 50)
	} else {
		log.Trace().Msg("admin fetching unapproved questions")
		questions, err = h.qadb.ListUnapprovedQuestions(ctx, math.MaxInt32-1, 50)
	}

	if err != nil {
		log.Err(err).Msg("admin querying questions")
		http.Error(w, "something went wrong", http.StatusInternalServerError)
		return
	}

	vms := make([]qa.QuestionViewModel, len(questions))
	for i, q := range questions {
		log.Debug().Bool("approved", q.Approved).Str("title", q.Title).Uint32("id", q.Id).Msg("fetchted question")
		vms[i] = ToQuestionViewModel(q)
	}
	log.Debug().Msg("fetched questions")

	questionListComponent := qa.QuestionList(vms)
	page := questionListComponent

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	err = page.Render(ctx, w)
	if err != nil {
		log.Err(err).Msg("Error rendering template")
		http.Error(w, "Error rendering template", http.StatusInternalServerError)
		return
	}

}
