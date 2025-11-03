package configurator

import (
	"context"
	"fmt"

	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

type Global struct {
	TLS         GlobalTLSConfig `mapstructure:"tls"`
	InsecureDev bool            `mapstructure:"insecure_dev"`
	Metrics     MetricsConfig   `mapstructure:"metrics"`
}

type contextKey struct{}

func setDefaults(conf *viper.Viper) {
	conf.SetDefault("metrics.prefix", "")
	conf.SetDefault("metrics.enabled", true)
}

func LoadAndAttach(parent context.Context, parentConf *viper.Viper) (context.Context, error) {
	conf := parentConf.Sub("global")

	if conf == nil {
		log.Warn().Msg("creating new viper")
		conf = viper.New()
	}

	setDefaults(conf)

	config := Global{}
	if err := UnmarshalWithFileResolution(conf, &config); err != nil {
		log.Err(err).Msg("unmarshaling session configuration")
		return nil, fmt.Errorf("unmarshaling config: %w", err)
	}

	if config.InsecureDev {
		log.Warn().Msg("insecure dev mode")
	}

	return context.WithValue(parent, contextKey{}, &config), nil
}

func FromContext(ctx context.Context) (*Global, bool) {
	globalCfg, ok := ctx.Value(contextKey{}).(*Global)
	return globalCfg, ok
}
