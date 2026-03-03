package server

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/htwr-aachen/backend/internal/configurator"
	"github.com/htwr-aachen/backend/internal/liveness"
	"github.com/htwr-aachen/backend/pkg/admin"
	"github.com/htwr-aachen/backend/pkg/config"
	"github.com/htwr-aachen/backend/pkg/panikzettel"
	"github.com/htwr-aachen/backend/pkg/qa"
	"github.com/quic-go/quic-go"
	"github.com/quic-go/quic-go/http3"
	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"golang.org/x/sync/errgroup"
)

type Startable interface {
	Start() error
	Shutdown(ctx context.Context) error
	String() string
}

// Server manages the HTTP servers and services
type Server struct {
	cfg      *config.Config
	services *Services
	servers  []Startable
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
func New(ctx context.Context) (*Server, error) {
	cfg, ok := configurator.FromContext(ctx)
	if !ok {
		log.Panic().Stack().Msg("no configuration context")
	}

	server := &Server{
		cfg:     cfg,
		servers: make([]Startable, 0),
	}

	err := server.setup(ctx)
	if err != nil {
		return nil, err
	}

	return server, nil
}

func (s *Server) setup(ctx context.Context) error {
	var err error
	log.Info().Msg("Initializing services and preparing servers...")

	s.services, err = s.initializeServices(ctx)
	if err != nil {
		return fmt.Errorf("failed to initialize services: %w", err)
	}

	// setup Routers
	publicRouter := s.setupPublicRouter()
	adminRouterBase := s.setupAdminRouter()
	adminRouter := otelhttp.NewHandler(adminRouterBase, "admin-api")
	metricsRouter := s.setupMetricsRouter()

	publicAddr := net.JoinHostPort(s.cfg.Public.Host, strconv.Itoa(s.cfg.Public.Port))
	adminAddr := net.JoinHostPort(s.cfg.Admin.Host, strconv.Itoa(s.cfg.Admin.Port))
	metricsAddr := net.JoinHostPort(s.cfg.MetricsServer.Host, strconv.Itoa(s.cfg.MetricsServer.Port))

	err = s.AddFullStackServer("Public", publicAddr, publicRouter,
		s.cfg.Public.ReadHeaderTimeout, s.cfg.Public.WriteTimeout, s.cfg.Public.IdleTimeout, &s.cfg.Public.ServerTLS)
	if err != nil {
		return err
	}

	if adminAddr != publicAddr {
		err = s.AddFullStackServer("Admin", adminAddr, adminRouter,
			s.cfg.Admin.ReadHeaderTimeout, s.cfg.Admin.WriteTimeout, s.cfg.Admin.IdleTimeout, &s.cfg.Admin.ServerTLS)
		if err != nil {
			return err
		}
	}

	if metricsAddr != adminAddr && metricsAddr != publicAddr {
		err = s.AddFullStackServer("Metrics", metricsAddr, metricsRouter,
			s.cfg.MetricsServer.ReadHeaderTimeout, s.cfg.MetricsServer.WriteTimeout, s.cfg.MetricsServer.IdleTimeout, &s.cfg.MetricsServer.ServerTLS)
		if err != nil {
			return err
		}
	}

	log.Info().Int("count", len(s.servers)).Msg("Server instances configured")
	return nil
}

// Run starts the server and blocks until context is cancelled
func (s *Server) Run(ctx context.Context) error {

	log.Info().Msg("Starting htwr-aachen backend server")
	if s.services == nil {
		return errors.New("server not setup: create with New() before Run()")
	}
	defer s.services.Close()

	g, gCtx := errgroup.WithContext(ctx)

	// Start all registered servers
	for _, srv := range s.servers {
		serverInstance := srv
		g.Go(func() error {
			if _, ok := srv.(*TLSReloader); !ok {
				log.Info().Str("server", serverInstance.String()).Msg("Starting server")
			}

			if err := serverInstance.Start(); err != nil {
				// If a server fails to start (e.g. port bind error), we return error
				// which cancels gCtx and triggers shutdown for everyone else
				return fmt.Errorf("%s failed: %w", serverInstance.String(), err)
			}
			return nil
		})
	}

	log.Info().Msg("All servers running. Waiting for shutdown signal...")

	// Wait for context cancellation (signal) or error in one of the servers
	<-gCtx.Done()

	if ctx.Err() != nil {
		log.Warn().Msg("Shutdown signal received (OS Signal)")
	} else {
		log.Error().Msg("One or more servers failed to start. Shutting down")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var wg sync.WaitGroup
	for _, srv := range s.servers {
		serverInstance := srv
		wg.Go(func() {
			log.Info().Str("server", serverInstance.String()).Msg("Shutting down")
			if err := serverInstance.Shutdown(shutdownCtx); err != nil {
				log.Error().Err(err).Str("server", serverInstance.String()).Msg("Shutdown error")
			}
		})
	}

	wg.Wait()
	log.Warn().Msg("Shutdown complete")

	if err := g.Wait(); err != nil {
		// This will reveal "bind: address already in use" or similar
		return err
	}

	return nil
}

func (s *Server) initializeServices(ctx context.Context) (*Services, error) {
	services := &Services{}

	services.Liveness = liveness.NewLivenessServer(nil)
	services.closers = append(services.closers, services.Liveness.Close)

	// Init QA subsystem
	if s.cfg.QA.Enabled {
		log.Info().Msg("Initializing QA service")
		qaHandler, qaCloser, err := qa.Init(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize qa service: %w", err)
		}
		services.QA = qaHandler
		services.closers = append(services.closers, qaCloser)
	}

	// Init Panikzettel subsystem
	if s.cfg.Panikzettel.Enabled {
		log.Info().Msg("Initializing Panikzettel service")
		panikzettelHandler, panikzettelCloser, err := panikzettel.Init(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize panikzettel service: %w", err)
		}
		services.Panikzettel = panikzettelHandler
		services.closers = append(services.closers, panikzettelCloser)
	}

	// Init Admin subsystem
	if s.cfg.Admin.Enabled {
		log.Info().Msg("Initializing Admin service")
		adminHandler, adminCloser, err := admin.Init(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize admin service: %w", err)
		}
		services.Admin = adminHandler
		services.closers = append(services.closers, adminCloser)
	}

	log.Info().Msg("All services initialized successfully")
	return services, nil
}

// httpServer wraps net/http.Server to satisfy Runnable
type httpServer struct {
	name   string
	server *http.Server
	isTLS  bool
}

func (h *httpServer) Start() error {
	// ListenAndServe always returns a non-nil error.
	// ErrServerClosed is normal during shutdown.
	var err error
	if h.isTLS {
		err = h.server.ListenAndServeTLS("", "")
	} else {
		err = h.server.ListenAndServe()
	}

	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return fmt.Errorf("http server %s failed: %w", h.name, err)
}

func (h *httpServer) Shutdown(ctx context.Context) error {
	return h.server.Shutdown(ctx)
}

func (h *httpServer) String() string {
	scheme := "HTTP"
	if h.isTLS {
		scheme = "HTTPS/HTTP2"
	}
	return fmt.Sprintf("%s-%s (%s)", h.name, scheme, h.server.Addr)
}

type http3Server struct {
	name   string
	server *http3.Server
}

func (h *http3Server) Start() error {
	err := h.server.ListenAndServe()
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}

	return fmt.Errorf("http3 server %s failed: %w", h.name, err)
}

func (h *http3Server) Shutdown(ctx context.Context) error {
	return h.server.Shutdown(ctx)
}

func (h *http3Server) String() string {
	return fmt.Sprintf("%s-HTTP3/QUIC (%s)", h.name, h.server.Addr)
}

// Helper to add a standard HTTP server to the list
func (s *Server) addHTTPServer(name, addr string, handler http.Handler, read, write, idle time.Duration) {
	srv := &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadHeaderTimeout: read,
		WriteTimeout:      write,
		IdleTimeout:       idle,
	}
	s.servers = append(s.servers, &httpServer{
		name:   name,
		server: srv,
	})
}

