package session

import (
	"context"
	"fmt"
	"net/http"

	"github.com/coreos/go-oidc"
	"github.com/htwr-aachen/backend/internal/configurator"
	"github.com/htwr-aachen/backend/pkg/config"
	"github.com/rs/zerolog/log"
	"golang.org/x/oauth2"
)

type oidcProviderConfig struct {
	name     string
	provider *oidc.Provider
	config   oauth2.Config
	verifier *oidc.IDTokenVerifier
}

type SessionSubsystem struct {
	db         *DB
	config     config.Session
	oidcConfig []oidcProviderConfig
	mux        *http.ServeMux
}

func New(ctx context.Context, parentConfig *config.SessionUsageConfig) (*SessionSubsystem, error) {
	gcfg, ok := configurator.FromContext(ctx)
	if !ok {
		log.Panic().Stack().Msg("no configuration context given")
	}

	cfg := configurator.MergeSession(&gcfg.Session, parentConfig)

	if cfg.Disabled {
		return &SessionSubsystem{
			db:         nil,
			config:     *cfg,
			oidcConfig: nil,
		}, nil
	}

	db, err := newSessionDB(ctx, *cfg)
	if err != nil {
		log.Err(err).Msg("creating session db")
		return nil, err
	}

	subsystem := SessionSubsystem{
		db: db,

		config:     *cfg,
		oidcConfig: make([]oidcProviderConfig, 0, len(cfg.Providers)),
	}

	authMux := http.NewServeMux()
	authMux.HandleFunc(fmt.Sprintf("GET %s/{name}/login", subsystem.config.AuthURLPrefix), subsystem.Login)
	authMux.HandleFunc(fmt.Sprintf("GET %s/{name}/callback", subsystem.config.AuthURLPrefix), subsystem.Callback)
	subsystem.mux = authMux

	if err := subsystem.initOIDCProviders(ctx); err != nil {
		return nil, err
	}

	return &subsystem, err
}

func (s *SessionSubsystem) initOIDCProviders(ctx context.Context) error {
	for name, provider := range s.config.Providers {
		oidcProvider, err := oidc.NewProvider(ctx, provider.Issuer)
		if err != nil {
			log.Err(err).Str("provider", name).Msg("failed to initialize OIDC provider")
			return fmt.Errorf("configuring oidc provider %s: %w", name, err)
		}

		verifier := oidcProvider.Verifier(&oidc.Config{
			ClientID: provider.ClientId,
		})

		oauthConfig := oauth2.Config{
			ClientID:     provider.ClientId,
			ClientSecret: provider.ClientSecret,
			Endpoint:     oidcProvider.Endpoint(),
			RedirectURL:  provider.RedirectURL,
			Scopes:       provider.Scopes,
		}

		// Auto-discover the auth method from provider
		var claims struct {
			TokenEndpointAuthMethodsSupported []string `json:"token_endpoint_auth_methods_supported"`
		}
		if err := oidcProvider.Claims(&claims); err == nil {
			// Check which methods are supported and prefer client_secret_basic
			for _, method := range claims.TokenEndpointAuthMethodsSupported {
				if method == "client_secret_basic" {
					oauthConfig.Endpoint.AuthStyle = oauth2.AuthStyleInHeader
					log.Debug().Str("provider", name).Msg("Using client_secret_basic")
					break
				} else if method == "client_secret_post" {
					oauthConfig.Endpoint.AuthStyle = oauth2.AuthStyleInParams
					log.Debug().Str("provider", name).Msg("Using client_secret_post")
					break
				}
			}
		}

		s.oidcConfig = append(s.oidcConfig, oidcProviderConfig{
			name:     name,
			provider: oidcProvider,
			config:   oauthConfig,
			verifier: verifier,
		})
	}
	return nil
}
