// Command space-sim-admin is the interactive control client for space-sim-server.
// It connects to a running server over gRPC and exposes SimulationService and
// WorldService operations as text commands or a script.
//
// Usage:
//
//	space-sim-admin [flags]
//
// Flags:
//
//	--addr      http://host:port of the space-sim-server (default "http://localhost:9090")
//	--script    path to a script file to execute (one command per line)
//	--decompose path to a system definition directory to split into local
//	            environments; performs offline authoring and exits without
//	            connecting to a server
package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"

	"github.com/digital-michael/space_sim/internal/client/repl"
	"github.com/digital-michael/space_sim/internal/sim/decompose"
)

func main() {
	addr := flag.String("addr", "http://localhost:9090", "space-sim-server address")
	script := flag.String("script", "", "path to a script file to execute")
	decomposeDir := flag.String("decompose", "", "system definition directory to split into local environments (offline; does not connect)")
	minSatellites := flag.Int("decompose-min-satellites", 4, "satellites a body needs before it gets its own environment")
	flag.Parse()

	// Offline authoring: runs without a server and exits. The partitioning core
	// lives in internal/sim/decompose so it can be reused if runtime
	// partitioning is ever wanted.
	if *decomposeDir != "" {
		chunks, err := decompose.Decompose(*decomposeDir, decompose.Options{
			MinSatellites: *minSatellites,
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "decompose: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("decomposed %s into %d environments:\n", *decomposeDir, len(chunks))
		for _, c := range chunks {
			fmt.Printf("  %-34s %-16s roots=%-2d bodies=%d\n",
				c.Dir, c.Label, len(c.Roots), len(c.Members))
		}
		return
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	var in io.Reader = os.Stdin
	if *script != "" {
		f, err := os.Open(*script)
		if err != nil {
			fmt.Fprintf(os.Stderr, "open script: %v\n", err)
			os.Exit(1)
		}
		defer f.Close()
		in = f
	}

	r := repl.New(*addr)
	if err := r.Run(ctx, in); err != nil {
		fmt.Fprintf(os.Stderr, "admin error: %v\n", err)
		os.Exit(1)
	}
}
