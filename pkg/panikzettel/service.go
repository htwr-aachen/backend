package panikzettel

import (
	"context"
	"fmt"
	"net/http"

	"github.com/htwr-aachen/backend/internal/httputils"
	"github.com/htwr-aachen/backend/internal/metrics"
	"github.com/htwr-aachen/backend/pkg/panikzettel/cloud"
	"github.com/htwr-aachen/backend/pkg/panikzettel/config"
	"github.com/htwr-aachen/backend/pkg/panikzettel/handlers"
	"github.com/htwr-aachen/backend/pkg/panikzettel/service"
	"github.com/rs/cors"
	"github.com/rs/zerolog/log"
	"github.com/slok/go-http-metrics/middleware"
	middlewarestd "github.com/slok/go-http-metrics/middleware/std"
	"github.com/spf13/viper"
	"gocloud.dev/blob"
)

type CloudClient interface {
	Bucket() *blob.Bucket
	Close()
}

func Init(ctx context.Context, conf *viper.Viper) (http.Handler, func(), error) {
	if conf == nil {
		log.Panic().Stack().Msg("nil conf given")
	}

	cfg, err := config.LoadConfig(ctx, conf)
	if err != nil {
		log.Err(err).Msg("validating panikzettel service configuration")
		return nil, nil, fmt.Errorf("validating panikzettel service: %w", err)
	}

	var cloudClient CloudClient
	switch cfg.CloudConfig.Provider {
	case config.CloudProviderGoogle:
		cloudClient, err = cloud.NewGCP(ctx, cfg)
	case config.CloudProviderAWS:
		cloudClient, err = cloud.NewAWS(ctx, cfg.CloudConfig)
	}
	if err != nil {
		return nil, nil, fmt.Errorf("failed setting up cloud connection: %w", err)
	}

	db := service.New(cfg, cloudClient.Bucket())
	panikHandler := handlers.NewPanikzettel(db)
	r := http.NewServeMux()
	r.HandleFunc("GET /", panikHandler.GetPanikzettelMeta)
	r.HandleFunc("GET /{filename}", panikHandler.GetPanikzettel)

	var handler http.Handler
	handler = r
	if recorder, ok := metrics.FromContext(ctx); cfg.GlobalConfig.Metrics.Enabled && cfg.Metrics.Enabled && ok {
		handler = middlewarestd.Handler("/api/panikzettel", middleware.New(
			middleware.Config{
				Recorder: recorder,
				Service:  "htwr-panikzettel",
			}), handler)
	} else if cfg.GlobalConfig.Metrics.Enabled && cfg.Metrics.Enabled && !ok {
		log.Error().Str("subsystem", "panikzettel").Msg("retrieving metrics recorder from context")

	}

	if cfg.GlobalConfig.InsecureDev {
		c := cors.AllowAll()
		handler = c.Handler(handler)
	}
	handler = httputils.LogMiddleware(handler)

	closer := func() {
		cloudClient.Close()
	}
	return handler, closer, nil
}
