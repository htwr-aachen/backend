package configurator

import (
	"errors"
	"net"
	"net/url"
	"strconv"

	"github.com/htwr-aachen/backend/pkg/config"
)

func dbHook(cfg *config.Config) error {

	err := createDBConnStr(&cfg.Database)
	return err

}

func createDBConnStr(config *config.DB) error {
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
	}
	return nil
}
