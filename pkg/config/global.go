package config

type Global struct {
	LogLevel    string          `koanf:"log_level"`
	InsecureDev bool            `koanf:"insecure_dev"`
	TLS         GlobalTLSConfig `koanf:"tls"`
	Metrics     Metrics         `koanf:"metrics"`
}
