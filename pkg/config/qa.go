package config

type QA struct {
	Enabled bool `koanf:"enabled"`

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
