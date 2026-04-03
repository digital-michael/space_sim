package grpcserver

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/digital-michael/space_sim/api/gen/spacesim/v1/spacesimv1connect"
)

// ServerConfig holds startup options for the ConnectRPC HTTP server.
type ServerConfig struct {
	// Addr is the TCP address to listen on (e.g. ":9090").
	Addr string

	// MaxConns is the maximum number of concurrent connections allowed.
	// 0 means no limit. Enforced in Phase 6d.
	MaxConns int

	// IdleTimeout is how long an idle connection is kept open before being
	// closed. 0 uses net/http's default. Enforced in Phase 6d.
	IdleTimeout time.Duration
}

// DefaultServerConfig returns a ServerConfig with safe defaults.
func DefaultServerConfig() ServerConfig {
	return ServerConfig{
		Addr:        ":9090",
		MaxConns:    100,
		IdleTimeout: 60 * time.Second,
	}
}

// Server wraps an HTTP server that speaks gRPC, gRPC-Web, and the Connect
// protocol simultaneously (ConnectRPC's default).
type Server struct {
	cfg   ServerConfig
	http  *http.Server
}

// New creates a Server. sim and world are the handler implementations; they
// are registered on the HTTP mux before the server starts.
//
// The server speaks all three ConnectRPC protocols (Connect, gRPC, gRPC-Web)
// on the same port. No proxy or separate gRPC-Web gateway is required.
func New(cfg ServerConfig, sim spacesimv1connect.SimulationServiceHandler, world spacesimv1connect.WorldServiceHandler) *Server {
	mux := http.NewServeMux()

	simPath, simHandler := spacesimv1connect.NewSimulationServiceHandler(sim)
	mux.Handle(simPath, simHandler)

	worldPath, worldHandler := spacesimv1connect.NewWorldServiceHandler(world)
	mux.Handle(worldPath, worldHandler)

	httpSrv := &http.Server{
		Addr:        cfg.Addr,
		Handler:     mux,
		IdleTimeout: cfg.IdleTimeout,
		// TODO (Phase 6d): wrap Handler with a connection-limit interceptor.
		// TODO (production): configure TLS via httpSrv.TLSConfig for non-local deployments.
		// For h2c (HTTP/2 cleartext) in production, wrap with golang.org/x/net/http2/h2c.
	}

	return &Server{
		cfg:  cfg,
		http: httpSrv,
	}
}

// Start begins listening and serving. It blocks until the context is cancelled
// or a fatal accept error occurs. Graceful shutdown is attempted on context
// cancellation.
func (s *Server) Start(ctx context.Context) error {
	shutdownDone := make(chan error, 1)

	go func() {
		<-ctx.Done()
		shutCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		shutdownDone <- s.http.Shutdown(shutCtx)
	}()

	if err := s.http.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("grpcserver: listen %s: %w", s.cfg.Addr, err)
	}

	return <-shutdownDone
}

// Addr returns the configured listen address.
func (s *Server) Addr() string {
	return s.cfg.Addr
}
