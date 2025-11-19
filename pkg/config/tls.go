package config

type TLSBaseConfig struct {
	MinVersionStr      string `koanf:"min_version" validate:"omitempty,oneof=tls1.2 tls1.3"`
	MinVersion         uint16 `koanf:"-"`
	MaxVersionStr      string `koanf:"max_version" validate:"omitempty,oneof=tls1.2 tls1.3"`
	MaxVersion         uint16 `koanf:"-"`
	InsecureSkipVerify bool   `koanf:"insecure_skip_verify"`
}

type ConnectionTLSConfig struct {
	TLSBaseConfig
	ServerName string `koanf:"server_name" validate:"omitempty"`
	ServerCert string `koanf:"server_cert" validate:"omitempty,file"`
	ClientCert string `koanf:"client_cert" validate:"omitempty,file"`
	ClientKey  string `koanf:"client_key" validate:"omitempty,file"`
}

type GlobalTLSConfig struct {
	TLSBaseConfig
	TrustBundle []string `koanf:"trust_bundle" validate:"omitempty"`
}

type ServerTLSConfig struct {
	TLSBaseConfig
	ServerCert string `koanf:"server_cert"`
	ServerKey  string `koanf:"server_key"`
}
