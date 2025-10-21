package config

import (
	"context"
	"fmt"
	"math"

	"github.com/htwr-aachen/backend/internal/configurator"
	"github.com/htwr-aachen/backend/internal/validation"
	"github.com/spf13/viper"
)

type Config struct {
	GlobalConfig *configurator.Global `mapstructure:"-"`
	APIConfig    APIConfig            `mapstructure:"api"`
}

type APIConfig struct {
	PaginationLimitDefault int        `mapstructure:"pagination_limit_default"`
	PaginationLimitMax     int        `mapstructure:"pagination_limit_max"`
	CorsConfig             CorsConfig `mapstructure:"cors"`
}

type CorsConfig struct {
	AllowedOrigins   []string `mapstructure:"allowed_origins"`
	AllowedMethods   []string `mapstructure:"allowed_methods"`
	AllowedHeaders   []string `mapstructure:"allowed_headers"`
	AllowCredentials bool     `mapstructure:"allow_credentials"`
}

func setDefaults(conf *viper.Viper) {
	conf.SetDefault("api.pagination_limit_default", 50)
	conf.SetDefault("api.pagination_limit_max", 150)

	conf.SetDefault("api.cors.allowed_origins", []string{"http://localhost:3000"})
	conf.SetDefault("api.cors.allowed_methods", []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"})
	conf.SetDefault("api.cors.allowed_headers", []string{"Accept", "Content-Type", "Authorization"})
	conf.SetDefault("api.cors.allow_credentials", false)
}

func Load(ctx context.Context, parentConf *viper.Viper) (*Config, error) {
	conf := parentConf.Sub("qa")
	if conf == nil {
		conf = viper.New()
	}

	setDefaults(conf)

	var config Config
	err := configurator.UnmarshalWithFileResolution(conf, &config)
	if err != nil {
		return nil, fmt.Errorf("failed unmashaling qa config from conf: %w", err)
	}

	var ok bool
	if config.GlobalConfig, ok = configurator.FromContext(ctx); !ok {
		return nil, fmt.Errorf("no global config in context")
	}

	err = validation.Validate.Struct(config)
	if err != nil {
		return nil, fmt.Errorf("validating qa configuration: %w", err)
	}

	return &config, nil
}
