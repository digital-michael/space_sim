// Command space-sim-grpc starts the Space Sim Raylib client wired to an
// embedded ConnectRPC server.
//
// Current state: stub — gRPC server and client wiring are added in Phase 6b.
// The binary builds and exits cleanly so the Makefile target is always valid.
//
// Long-term target (Option B): this binary becomes the client only; the server
// is a separate binary that clients (Go Raylib, JS browser) connect to over the
// network. Player identification and registration on gRPC connection will be
// added as part of that split.
package main

import "fmt"

func main() {
	fmt.Println("space-sim-grpc: Phase 6 gRPC wiring not yet implemented")
}
