package config

import "time"

type SessionProvider struct {
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

type Session struct {
	Disabled             bool                       `koanf:"disabled"`
	CacheExpiration      time.Duration              `koanf:"cache_expiration" validate:"required"`
	CacheCleanupInterval time.Duration              `koanf:"cache_cleanup_interval" validate:"required"`
	Providers            map[string]SessionProvider `koanf:"providers" validate:"required,min=1,dive"`

	SessionCookieName     string `koanf:"cookie_name" validate:"required"`
	SessionCookieNameFile string `koanf:"cookie_name_file"`
	SessionCookieSecure   bool   `koanf:"cookie_secure"`
	SessionCookieHttpOnly bool   `koanf:"cookie_http_only"`
	SessionCookieSameSite string `koanf:"cookie_same_site"`

	SessionExpiration time.Duration `koanf:"expiration" validate:"required"`

	RoleMap map[string]string `koanf:"role_map"`

	AuthURLPrefix     string `koanf:"auth_url_prefix" validate:"uri"`
	AuthURLPrefixFile string `koanf:"auth_url_prefix_file"`
	AuthLoginURL      string `koanf:"auth_login_url" validate:"uri"`
	AuthLoginURLFile  string `koanf:"auth_login_url_file"`
}

type SessionUsageConfig struct {
	AuthLoginURL string
}
