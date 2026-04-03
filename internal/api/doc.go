// Package api defines the transport-agnostic contract between Space Sim's
// server and client layers.
//
// Each interface in this package is a port in the ports-and-adapters sense:
// the server implements the server-side ports, clients implement the
// client-side ports, and any transport (in-process call, gRPC, WebSocket)
// is an adapter that bridges the two.
//
// Sub-files:
//
//   - client.go  Camera control and player point-of-view contracts
//   - server.go  Simulation control and animation contracts
//
// None of the interfaces carry concrete types from internal/sim/engine or
// internal/client so that any implementation (Raylib, web, headless test)
// can satisfy them without pulling in unrelated dependencies.
package api
