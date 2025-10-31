package metrics

import (
	"context"
	"fmt"

	"github.com/htwr-aachen/backend/internal/configurator"
	"github.com/rs/zerolog/log"
	"github.com/slok/go-http-metrics/metrics"
	promMetrics "github.com/slok/go-http-metrics/metrics/prometheus"
	"github.com/spf13/viper"
)

type contextKey struct{}

func CreateAndAttach(parent context.Context, conf *viper.Viper) (context.Context, error) {
	if conf == nil {
		log.Panic().Stack().Msg("nil conf given")
	}

	globalCfg, ok := configurator.FromContext(parent)
	if !ok {
		return parent, fmt.Errorf("loading global metrics configuration")
	}

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
