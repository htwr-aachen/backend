package qa

import (
	"context"
	"fmt"
	"net/http"

	"github.com/htwr-aachen/backend/internal/httputils"
	"github.com/htwr-aachen/backend/internal/metrics"
	"github.com/htwr-aachen/backend/pkg/qa/config"
	"github.com/htwr-aachen/backend/pkg/qa/db"
	"github.com/htwr-aachen/backend/pkg/qa/handlers"
	"github.com/rs/cors"
	"github.com/rs/zerolog/log"
	"github.com/slok/go-http-metrics/middleware"
	middlewarestd "github.com/slok/go-http-metrics/middleware/std"
	"github.com/spf13/viper"
)

type service struct {
	db *db.DB
}

func Init(ctx context.Context, conf *viper.Viper) (http.Handler, func(), error) {
	if conf == nil {
		log.Panic().Stack().Msg("nil conf given")
	}

	var service service

	log.Debug().Msg("Loading QA Config")

	cfg, err := config.Load(ctx, conf)
	if err != nil {
		return nil, nil, fmt.Errorf("loading qa config: %w", err)
	}

	service.db, err = db.New(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("initializing qa service qa db: %w", err)
	}

	r := http.NewServeMux()

	//questions
	public, err := handlers.New(&cfg.APIConfig, service.db)
	if err != nil {
		log.Err(err).Msg("setting up public qa handler")
		return nil, nil, fmt.Errorf("configuring qa http handler: %w", err)
	}
	r.HandleFunc("GET /questions", public.GetQuestions)
	r.HandleFunc("POST /questions", public.NewQuestion)
	r.HandleFunc("POST /questions/{qid}/answers", public.NewAnswer)

	// TODO: delete requests

	// TODO: answer scrolling

	var handler http.Handler
	handler = r
	if recorder, ok := metrics.FromContext(ctx); cfg.GlobalConfig.Metrics.Enabled && cfg.Metrics.Enabled && ok {
		handler = middlewarestd.Handler("/api/qa", middleware.New(
			middleware.Config{
				Recorder: recorder,
				Service:  "htwr-qa",
			}), handler)
	} else if cfg.GlobalConfig.Metrics.Enabled && cfg.Metrics.Enabled && !ok {
		log.Error().Str("subsystem", "panikzettel").Msg("retrieving metrics recorder from context")

	}
	if cfg.GlobalConfig.InsecureDev {
		c := cors.AllowAll()
		handler = c.Handler(handler)
	}
	handler = httputils.LogMiddleware(handler)

	return handler, service.Close, nil
}

func (s *service) Close() {
}
