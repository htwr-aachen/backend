package config

import "time"

type DB struct {
	DBConnStr     string `koanf:"connection_string" validate:"required"`
	DBConnStrFile string `koanf:"connection_string_file"`

	DBHost         string `koanf:"host" validate:"omitempty,hostname"`
	DBPort         uint   `koanf:"port" validate:"omitempty,port"`
	DBUser         string `koanf:"user"`
	DBUserFile     string `koanf:"user_file"`
	DBPassword     string `koanf:"password"`
	DBPasswordFile string `koanf:"password_file"`
	DBName         string `koanf:"dbname"`
	DBSSLMode      string `koanf:"ssl_mode"`
	DBSSLRootCert  string `koanf:"ssl_root_cert_file" fileconfig:"skip" validate:"omitempty,file"`
	DBSSLCert      string `koanf:"ssl_client_cert_file" fileconfig:"skip" validate:"omitempty,file"`
	DBSSLKey       string `koanf:"ssl_client_key_file" fileconfig:"skip" validate:"omitempty,file"`

	dbSetConfig             bool
	DBMaxConns              int32         `koanf:"max_conns"`
	DBMaxIdleConns          int32         `koanf:"max_idle_conns"`
	DBMinConns              int32         `koanf:"min_conns"`
	DBConnMaxLifetime       time.Duration `koanf:"max_conn_lifetime"`
	DBTimeout               time.Duration `koanf:"ping_timeout"`
	DBConnHealthCheckPeriod time.Duration `koanf:"conn_health_check_period"`
	DBConnMaxIdleLifetime   time.Duration `koanf:"max_idle_conn_lifetime"`
}
