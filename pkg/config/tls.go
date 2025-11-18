package config

type TLSBaseConfig struct {
	MinVersionStr      string `mapstructure:"min_version" validate:"omitempty,oneof=tls1.2 tls1.3"`
	MinVersion         uint16 `mapstructure:"-"`
	MaxVersionStr      string `mapstructure:"max_version" validate:"omitempty,oneof=tls1.2 tls1.3"`
	MaxVersion         uint16 `mapstructure:"-"`
	InsecureSkipVerify bool   `mapstructure:"insecure_skip_verify"`
}

type ConnectionTLSConfig struct {
	TLSBaseConfig
	ServerName string `mapstructure:"server_name" validate:"omitempty"`
	ServerCert string `mapstructure:"server_cert" validate:"omitempty,file"`
	ClientCert string `mapstructure:"client_cert" validate:"omitempty,file"`
	ClientKey  string `mapstructure:"client_key" validate:"omitempty,file"`
}

type GlobalTLSConfig struct {
	TLSBaseConfig
	TrustBundle []string `mapstructure:"trust_bundle" validate:"omitempty"`
}

type ServerTLSConfig struct {
	TLSBaseConfig
	ServerCert string `mapstructure:"server_cert"`
	ServerKey  string `mapstructure:"server_key"`
}
