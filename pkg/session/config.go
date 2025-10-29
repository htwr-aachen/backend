package session

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/coreos/go-oidc"
	"github.com/htwr-aachen/backend/internal/configurator"
	"github.com/htwr-aachen/backend/internal/validation"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

type SessionConfig struct {
	GlobalConfig         *configurator.Global `mapstructure:"-"`
	Disabled             bool                 `mapstructure:"disabled"`
	CacheExpiration      time.Duration        `mapstructure:"cache_expiration" validate:"required"`
	CacheCleanupInterval time.Duration        `mapstructure:"cache_cleanup_interval" validate:"required"`
	Providers            map[string]Provider  `mapstructure:"providers" validate:"required,min=1,dive"`

	SessionCookieName     string `mapstructure:"cookie_name" validate:"required"`
	SessionCookieNameFile string `mapstructure:"cookie_name_file"`
	SessionCookieSecure   bool   `mapstructure:"cookie_secure"`
	SessionCookieHttpOnly bool   `mapstructure:"cookie_http_only"`
	SessionCookieSameSite string `mapstructure:"cookie_same_site"`

	SessionExpiration time.Duration `mapstructure:"expiration" validate:"required"`

	RoleMap map[string]string `mapstructure:"role_map"`

	AuthURLPrefix     string `mapstructure:"auth_url_prefix" validate:"uri"`
	AuthURLPrefixFile string `mapstructure:"auth_url_prefix_file"`
	AuthLoginURL      string `mapstructure:"auth_login_url" validate:"uri"`
	AuthLoginURLFile  string `mapstructure:"auth_login_url_file"`
}

type SessionUsageConfig struct {
	AuthLoginURL string
}

type Provider struct {
	Name             string   `mapstructure:"name" validate:"required"`
	Issuer           string   `mapstructure:"issuer" validate:"required"`
	IssuerFile       string   `mapstructure:"issuer_file"`
	ClientId         string   `mapstructure:"client_id" validate:"required"`
	ClientIdFile     string   `mapstructure:"client_id_file"`
	ClientSecret     string   `mapstructure:"client_secret" validate:"required"`
	ClientSecretFile string   `mapstructure:"client_secret_file"`
	Endpoint         string   `mapstructure:"endpoint" validate:"required,url"`
	EndpointFile     string   `mapstructure:"endpoint_file"`
	RedirectURL      string   `mapstructure:"redirect_url" validate:"required,url"`
	RedirectURLFile  string   `mapstructure:"redirect_url_file"`
	Scopes           []string `mapstructure:"scopes" validate:"min=1"`
}

func setDefaults(conf *viper.Viper) {
	conf.SetDefault("cache_expiration", time.Minute*5)
	conf.SetDefault("cookie_name", "session")
	conf.SetDefault("cookie_secure", true)
	conf.SetDefault("cookie_http_only", true)
	conf.SetDefault("expiration", time.Hour*24)
	conf.SetDefault("auth_url_prefix", "/auth")

	conf.SetDefault("role_map.admin", "admin")
	conf.SetDefault("role_map.user", "user")

	if !conf.IsSet("cache_cleanup_interval") {
		conf.SetDefault("cache_cleanup_interval", 2*conf.GetDuration("cache_expiration"))
	}
}

func LoadConfig(ctx context.Context, parentConf *viper.Viper, useConf *SessionUsageConfig) (*SessionConfig, error) {
	conf := parentConf.Sub("session")
	if conf == nil {
		conf = viper.New()
	}

	setDefaults(conf)

	config := &SessionConfig{}
	if err := configurator.UnmarshalWithFileResolution(conf, config); err != nil {
		log.Err(err).Msg("unmarshaling session configuration")
		return nil, fmt.Errorf("unmarshaling config: %w", err)
	}

	var ok bool
	if config.GlobalConfig, ok = configurator.FromContext(ctx); !ok {
		return nil, fmt.Errorf("no global config in context")
	}

	if config.Disabled {
		return config, nil
	}

	if config.Providers == nil {
		config.Providers = make(map[string]Provider)
	}

	config.loadFromParentService(useConf)
	if err := config.loadProvidersFromEnv(conf); err != nil {
		return nil, fmt.Errorf("loading providers from environment: %w", err)
	}

	for name, provider := range config.Providers {
		if provider.Name == "" {
			provider.Name = name
		}

		if len(provider.Scopes) == 0 {
			provider.Scopes = []string{oidc.ScopeOpenID}
			log.Warn().Str("provider", name).Msg("no scopes configured, using default 'openid' scope")
		}

		config.Providers[name] = provider
	}

	config.AuthURLPrefix = strings.TrimRight(config.AuthURLPrefix, "/")

	if err := validation.Validate.Struct(config); err != nil {
		return nil, formatValidationError(err)
	}

	return config, nil
}

