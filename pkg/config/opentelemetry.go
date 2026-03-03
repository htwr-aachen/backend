package config

// OpenTelemetry configuration
type OpenTelemetry struct {
	Enabled    bool   `koanf:"enabled"`
	ServiceName string `koanf:"service_name"`
	Endpoint   string `koanf:"endpoint"`
}
