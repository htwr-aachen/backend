package config

import (
	"fmt"
	"time"
)

const (
	CloudProviderGoogle string = "gcp"
	CloudProviderAWS    string = "aws"

	CloudAuthMethodDefault        string = "default"
	CloudAuthMethodServiceAccount string = "service_account"
	CloudAuthMethodAPIKey         string = "api_key"
)

type CloudConfig struct {
	BucketName         string              `koanf:"bucket" validate:"required"`
	BucketNameFile     string              `koanf:"bucket_file"`
	Provider           string              `koanf:"provider" validate:"oneof=gcp aws"`
	ProviderFile       string              `koanf:"provider_file"`
	AuthMethod         string              `koanf:"auth_method" validate:"omitempty,oneof=default service_account api_key"`
	AuthMethodFile     string              `koanf:"auth_method_file"`
	ServiceAccount     string              `koanf:"service_account_path" fileconfig:"skip" validate:"required_if=AuthMethod service_account,omitempty,file"`
	ServiceAccountJSON string              `koanf:"service_account_json" fileconfig:"skip" validate:"excluded_with=ServiceAccountPath"`
	APIKey             string              `koanf:"api_key" validate:"required_if=AuthMethod api_key"`
	APIKeyFile         string              `koanf:"api_key_file"`
	ProjectId          string              `koanf:"project_id" validate:"omitempty"`
	ProjectIdFile      string              `koanf:"project_id_file"`
	GoogleAccessId     string              `koanf:"google_access_id" validate:"omitempty"`
	GoogleAccessIdFile string              `koanf:"google_access_id_file"`
	PrivateKey         string              `koanf:"private_key" fileconfig:"skip" validate:"omitempty,file"`
	TLSConfig          ConnectionTLSConfig `koanf:"tls"`
}

type Panikzettel struct {
	Enabled              bool          `koanf:"enabled"`
	AutoDownload         bool          `koanf:"auto_download"`
	Metrics              Metrics       `koanf:"metrics"`
	BaseURL              string        `koanf:"base_url"`
	BaseURLFile          string        `koanf:"base_url_file"`
	MetadataFilename     string        `koanf:"metadata_filename"`
	MetadataFilenameFile string        `koanf:"metadata_filename_file"`
	MaxFileSize          int64         `koanf:"max_file_size"`
	CacheDuration        time.Duration `koanf:"cache_duration"`
	CacheCleanupInterval time.Duration `koanf:"cache_cleanup_interval" validate:"omitempty"`
	CloudConfig          CloudConfig   `koanf:"cloud"`
}

// Validate performs custom validation on CloudConfig
func (c CloudConfig) Validate() error {
	// Validate service account auth requirements
	if c.AuthMethod == CloudAuthMethodServiceAccount {
		if c.ServiceAccount == "" && c.ServiceAccountJSON == "" {
			return fmt.Errorf("service_account auth requires either service_account_path or service_account_json")
		}
		if c.ServiceAccount != "" && c.ServiceAccountJSON != "" {
			return fmt.Errorf("specify only one of service_account_path or service_account_json, not both")
		}
	}

	// Validate API key auth requirements
	if c.AuthMethod == CloudAuthMethodAPIKey && c.APIKey == "" {
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
