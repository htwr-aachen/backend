package configurator

import (
	"fmt"
	"strings"

	"github.com/coreos/go-oidc"
	"github.com/htwr-aachen/backend/pkg/config"
	"github.com/knadh/koanf/v2"
	"github.com/rs/zerolog/log"
)

func sessionHook(conf *koanf.Koanf, gcfg *config.Config) error {
	cfg := &gcfg.Session
	if cfg.Disabled {
		return nil
	}

	if cfg.Providers == nil {
		cfg.Providers = make(map[string]config.SessionProvider)
	}

	if err := loadProvidersFromEnv(conf, cfg); err != nil {
		return fmt.Errorf("loading providers from environment: %w", err)
	}

	for name, provider := range cfg.Providers {
		if provider.Name == "" {
			provider.Name = name
		}

		if len(provider.Scopes) == 0 {
			provider.Scopes = []string{oidc.ScopeOpenID}
			log.Warn().Str("provider", name).Msg("no scopes configured, using default 'openid' scope")
		}

		cfg.Providers[name] = provider
	}

	cfg.AuthURLPrefix = strings.TrimRight(cfg.AuthURLPrefix, "/")
	return nil
}

func loadFromParentService(cfg *config.Session, parentConfig *config.SessionUsageConfig) {
}

func loadProvidersFromEnv(conf *koanf.Koanf, cfg *config.Session) error {
	providerNames := conf.String("session.providers")

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
		clientId, err := GetStringOrFile(conf, prefix+".client_id")
		if err != nil {
			return fmt.Errorf("loading client_id for provider %s: %w", name, err)
		}

		clientSecret, err := GetStringOrFile(conf, prefix+".client_secret")
		if err != nil {
			return fmt.Errorf("loading client_secret for provider %s: %w", name, err)
		}

		issuer, err := GetStringOrFile(conf, prefix+".issuer")
		if err != nil {
			return fmt.Errorf("loading issuer for provider %s: %w", name, err)
		}

		endpoint, err := GetStringOrFile(conf, prefix+".endpoint")
		if err != nil {
			return fmt.Errorf("loading endpoint for provider %s: %w", name, err)
		}

		redirectURL, err := GetStringOrFile(conf, prefix+".redirect_url")
		if err != nil {
			return fmt.Errorf("loading redirect_url for provider %s: %w", name, err)
		}

		// Parse scopes (comma-separated)
		var scopes []string
		scopesStr, err := GetStringOrFile(conf, prefix+".scopes")
		if err != nil {
			return fmt.Errorf("loading scopes for provider %s: %w", name, err)
		}
		if scopesStr != "" {
			scopes = strings.Split(scopesStr, ",")
			for i, scope := range scopes {
				scopes[i] = strings.TrimSpace(scope)
			}
		}

		envProvider := config.SessionProvider{
			Name:         name,
			Issuer:       issuer,
			ClientId:     clientId,
			ClientSecret: clientSecret,
			Endpoint:     endpoint,
			RedirectURL:  redirectURL,
			Scopes:       scopes,
		}

		// In loadProvidersFromEnv
		if existing, exists := cfg.Providers[name]; exists {
			cfg.Providers[name] = mergeProvider(existing, envProvider, withMergeScopes())
		} else {
			if envProvider.ClientId != "" {
				cfg.Providers[name] = envProvider
			}
		}
	}

	return nil
}

func mergeProvider(base, override config.SessionProvider, opts ...mergeOption) config.SessionProvider {
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
type mergeOption func(*config.SessionProvider, config.SessionProvider)

// WithMergeScopes appends scopes instead of replacing
func withMergeScopes() mergeOption {
	return func(base *config.SessionProvider, override config.SessionProvider) {
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
	return func(base *config.SessionProvider, override config.SessionProvider) {
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

func MergeSession(cfg *config.Session, usageCfg *config.SessionUsageConfig) *config.Session {
	tmpcfg := *cfg
	copyCfg := tmpcfg

	copyCfg.AuthLoginURL = usageCfg.AuthLoginURL
	return &copyCfg
}
