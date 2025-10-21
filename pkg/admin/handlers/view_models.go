package handlers

import (
	"github.com/htwr-aachen/backend/components/qa"
	"github.com/htwr-aachen/backend/pkg/qa/schema"
)

func ToAnswerViewModel(a *schema.Answer) qa.AnswerViewModel {
	return qa.AnswerViewModel{
		Id:         a.Id,
		QuestionId: a.QuestionId,
		Answer:     a.Answer,
		Approved:   a.Approved,
		CreatedAt:  a.CreatedAt,
		KnownSince: a.KnownSince,
	}
}

func ToBareQuestionViewModel(q *schema.Question) qa.BareQuestionViewModel {
	description := ""
	if q.Description.Valid {
		description = q.Description.String
	}
	return qa.BareQuestionViewModel{
		Id:          q.Id,
		Approved:    q.Approved,
		Title:       q.Title,
		Description: description,
		CreatedAt:   q.CreatedAt,
	}
}
func ToQuestionViewModel(q schema.Question) qa.QuestionViewModel {

	vm := qa.QuestionViewModel{
		BareQuestionViewModel: ToBareQuestionViewModel(&q),
		Answers:               make([]qa.AnswerViewModel, len(q.Answers)),
	}

	// copying is fine for this little data
	for i, a := range q.Answers {
		vm.Answers[i] = ToAnswerViewModel(&a)
	}

	return vm
}
