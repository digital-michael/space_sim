// Package repl implements an interactive read-eval-print loop that connects
// to a running space-sim-grpc server and exposes SimulationService and
// WorldService operations as simple text commands.
//
// Commands are parsed by [commands.Parse]; each maps to one ConnectRPC call.
// Errors from the server are printed and the loop continues — only 'quit'
// or EOF exit cleanly.
package repl

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"connectrpc.com/connect"
	v1 "github.com/digital-michael/space_sim/api/gen/spacesim/v1"
	"github.com/digital-michael/space_sim/api/gen/spacesim/v1/spacesimv1connect"
	"github.com/digital-michael/space_sim/internal/client/commands"
)

// REPL holds the live ConnectRPC client connections and executes parsed
// commands against them.
type REPL struct {
	simClient   spacesimv1connect.SimulationServiceClient
	worldClient spacesimv1connect.WorldServiceClient
	out         io.Writer
}

// New creates a REPL connected to addr (e.g. "http://localhost:9090").
// The HTTP client used is net/http's default — suitable for local Connect
// protocol. For gRPC wire format over HTTP/2 cleartext pass
// connect.WithGRPC() + an h2c transport as opts.
func New(addr string, opts ...connect.ClientOption) *REPL {
	return &REPL{
		simClient:   spacesimv1connect.NewSimulationServiceClient(http.DefaultClient, addr, opts...),
		worldClient: spacesimv1connect.NewWorldServiceClient(http.DefaultClient, addr, opts...),
		out:         os.Stdout,
	}
}

// Run starts the interactive loop. It reads from in (typically os.Stdin)
// until EOF, 'quit', or ctx is cancelled.
func (r *REPL) Run(ctx context.Context, in io.Reader) error {
	scanner := bufio.NewScanner(in)
	r.printf("Space Sim REPL — type 'help' for commands\n")
	r.prompt()

	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		line := strings.TrimSpace(scanner.Text())
		cmd, err := commands.Parse(line)
		if err != nil {
			r.printf("error: %v\n", err)
			r.prompt()
			continue
		}
		if cmd == nil {
			// blank / comment
			r.prompt()
			continue
		}

		done, execErr := r.exec(ctx, cmd)
		if execErr != nil {
			r.printf("error: %v\n", execErr)
		}
		if done {
			return nil
		}
		r.prompt()
	}

	return scanner.Err()
}

// exec dispatches cmd and returns (done=true) when the REPL should exit.
func (r *REPL) exec(ctx context.Context, cmd commands.Cmd) (bool, error) {
	switch c := cmd.(type) {

	case commands.SetSpeed:
		resp, err := r.simClient.SetSpeed(ctx, connect.NewRequest(&v1.SetSpeedRequest{
			SecondsPerSecond: c.SecondsPerSecond,
		}))
		if err != nil {
			return false, err
		}
		ack := resp.Msg.Ack
		r.printf("ok  event_id=%s  status=%s\n", ack.GetEventId(), ack.GetStatus())

	case commands.GetSpeed:
		resp, err := r.simClient.GetSpeed(ctx, connect.NewRequest(&v1.GetSpeedRequest{}))
		if err != nil {
			return false, err
		}
		r.printf("speed = %.4g s/s\n", resp.Msg.SecondsPerSecond)

	case commands.SetDataset:
		level := levelToProto(c.Level)
		resp, err := r.simClient.SetDataset(ctx, connect.NewRequest(&v1.SetDatasetRequest{Level: level}))
		if err != nil {
			return false, err
		}
		ack := resp.Msg.Ack
		r.printf("ok  event_id=%s  status=%s\n", ack.GetEventId(), ack.GetStatus())

	case commands.GetDataset:
		resp, err := r.simClient.GetDataset(ctx, connect.NewRequest(&v1.GetDatasetRequest{}))
		if err != nil {
			return false, err
		}
		r.printf("dataset = %s\n", strings.ToLower(strings.TrimPrefix(resp.Msg.Level.String(), "DATASET_LEVEL_")))

	case commands.GetTime:
		resp, err := r.simClient.GetSimulationTime(ctx, connect.NewRequest(&v1.GetSimulationTimeRequest{}))
		if err != nil {
			return false, err
		}
		r.printf("simulation_time = %.2f s (J2000)\n", resp.Msg.SecondsSinceJ2000)

	case commands.Stream:
		return false, r.runStream(ctx)

	case commands.Help:
		r.printHelp()

	case commands.Quit:
		r.printf("bye\n")
		return true, nil
	}
	return false, nil
}

func (r *REPL) runStream(ctx context.Context) error {
	stream, err := r.worldClient.StreamSnapshot(ctx, connect.NewRequest(&v1.StreamSnapshotRequest{}))
	if err != nil {
		return err
	}
	r.printf("streaming — press Ctrl-C to stop\n")
	for stream.Receive() {
		msg := stream.Msg()
		r.printf("t=%.0f  speed=%.4g  bodies=%d\n",
			msg.SimulationTime, msg.Speed, len(msg.Bodies))
	}
	return stream.Err()
}

func (r *REPL) printHelp() {
	r.printf(`Commands:
  setspeed <n>                  set simulation speed (seconds/second, n > 0)
  getspeed                      print current simulation speed
  setdataset <small|medium|large|huge>  change asteroid dataset
  getdataset                    print active asteroid dataset
  gettime                       print simulation time (seconds since J2000)
  stream                        print a live snapshot line per physics frame (Ctrl-C to stop)
  help                          print this message
  quit / exit                   exit the REPL
`)
}

func (r *REPL) printf(format string, args ...interface{}) {
	fmt.Fprintf(r.out, format, args...)
}

func (r *REPL) prompt() {
	fmt.Fprint(r.out, "> ")
}

// levelToProto converts a validated lower-case level string to the proto enum.
func levelToProto(level string) v1.DatasetLevel {
	switch level {
	case "small":
		return v1.DatasetLevel_DATASET_LEVEL_SMALL
	case "medium":
		return v1.DatasetLevel_DATASET_LEVEL_MEDIUM
	case "large":
		return v1.DatasetLevel_DATASET_LEVEL_LARGE
	case "huge":
		return v1.DatasetLevel_DATASET_LEVEL_HUGE
	default:
		return v1.DatasetLevel_DATASET_LEVEL_UNSPECIFIED
	}
}
