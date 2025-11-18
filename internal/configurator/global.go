package configurator

import (
	"context"
	"errors"
	"fmt"

	"github.com/htwr-aachen/backend/internal/validation"
	"github.com/htwr-aachen/backend/pkg/config"
	"github.com/knadh/koanf/v2"
	"github.com/rs/zerolog/log"
)

type contextKey struct{}

func load(conf *koanf.Koanf) (*config.Config, error) {
	cfg := new(config.Config)
	err := UnmarshalWithFileResolution(conf, cfg)
	if err != nil {
		log.Err(err).Msg("unmarshaling config")
		return nil, fmt.Errorf("could not load configuration: %w", err)
	}

	if cfg.Global.InsecureDev {
		log.Warn().Msg("insecure development mode active")
	}

	return cfg, nil
}

func validate(cfg *config.Config) error {
	var errs error

	// Helper to reduce boilerplate and wrap errors with context
	check := func(s interface{}, name string) {
		if err := validation.Validate.Struct(s); err != nil {
			errs = errors.Join(errs, fmt.Errorf("%s config: %w", name, err))
		}
	}

	check(&cfg.Global, "global")

	check(&cfg.Database, "database")

	// 2. Conditional Validations
	if cfg.Admin.Enabled {
		check(&cfg.Admin, "admin")
	}

	if !cfg.Session.Disabled {
		check(&cfg.Session, "session")
	}

	// 4. Feature Flags
	// Fixed logic: Validate the STRUCT (not the bool flag) only if it IS enabled.
	if cfg.QA.Enabled {
		check(&cfg.QA, "qa")
	}

	if cfg.Panikzettel.Enabled {
		check(&cfg.Panikzettel, "panikzettel")
	}

	if cfg.MetricsServer.Enabled {
		check(&cfg.MetricsServer, "metrics_server")
	}

	// 5. Return combined errors
	if errs != nil {
		log.Err(errs).Msg("configuration validation failed")
		return errs
	}

	return nil
}

func attach(parent context.Context, cfg *config.Config) context.Context {
	return context.WithValue(parent, contextKey{}, cfg)
}

func LoadAndAttach(parent context.Context, conf *koanf.Koanf) (context.Context, error) {
	if conf == nil {
		log.Fatal().Msg("no koanf supplied")
	}

	cfg, err := load(conf)
	if err != nil {
		return parent, err
	}

	err = dbHook(cfg)
	if err != nil {
		return parent, err
	}

	err = sessionHook(conf, cfg)
	if err != nil {
		return parent, err
	}

	err = panikzettelHook(conf, cfg)
	if err != nil {
		return parent, err
	}

	err = validate(cfg)
	if err != nil {
		return parent, err
	}

	if cfg.Global.InsecureDev {
		log.Warn().Msg("insecure dev mode")
	}

	return attach(parent, cfg), nil
}

func FromContext(ctx context.Context) (*config.Config, bool) {
	cfg, ok := ctx.Value(contextKey{}).(*config.Config)
	return cfg, ok
}
