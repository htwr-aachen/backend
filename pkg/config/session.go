package config

import "time"

type SessionProvider struct {
	Name             string   `koanf:"name" validate:"required"`
	Issuer           string   `koanf:"issuer" validate:"required"`
	IssuerFile       string   `koanf:"issuer_file"`
	ClientId         string   `koanf:"client_id" validate:"required"`
	ClientIdFile     string   `koanf:"client_id_file"`
	ClientSecret     string   `koanf:"client_secret" validate:"required"`
	ClientSecretFile string   `koanf:"client_secret_file"`
	Endpoint         string   `koanf:"endpoint" validate:"required,url"`
	EndpointFile     string   `koanf:"endpoint_file"`
	RedirectURL      string   `koanf:"redirect_url" validate:"required,url"`
	RedirectURLFile  string   `koanf:"redirect_url_file"`
	Scopes           []string `koanf:"scopes" validate:"min=1"`
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
