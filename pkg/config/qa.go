package config

import "time"

type QA struct {
	Enabled           bool            `koanf:"enabled"`
	Host              string          `koanf:"host"`
	Port              int             `koanf:"port"`
	ReadHeaderTimeout time.Duration   `koanf:"read_header_timeout"`
	WriteTimeout      time.Duration   `koanf:"write_timeout"`
	IdleTimeout       time.Duration   `koanf:"idle_timeout"`
	ServerTLS         ServerTLSConfig `koanf:"server_tls"`

	Metrics   Metrics `koanf:"metrics"`
	APIConfig QAAPI   `koanf:"api"`
}

type QAAPI struct {
	PaginationLimitDefault int  `koanf:"pagination_limit_default"`
	PaginationLimitMax     int  `koanf:"pagination_limit_max"`
	CorsConfig             CORS `koanf:"cors"`
}

type CORS struct {
	AllowedOrigins   []string `koanf:"allowed_origins"`
	AllowedMethods   []string `koanf:"allowed_methods"`
	AllowedHeaders   []string `koanf:"allowed_headers"`
	AllowCredentials bool     `koanf:"allow_credentials"`
}
