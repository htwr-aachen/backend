package defaults

import (
	"time"

	"github.com/htwr-aachen/backend/pkg/config"
)

func GetDefaultConfig() *config.Config {
	return &config.Config{
		Global: config.Global{
			TLS:         config.GlobalTLSConfig{},
			InsecureDev: false,
			Metrics: config.Metrics{
				Prefix:  "",
				Enabled: true,
			},
			OpenTelemetry: config.OpenTelemetry{
				Enabled:     false,
				ServiceName: "htwr-backend",
				Endpoint:    "http://localhost:4318",
			},
		},
		Database: config.DB{
			DBHost:                  "localhost",
			DBPort:                  5432,
			DBUser:                  "postgres",
			DBName:                  "postgres",
			DBSSLMode:               "disable",
			DBMaxConns:              5,
			DBMaxIdleConns:          5,
			DBMinConns:              1,
			DBConnMaxLifetime:       time.Hour,
			DBTimeout:               30 * time.Second,
			DBConnHealthCheckPeriod: 5 * time.Minute,
			DBConnMaxIdleLifetime:   30 * time.Minute,
		},
		Session: config.Session{
			Disabled:              false,
			CacheExpiration:       time.Hour * 24,
			CacheCleanupInterval:  2 * time.Hour * 24,
			SessionCookieName:     "session",
			SessionCookieSecure:   true,
			SessionCookieHttpOnly: true,
			SessionCookieSameSite: "lax",
			SessionExpiration:     time.Hour * 24,
			RoleMap: map[string]string{
				"admin": "admin",
				"user":  "user",
			},
			AuthURLPrefix: "/auth",
		},
		Public: config.Public{
			Enabled:           false,
			Host:              "::",
			Port:              8080,
			ReadHeaderTimeout: time.Minute,
			WriteTimeout:      time.Minute,
			IdleTimeout:       5 * time.Minute,
		},
		Admin: config.Admin{
			Enabled:           true,
			Host:              "::",
			Port:              8081,
			ReadHeaderTimeout: time.Minute,
			WriteTimeout:      time.Minute,
			IdleTimeout:       5 * time.Minute,
			Metrics: config.Metrics{
				Enabled: true,
			},
		},
		QA: config.QA{
			Enabled: true,
			Metrics: config.Metrics{
				Enabled: true,
			},
			APIConfig: config.QAAPI{
				PaginationLimitDefault: 50,
				PaginationLimitMax:     150,
				CorsConfig: config.CORS{
					AllowedOrigins:   []string{"http://localhost:3000"},
					AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
					AllowedHeaders:   []string{"Accept", "Content-Type", "Authorization"},
					AllowCredentials: false,
				},
			},
		},
		Panikzettel: config.Panikzettel{
			Enabled:      true,
			AutoDownload: false,
			Metrics: config.Metrics{
				Enabled: false,
			},
			BaseURL:              "/api/panikzettel",
			MetadataFilename:     "metadata.json",
			MaxFileSize:          512 * 1024 * 1024, // 512MB
			CacheDuration:        time.Hour * 6,
			CacheCleanupInterval: 2 * time.Hour * 6,
			CloudConfig: config.CloudConfig{
				Provider:   "aws",
				AuthMethod: "default",
			},
			Downloads: config.PanikzettelDownloads{
				Enabled:       true,
				FlushInterval: 30 * time.Second,
				FlushTimeout:  10 * time.Second,
			},
		},

		MetricsServer: config.MetricsServer{
			Enabled:           true,
			Host:              "::",
			Port:              9090,
			ReadHeaderTimeout: time.Minute,
			WriteTimeout:      time.Minute,
			IdleTimeout:       5 * time.Minute,
		},
	}
}
