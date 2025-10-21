package database

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"time"

	"github.com/htwr-aachen/backend/internal/configurator"
	"github.com/htwr-aachen/backend/internal/validation"
	"github.com/spf13/viper"
)

type DBConfig struct {
	DBConnStr     string `mapstructure:"connection_string" validate:"url"`
	DBConnStrFile string `mapstructure:"connection_string_file"`

	DBHost         string `mapstructure:"host" validate:"omitempty,hostname"`
	DBPort         uint   `mapstructure:"port" validate:"omitempty,port"`
	DBUser         string `mapstructure:"user"`
	DBUserFile     string `mapstructure:"user_file"`
	DBPassword     string `mapstructure:"password"`
	DBPasswordFile string `mapstructure:"password_file"`
	DBName         string `mapstructure:"dbname"`
	DBSSLMode      string `mapstructure:"ssl_mode"`
	DBSSLRootCert  string `mapstructure:"ssl_root_cert_file" fileconfig:"skip" validate:"omitempty,file"`
	DBSSLCert      string `mapstructure:"ssl_client_cert_file" fileconfig:"skip" validate:"omitempty,file"`
	DBSSLKey       string `mapstructure:"ssl_client_key_file" fileconfig:"skip" validate:"omitempty,file"`

	dbSetConfig             bool
	DBMaxConns              int32         `mapstructure:"max_conns"`
	DBMaxIdleConns          int32         `mapstructure:"max_idle_conns"`
	DBMinConns              int32         `mapstructure:"min_conns"`
	DBConnMaxLifetime       time.Duration `mapstructure:"max_conn_lifetime"`
	DBTimeout               time.Duration `mapstructure:"ping_timeout"`
	DBConnHealthCheckPeriod time.Duration `mapstructure:"conn_health_check_period"`
	DBConnMaxIdleLifetime   time.Duration `mapstructure:"max_idle_conn_lifetime"`

	GlobalConfig *configurator.Global `mapstructure:"-"`
}

func setDefaults(conf *viper.Viper) {
	conf.SetDefault("max_conns", 5)
	conf.SetDefault("max_idle_conns", 5)
	conf.SetDefault("min_conns", 1)
	conf.SetDefault("max_conn_lifetime", time.Hour)
	conf.SetDefault("conn_health_check_period", time.Minute)
	conf.SetDefault("max_idle_conn_lifetime", 30*time.Minute)
	conf.SetDefault("ping_timeout", 30*time.Second)

	// Individual connection defaults (useful for local development)
	conf.SetDefault("host", "localhost")
	conf.SetDefault("port", 5432)
	conf.SetDefault("user", "postgres")
	conf.SetDefault("dbname", "postgres")
	conf.SetDefault("sslmode", "disable") // Default to insecure for local dev.
	// No defaults for password or cert paths, as they should be explicitly set.
	conf.SetDefault("ssl_root_cert_file", "")
	conf.SetDefault("ssl_client_cert_file", "")
	conf.SetDefault("ssl_client_key_file", "")
}

func LoadDBConfig(ctx context.Context, parentConf *viper.Viper) (*DBConfig, error) {

	conf := parentConf.Sub("database")
	if conf == nil {
		conf = viper.New()
	}

	setDefaults(conf)

	var config DBConfig
	err := configurator.UnmarshalWithFileResolution(conf, &config)
	if err != nil {
		return nil, fmt.Errorf("unmarshaling database config: %w", err)
	}

	var ok bool
	if config.GlobalConfig, ok = configurator.FromContext(ctx); !ok {
		return nil, fmt.Errorf("no global config in context")
	}

	err = config.createDBConnStr()
	if err != nil {
		return nil, err
	}

	err = validation.Validate.Struct(config)
	if err != nil {
		return nil, fmt.Errorf("validating database configuration: %w", err)
	}

	return &config, nil
}

func (config *DBConfig) createDBConnStr() error {
	if config.DBConnStr == "" {
		if config.DBUser == "" || config.DBName == "" {
			return errors.New("database configuration incomplete: user and dbname are required")
		}

		dsn := url.URL{
			Scheme: "postgres",
			User:   url.UserPassword(config.DBUser, config.DBPassword),
			Host:   net.JoinHostPort(config.DBHost, strconv.FormatUint(uint64(config.DBPort), 10)),
			Path:   "/" + config.DBName,
		}

		query := url.Values{}
		if config.DBSSLMode != "" {
			query.Set("sslmode", config.DBSSLMode)
		}
		if config.DBSSLRootCert != "" {
			query.Set("sslrootcert", config.DBSSLRootCert)
		}
		if config.DBSSLCert != "" {
			query.Set("sslcert", config.DBSSLCert)
		}
		if config.DBSSLKey != "" {
			query.Set("sslkey", config.DBSSLKey)
		}
		dsn.RawQuery = query.Encode()
		config.DBConnStr = dsn.String()
		config.dbSetConfig = true
	}
	return nil
}
