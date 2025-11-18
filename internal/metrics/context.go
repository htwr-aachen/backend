package metrics

import (
	"context"

	"github.com/htwr-aachen/backend/internal/configurator"
	"github.com/rs/zerolog/log"
	"github.com/slok/go-http-metrics/metrics"
	promMetrics "github.com/slok/go-http-metrics/metrics/prometheus"
)

type contextKey struct{}

func CreateAndAttach(parent context.Context) (context.Context, error) {

	cfg, ok := configurator.FromContext(parent)
	if !ok {
		log.Fatal().Stack().Msg("no configuration context")
	}

	globalCfg := cfg.Global

	recorder := promMetrics.NewRecorder(promMetrics.Config{
		Prefix: globalCfg.Metrics.Prefix,
	})
	return WithRecorder(parent, recorder), nil
}

func WithRecorder(ctx context.Context, recorder metrics.Recorder) context.Context {
	return context.WithValue(ctx, contextKey{}, recorder)
}

func FromContext(ctx context.Context) (metrics.Recorder, bool) {
	recorder, ok := ctx.Value(contextKey{}).(metrics.Recorder)
	return recorder, ok
}
