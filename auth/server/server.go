package server

import (
	"fmt"
	"log"
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/status-im/proxy-common/auth/config"
	"github.com/status-im/proxy-common/auth/handlers"
	"github.com/status-im/proxy-common/auth/metrics"
)

type Server struct {
	config         *config.Config
	handlers       *handlers.Handlers
	mux            *http.ServeMux
	enableMetrics  bool
	metricsPath    string
	enableTestMode bool
}

type Option func(*Server)

func WithConfig(cfg *config.Config) Option {
	return func(s *Server) {
		s.config = cfg
	}
}

func WithMetrics(enable bool) Option {
	return func(s *Server) {
		s.enableMetrics = enable
	}
}

func WithMetricsPath(path string) Option {
	return func(s *Server) {
		s.metricsPath = path
	}
}

func WithTestMode(enable bool) Option {
	return func(s *Server) {
		s.enableTestMode = enable
	}
}

func WithCustomMetrics(m metrics.MetricsRecorder) Option {
	return func(s *Server) {
		// This will be applied when creating handlers
		if s.handlers == nil {
			s.handlers = handlers.New(s.config, handlers.WithMetrics(m))
		}
	}
}

func New(opts ...Option) (*Server, error) {
	s := &Server{
		enableMetrics:  true,
		metricsPath:    "/metrics",
		enableTestMode: false,
	}

	for _, opt := range opts {
		opt(s)
	}

	// If no config was provided, try to load from default sources
	if s.config == nil {
		cfg, err := config.Load()
		if err != nil {
			return nil, fmt.Errorf("failed to load config: %w", err)
		}
		s.config = cfg
	}

	if err := s.config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	if s.handlers == nil {
		var metricsRecorder metrics.MetricsRecorder
		if s.enableMetrics {
			metricsRecorder = metrics.NewPrometheusMetrics()
		} else {
			metricsRecorder = metrics.NewNoopMetrics()
		}
		s.handlers = handlers.New(s.config, handlers.WithMetrics(metricsRecorder))
	}

	s.setupRoutes()

	return s, nil
}

func (s *Server) setupRoutes() {
	s.mux = http.NewServeMux()

	s.mux.HandleFunc("/auth/puzzle", s.handlers.PuzzleHandler)
	s.mux.HandleFunc("/auth/solve", s.handlers.SolveHandler)
	s.mux.HandleFunc("/auth/verify", s.handlers.VerifyHandler)
	s.mux.HandleFunc("/auth/status", s.handlers.StatusHandler)

	if s.enableTestMode {
		s.mux.HandleFunc("/dev/test-solve", s.handlers.TestSolveHandler)
	}

	if s.enableMetrics {
		s.mux.Handle(s.metricsPath, promhttp.Handler())
	}
}

func (s *Server) ListenAndServe(addr string) error {
	log.Printf("[go-auth-service] starting on %s", addr)
	log.Printf("[go-auth-service] algorithm: %s, memory: %dKB, time: %d, token expiry: %d minutes",
		s.config.Algorithm, s.config.Argon2Params.MemoryKB, s.config.Argon2Params.Time, s.config.TokenExpiryMinutes)

	if s.enableMetrics {
		log.Printf("[go-auth-service] metrics available at %s", s.metricsPath)
	}

	if s.enableTestMode {
		log.Printf("[go-auth-service] test mode enabled - /dev/test-solve available")
	}

	return http.ListenAndServe(addr, s.mux)
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

// Handler returns the http.Handler for the server
// Useful for integrating with other HTTP servers or routers
func (s *Server) Handler() http.Handler {
	return s.mux
}

func (s *Server) Config() *config.Config {
	return s.config
}

// Quick creates a server with minimal configuration
// Loads config from environment or default file
func Quick() (*Server, error) {
	return New()
}

// MustNew creates a new server and panics on error
// Useful for quick prototyping
func MustNew(opts ...Option) *Server {
	srv, err := New(opts...)
	if err != nil {
		panic(fmt.Sprintf("failed to create server: %v", err))
	}
	return srv
}
