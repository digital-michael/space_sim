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

	// MaxConns is the maximum number of concurrent in-flight requests allowed.
	// Excess requests receive a ResourceExhausted error immediately.
	// 0 means no limit.
	MaxConns int

	// IdleTimeout is how long an idle connection is kept open before being
	// closed by net/http. 0 uses net/http's default (no idle timeout).
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
	cfg     ServerConfig
	http    *http.Server
	limiter *connLimitHandler // nil when MaxConns == 0
}

// New creates a Server. sim and world are the handler implementations; they
// are registered on the HTTP mux before the server starts.
//
// Connection limiting (MaxConns) is enforced via a middleware wrapper.
// Idle timeout (IdleTimeout) is enforced by net/http natively.
//
// The server speaks all three ConnectRPC protocols (Connect, gRPC, gRPC-Web)
// on the same port. No proxy or gRPC-Web gateway is required.
//
// NOTE (production): for non-local deployments wrap the mux with h2c
// (golang.org/x/net/http2/h2c) for HTTP/2 cleartext, or configure
// httpSrv.TLSConfig for TLS.
func New(cfg ServerConfig, sim spacesimv1connect.SimulationServiceHandler, world spacesimv1connect.WorldServiceHandler) *Server {
	mux := http.NewServeMux()

	simPath, simHandler := spacesimv1connect.NewSimulationServiceHandler(sim)
	mux.Handle(simPath, simHandler)

	worldPath, worldHandler := spacesimv1connect.NewWorldServiceHandler(world)
	mux.Handle(worldPath, worldHandler)

	var handler http.Handler = mux
	var limiter *connLimitHandler

	if cfg.MaxConns > 0 {
		l := newConnLimitHandler(cfg.MaxConns, mux).(*connLimitHandler)
		limiter = l
		handler = l
	}

	httpSrv := &http.Server{
		Addr:        cfg.Addr,
		Handler:     handler,
		IdleTimeout: cfg.IdleTimeout,
	}

	return &Server{
		cfg:     cfg,
		http:    httpSrv,
		limiter: limiter,
	}
}

// Start begins listening and serving. It blocks until the context is cancelled
// or a fatal accept error occurs. Graceful shutdown is attempted on context
// cancellation with a 5-second deadline.
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

// ActiveConns returns the number of currently in-flight requests.
// Returns 0 when MaxConns is 0 (no limiting configured).
func (s *Server) ActiveConns() int64 {
	if s.limiter == nil {
		return 0
	}
	return s.limiter.ActiveConns()
}
