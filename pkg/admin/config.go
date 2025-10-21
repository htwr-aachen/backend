package admin

import (
	"context"
	"fmt"

	"github.com/htwr-aachen/backend/internal/configurator"
	"github.com/spf13/viper"
)

type AdminConfig struct {
	GlobalConfig *configurator.Global `mapstructure:"-"`
}

func LoadConfig(ctx context.Context, parentConf *viper.Viper) (*AdminConfig, error) {
	config := AdminConfig{}
	var ok bool
	if config.GlobalConfig, ok = configurator.FromContext(ctx); !ok {
		return nil, fmt.Errorf("no global config in context")
	}
	return &config, nil
}
