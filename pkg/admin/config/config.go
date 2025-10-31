package config

import (
	"context"
	"fmt"

	"github.com/htwr-aachen/backend/internal/configurator"
	"github.com/htwr-aachen/backend/internal/validation"
	"github.com/spf13/viper"
)

type Admin struct {
	GlobalConfig *configurator.Global       `mapstructure:"-"`
	Metrics      configurator.MetricsConfig `mapstructure:"metrics"`
}

func setDefaults(conf *viper.Viper) {
	conf.SetDefault("metrics.enabled", true)
}

func LoadConfig(ctx context.Context, parentConf *viper.Viper) (*Admin, error) {
	conf := parentConf.Sub("admin")
	if conf == nil {
		conf = viper.New()
	}

	setDefaults(conf)

	var config Admin
	err := configurator.UnmarshalWithFileResolution(conf, &config)
	if err != nil {
		return nil, fmt.Errorf("failed unmashaling admin config from conf: %w", err)
	}

	var ok bool
	if config.GlobalConfig, ok = configurator.FromContext(ctx); !ok {
		return nil, fmt.Errorf("no global config in context")
	}

	err = validation.Validate.Struct(config)
	if err != nil {
		return nil, fmt.Errorf("validating admin configuration: %w", err)
	}
	return &config, nil
}
