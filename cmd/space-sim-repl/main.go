// Command space-sim-repl is an interactive command-line client for a running
// space-sim-grpc server. It dials the server over the Connect protocol and
// exposes SimulationService and WorldService operations as text commands.
//
// Usage:
//
//	space-sim-repl [--addr http://localhost:9090] [--script path/to/script.txt]
//
// Run 'help' inside the REPL for the full command reference.
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
)

func main() {
	addr := flag.String("addr", "http://localhost:9090", "space-sim-grpc server address")
	script := flag.String("script", "", "path to a script file to replay (one command per line)")
	flag.Parse()

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
		fmt.Fprintf(os.Stderr, "repl error: %v\n", err)
		os.Exit(1)
	}
}
