package config

import "time"

type ServerConfig struct {
	Enabled           bool            `koanf:"enabled"`
	Host              string          `koanf:"host"`
	Port              int             `koanf:"port"`
	ReadHeaderTimeout time.Duration   `koanf:"read_header_timeout"`
	WriteTimeout      time.Duration   `koanf:"write_timeout"`
	IdleTimeout       time.Duration   `koanf:"idle_timeout"`
	ServerTLS         ServerTLSConfig `koanf:"server_tls"`
}
