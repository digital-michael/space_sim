package grpcserver

import (
	"net/http"
	"sync/atomic"

	"connectrpc.com/connect"
)

// connLimitHandler wraps an http.Handler and enforces a maximum number of
// concurrent in-flight requests. When the limit is exceeded it returns a
// ConnectRPC-formatted ResourceExhausted error so all three protocols
// (Connect, gRPC, gRPC-Web) receive a well-formed error response.
type connLimitHandler struct {
	max     int64
	active  atomic.Int64
	wrapped http.Handler
}

// newConnLimitHandler returns an http.Handler that rejects requests once
// active in-flight requests reach max. Pass max <= 0 to disable limiting.
func newConnLimitHandler(max int, wrapped http.Handler) http.Handler {
	if max <= 0 {
		return wrapped
	}
	return &connLimitHandler{max: int64(max), wrapped: wrapped}
}

func (h *connLimitHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	n := h.active.Add(1)
	defer h.active.Add(-1)

	if n > h.max {
		// Write a ConnectRPC-compatible ResourceExhausted error.
		// ConnectRPC's connect.NewError + connect.CodeResourceExhausted produces
		// status 429 with a well-formed JSON body in the Connect protocol. gRPC
		// and gRPC-Web clients see the grpc-status trailer correctly.
		cerr := connect.NewError(connect.CodeResourceExhausted, nil)
		cerr.Meta().Set("x-space-sim-reason", "connection limit reached")
		http.Error(w, cerr.Error(), http.StatusTooManyRequests)
		return
	}

	h.wrapped.ServeHTTP(w, r)
}

// ActiveConns returns the current number of in-flight requests. Safe to call
// from any goroutine; useful for metrics and tests.
func (h *connLimitHandler) ActiveConns() int64 {
	return h.active.Load()
}
