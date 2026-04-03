package repl

import (
	"bytes"
	"context"
	"net/http/httptest"
	"strings"
	"testing"

	"connectrpc.com/connect"
	v1 "github.com/digital-michael/space_sim/api/gen/spacesim/v1"
	"github.com/digital-michael/space_sim/api/gen/spacesim/v1/spacesimv1connect"
	"github.com/google/uuid"
	"net/http"
)

// ─── Stub server handlers ────────────────────────────────────────────────

type stubSim struct {
	speed float32
}

func (s *stubSim) GetType() string { return "stub" }

func (s *stubSim) SetSpeed(_ context.Context, req *connect.Request[v1.SetSpeedRequest]) (*connect.Response[v1.SetSpeedResponse], error) {
	s.speed = req.Msg.SecondsPerSecond
	return connect.NewResponse(&v1.SetSpeedResponse{
		Version: 1,
		Ack:     &v1.CommandAck{EventId: uuid.NewString(), Status: v1.AckStatus_ACK_STATUS_QUEUED},
	}), nil
}

func (s *stubSim) GetSpeed(_ context.Context, _ *connect.Request[v1.GetSpeedRequest]) (*connect.Response[v1.GetSpeedResponse], error) {
	return connect.NewResponse(&v1.GetSpeedResponse{Version: 1, SecondsPerSecond: s.speed}), nil
}

func (s *stubSim) SetDataset(_ context.Context, req *connect.Request[v1.SetDatasetRequest]) (*connect.Response[v1.SetDatasetResponse], error) {
	return connect.NewResponse(&v1.SetDatasetResponse{
		Version: 1,
		Ack:     &v1.CommandAck{EventId: uuid.NewString(), Status: v1.AckStatus_ACK_STATUS_QUEUED},
	}), nil
}

func (s *stubSim) GetDataset(_ context.Context, _ *connect.Request[v1.GetDatasetRequest]) (*connect.Response[v1.GetDatasetResponse], error) {
	return connect.NewResponse(&v1.GetDatasetResponse{Version: 1, Level: v1.DatasetLevel_DATASET_LEVEL_SMALL}), nil
}

func (s *stubSim) GetSimulationTime(_ context.Context, _ *connect.Request[v1.GetSimulationTimeRequest]) (*connect.Response[v1.GetSimulationTimeResponse], error) {
	return connect.NewResponse(&v1.GetSimulationTimeResponse{Version: 1, SecondsSinceJ2000: 812345678.0}), nil
}

type stubWorld struct{}

func (s *stubWorld) StreamSnapshot(_ context.Context, _ *connect.Request[v1.StreamSnapshotRequest], _ *connect.ServerStream[v1.StreamSnapshotResponse]) error {
	return nil
}

func newTestServer(t *testing.T) (*httptest.Server, *bytes.Buffer, *REPL) {
	t.Helper()
	mux := http.NewServeMux()
	stub := &stubSim{speed: 1.0}
	simPath, simH := spacesimv1connect.NewSimulationServiceHandler(stub)
	mux.Handle(simPath, simH)
	worldPath, worldH := spacesimv1connect.NewWorldServiceHandler(&stubWorld{})
	mux.Handle(worldPath, worldH)

	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	var out bytes.Buffer
	r := New(srv.URL)
	r.out = &out
	return srv, &out, r
}

// ─── Tests ───────────────────────────────────────────────────────────────

func TestREPL_GetSpeed(t *testing.T) {
	_, out, r := newTestServer(t)
	r.Run(context.Background(), strings.NewReader("getspeed\nquit\n")) //nolint:errcheck
	if !strings.Contains(out.String(), "speed =") {
		t.Errorf("expected 'speed =' in output, got:\n%s", out.String())
	}
}

func TestREPL_SetSpeed_RoundTrip(t *testing.T) {
	_, out, r := newTestServer(t)
	r.Run(context.Background(), strings.NewReader("setspeed 20\ngetspeed\nquit\n")) //nolint:errcheck
	output := out.String()
	if !strings.Contains(output, "ok") {
		t.Errorf("expected ack 'ok' after setspeed, got:\n%s", output)
	}
	if !strings.Contains(output, "20") {
		t.Errorf("expected '20' in getspeed output, got:\n%s", output)
	}
}

func TestREPL_GetDataset(t *testing.T) {
	_, out, r := newTestServer(t)
	r.Run(context.Background(), strings.NewReader("getdataset\nquit\n")) //nolint:errcheck
	if !strings.Contains(out.String(), "small") {
		t.Errorf("expected 'small' in output, got:\n%s", out.String())
	}
}

func TestREPL_GetTime(t *testing.T) {
	_, out, r := newTestServer(t)
	r.Run(context.Background(), strings.NewReader("gettime\nquit\n")) //nolint:errcheck
	if !strings.Contains(out.String(), "812345678") {
		t.Errorf("expected simulation time in output, got:\n%s", out.String())
	}
}

func TestREPL_Help(t *testing.T) {
	_, out, r := newTestServer(t)
	r.Run(context.Background(), strings.NewReader("help\nquit\n")) //nolint:errcheck
	if !strings.Contains(out.String(), "setspeed") {
		t.Errorf("expected help text, got:\n%s", out.String())
	}
}

func TestREPL_UnknownCommand_PrintsError(t *testing.T) {
	_, out, r := newTestServer(t)
	r.Run(context.Background(), strings.NewReader("frobulate\nquit\n")) //nolint:errcheck
	if !strings.Contains(out.String(), "unknown command") {
		t.Errorf("expected unknown-command error, got:\n%s", out.String())
	}
}

func TestREPL_Quit_ExitsCleanly(t *testing.T) {
	_, out, r := newTestServer(t)
	err := r.Run(context.Background(), strings.NewReader("quit\n"))
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if !strings.Contains(out.String(), "bye") {
		t.Errorf("expected 'bye', got:\n%s", out.String())
	}
}

func TestREPL_EOF_ExitsCleanly(t *testing.T) {
	_, _, r := newTestServer(t)
	err := r.Run(context.Background(), strings.NewReader(""))
	if err != nil {
		t.Errorf("unexpected error on EOF: %v", err)
	}
}

func TestREPL_SetDataset(t *testing.T) {
	_, out, r := newTestServer(t)
	r.Run(context.Background(), strings.NewReader("setdataset large\nquit\n")) //nolint:errcheck
	if !strings.Contains(out.String(), "ok") {
		t.Errorf("expected ack 'ok' after setdataset, got:\n%s", out.String())
	}
}
