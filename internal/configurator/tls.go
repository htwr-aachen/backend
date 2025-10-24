package configurator

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"

	"github.com/rs/zerolog/log"
)

const (
	TLSVersion12 = "tls1.2"
	TLSVersion13 = "tls1.3"
)

type baseConfig struct {
	MinVersionStr      string `mapstructure:"min_version" validate:"omitempty,oneof=tls1.2 tls1.3"`
	MinVersion         uint16 `mapstructure:"-"`
	MaxVersionStr      string `mapstructure:"max_version" validate:"omitempty,oneof=tls1.2 tls1.3"`
	MaxVersion         uint16 `mapstructure:"-"`
	InsecureSkipVerify bool   `mapstructure:"insecure_skip_verify"`
}

type ConnectionTLSConfig struct {
	baseConfig
	ServerName string `mapstructure:"server_name" validate:"omitempty"`
	ServerCert string `mapstructure:"server_cert" validate:"omitempty,file"`
	ClientCert string `mapstructure:"client_cert" validate:"omitempty,file"`
	ClientKey  string `mapstructure:"client_key" validate:"omitempty,file"`
}

type GlobalTLSConfig struct {
	baseConfig
	TrustBundle []string `mapstructure:"trust_bundle" validate:"omitempty"`
}

func ParseTLSVersion(versionStr string) (uint16, error) {
	switch versionStr {
	case TLSVersion12:
		return tls.VersionTLS12, nil
	case TLSVersion13:
		return tls.VersionTLS13, nil
	default:
		return tls.VersionTLS12, fmt.Errorf("could not parse unknown '%s' tls version string", versionStr)
	}
}

func mergeFromConfig(cfg *baseConfig, tlsConfig *tls.Config) {
	if cfg.MinVersionStr != "" {
		tlsConfig.MinVersion = cfg.MinVersion
	}
	if cfg.MaxVersionStr != "" {
		tlsConfig.MaxVersion = cfg.MaxVersion
	}

	if cfg.InsecureSkipVerify {
		tlsConfig.InsecureSkipVerify = true
		log.Warn().
			Msg("TLS certificate verification is DISABLED - this is insecure and should only be used for testing")
	}

}

func MergeFromConnConfig(cfg *ConnectionTLSConfig, tlsConfig *tls.Config) error {
	mergeFromConfig(&cfg.baseConfig, tlsConfig)

	// Handle custom certificate chain
	if cfg.ServerCert != "" {
		caCert, err := os.ReadFile(cfg.ServerCert)
		if err != nil {
			return fmt.Errorf("reading TLS certificate from '%s': %w", cfg.ServerCert, err)
		}

		caCertPool := x509.NewCertPool()
		if !caCertPool.AppendCertsFromPEM(caCert) {
			return fmt.Errorf("failed to parse TLS certificate from '%s'", cfg.ServerCert)
		}

		tlsConfig.RootCAs = caCertPool
		log.Info().
			Str("cert_path", cfg.ServerCert).
			Msg("using custom CA certificate for TLS verification")
	}

	tlsConfig.ServerName = cfg.ServerName

	if cfg.ClientCert != "" || cfg.ClientKey != "" {
		cert, err := tls.LoadX509KeyPair(cfg.ClientCert, cfg.ClientKey)
		if err != nil {
			return fmt.Errorf("could not load client cert and key file: %w", err)
		}

		if tlsConfig.Certificates == nil {
			tlsConfig.Certificates = []tls.Certificate{cert}
		} else {
			tlsConfig.Certificates = append(tlsConfig.Certificates, cert)
		}
	}

	return nil
}
func MergeFromGlobalConfig(cfg *GlobalTLSConfig, tlsConfig *tls.Config) error {
	mergeFromConfig(&cfg.baseConfig, tlsConfig)

	// Handle custom certificate chain
	if len(cfg.TrustBundle) > 0 {
		caCertPool := x509.NewCertPool()

		for _, bundlePath := range cfg.TrustBundle {
			caCert, err := os.ReadFile(bundlePath)
			if err != nil {
				return fmt.Errorf("reading TLS certificate from trust bundle '%s': %w", cfg.TrustBundle, err)
			}

			if !caCertPool.AppendCertsFromPEM(caCert) {
				return fmt.Errorf("failed to parse TLS certificate from trust bundle '%s'", cfg.TrustBundle)
			}
		}

		tlsConfig.RootCAs = caCertPool
	}

	return nil
}
