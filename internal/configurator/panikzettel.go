package configurator

import (
	"github.com/htwr-aachen/backend/pkg/config"
	"github.com/knadh/koanf/v2"
)

func panikzettelHook(conf *koanf.Koanf, gcfg *config.Config) error {
	cfg := &gcfg.Panikzettel

	if cfg.CacheCleanupInterval == 0 {
		cfg.CacheCleanupInterval = 2 * cfg.CacheDuration
	}

	return nil
}
