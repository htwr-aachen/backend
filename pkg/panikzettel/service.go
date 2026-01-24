package panikzettel

import (
	"context"
	"fmt"
	"net/http"

	"github.com/htwr-aachen/backend/internal/configurator"
	"github.com/htwr-aachen/backend/internal/httputils"
	"github.com/htwr-aachen/backend/internal/metrics"
	"github.com/htwr-aachen/backend/pkg/config"
	"github.com/htwr-aachen/backend/pkg/panikzettel/cloud"
	"github.com/htwr-aachen/backend/pkg/panikzettel/handlers"
	"github.com/htwr-aachen/backend/pkg/panikzettel/service"
	"github.com/rs/cors"
	"github.com/rs/zerolog/log"
	"github.com/slok/go-http-metrics/middleware"
	middlewarestd "github.com/slok/go-http-metrics/middleware/std"
	"gocloud.dev/blob"
)

type CloudClient interface {
	Bucket() *blob.Bucket
	Close()
}

func Init(ctx context.Context) (http.Handler, func(), error) {
	var err error
	gcfg, ok := configurator.FromContext(ctx)
	if !ok {
		log.Panic().Stack().Msg("no configuration context given")

	}

	cfg := &gcfg.Panikzettel

	var cloudClient CloudClient
	switch cfg.CloudConfig.Provider {
	case config.CloudProviderGoogle:
		cloudClient, err = cloud.NewGCP(ctx, gcfg)
	case config.CloudProviderAWS:
		cloudClient, err = cloud.NewAWS(ctx, gcfg)
	}
	if err != nil {
		return nil, nil, fmt.Errorf("failed setting up cloud connection: %w", err)
	}

	db := service.New(gcfg, cloudClient.Bucket())
	panikHandler := handlers.NewPanikzettel(db, cfg)
	r := http.NewServeMux()
	r.HandleFunc("GET /", panikHandler.GetPanikzettelMeta)
	r.HandleFunc("GET /{filename}", panikHandler.GetPanikzettel)

	var handler http.Handler
	handler = r
	if recorder, ok := metrics.FromContext(ctx); gcfg.Global.Metrics.Enabled && cfg.Metrics.Enabled && ok {
		handler = middlewarestd.Handler("/api/panikzettel", middleware.New(
			middleware.Config{
				Recorder: recorder,
				Service:  "htwr-panikzettel",
			}), handler)
	} else if gcfg.Global.Metrics.Enabled && cfg.Metrics.Enabled && !ok {
		log.Error().Str("subsystem", "panikzettel").Msg("retrieving metrics recorder from context")

	}

	if gcfg.Global.InsecureDev {
		c := cors.AllowAll()
		handler = c.Handler(handler)
	}
	handler = httputils.LogMiddleware(handler)

	closer := func() {
		cloudClient.Close()
	}
	return handler, closer, nil
}
