package handlers

import (
	"context"
	"net/http"

	"github.com/htwr-aachen/backend/internal/httputils"
	"github.com/htwr-aachen/backend/internal/metrics"
	"github.com/htwr-aachen/backend/pkg/admin/config"
	qaschema "github.com/htwr-aachen/backend/pkg/qa/schema"
	"github.com/htwr-aachen/backend/pkg/schema"
	"github.com/rs/zerolog/log"
	"github.com/slok/go-http-metrics/middleware"
	middlewarestd "github.com/slok/go-http-metrics/middleware/std"
)

type QADB interface {
	ApproveQuestion(ctx context.Context, id uint32, by string) (*qaschema.Question, error)
	ApproveAnswer(ctx context.Context, id uint32, by string) (*qaschema.Answer, error)
	ApproveDeletionRequest(ctx context.Context, deletionRequestId uint32) error

	DeleteQuestion(ctx context.Context, id uint32) error
	DeleteAnswer(ctx context.Context, id uint32) error
	RequestQuestionDeletion(ctx context.Context, id uint32, reason string) (*qaschema.DeletionRequest, error)
	RequestAnswerDeletion(ctx context.Context, id uint32, reason string) (*qaschema.DeletionRequest, error)

	ListApprovedAnswers(ctx context.Context, questionId uint32, beforeId uint32, limit int) (map[uint32][]qaschema.Answer, error)
	ListUnapprovedAnswers(ctx context.Context, questionId uint32, beforeId uint32, limit int) (map[uint32][]qaschema.Answer, error)
	ListAnswers(ctx context.Context, questionIDs []uint32, approvedFilter qaschema.ApprovedFilter, beforeId uint32, limit int) (map[uint32][]qaschema.Answer, error)

	ListApprovedQuestions(ctx context.Context, beforeId uint32, limit int) ([]qaschema.Question, error)
	ListUnapprovedQuestions(ctx context.Context, beforeId uint32, limit int) ([]qaschema.Question, error)
	ListQuestions(ctx context.Context, approvedFilter qaschema.ApprovedFilter, beforeId uint32, limit int) ([]qaschema.Question, error)
	ListQuestionsWithAnswers(ctx context.Context, approved bool, orderBy string, limit int) ([]qaschema.Question, error)

	GetAnswerDeletionRequests(ctx context.Context, id uint32) ([]qaschema.DeletionRequest, error)
	GetQuestionDeletionRequests(ctx context.Context, id uint32) ([]qaschema.DeletionRequest, error)

	GetQuestion(ctx context.Context, id uint32) (*qaschema.Question, error)
	GetAnswer(ctx context.Context, id uint32) (*qaschema.Answer, error)
}

// SessionProvider interface for the admin service
type SessionProvider interface {
	// Retrieves the authenticated user of this request. MUST return user only when authenticated and err != nil => Unauthenticated{} if unauthenticated
	RequestUser(r *http.Request) (*schema.User, error)
	AuthMiddleware(next http.Handler) http.Handler
}

type AdminHandler struct {
	sessions SessionProvider
	router   *http.ServeMux
	handler  http.Handler
	assets   http.Handler
	qadb     QADB
}

func New(ctx context.Context, cfg *config.Admin, qadb QADB, sessions SessionProvider) *AdminHandler {
	h := &AdminHandler{
		qadb:     qadb,
		sessions: sessions,
	}
	router := http.NewServeMux()

	assetsHandler := NewAssets()
	h.assets = assetsHandler

	router.HandleFunc("GET /", h.Landing) // done
	router.Handle("GET /assets/", http.StripPrefix("/assets", h.assets))
	router.HandleFunc("GET /home", h.Home) // done

	router.HandleFunc("GET /questions", h.Questions)
	// router.HandleFunc("GET /questions/{id}", h.Question)
	// router.HandleFunc("GET /questions/{qid}/answers", h.Answers)
	// router.HandleFunc("GET /answers/{id}", h.Answer)

	router.HandleFunc("GET /answers", h.Answers)
	// // question functionality
	// router.HandleFunc("PUT /questions/{id}/priority", h.UpdateQuestionPriority)
	router.HandleFunc("POST /questions/{id}/approve", h.ApproveQuestion)
	router.HandleFunc("POST /questions/{id}/reject", h.DeleteQuestion)
	router.HandleFunc("DELETE /questions/{id}", h.DeleteQuestion)
	//
	// // answer functionality
	// router.HandleFunc("PUT /answers/{id}/priority", h.UpdateAnswerPriority)
	router.HandleFunc("POST /answers/{id}/approve", h.ApproveAnswer)
	router.HandleFunc("POST /answers/{id}/reject", h.DeleteAnswer)
	router.HandleFunc("DELETE /answers/{id}", h.DeleteAnswer)
	//
	h.router = router

	var handler http.Handler
	handler = h.router
	if recorder, ok := metrics.FromContext(ctx); cfg.GlobalConfig.Metrics.Enabled && cfg.Metrics.Enabled && ok {
		handler = middlewarestd.Handler("/api/qa", middleware.New(
			middleware.Config{
				Recorder: recorder,
				Service:  "htwr-qa",
			}), handler)
	} else if cfg.GlobalConfig.Metrics.Enabled && cfg.Metrics.Enabled && !ok {
		log.Error().Str("subsystem", "panikzettel").Msg("retrieving metrics recorder from context")

	}

	handler = sessions.AuthMiddleware(handler)
	handler = httputils.LogMiddleware(handler)
	h.handler = handler
	return h
}

func (h *AdminHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.handler.ServeHTTP(w, r)
}