func (config *SessionConfig) loadFromParentService(parentConfig *SessionUsageConfig) {
	config.AuthLoginURL = parentConfig.AuthLoginURL
}

func (config *SessionConfig) loadProvidersFromEnv(conf *viper.Viper) error {
	providerNames := conf.GetString("providers")

	if providerNames == "" {
		return nil
	}

	for _, name := range strings.Split(providerNames, ",") {
		name := strings.TrimSpace(name)
		if name == "" {
			continue
		}

		// Build provider from environment variables
		prefix := fmt.Sprintf("providers.%s", name)

		// Load values with file support using the helper
		clientId, err := configurator.GetStringOrFile(conf, prefix+".client_id")
		if err != nil {
			return fmt.Errorf("loading client_id for provider %s: %w", name, err)
		}

		clientSecret, err := configurator.GetStringOrFile(conf, prefix+".client_secret")
		if err != nil {
			return fmt.Errorf("loading client_secret for provider %s: %w", name, err)
		}

		issuer, err := configurator.GetStringOrFile(conf, prefix+".issuer")
		if err != nil {
			return fmt.Errorf("loading issuer for provider %s: %w", name, err)
		}

		endpoint, err := configurator.GetStringOrFile(conf, prefix+".endpoint")
		if err != nil {
			return fmt.Errorf("loading endpoint for provider %s: %w", name, err)
		}

		redirectURL, err := configurator.GetStringOrFile(conf, prefix+".redirect_url")
		if err != nil {
			return fmt.Errorf("loading redirect_url for provider %s: %w", name, err)
		}

		// Parse scopes (comma-separated)
		var scopes []string
		scopesStr, err := configurator.GetStringOrFile(conf, prefix+".scopes")
		if err != nil {
			return fmt.Errorf("loading scopes for provider %s: %w", name, err)
		}
		if scopesStr != "" {
			scopes = strings.Split(scopesStr, ",")
			for i, scope := range scopes {
				scopes[i] = strings.TrimSpace(scope)
			}
		}

		envProvider := Provider{
			Name:         name,
			Issuer:       issuer,
			ClientId:     clientId,
			ClientSecret: clientSecret,
			Endpoint:     endpoint,
			RedirectURL:  redirectURL,
			Scopes:       scopes,
		}

		// In loadProvidersFromEnv
		if existing, exists := config.Providers[name]; exists {
			config.Providers[name] = mergeProvider(existing, envProvider, withMergeScopes())
		} else {
			if envProvider.ClientId != "" {
				config.Providers[name] = envProvider
			}
		}
	}

	return nil
}

func mergeProvider(base, override Provider, opts ...mergeOption) Provider {
	merged := base

	// Apply default merge
	withOverrideAll()(&merged, override)

	// Apply custom options
	for _, opt := range opts {
		opt(&merged, override)
	}

	return merged
}

// mergeOption defines how to merge provider fields
type mergeOption func(*Provider, Provider)

// WithMergeScopes appends scopes instead of replacing
func withMergeScopes() mergeOption {
	return func(base *Provider, override Provider) {
		if len(override.Scopes) > 0 {
			// Create a map to avoid duplicates
			scopeSet := make(map[string]bool)
			for _, scope := range base.Scopes {
				scopeSet[scope] = true
			}
			for _, scope := range override.Scopes {
				scopeSet[scope] = true
			}

			// Convert back to slice
			merged := make([]string, 0, len(scopeSet))
			for scope := range scopeSet {
				merged = append(merged, scope)
			}
			base.Scopes = merged
		}
	}
}

// WithOverrideAll overrides all non-zero values
func withOverrideAll() mergeOption {
	return func(base *Provider, override Provider) {
		if override.ClientId != "" {
			base.ClientId = override.ClientId
		}
		if override.ClientSecret != "" {
			base.ClientSecret = override.ClientSecret
		}
		if override.Endpoint != "" {
			base.Endpoint = override.Endpoint
		}
		if override.RedirectURL != "" {
			base.RedirectURL = override.RedirectURL
		}
		if override.Name != "" {
			base.Name = override.Name
		}
		if len(override.Scopes) > 0 {
			base.Scopes = override.Scopes
		}
	}
}
