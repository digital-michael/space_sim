package grpcserver

import (
	"context"
	"errors"
	"testing"
	"time"

	"connectrpc.com/connect"
	v1 "github.com/digital-michael/space_sim/api/gen/spacesim/v1"
	rayapp "github.com/digital-michael/space_sim/internal/client/go/raylib/app"
	"github.com/digital-michael/space_sim/internal/sim/engine"
)

// ─── stub helpers ─────────────────────────────────────────────────────────────

// nopSend accepts every command and drops it.
func nopSend(rayapp.AppCmd) bool { return true }

// fullSend simulates a full channel — always returns false.
func fullSend(rayapp.AppCmd) bool { return false }

// captureSend captures the first command sent by a handler and returns true.
// Useful when verifying the type / contents of a queued cmd.
func captureSend(captured *rayapp.AppCmd) func(rayapp.AppCmd) bool {
	return func(cmd rayapp.AppCmd) bool {
		*captured = cmd
		return true
	}
}

// ─── ShutdownHandler ──────────────────────────────────────────────────────────

func TestShutdownHandler_CallsCancel(t *testing.T) {
	called := false
	h := NewShutdownHandler(func() { called = true })
	_, err := h.Shutdown(context.Background(), connect.NewRequest(&v1.ShutdownRequest{}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !called {
		t.Error("cancel was not called")
	}
}

// ─── SystemHandler ─────────────────────────────────────────────────────────────

func TestSystemHandler_LoadSystem_EmptyPath_InvalidArgument(t *testing.T) {
	h := NewSystemHandler(nopSend)
	_, err := h.LoadSystem(context.Background(), connect.NewRequest(&v1.LoadSystemRequest{Path: ""}))
	assertCode(t, err, connect.CodeInvalidArgument)
}

func TestSystemHandler_LoadSystem_PathTraversal_InvalidArgument(t *testing.T) {
	h := NewSystemHandler(nopSend)
	for _, bad := range []string{"../../etc/passwd", "../secret.json", "data/systems/../../ok"} {
		_, err := h.LoadSystem(context.Background(), connect.NewRequest(&v1.LoadSystemRequest{Path: bad}))
		if err == nil {
			t.Errorf("path %q: expected error, got nil", bad)
			continue
		}
		var cerr *connect.Error
		if errors.As(err, &cerr) && cerr.Code() != connect.CodeInvalidArgument {
			t.Errorf("path %q: want CodeInvalidArgument, got %v", bad, cerr.Code())
		}
	}
}

func TestSystemHandler_LoadSystem_ValidPath_QueuesCmd(t *testing.T) {
	var captured rayapp.AppCmd
	h := NewSystemHandler(captureSend(&captured))
	path := "data/systems/solar_system.json"
	resp, err := h.LoadSystem(context.Background(), connect.NewRequest(&v1.LoadSystemRequest{Path: path}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Msg.Ack == nil {
		t.Fatal("expected non-nil ack")
	}
	lc, ok := captured.(rayapp.LoadSystemCmd)
	if !ok {
		t.Fatalf("expected LoadSystemCmd, got %T", captured)
	}
	if lc.Path != path {
		t.Errorf("path: want %q, got %q", path, lc.Path)
	}
}

func TestSystemHandler_LoadSystem_FullChannel_Unavailable(t *testing.T) {
	h := NewSystemHandler(fullSend)
	_, err := h.LoadSystem(context.Background(), connect.NewRequest(&v1.LoadSystemRequest{Path: "data/systems/solar_system.json"}))
	assertCode(t, err, connect.CodeUnavailable)
}

func TestSystemHandler_GetActiveSystem_CtxCancel_DeadlineExceeded(t *testing.T) {
	// sendCmd that never fills respCh — ctx cancel must unblock the handler.
	h := NewSystemHandler(nopSend)
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately
	_, err := h.GetActiveSystem(ctx, connect.NewRequest(&v1.GetActiveSystemRequest{}))
	assertCode(t, err, connect.CodeDeadlineExceeded)
}

func TestSystemHandler_GetActiveSystem_RoundTrip(t *testing.T) {
	const wantPath = "data/systems/solar_system.json"
	send := func(cmd rayapp.AppCmd) bool {
		gc, ok := cmd.(rayapp.GetActiveSystemCmd)
		if !ok {
			return false
		}
		gc.RespCh <- wantPath
		return true
	}
	h := NewSystemHandler(send)
	resp, err := h.GetActiveSystem(context.Background(), connect.NewRequest(&v1.GetActiveSystemRequest{}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := resp.Msg.GetActive().GetPath(); got != wantPath {
		t.Errorf("path: want %q, got %q", wantPath, got)
	}
}

// ─── WindowHandler ─────────────────────────────────────────────────────────────

func TestWindowHandler_SetWindowSize_ZeroDims_InvalidArgument(t *testing.T) {
	h := NewWindowHandler(nopSend)
	for _, tc := range []struct{ w, h int32 }{{0, 1080}, {1920, 0}, {-1, 1080}} {
		_, err := h.SetWindowSize(context.Background(), connect.NewRequest(&v1.SetWindowSizeRequest{
			Width: tc.w, Height: tc.h,
		}))
		if err == nil {
			t.Errorf("(%d,%d): expected error, got nil", tc.w, tc.h)
			continue
		}
		assertCode(t, err, connect.CodeInvalidArgument)
	}
}

func TestWindowHandler_SetWindowSize_ValidDims_QueuesCmd(t *testing.T) {
	var captured rayapp.AppCmd
	h := NewWindowHandler(captureSend(&captured))
	_, err := h.SetWindowSize(context.Background(), connect.NewRequest(&v1.SetWindowSizeRequest{Width: 1920, Height: 1080}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	sc, ok := captured.(rayapp.WindowSizeCmd)
	if !ok {
		t.Fatalf("expected WindowSizeCmd, got %T", captured)
	}
	if sc.Width != 1920 || sc.Height != 1080 {
		t.Errorf("size: want 1920x1080, got %dx%d", sc.Width, sc.Height)
	}
}

func TestWindowHandler_GetWindow_CtxCancel_DeadlineExceeded(t *testing.T) {
	h := NewWindowHandler(nopSend)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := h.GetWindow(ctx, connect.NewRequest(&v1.GetWindowRequest{}))
	assertCode(t, err, connect.CodeDeadlineExceeded)
}

func TestWindowHandler_GetWindow_RoundTrip(t *testing.T) {
	send := func(cmd rayapp.AppCmd) bool {
		gc, ok := cmd.(rayapp.GetWindowCmd)
		if !ok {
			return false
		}
		gc.RespCh <- rayapp.WindowSnapshot{Width: 2560, Height: 1440, Fullscreen: true}
		return true
	}
	h := NewWindowHandler(send)
	resp, err := h.GetWindow(context.Background(), connect.NewRequest(&v1.GetWindowRequest{}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	w := resp.Msg.GetWindow()
	if w.GetWidth() != 2560 || w.GetHeight() != 1440 {
		t.Errorf("size: want 2560x1440, got %dx%d", w.GetWidth(), w.GetHeight())
	}
	if !w.GetFullscreen() {
		t.Error("fullscreen: want true")
	}
}

func TestWindowHandler_SetWindowMaximize_FullChannel_Unavailable(t *testing.T) {
	h := NewWindowHandler(fullSend)
	_, err := h.SetWindowMaximize(context.Background(), connect.NewRequest(&v1.SetWindowMaximizeRequest{}))
	assertCode(t, err, connect.CodeUnavailable)
}

// ─── CameraHandler ─────────────────────────────────────────────────────────────

func TestCameraHandler_GetCamera_CtxCancel_DeadlineExceeded(t *testing.T) {
	h := NewCameraHandler(nopSend)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := h.GetCamera(ctx, connect.NewRequest(&v1.GetCameraRequest{}))
	assertCode(t, err, connect.CodeDeadlineExceeded)
}

func TestCameraHandler_GetCamera_RoundTrip(t *testing.T) {
	send := func(cmd rayapp.AppCmd) bool {
		gc, ok := cmd.(rayapp.GetCameraCmd)
		if !ok {
			return false
		}
		gc.RespCh <- rayapp.CameraSnapshot{
			YawDeg:   45.0,
			PitchDeg: -10.0,
			Position: engine.Vector3{X: 1.0, Y: 2.0, Z: 3.0},
			Mode:     0, // free
		}
		return true
	}
	h := NewCameraHandler(send)
	resp, err := h.GetCamera(context.Background(), connect.NewRequest(&v1.GetCameraRequest{}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	cam := resp.Msg.GetCamera()
	if cam.GetYawDeg() != 45.0 {
		t.Errorf("yaw: want 45.0, got %v", cam.GetYawDeg())
	}
	if cam.GetPosX() != 1.0 || cam.GetPosY() != 2.0 || cam.GetPosZ() != 3.0 {
		t.Errorf("pos: want (1,2,3), got (%v,%v,%v)", cam.GetPosX(), cam.GetPosY(), cam.GetPosZ())
	}
}

func TestCameraHandler_SetCameraOrient_QueuesCmd(t *testing.T) {
	var captured rayapp.AppCmd
	h := NewCameraHandler(captureSend(&captured))
	_, err := h.SetCameraOrient(context.Background(), connect.NewRequest(&v1.SetCameraOrientRequest{YawDeg: 90, PitchDeg: -15}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	oc, ok := captured.(rayapp.CameraOrientCmd)
	if !ok {
		t.Fatalf("expected CameraOrientCmd, got %T", captured)
	}
	if oc.YawDeg != 90 || oc.PitchDeg != -15 {
		t.Errorf("orient: want (90,-15), got (%v,%v)", oc.YawDeg, oc.PitchDeg)
	}
}

// ─── NavigationHandler ────────────────────────────────────────────────────────

func TestNavigationHandler_JumpTo_EmptyNames_InvalidArgument(t *testing.T) {
	h := NewNavigationHandler(nopSend)
	_, err := h.JumpTo(context.Background(), connect.NewRequest(&v1.JumpToRequest{Names: nil}))
	assertCode(t, err, connect.CodeInvalidArgument)
}

func TestNavigationHandler_JumpTo_ValidNames_QueuesCmd(t *testing.T) {
	var captured rayapp.AppCmd
	h := NewNavigationHandler(captureSend(&captured))
	_, err := h.JumpTo(context.Background(), connect.NewRequest(&v1.JumpToRequest{Names: []string{"Earth", "Saturn"}}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	jc, ok := captured.(rayapp.JumpToCmd)
	if !ok {
		t.Fatalf("expected JumpToCmd, got %T", captured)
	}
	if len(jc.Names) != 2 || jc.Names[0] != "Earth" || jc.Names[1] != "Saturn" {
		t.Errorf("names: want [Earth Saturn], got %v", jc.Names)
	}
}

func TestNavigationHandler_GetVelocity_CtxCancel_DeadlineExceeded(t *testing.T) {
	h := NewNavigationHandler(nopSend)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := h.GetVelocity(ctx, connect.NewRequest(&v1.GetVelocityRequest{}))
	assertCode(t, err, connect.CodeDeadlineExceeded)
}

func TestNavigationHandler_GetVelocity_RoundTrip(t *testing.T) {
	send := func(cmd rayapp.AppCmd) bool {
		gc, ok := cmd.(rayapp.GetVelocityCmd)
		if !ok {
			return false
		}
		gc.RespCh <- engine.Vector3{X: 0.5, Y: 0, Z: -1.0}
		return true
	}
	h := NewNavigationHandler(send)
	resp, err := h.GetVelocity(context.Background(), connect.NewRequest(&v1.GetVelocityRequest{}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	vel := resp.Msg.GetVelocity()
	if vel.GetX() != 0.5 || vel.GetZ() != -1.0 {
		t.Errorf("velocity: want (0.5,0,-1), got (%v,%v,%v)", vel.GetX(), vel.GetY(), vel.GetZ())
	}
}

func TestNavigationHandler_SetVelocity_FullChannel_Unavailable(t *testing.T) {
	h := NewNavigationHandler(fullSend)
	_, err := h.SetVelocity(context.Background(), connect.NewRequest(&v1.SetVelocityRequest{}))
	assertCode(t, err, connect.CodeUnavailable)
}

// ─── PerformanceHandler ───────────────────────────────────────────────────────

func TestPerformanceHandler_GetPerformance_CtxCancel_DeadlineExceeded(t *testing.T) {
	h := NewPerformanceHandler(nopSend)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := h.GetPerformance(ctx, connect.NewRequest(&v1.GetPerformanceRequest{}))
	assertCode(t, err, connect.CodeDeadlineExceeded)
}

func TestPerformanceHandler_GetPerformance_RoundTrip(t *testing.T) {
	send := func(cmd rayapp.AppCmd) bool {
		gc, ok := cmd.(rayapp.GetPerfCmd)
		if !ok {
			return false
		}
		gc.RespCh <- rayapp.PerfSnapshot{CameraSpeed: 2.5, NumWorkers: 4}
		return true
	}
	h := NewPerformanceHandler(send)
	resp, err := h.GetPerformance(context.Background(), connect.NewRequest(&v1.GetPerformanceRequest{}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	st := resp.Msg.GetState()
	if st.GetCameraSpeed() != 2.5 {
		t.Errorf("camera_speed: want 2.5, got %v", st.GetCameraSpeed())
	}
	if st.GetNumWorkers() != 4 {
		t.Errorf("workers: want 4, got %v", st.GetNumWorkers())
	}
}

func TestPerformanceHandler_SetPerformance_FullChannel_Unavailable(t *testing.T) {
	h := NewPerformanceHandler(fullSend)
	_, err := h.SetPerformance(context.Background(), connect.NewRequest(&v1.SetPerformanceRequest{
		State:             &v1.PerformanceState{FrustumCulling: true},
		SetFrustumCulling: true,
	}))
	assertCode(t, err, connect.CodeUnavailable)
}

// ─── isAllowedSystemPath (edge cases) ─────────────────────────────────────────

func TestIsAllowedSystemPath(t *testing.T) {
	cases := []struct {
		path  string
		allow bool
	}{
		{"solar_system.json", true},
		{"data/systems/solar_system.json", true},
		{"data/systems/alpha_centauri_system.json", true},
		{"../etc/passwd", false},
		{"data/systems/../../secret", false},
		{"../../root", false},
		{"/absolute/path.json", false},
	}
	for _, tc := range cases {
		got := isAllowedSystemPath(tc.path)
		if got != tc.allow {
			t.Errorf("isAllowedSystemPath(%q) = %v; want %v", tc.path, got, tc.allow)
		}
	}
}

// ─── helpers ──────────────────────────────────────────────────────────────────

// assertCode fails the test unless err is a *connect.Error with the given code.
func assertCode(t *testing.T, err error, want connect.Code) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected error with code %v, got nil", want)
	}
	var cerr *connect.Error
	if !errors.As(err, &cerr) {
		t.Fatalf("expected *connect.Error, got %T: %v", err, err)
	}
	if cerr.Code() != want {
		t.Errorf("code: want %v, got %v", want, cerr.Code())
	}
}

// Ensure the time package is used (via context timeout tests).
var _ = time.Second
