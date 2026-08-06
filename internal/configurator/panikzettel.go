package configurator

import (
	"time"

	"github.com/htwr-aachen/backend/pkg/config"
	"github.com/knadh/koanf/v2"
)

const (
	defaultDownloadFlushInterval = 30 * time.Second
	defaultDownloadFlushTimeout  = 10 * time.Second
)

func panikzettelHook(conf *koanf.Koanf, gcfg *config.Config) error {
	cfg := &gcfg.Panikzettel

	if cfg.CacheCleanupInterval == 0 {
		cfg.CacheCleanupInterval = 2 * cfg.CacheDuration
	}

	if cfg.Downloads.FlushInterval <= 0 {
		cfg.Downloads.FlushInterval = defaultDownloadFlushInterval
	}

	if cfg.Downloads.FlushTimeout <= 0 {
		cfg.Downloads.FlushTimeout = defaultDownloadFlushTimeout
	}

	return nil
}