// addSecureServer adds a HTTPS server (HTTP/1.1 + HTTP/2)
func (s *Server) addSecureServer(name, addr string, tlsConfig *tls.Config, handler http.Handler, read, write, idle time.Duration) {
	srv := &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadHeaderTimeout: read,
		WriteTimeout:      write,
		IdleTimeout:       idle,
		TLSConfig:         tlsConfig,
	}

	// Go's net/http enables HTTP/2 automatically when ListenAndServeTLS is called.

	s.servers = append(s.servers, &httpServer{
		name:   name,
		server: srv,
		isTLS:  tlsConfig != nil,
	})
}

// addHTTP3Server adds a QUIC-based HTTP/3 server
func (s *Server) addHTTP3Server(name, addr string, tlsConfig *tls.Config, handler http.Handler, idle time.Duration) {
	srv := &http3.Server{
		Addr:        addr,
		Handler:     handler,
		IdleTimeout: idle,
		TLSConfig:   http3.ConfigureTLSConfig(tlsConfig),
		QUICConfig:  &quic.Config{},
		// QuicConfig can be added here for fine-tuning
	}

	s.servers = append(s.servers, &http3Server{
		name:   name,
		server: srv,
	})
}

func (s *Server) AddFullStackServer(name, addr string, handler http.Handler, read, write, idle time.Duration, tlsSrvCfg *config.ServerTLSConfig) error {

	if tlsSrvCfg == nil || tlsSrvCfg.ServerCert == "" || tlsSrvCfg.ServerKey == "" {
		s.addHTTPServer(name, addr, handler, read, write, idle)
		return nil
	}

	reloader, err := NewTLSReloader(tlsSrvCfg.ServerCert, tlsSrvCfg.ServerKey, 30*time.Minute)
	if err != nil {
		return fmt.Errorf("failed to initialize tls reloader for %s: %w", name, err)
	}
	s.servers = append(s.servers, reloader)

	baseTLS := &tls.Config{
		GetCertificate: reloader.GetCertificateFunc,
		MinVersion:     tlsSrvCfg.MinVersion,
		MaxVersion:     tlsSrvCfg.MaxVersion,
		NextProtos:     []string{"h2", "http/1.1"},
	}

	_, portStr, err := net.SplitHostPort(addr)
	if err != nil {
		return err
	}
	port, err := strconv.Atoi(portStr)
	if err != nil {
		return err
	}
	quicServer := &http3.Server{Addr: addr, Port: port}
	h3Handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.ProtoMajor < 3 {
			err := quicServer.SetQUICHeaders(w.Header())
			if err != nil {
				log.Err(err).Msg("adding quic headers")
			}
		}
		handler.ServeHTTP(w, r)
	})

	s.addSecureServer(name+"-tcp", addr, baseTLS.Clone(), h3Handler, read, write, idle)

	quicServer.IdleTimeout = idle
	quicServer.TLSConfig = http3.ConfigureTLSConfig(baseTLS.Clone())
	quicServer.Handler = h3Handler

	s.servers = append(s.servers, &http3Server{
		name:   name,
		server: quicServer,
	})
	return nil
}
