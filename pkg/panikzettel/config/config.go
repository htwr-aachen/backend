package config

import (
	"context"
	"fmt"
	"time"

	"github.com/htwr-aachen/backend/internal/configurator"
	"github.com/htwr-aachen/backend/internal/validation"
	"github.com/spf13/viper"
)

const (
	CloudProviderGoogle string = "gcp"
	CloudProviderAWS    string = "aws"

	AuthMethodDefault        string = "default"
	AuthMethodServiceAccount string = "service_account"
	AuthMethodAPIKey         string = "api_key"
)

type CloudConfig struct {
	BucketName         string                           `mapstructure:"bucket" validate:"required"`
	BucketNameFile     string                           `mapstructure:"bucket_file"`
	Provider           string                           `mapstructure:"provider" validate:"oneof=gcp aws"`
	ProviderFile       string                           `mapstructure:"provider_file"`
	AuthMethod         string                           `mapstructure:"auth_method" validate:"omitempty,oneof=default service_account api_key"`
	AuthMethodFile     string                           `mapstructure:"auth_method_file"`
	ServiceAccount     string                           `mapstructure:"service_account_path" fileconfig:"skip" validate:"required_if=AuthMethod service_account,omitempty,file"`
	ServiceAccountJSON string                           `mapstructure:"service_account_json" fileconfig:"skip" validate:"excluded_with=ServiceAccountPath"`
	APIKey             string                           `mapstructure:"api_key" validate:"required_if=AuthMethod api_key"`
	APIKeyFile         string                           `mapstructure:"api_key_file"`
	ProjectId          string                           `mapstructure:"project_id" validate:"omitempty"`
	ProjectIdFile      string                           `mapstructure:"project_id_file"`
	GoogleAccessId     string                           `mapstructure:"google_access_id" validate:"omitempty"`
	GoogleAccessIdFile string                           `mapstructure:"google_access_id_file"`
	PrivateKey         string                           `mapstructure:"private_key" fileconfig:"skip" validate:"omitempty,file"`
	TLSConfig          configurator.ConnectionTLSConfig `mapstructure:"tls"`
}

type Config struct {
	GlobalConfig         *configurator.Global `mapstructure:"-"`
	BaseURL              string               `mapstructure:"base_url"`
	BaseURLFile          string               `mapstructure:"base_url_file"`
	MetadataFilename     string               `mapstructure:"metadata_filename"`
	MetadataFilenameFile string               `mapstructure:"metadata_filename_file"`
	MaxFileSize          int64                `mapstructure:"max_file_size"`
	CacheDuration        time.Duration        `mapstructure:"cache_duration"`
	CacheCleanupInterval time.Duration        `mapstructure:"cache_cleanup_interval" validate:"omitempty"`
	CloudConfig          CloudConfig          `mapstructure:"cloud"`
}

func setDefaults(conf *viper.Viper) {
	conf.SetDefault("base_url", "/api/panikzettel")
	conf.SetDefault("metadata_filename", "metadata.json")
	conf.SetDefault("cache_duration", time.Hour*6)
	conf.SetDefault("max_file_size", 512*1024*1024) // 512MB

	conf.SetDefault("cloud.provider", "aws")
	conf.SetDefault("cloud.auth_method", "default")
}

func LoadConfig(ctx context.Context, parentConf *viper.Viper) (*Config, error) {

	conf := parentConf.Sub("panikzettel")
	if conf == nil {
		conf = viper.New()
	}

	setDefaults(conf)

	var config Config
	err := configurator.UnmarshalWithFileResolution(conf, &config)
	if err != nil {
		return nil, fmt.Errorf("failed unmarshaling panikzettel config: %w", err)
	}

	var ok bool
	if config.GlobalConfig, ok = configurator.FromContext(ctx); !ok {
		return nil, fmt.Errorf("no global config in context")
	}

	if config.CacheCleanupInterval == 0 {
		config.CacheCleanupInterval = config.CacheDuration
	}

	err = validation.Validate.Struct(config)
	if err != nil {
		return nil, fmt.Errorf("validating panikzettel configuration: %w", err)
	}
	return &config, nil
}

// Validate performs custom validation on CloudConfig
func (c CloudConfig) Validate() error {
	// Validate service account auth requirements
	if c.AuthMethod == AuthMethodServiceAccount {
		if c.ServiceAccount == "" && c.ServiceAccountJSON == "" {
			return fmt.Errorf("service_account auth requires either service_account_path or service_account_json")
		}
		if c.ServiceAccount != "" && c.ServiceAccountJSON != "" {
			return fmt.Errorf("specify only one of service_account_path or service_account_json, not both")
		}
	}

	// Validate API key auth requirements
	if c.AuthMethod == AuthMethodAPIKey && c.APIKey == "" {
		return fmt.Errorf("api_key auth requires api_key to be set")
	}

	// Validate signed URL requirements (if any fields are set)
	if c.GoogleAccessId != "" || c.PrivateKey != "" {
		if c.GoogleAccessId == "" {
			return fmt.Errorf("private_key_path requires google_access_id to be set")
		}
		if c.PrivateKey == "" {
			return fmt.Errorf("google_access_id requires private_key_path to be set")
		}
	}

	return nil
}
