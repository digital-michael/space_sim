// Package grpcserver provides the ConnectRPC transport layer for Space Sim.
//
// It owns the HTTP server lifecycle, handler registration, connection limiting,
// and idle timeout enforcement. Handlers in this package delegate to
// internal/server packages; they never import internal/client.
//
// Transport shape:
//
//	SimulationService — unary RPCs for simulation commands and queries.
//	WorldService      — server-streaming RPC delivering WorldSnapshot frames.
//
// Connection management and idle-timeout interceptors are added in Phase 6d.
package grpcserver
