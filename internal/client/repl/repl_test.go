package repl

import (
	"bytes"
	"context"
	"net/http/httptest"
	"strings"
	"testing"

	"net/http"

	"connectrpc.com/connect"
	v1 "github.com/digital-michael/space_sim/api/gen/spacesim/v1"
	"github.com/digital-michael/space_sim/api/gen/spacesim/v1/spacesimv1connect"
	"github.com/google/uuid"
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

	sysPath, sysH := spacesimv1connect.NewSystemServiceHandler(&stubSystem{})
	mux.Handle(sysPath, sysH)
	winPath, winH := spacesimv1connect.NewWindowServiceHandler(&stubWindow{})
	mux.Handle(winPath, winH)
	camPath, camH := spacesimv1connect.NewCameraServiceHandler(&stubCamera{})
	mux.Handle(camPath, camH)
	navPath, navH := spacesimv1connect.NewNavigationServiceHandler(&stubNavigation{})
	mux.Handle(navPath, navH)
	perfPath, perfH := spacesimv1connect.NewPerformanceServiceHandler(&stubPerformance{})
	mux.Handle(perfPath, perfH)
	sdPath, sdH := spacesimv1connect.NewShutdownServiceHandler(&stubShutdown{})
	mux.Handle(sdPath, sdH)

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

func TestREPL_Pause_Resumes(t *testing.T) {
	_, out, r := newTestServer(t)
	// setspeed first so lastSpeed is known, then pause, then resume.
	r.Run(context.Background(), strings.NewReader("setspeed 5\npause\nresume\nquit\n")) //nolint:errcheck
	output := out.String()
	if !strings.Contains(output, "paused") {
		t.Errorf("expected 'paused' in output, got:\n%s", output)
	}
	if !strings.Contains(output, "5") {
		t.Errorf("expected speed '5' in resume output, got:\n%s", output)
	}
}

func TestREPL_SetSpeed_Zero_Accepted(t *testing.T) {
	_, out, r := newTestServer(t)
	r.Run(context.Background(), strings.NewReader("setspeed 0\nquit\n")) //nolint:errcheck
	if !strings.Contains(out.String(), "ok") {
		t.Errorf("expected ack 'ok' for setspeed 0, got:\n%s", out.String())
	}
}

func TestREPL_Bodies_NoSnapshot(t *testing.T) {
	_, out, r := newTestServer(t)
	r.Run(context.Background(), strings.NewReader("bodies\nquit\n")) //nolint:errcheck
	if !strings.Contains(out.String(), "no bodies") {
		t.Errorf("expected 'no bodies' when stream is empty, got:\n%s", out.String())
	}
}

func TestREPL_Inspect_NotFound(t *testing.T) {
	_, out, r := newTestServer(t)
	r.Run(context.Background(), strings.NewReader("inspect Earth\nquit\n")) //nolint:errcheck
	if !strings.Contains(out.String(), "not found") {
		t.Errorf("expected 'not found' error, got:\n%s", out.String())
	}
}

func TestREPL_Status(t *testing.T) {
	_, out, r := newTestServer(t)
	r.Run(context.Background(), strings.NewReader("status\nquit\n")) //nolint:errcheck
	output := out.String()
	if !strings.Contains(output, "speed:") {
		t.Errorf("expected 'speed:' in status output, got:\n%s", output)
	}
	if !strings.Contains(output, "dataset:") {
		t.Errorf("expected 'dataset:' in status output, got:\n%s", output)
	}
	if !strings.Contains(output, "simtime:") {
		t.Errorf("expected 'simtime:' in status output, got:\n%s", output)
	}
}

func TestREPL_Help_ShowsCategories(t *testing.T) {
	_, out, r := newTestServer(t)
	r.Run(context.Background(), strings.NewReader("help\nquit\n")) //nolint:errcheck
	output := out.String()
	for _, want := range []string{"Simulation", "Dataset", "World", "Streaming", "System", "Window", "Camera", "Navigation", "Performance", "Meta"} {
		if !strings.Contains(output, want) {
			t.Errorf("expected %q section in help, got:\n%s", want, output)
		}
	}
}

// ─── Stub handlers for new services ──────────────────────────────────────────

type stubSystem struct{}

func (s *stubSystem) ListSystems(_ context.Context, _ *connect.Request[v1.ListSystemsRequest]) (*connect.Response[v1.ListSystemsResponse], error) {
	return connect.NewResponse(&v1.ListSystemsResponse{
		Version: 1,
		Systems: []*v1.SystemInfo{
			{Label: "solar_system.json", Path: "data/systems/solar_system.json"},
			{Label: "alpha_centauri_system.json", Path: "data/systems/alpha_centauri_system.json"},
		},
	}), nil
}

func (s *stubSystem) GetActiveSystem(_ context.Context, _ *connect.Request[v1.GetActiveSystemRequest]) (*connect.Response[v1.GetActiveSystemResponse], error) {
	return connect.NewResponse(&v1.GetActiveSystemResponse{
		Version: 1,
		Active:  &v1.SystemInfo{Label: "solar_system.json", Path: "data/systems/solar_system.json"},
	}), nil
}

func (s *stubSystem) LoadSystem(_ context.Context, _ *connect.Request[v1.LoadSystemRequest]) (*connect.Response[v1.LoadSystemResponse], error) {
	return connect.NewResponse(&v1.LoadSystemResponse{
		Version: 1,
		Ack:     &v1.CommandAck{EventId: uuid.NewString(), Status: v1.AckStatus_ACK_STATUS_QUEUED},
	}), nil
}

type stubWindow struct{}

func (s *stubWindow) GetWindow(_ context.Context, _ *connect.Request[v1.GetWindowRequest]) (*connect.Response[v1.GetWindowResponse], error) {
	return connect.NewResponse(&v1.GetWindowResponse{
		Version: 1,
		Window:  &v1.WindowState{Width: 1920, Height: 1080, Fullscreen: false, Maximized: false},
	}), nil
}

func (s *stubWindow) SetWindowSize(_ context.Context, _ *connect.Request[v1.SetWindowSizeRequest]) (*connect.Response[v1.SetWindowSizeResponse], error) {
	return connect.NewResponse(&v1.SetWindowSizeResponse{
		Version: 1,
		Ack:     &v1.CommandAck{EventId: uuid.NewString(), Status: v1.AckStatus_ACK_STATUS_QUEUED},
	}), nil
}

func (s *stubWindow) SetWindowMaximize(_ context.Context, _ *connect.Request[v1.SetWindowMaximizeRequest]) (*connect.Response[v1.SetWindowMaximizeResponse], error) {
	return connect.NewResponse(&v1.SetWindowMaximizeResponse{
		Version: 1,
		Ack:     &v1.CommandAck{EventId: uuid.NewString(), Status: v1.AckStatus_ACK_STATUS_QUEUED},
	}), nil
}

func (s *stubWindow) SetWindowRestore(_ context.Context, _ *connect.Request[v1.SetWindowRestoreRequest]) (*connect.Response[v1.SetWindowRestoreResponse], error) {
	return connect.NewResponse(&v1.SetWindowRestoreResponse{
		Version: 1,
		Ack:     &v1.CommandAck{EventId: uuid.NewString(), Status: v1.AckStatus_ACK_STATUS_QUEUED},
	}), nil
}

func (s *stubWindow) SetWindowFullscreen(_ context.Context, _ *connect.Request[v1.SetWindowFullscreenRequest]) (*connect.Response[v1.SetWindowFullscreenResponse], error) {
	return connect.NewResponse(&v1.SetWindowFullscreenResponse{
		Version: 1,
		Ack:     &v1.CommandAck{EventId: uuid.NewString(), Status: v1.AckStatus_ACK_STATUS_QUEUED},
	}), nil
}

type stubCamera struct{}

func (s *stubCamera) GetCamera(_ context.Context, _ *connect.Request[v1.GetCameraRequest]) (*connect.Response[v1.GetCameraResponse], error) {
	return connect.NewResponse(&v1.GetCameraResponse{
		Version: 1,
		Camera: &v1.WireCameraState{
			YawDeg: 90, PitchDeg: -15,
			PosX: 1.0, PosY: 2.0, PosZ: 3.0,
			Mode: "free", TrackTarget: "",
		},
	}), nil
}

func (s *stubCamera) SetCameraOrient(_ context.Context, _ *connect.Request[v1.SetCameraOrientRequest]) (*connect.Response[v1.SetCameraOrientResponse], error) {
	return connect.NewResponse(&v1.SetCameraOrientResponse{
		Version: 1,
		Ack:     &v1.CommandAck{EventId: uuid.NewString(), Status: v1.AckStatus_ACK_STATUS_QUEUED},
	}), nil
}

func (s *stubCamera) SetCameraPosition(_ context.Context, _ *connect.Request[v1.SetCameraPositionRequest]) (*connect.Response[v1.SetCameraPositionResponse], error) {
	return connect.NewResponse(&v1.SetCameraPositionResponse{
		Version: 1,
		Ack:     &v1.CommandAck{EventId: uuid.NewString(), Status: v1.AckStatus_ACK_STATUS_QUEUED},
	}), nil
}

func (s *stubCamera) SetCameraTrack(_ context.Context, _ *connect.Request[v1.SetCameraTrackRequest]) (*connect.Response[v1.SetCameraTrackResponse], error) {
	return connect.NewResponse(&v1.SetCameraTrackResponse{
		Version: 1,
		Ack:     &v1.CommandAck{EventId: uuid.NewString(), Status: v1.AckStatus_ACK_STATUS_QUEUED},
	}), nil
}

func (s *stubCamera) StartOrbit(_ context.Context, _ *connect.Request[v1.StartOrbitRequest]) (*connect.Response[v1.StartOrbitResponse], error) {
	return connect.NewResponse(&v1.StartOrbitResponse{
		Version: 1,
		Ack:     &v1.CommandAck{EventId: uuid.NewString(), Status: v1.AckStatus_ACK_STATUS_QUEUED},
	}), nil
}

type stubNavigation struct{}

func (s *stubNavigation) GetVelocity(_ context.Context, _ *connect.Request[v1.GetVelocityRequest]) (*connect.Response[v1.GetVelocityResponse], error) {
	return connect.NewResponse(&v1.GetVelocityResponse{
		Version:  1,
		Velocity: &v1.Velocity3{X: 0.5, Y: 0, Z: -1.0},
	}), nil
}

func (s *stubNavigation) SetVelocity(_ context.Context, _ *connect.Request[v1.SetVelocityRequest]) (*connect.Response[v1.SetVelocityResponse], error) {
	return connect.NewResponse(&v1.SetVelocityResponse{
		Version: 1,
		Ack:     &v1.CommandAck{EventId: uuid.NewString(), Status: v1.AckStatus_ACK_STATUS_QUEUED},
	}), nil
}

func (s *stubNavigation) JumpTo(_ context.Context, _ *connect.Request[v1.JumpToRequest]) (*connect.Response[v1.JumpToResponse], error) {
	return connect.NewResponse(&v1.JumpToResponse{
		Version: 1,
		Ack:     &v1.CommandAck{EventId: uuid.NewString(), Status: v1.AckStatus_ACK_STATUS_QUEUED},
	}), nil
}

type stubPerformance struct{}

func (s *stubPerformance) GetPerformance(_ context.Context, _ *connect.Request[v1.GetPerformanceRequest]) (*connect.Response[v1.GetPerformanceResponse], error) {
	return connect.NewResponse(&v1.GetPerformanceResponse{
		Version: 1,
		State: &v1.PerformanceState{
			FrustumCulling: true,
			LodEnabled:     true,
			CameraSpeed:    1.5,
			NumWorkers:     4,
		},
	}), nil
}

func (s *stubPerformance) SetPerformance(_ context.Context, _ *connect.Request[v1.SetPerformanceRequest]) (*connect.Response[v1.SetPerformanceResponse], error) {
	return connect.NewResponse(&v1.SetPerformanceResponse{
		Version: 1,
		Ack:     &v1.CommandAck{EventId: uuid.NewString(), Status: v1.AckStatus_ACK_STATUS_QUEUED},
	}), nil
}

type stubShutdown struct{}

func (s *stubShutdown) Shutdown(_ context.Context, _ *connect.Request[v1.ShutdownRequest]) (*connect.Response[v1.ShutdownResponse], error) {
	return connect.NewResponse(&v1.ShutdownResponse{Version: 1}), nil
}

// ─── System integration tests ─────────────────────────────────────────────────

func TestREPL_SystemList(t *testing.T) {
	_, out, r := newTestServer(t)
	r.Run(context.Background(), strings.NewReader("system list\nquit\n")) //nolint:errcheck
	output := out.String()
	if !strings.Contains(output, "solar_system.json") {
		t.Errorf("expected system label in output, got:\n%s", output)
	}
}

func TestREPL_SystemGet(t *testing.T) {
	_, out, r := newTestServer(t)
	r.Run(context.Background(), strings.NewReader("system get\nquit\n")) //nolint:errcheck
	if !strings.Contains(out.String(), "solar_system.json") {
		t.Errorf("expected active system label, got:\n%s", out.String())
	}
}

func TestREPL_SystemLoad(t *testing.T) {
	_, out, r := newTestServer(t)
	r.Run(context.Background(), strings.NewReader("system load data/systems/solar_system.json\nquit\n")) //nolint:errcheck
	if !strings.Contains(out.String(), "ok") {
		t.Errorf("expected ack 'ok', got:\n%s", out.String())
	}
}

func TestREPL_SystemLoad_MissingLabel_PrintsError(t *testing.T) {
	_, out, r := newTestServer(t)
	r.Run(context.Background(), strings.NewReader("system load\nquit\n")) //nolint:errcheck
	if !strings.Contains(out.String(), "label required") {
		t.Errorf("expected label-required error, got:\n%s", out.String())
	}
}

// ─── Window integration tests ─────────────────────────────────────────────────

func TestREPL_WindowGet(t *testing.T) {
	_, out, r := newTestServer(t)
	r.Run(context.Background(), strings.NewReader("window get\nquit\n")) //nolint:errcheck
	output := out.String()
	if !strings.Contains(output, "1920") || !strings.Contains(output, "1080") {
		t.Errorf("expected size in output, got:\n%s", output)
	}
}

func TestREPL_WindowSize(t *testing.T) {
	_, out, r := newTestServer(t)
	r.Run(context.Background(), strings.NewReader("window size 2560x1440\nquit\n")) //nolint:errcheck
	if !strings.Contains(out.String(), "ok") {
		t.Errorf("expected ack 'ok', got:\n%s", out.String())
	}
}

func TestREPL_WindowSize_BadFormat_PrintsError(t *testing.T) {
	_, out, r := newTestServer(t)
	r.Run(context.Background(), strings.NewReader("window size notadim\nquit\n")) //nolint:errcheck
	if !strings.Contains(out.String(), "error") {
		t.Errorf("expected parse error, got:\n%s", out.String())
	}
}

func TestREPL_WindowMaximize(t *testing.T) {
	_, out, r := newTestServer(t)
	r.Run(context.Background(), strings.NewReader("window maximize\nquit\n")) //nolint:errcheck
	if !strings.Contains(out.String(), "ok") {
		t.Errorf("expected ack 'ok', got:\n%s", out.String())
	}
}

func TestREPL_WindowRestore(t *testing.T) {
	_, out, r := newTestServer(t)
	r.Run(context.Background(), strings.NewReader("window restore\nquit\n")) //nolint:errcheck
	if !strings.Contains(out.String(), "ok") {
		t.Errorf("expected ack 'ok', got:\n%s", out.String())
	}
}

func TestREPL_WindowFullOn(t *testing.T) {
	_, out, r := newTestServer(t)
	r.Run(context.Background(), strings.NewReader("window full on\nquit\n")) //nolint:errcheck
	if !strings.Contains(out.String(), "ok") {
		t.Errorf("expected ack 'ok', got:\n%s", out.String())
	}
}

func TestREPL_WindowFullOff(t *testing.T) {
	_, out, r := newTestServer(t)
	r.Run(context.Background(), strings.NewReader("window full off\nquit\n")) //nolint:errcheck
	if !strings.Contains(out.String(), "ok") {
		t.Errorf("expected ack 'ok', got:\n%s", out.String())
	}
}

func TestREPL_WindowFullMissingArg_PrintsError(t *testing.T) {
	_, out, r := newTestServer(t)
	r.Run(context.Background(), strings.NewReader("window full\nquit\n")) //nolint:errcheck
	if !strings.Contains(out.String(), "on|off") {
		t.Errorf("expected on|off error, got:\n%s", out.String())
	}
}

// ─── Camera integration tests ─────────────────────────────────────────────────

func TestREPL_CameraGet(t *testing.T) {
	_, out, r := newTestServer(t)
	r.Run(context.Background(), strings.NewReader("camera get\nquit\n")) //nolint:errcheck
	output := out.String()
	if !strings.Contains(output, "position") || !strings.Contains(output, "orient") {
		t.Errorf("expected camera info, got:\n%s", output)
	}
}

func TestREPL_CameraOrient(t *testing.T) {
	_, out, r := newTestServer(t)
	r.Run(context.Background(), strings.NewReader("camera orient 45 -10\nquit\n")) //nolint:errcheck
	if !strings.Contains(out.String(), "ok") {
		t.Errorf("expected ack 'ok', got:\n%s", out.String())
	}
}

func TestREPL_CameraPosition(t *testing.T) {
	_, out, r := newTestServer(t)
	r.Run(context.Background(), strings.NewReader("camera position 1.0 2.0 3.0\nquit\n")) //nolint:errcheck
	if !strings.Contains(out.String(), "ok") {
		t.Errorf("expected ack 'ok', got:\n%s", out.String())
	}
}

func TestREPL_CameraTrack(t *testing.T) {
	_, out, r := newTestServer(t)
	r.Run(context.Background(), strings.NewReader("camera track Earth\nquit\n")) //nolint:errcheck
	if !strings.Contains(out.String(), "ok") {
		t.Errorf("expected ack 'ok', got:\n%s", out.String())
	}
}

// ─── Navigation integration tests ─────────────────────────────────────────────

func TestREPL_NavStop(t *testing.T) {
	_, out, r := newTestServer(t)
	r.Run(context.Background(), strings.NewReader("nav stop\nquit\n")) //nolint:errcheck
	if !strings.Contains(out.String(), "ok") {
		t.Errorf("expected ack 'ok', got:\n%s", out.String())
	}
}

func TestREPL_NavVelocity(t *testing.T) {
	_, out, r := newTestServer(t)
	r.Run(context.Background(), strings.NewReader("nav velocity\nquit\n")) //nolint:errcheck
	output := out.String()
	if !strings.Contains(output, "velocity") {
		t.Errorf("expected velocity in output, got:\n%s", output)
	}
}

func TestREPL_NavForward(t *testing.T) {
	_, out, r := newTestServer(t)
	r.Run(context.Background(), strings.NewReader("nav forward 0.5\nquit\n")) //nolint:errcheck
	if !strings.Contains(out.String(), "ok") {
		t.Errorf("expected ack 'ok', got:\n%s", out.String())
	}
}

func TestREPL_NavJump(t *testing.T) {
	_, out, r := newTestServer(t)
	r.Run(context.Background(), strings.NewReader("nav jump Earth Saturn\nquit\n")) //nolint:errcheck
	if !strings.Contains(out.String(), "ok") {
		t.Errorf("expected ack 'ok', got:\n%s", out.String())
	}
}

func TestREPL_NavJump_NoNames_PrintsError(t *testing.T) {
	_, out, r := newTestServer(t)
	r.Run(context.Background(), strings.NewReader("nav jump\nquit\n")) //nolint:errcheck
	if !strings.Contains(out.String(), "required") {
		t.Errorf("expected required-names error, got:\n%s", out.String())
	}
}

// ─── Performance integration tests ───────────────────────────────────────────

func TestREPL_PerfGet(t *testing.T) {
	_, out, r := newTestServer(t)
	r.Run(context.Background(), strings.NewReader("perf get\nquit\n")) //nolint:errcheck
	output := out.String()
	if !strings.Contains(output, "frustum_culling") {
		t.Errorf("expected perf knobs in output, got:\n%s", output)
	}
}

func TestREPL_PerfSet(t *testing.T) {
	_, out, r := newTestServer(t)
	r.Run(context.Background(), strings.NewReader("perf set frustum_culling true\nquit\n")) //nolint:errcheck
	if !strings.Contains(out.String(), "ok") {
		t.Errorf("expected ack 'ok', got:\n%s", out.String())
	}
}

func TestREPL_PerfSet_UnknownField_PrintsError(t *testing.T) {
	_, out, r := newTestServer(t)
	r.Run(context.Background(), strings.NewReader("perf set badfield true\nquit\n")) //nolint:errcheck
	if !strings.Contains(out.String(), "unknown field") {
		t.Errorf("expected 'unknown field' error, got:\n%s", out.String())
	}
}

// ─── Shutdown integration test ─────────────────────────────────────────────────

func TestREPL_Shutdown(t *testing.T) {
	_, out, r := newTestServer(t)
	err := r.Run(context.Background(), strings.NewReader("shutdown\n"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out.String(), "shutting down") {
		t.Errorf("expected shutdown message, got:\n%s", out.String())
	}
}

// ─── For-loop scripting tests ─────────────────────────────────────────────────

// stubWorldWithBodies sends a single snapshot containing the given bodies.
type stubWorldWithBodies struct {
	bodies []*v1.BodyState
}

func (s *stubWorldWithBodies) StreamSnapshot(_ context.Context, _ *connect.Request[v1.StreamSnapshotRequest], stream *connect.ServerStream[v1.StreamSnapshotResponse]) error {
	return stream.Send(&v1.StreamSnapshotResponse{
		Version: 1,
		Bodies:  s.bodies,
	})
}

// newTestServerWithWorld builds a test server using a custom world handler.
func newTestServerWithWorld(t *testing.T, world spacesimv1connect.WorldServiceHandler) (*httptest.Server, *bytes.Buffer, *REPL) {
	t.Helper()
	mux := http.NewServeMux()
	stub := &stubSim{speed: 1.0}
	simPath, simH := spacesimv1connect.NewSimulationServiceHandler(stub)
	mux.Handle(simPath, simH)
	worldPath, worldH := spacesimv1connect.NewWorldServiceHandler(world)
	mux.Handle(worldPath, worldH)
	sysPath, sysH := spacesimv1connect.NewSystemServiceHandler(&stubSystem{})
	mux.Handle(sysPath, sysH)
	winPath, winH := spacesimv1connect.NewWindowServiceHandler(&stubWindow{})
	mux.Handle(winPath, winH)
	camPath, camH := spacesimv1connect.NewCameraServiceHandler(&stubCamera{})
	mux.Handle(camPath, camH)
	navPath, navH := spacesimv1connect.NewNavigationServiceHandler(&stubNavigation{})
	mux.Handle(navPath, navH)
	perfPath, perfH := spacesimv1connect.NewPerformanceServiceHandler(&stubPerformance{})
	mux.Handle(perfPath, perfH)
	sdPath, sdH := spacesimv1connect.NewShutdownServiceHandler(&stubShutdown{})
	mux.Handle(sdPath, sdH)

	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	var out bytes.Buffer
	r := New(srv.URL)
	r.out = &out
	return srv, &out, r
}

func TestParseForHeader(t *testing.T) {
	cases := []struct {
		input     string
		group     string
		varName   string
		sliceSpec string
		ok        bool
	}{
		{"for planets as X:", "planets", "X", "", true},
		{"for planets as X", "planets", "X", "", true},  // no trailing colon
		{"for PLANETS AS X:", "planets", "X", "", true}, // case-insensitive keywords
		{"for dwarf_planets as P:", "dwarf_planets", "P", "", true},
		{"for moons as M:", "moons", "M", "", true},
		{"for stars as S:", "stars", "S", "", true},
		{"for asteroids as A:", "asteroids", "A", "", true},
		{"for planets[3:] as X:", "planets", "X", "[3:]", true},
		{"for planets[:10] as X:", "planets", "X", "[:10]", true},
		{"for planets[3:10] as X:", "planets", "X", "[3:10]", true},
		{"for planets[-5:] as X:", "planets", "X", "[-5:]", true},
		{"nav jump X", "", "", "", false},            // not a for header
		{"for planets X:", "", "", "", false},        // missing "as"
		{"for planets:", "", "", "", false},          // missing var and "as"
		{"for:", "", "", "", false},                  // bare for
		{"foreach planets as X:", "", "", "", false}, // wrong keyword
	}
	for _, c := range cases {
		group, varName, sliceSpec, ok := parseForHeader(c.input)
		if ok != c.ok {
			t.Errorf("parseForHeader(%q): ok=%v want %v", c.input, ok, c.ok)
			continue
		}
		if ok && group != c.group {
			t.Errorf("parseForHeader(%q): group=%q want %q", c.input, group, c.group)
		}
		if ok && varName != c.varName {
			t.Errorf("parseForHeader(%q): varName=%q want %q", c.input, varName, c.varName)
		}
		if ok && sliceSpec != c.sliceSpec {
			t.Errorf("parseForHeader(%q): sliceSpec=%q want %q", c.input, sliceSpec, c.sliceSpec)
		}
	}
}

func TestREPL_ForLoop_Planets(t *testing.T) {
	world := &stubWorldWithBodies{
		bodies: []*v1.BodyState{
			{Name: "Mercury", Category: "planet"},
			{Name: "Venus", Category: "planet"},
			{Name: "Earth", Category: "planet"},
		},
	}
	_, out, r := newTestServerWithWorld(t, world)

	// Body: "nav jump X" with blank line terminating the loop, then quit.
	input := "for planets as X:\nnav jump X\n\nquit\n"
	r.Run(context.Background(), strings.NewReader(input)) //nolint:errcheck

	output := out.String()
	// Each planet should produce a queued ack line.
	for _, name := range []string{"Mercury", "Venus", "Earth"} {
		if strings.Count(output, "ok") < 3 {
			t.Errorf("expected 3 'ok' acks (one per planet), got:\n%s", output)
			break
		}
		_ = name // counted above
	}
}

func TestREPL_ForLoop_UnknownGroup(t *testing.T) {
	_, out, r := newTestServer(t)
	input := "for galaxies as X:\nnav jump X\n\nquit\n"
	r.Run(context.Background(), strings.NewReader(input)) //nolint:errcheck
	if !strings.Contains(out.String(), "unknown group") {
		t.Errorf("expected 'unknown group' error, got:\n%s", out.String())
	}
}

func TestREPL_ForLoop_EmptyBody(t *testing.T) {
	world := &stubWorldWithBodies{
		bodies: []*v1.BodyState{
			{Name: "Earth", Category: "planet"},
		},
	}
	_, out, r := newTestServerWithWorld(t, world)
	// Blank line immediately after header — empty body, no commands run.
	input := "for planets as X:\n\ngetspeed\nquit\n"
	r.Run(context.Background(), strings.NewReader(input)) //nolint:errcheck
	// getspeed should still run after the empty loop
	if !strings.Contains(out.String(), "speed =") {
		t.Errorf("expected getspeed to run after empty for-loop, got:\n%s", out.String())
	}
}

func TestREPL_ForLoop_NoBodiesInGroup(t *testing.T) {
	// World has planets but no moons — loop over moons should print a warning.
	world := &stubWorldWithBodies{
		bodies: []*v1.BodyState{
			{Name: "Earth", Category: "planet"},
		},
	}
	_, out, r := newTestServerWithWorld(t, world)
	input := "for moons as M:\nnav jump M\n\nquit\n"
	r.Run(context.Background(), strings.NewReader(input)) //nolint:errcheck
	if !strings.Contains(out.String(), "no bodies found") {
		t.Errorf("expected 'no bodies found' warning, got:\n%s", out.String())
	}
}

func TestREPL_ForLoop_IndentedBody_Spaces(t *testing.T) {
	world := &stubWorldWithBodies{
		bodies: []*v1.BodyState{
			{Name: "Earth", Category: "planet"},
		},
	}
	_, out, r := newTestServerWithWorld(t, world)
	// Body lines indented with spaces — should be accepted as body lines, not ignored.
	input := "for planets as X:\n    nav jump X\n\nquit\n"
	r.Run(context.Background(), strings.NewReader(input)) //nolint:errcheck
	if !strings.Contains(out.String(), "ok") {
		t.Errorf("expected 'ok' ack with space-indented body, got:\n%s", out.String())
	}
}

func TestREPL_ForLoop_IndentedBody_Tabs(t *testing.T) {
	world := &stubWorldWithBodies{
		bodies: []*v1.BodyState{
			{Name: "Earth", Category: "planet"},
		},
	}
	_, out, r := newTestServerWithWorld(t, world)
	// Body lines indented with a tab — should be accepted as body lines, not ignored.
	input := "for planets as X:\n\tnav jump X\n\nquit\n"
	r.Run(context.Background(), strings.NewReader(input)) //nolint:errcheck
	if !strings.Contains(out.String(), "ok") {
		t.Errorf("expected 'ok' ack with tab-indented body, got:\n%s", out.String())
	}
}

func TestREPL_Labels_On(t *testing.T) {
	_, out, r := newTestServer(t)
	r.Run(context.Background(), strings.NewReader("labels on\nquit\n")) //nolint:errcheck
	if !strings.Contains(out.String(), "ok") {
		t.Errorf("expected 'ok' ack for 'labels on', got:\n%s", out.String())
	}
}

func TestREPL_Labels_Off(t *testing.T) {
	_, out, r := newTestServer(t)
	r.Run(context.Background(), strings.NewReader("labels off\nquit\n")) //nolint:errcheck
	if !strings.Contains(out.String(), "ok") {
		t.Errorf("expected 'ok' ack for 'labels off', got:\n%s", out.String())
	}
}

func TestREPL_Labels_Nearest(t *testing.T) {
	_, out, r := newTestServer(t)
	r.Run(context.Background(), strings.NewReader("labels nearest\nquit\n")) //nolint:errcheck
	if !strings.Contains(out.String(), "ok") {
		t.Errorf("expected 'ok' ack for 'labels nearest', got:\n%s", out.String())
	}
}

// ─── comment tests ────────────────────────────────────────────────────────────

func TestStripLineComment_RemovesTrailingComment(t *testing.T) {
	cases := []struct{ in, want string }{
		{"setspeed 5 // comment", "setspeed 5"},
		{"// whole line", ""},
		{"setspeed 5", "setspeed 5"},
		{`orbit "X//Y" 10 1`, `orbit "X//Y" 10 1`}, // inside quotes — preserved
		{"setspeed 5 // a // b", "setspeed 5"},     // only first // matters
	}
	for _, c := range cases {
		got := stripLineComment(c.in)
		if got != c.want {
			t.Errorf("stripLineComment(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestREPL_LineComment_WholeLineIgnored(t *testing.T) {
	_, out, r := newTestServer(t)
	input := "// this whole line is a comment\ngetspeed\nquit\n"
	r.Run(context.Background(), strings.NewReader(input)) //nolint:errcheck
	if !strings.Contains(out.String(), "speed =") {
		t.Errorf("expected getspeed to run after comment line, got:\n%s", out.String())
	}
}

func TestREPL_LineComment_InlineOnSet(t *testing.T) {
	_, out, r := newTestServer(t)
	// RHS should be "20", not "20 // comment"
	input := "set $x 20 // inline comment\nsetspeed $x\nquit\n"
	r.Run(context.Background(), strings.NewReader(input)) //nolint:errcheck
	if !strings.Contains(out.String(), "ok") {
		t.Errorf("expected setspeed ack after inline comment on set, got:\n%s", out.String())
	}
}

func TestREPL_BlockComment_InlineSameLine(t *testing.T) {
	_, out, r := newTestServer(t)
	input := "/* ignore this */ getspeed\nquit\n"
	r.Run(context.Background(), strings.NewReader(input)) //nolint:errcheck
	if !strings.Contains(out.String(), "speed =") {
		t.Errorf("expected getspeed to run after inline block comment, got:\n%s", out.String())
	}
}

func TestREPL_BlockComment_MultiLine(t *testing.T) {
	_, out, r := newTestServer(t)
	input := "/*\nignore line 1\nignore line 2\n*/\ngetspeed\nquit\n"
	r.Run(context.Background(), strings.NewReader(input)) //nolint:errcheck
	if !strings.Contains(out.String(), "speed =") {
		t.Errorf("expected getspeed to run after multi-line block comment, got:\n%s", out.String())
	}
}

func TestREPL_BlockComment_MultiplOnOneLine(t *testing.T) {
	_, out, r := newTestServer(t)
	// Two block comments on one line; command in the middle should execute.
	input := "/* a */ getspeed /* b */\nquit\n"
	r.Run(context.Background(), strings.NewReader(input)) //nolint:errcheck
	if !strings.Contains(out.String(), "speed =") {
		t.Errorf("expected getspeed to run with comments on both sides, got:\n%s", out.String())
	}
}

func TestREPL_ForLoop_BodyWithLineComment(t *testing.T) {
	world := &stubWorldWithBodies{
		bodies: []*v1.BodyState{{Name: "Earth", Category: "planet"}},
	}
	_, out, r := newTestServerWithWorld(t, world)
	input := "for planets as X:\n    nav jump X // jump to it\n\nquit\n"
	r.Run(context.Background(), strings.NewReader(input)) //nolint:errcheck
	if !strings.Contains(out.String(), "ok") {
		t.Errorf("expected nav jump ack with inline comment in body, got:\n%s", out.String())
	}
}

func TestApplyForSlice(t *testing.T) {
	names := []string{"a", "b", "c", "d", "e", "f", "g", "h", "i", "j"} // 10 items
	cases := []struct {
		spec    string
		want    []string
		wantErr bool
	}{
		{"", names, false},     // no slice
		{"[0:]", names, false}, // all
		{"[3:]", []string{"d", "e", "f", "g", "h", "i", "j"}, false}, // skip first 3
		{"[:4]", []string{"a", "b", "c", "d"}, false},                // first 4
		{"[3:7]", []string{"d", "e", "f", "g"}, false},               // middle slice
		{"[-3:]", []string{"h", "i", "j"}, false},                    // last 3
		{"[0:10]", names, false},                                     // full range explicit
		{"[5:5]", []string{}, false},                                 // empty result
		{"[100:]", []string{}, false},                                // start beyond end
		{"[-100:]", names, false},                                    // large negative clamped to 0
		{"[:-3]", nil, true},                                         // negative end → error
		{"[bad:]", nil, true},                                        // non-numeric start
		{"[:bad]", nil, true},                                        // non-numeric end
		{"nope", nil, true},                                          // missing brackets
		{"[1]", nil, true},                                           // missing colon
	}
	for _, c := range cases {
		got, err := applyForSlice(names, c.spec)
		if c.wantErr {
			if err == nil {
				t.Errorf("applyForSlice(%q): expected error, got %v", c.spec, got)
			}
			continue
		}
		if err != nil {
			t.Errorf("applyForSlice(%q): unexpected error: %v", c.spec, err)
			continue
		}
		if len(got) != len(c.want) {
			t.Errorf("applyForSlice(%q): len=%d want %d  got=%v", c.spec, len(got), len(c.want), got)
			continue
		}
		for i := range got {
			if got[i] != c.want[i] {
				t.Errorf("applyForSlice(%q)[%d]: %q want %q", c.spec, i, got[i], c.want[i])
			}
		}
	}
}

func TestREPL_ForLoop_Slice_SkipFirst(t *testing.T) {
	world := &stubWorldWithBodies{
		bodies: []*v1.BodyState{
			{Name: "Mercury", Category: "planet"},
			{Name: "Venus", Category: "planet"},
			{Name: "Earth", Category: "planet"},
			{Name: "Mars", Category: "planet"},
		},
	}
	_, out, r := newTestServerWithWorld(t, world)
	// [2:] should visit only Earth and Mars.
	input := "for planets[2:] as X:\nnav jump X\n\nquit\n"
	r.Run(context.Background(), strings.NewReader(input)) //nolint:errcheck
	output := out.String()
	if strings.Count(output, "ok") != 2 {
		t.Errorf("expected 2 'ok' acks (Earth + Mars), got:\n%s", output)
	}
}

func TestREPL_ForLoop_Slice_LastN(t *testing.T) {
	world := &stubWorldWithBodies{
		bodies: []*v1.BodyState{
			{Name: "Mercury", Category: "planet"},
			{Name: "Venus", Category: "planet"},
			{Name: "Earth", Category: "planet"},
			{Name: "Mars", Category: "planet"},
		},
	}
	_, out, r := newTestServerWithWorld(t, world)
	// [-2:] should visit only Earth and Mars.
	input := "for planets[-2:] as X:\nnav jump X\n\nquit\n"
	r.Run(context.Background(), strings.NewReader(input)) //nolint:errcheck
	output := out.String()
	if strings.Count(output, "ok") != 2 {
		t.Errorf("expected 2 'ok' acks (Earth + Mars), got:\n%s", output)
	}
}

// ─── set $var tests ───────────────────────────────────────────────────────────

func TestParseSetVar_SpaceSeparated(t *testing.T) {
	name, val, ok := parseSetVar("set $speed 15")
	if !ok || name != "$speed" || val != "15" {
		t.Errorf("got (%q, %q, %v), want ($speed, 15, true)", name, val, ok)
	}
}

func TestParseSetVar_EqualsSeparated(t *testing.T) {
	name, val, ok := parseSetVar("set $orbit_count=1")
	if !ok || name != "$orbit_count" || val != "1" {
		t.Errorf("got (%q, %q, %v), want ($orbit_count, 1, true)", name, val, ok)
	}
}

func TestParseSetVar_ColonEqualsSeparated(t *testing.T) {
	name, val, ok := parseSetVar("set $target:=Earth")
	if !ok || name != "$target" || val != "Earth" {
		t.Errorf("got (%q, %q, %v), want ($target, Earth, true)", name, val, ok)
	}
}

func TestParseSetVar_QuotedValue(t *testing.T) {
	name, val, ok := parseSetVar(`set $target "Alpha Centauri A"`)
	if !ok || name != "$target" || val != "Alpha Centauri A" {
		t.Errorf("got (%q, %q, %v), want ($target, Alpha Centauri A, true)", name, val, ok)
	}
}

func TestParseSetVar_NoSigil_NotMatched(t *testing.T) {
	_, _, ok := parseSetVar("set speed 15")
	if ok {
		t.Error("expected no match for set without sigil")
	}
}

func TestParseSetVar_NotSetLine(t *testing.T) {
	_, _, ok := parseSetVar("setspeed 15")
	if ok {
		t.Error("expected no match for 'setspeed'")
	}
}

func TestExpandVars_Basic(t *testing.T) {
	r := &REPL{vars: map[string]string{"$speed": "15", "$target": "Earth"}}
	got := r.expandVars("orbit $target $speed 1")
	want := "orbit Earth 15 1"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestExpandVars_LongestFirst(t *testing.T) {
	// $speed_max must not accidentally match before $speed is tried.
	r := &REPL{vars: map[string]string{"$speed": "10", "$speed_max": "30"}}
	got := r.expandVars("orbit Earth $speed_max 1")
	want := "orbit Earth 30 1"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestExpandVars_NoVars_Passthrough(t *testing.T) {
	r := &REPL{vars: make(map[string]string)}
	line := "orbit Earth 15 1"
	if got := r.expandVars(line); got != line {
		t.Errorf("expected passthrough, got %q", got)
	}
}

func TestREPL_Set_StoresAndSubstitutes(t *testing.T) {
	_, out, r := newTestServer(t)
	// set $speed, then use it in setspeed; stub accepts any value.
	input := "set $speed 20\nsetspeed $speed\nquit\n"
	r.Run(context.Background(), strings.NewReader(input)) //nolint:errcheck
	output := out.String()
	if !strings.Contains(output, `set $speed = "20"`) {
		t.Errorf("expected set confirmation, got:\n%s", output)
	}
	if !strings.Contains(output, "ok") {
		t.Errorf("expected setspeed ack after var substitution, got:\n%s", output)
	}
}
