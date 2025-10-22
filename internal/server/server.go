package server

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/htwr-aachen/backend/internal/liveness"
	"github.com/htwr-aachen/backend/pkg/admin"
	"github.com/htwr-aachen/backend/pkg/panikzettel"
	"github.com/htwr-aachen/backend/pkg/qa"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

type Config struct {
	PublicHost              string
	PublicPort              string
	PublicReadHeaderTimeout time.Duration
	PublicWriteTimeout      time.Duration
	PublicIdleTimeout       time.Duration

	AdminHost              string
	AdminPort              string
	AdminReadHeaderTimeout time.Duration
	AdminWriteTimeout      time.Duration
	AdminIdleTimeout       time.Duration

	MetricsHost              string
	MetricsPort              string
	MetricsReadHeaderTimeout time.Duration
	MetricsWriteTimeout      time.Duration
	MetricsIdleTimeout       time.Duration

	PanikzettelEnabled bool
	QAEnabled          bool
	AdminEnabled       bool
}

func setDefaults(conf *viper.Viper) {
	conf.SetDefault("qa.host", "::")
	conf.SetDefault("qa.port", "8080")
	conf.SetDefault("qa.read_header_timeout", time.Minute)
	conf.SetDefault("qa.write_timeout", time.Minute)
	conf.SetDefault("qa.idle_timeout", 5*time.Minute)
	conf.SetDefault("admin.host", "::")
	conf.SetDefault("admin.port", "8081")
	conf.SetDefault("admin.read_header_timeout", time.Minute)
	conf.SetDefault("admin.write_timeout", time.Minute)
	conf.SetDefault("admin.idle_timeout", 5*time.Minute)
	conf.SetDefault("metrics.host", "::")
	conf.SetDefault("metrics.port", "9090")
	conf.SetDefault("metrics.read_header_timeout", time.Minute)
	conf.SetDefault("metrics.write_timeout", time.Minute)
	conf.SetDefault("metrics.idle_timeout", 5*time.Minute)
	conf.SetDefault("panikzettel.disabled", false)
	conf.SetDefault("qa.disabled", false)
	conf.SetDefault("admin.disabled", false)

}

func (s *Server) loadConfig(conf *viper.Viper) {
	setDefaults(conf)

	s.config = &Config{
		PublicHost:               conf.GetString("qa.host"),
		PublicPort:               conf.GetString("qa.port"),
		PublicReadHeaderTimeout:  conf.GetDuration("qa.read_header_timeout"),
		PublicWriteTimeout:       conf.GetDuration("qa.write_timeout"),
		PublicIdleTimeout:        conf.GetDuration("qa.idle_timeout"),
		AdminHost:                conf.GetString("admin.host"),
		AdminPort:                conf.GetString("admin.port"),
		AdminReadHeaderTimeout:   conf.GetDuration("admin.read_header_timeout"),
		AdminWriteTimeout:        conf.GetDuration("admin.write_timeout"),
		AdminIdleTimeout:         conf.GetDuration("admin.idle_timeout"),
		MetricsHost:              conf.GetString("metrics.host"),
		MetricsPort:              conf.GetString("metrics.port"),
		MetricsReadHeaderTimeout: conf.GetDuration("metrics.read_header_timeout"),
		MetricsWriteTimeout:      conf.GetDuration("metrics.write_timeout"),
		MetricsIdleTimeout:       conf.GetDuration("metrics.idle_timeout"),

		PanikzettelEnabled: !conf.GetBool("panikzettel.disabled"),
		QAEnabled:          !conf.GetBool("qa.disabled"),
		AdminEnabled:       !conf.GetBool("admin.disabled"),
	}
}

// Server manages the HTTP servers and services
type Server struct {
	config   *Config
	services *Services
}

// Services holds all initialized service handlers
type Services struct {
	Liveness    *liveness.Server
	QA          http.Handler
	Panikzettel http.Handler
	Admin       http.Handler
	closers     []func()
}

// Close cleanly shuts down all services
func (s *Services) Close() {
	for i := len(s.closers) - 1; i >= 0; i-- {
		s.closers[i]()
	}
}

// New creates a new server instance
func New(conf *viper.Viper) (*Server, error) {
	if conf == nil {
		log.Panic().Stack().Msg("nil viper conf given")
	}

	server := &Server{}
	server.loadConfig(conf)
	return server, nil
}

// Run starts the server and blocks until context is cancelled
func (s *Server) Run(ctx context.Context, conf *viper.Viper) error {
	if conf == nil {
		log.Panic().Stack().Msg("nil conf given")
	}

	log.Info().Msg("Starting htwr-aachen backend server")

	// Init services
	services, err := s.initializeServices(ctx, conf)
	if err != nil {
		return fmt.Errorf("failed to initialize services: %w", err)
	}
	defer services.Close()
	s.services = services

	// Setup routers
	publicRouter := s.setupPublicRouter()
	adminRouter := s.setupAdminRouter()
	metricsRouter := s.setupMetricsRouter()

	// Create servers
	publicAddr := net.JoinHostPort(s.config.PublicHost, s.config.PublicPort)
	adminAddr := net.JoinHostPort(s.config.AdminHost, s.config.AdminPort)
	metricsAddr := net.JoinHostPort(s.config.MetricsHost, s.config.MetricsPort)

	separateAdmin := adminAddr != publicAddr
	separateMetrics := metricsAddr != adminAddr && metricsAddr != publicAddr

	publicServer := &http.Server{
		Addr:              publicAddr,
		Handler:           publicRouter,
		ReadHeaderTimeout: s.config.PublicReadHeaderTimeout,
		WriteTimeout:      s.config.PublicWriteTimeout,
		IdleTimeout:       s.config.PublicIdleTimeout,
	}

	adminServer := &http.Server{
		Addr:              adminAddr,
		Handler:           adminRouter,
		ReadHeaderTimeout: s.config.AdminReadHeaderTimeout,
		WriteTimeout:      s.config.AdminWriteTimeout,
		IdleTimeout:       s.config.AdminIdleTimeout,
	}
	metricsServer := &http.Server{
		Addr:              metricsAddr,
		Handler:           metricsRouter,
		ReadHeaderTimeout: s.config.MetricsReadHeaderTimeout,
		WriteTimeout:      s.config.MetricsWriteTimeout,
		IdleTimeout:       s.config.MetricsIdleTimeout,
	}

	// Start servers
	var wg sync.WaitGroup
	errChan := make(chan error, 2)

	// Start public server
	wg.Go(func() {
		log.Info().Str("address", publicAddr).Msg("Starting public API server")
		if err := publicServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errChan <- fmt.Errorf("public server error: %w", err)
		}
	})

	if separateAdmin {
		// Start admin server
		wg.Go(func() {
			log.Info().Str("address", adminAddr).Msg("Starting admin server")
			if err := adminServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				errChan <- fmt.Errorf("admin server error: %w", err)
			}
		})
	}

	if separateMetrics {
		// Start metrics server
		wg.Go(func() {
			log.Info().Str("address", metricsAddr).Msg("Starting metrics server")
			if err := metricsServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				errChan <- fmt.Errorf("metrics server error: %w", err)
			}
		})
	}

	log.Info().Msg("Servers started successfully")
	log.Info().Msgf("Public API available %s", "http://"+publicAddr)
	log.Info().Msgf("Admin interface available %s", "http://"+adminAddr)
	log.Info().Msgf("Metrics API available %s", "http://"+metricsAddr)

	// Wait for context cancellation or server error
	select {
	case <-ctx.Done():
		log.Info().Msg("Shutdown signal received")
	case err := <-errChan:
		return err
	}

	// Graceful shutdown
	log.Info().Msg("Shutting down servers...")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var shutdownWg sync.WaitGroup
	shutdownWg.Go(func() {
		if err := publicServer.Shutdown(shutdownCtx); err != nil {
			log.Err(err).Msg("Main server shutdown error")
		}
	})

	shutdownWg.Go(func() {
		if err := adminServer.Shutdown(shutdownCtx); err != nil {
			log.Err(err).Msg("Admin server shutdown error")
		}
	})

	shutdownWg.Go(func() {
		if err := metricsServer.Shutdown(shutdownCtx); err != nil {
			log.Err(err).Msg("Metrics server shutdown error")
		}
	})

	shutdownWg.Wait()
	wg.Wait()

	log.Info().Msg("Server shutdown complete")
	return nil
}

func (s *Server) initializeServices(ctx context.Context, conf *viper.Viper) (*Services, error) {
	services := &Services{}

	services.Liveness = liveness.NewLivenessServer(nil)
	services.closers = append(services.closers, services.Liveness.Close)

	// Init QA subsystem
	if s.config.QAEnabled {
		log.Info().Msg("Initializing QA service")
		qaHandler, qaCloser, err := qa.Init(ctx, conf)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize qa service: %w", err)
		}
		services.QA = qaHandler
		services.closers = append(services.closers, qaCloser)
	}

	// Init Panikzettel subsystem
	if s.config.PanikzettelEnabled {
		log.Info().Msg("Initializing Panikzettel service")
		panikzettelHandler, panikzettelCloser, err := panikzettel.Init(ctx, conf)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize panikzettel service: %w", err)
		}
		services.Panikzettel = panikzettelHandler
		services.closers = append(services.closers, panikzettelCloser)
	}

	// Init Admin subsystem
	if s.config.AdminEnabled {
		log.Info().Msg("Initializing Admin service")
		adminHandler, adminCloser, err := admin.Init(ctx, conf)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize admin service: %w", err)
		}
		services.Admin = adminHandler
		services.closers = append(services.closers, adminCloser)
	}

	log.Info().Msg("All services initialized successfully")
	return services, nil
}
