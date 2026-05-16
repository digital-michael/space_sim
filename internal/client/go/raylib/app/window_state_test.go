package app

import "testing"

// TestScaledDimensions_Retina2x verifies that physical pixels are halved on a
// 2× HiDPI (Retina) display. This is the core of the fullscreen width-growth
// defect: GetRenderWidth returns physical pixels on every platform with a HiDPI
// window, but SetWindowSize and all config values use logical pixels. Without
// the division, WindowedWidth doubles on each fullscreen round-trip.
func TestScaledDimensions_Retina2x(t *testing.T) {
	w, h := scaledDimensions(2560, 1440, 2.0, 2.0)
	if w != 1280 || h != 720 {
		t.Fatalf("Retina 2×: got %d×%d, want 1280×720", w, h)
	}
}

// TestScaledDimensions_Standard1x verifies that 1× displays are unchanged.
func TestScaledDimensions_Standard1x(t *testing.T) {
	w, h := scaledDimensions(1280, 720, 1.0, 1.0)
	if w != 1280 || h != 720 {
		t.Fatalf("1×: got %d×%d, want 1280×720", w, h)
	}
}

// TestSyncWindowState_FullscreenRoundTripPreservesWindowedSize verifies that
// WindowedWidth/Height are unchanged after a simulated fullscreen round-trip
// including the one-frame-lag case on both enter and exit.
func TestSyncWindowState_FullscreenRoundTripPreservesWindowedSize(t *testing.T) {
	const (
		windowedW int32 = 1280
		windowedH int32 = 720
		monitorW  int32 = 2560
		monitorH  int32 = 1600
	)
	ctx := &RuntimeContext{
		WindowedWidth:  windowedW,
		WindowedHeight: windowedH,
	}

	// Stable windowed state.
	ctx.SyncWindowState(windowedW, windowedH, false)
	if ctx.WindowedWidth != windowedW || ctx.WindowedHeight != windowedH {
		t.Fatalf("windowed: got %d×%d, want %d×%d",
			ctx.WindowedWidth, ctx.WindowedHeight, windowedW, windowedH)
	}

	// Enter fullscreen: guard set, then fullscreen confirmed.
	ctx.protectWindowedSize = true
	ctx.SyncWindowState(monitorW, monitorH, true)
	if ctx.WindowedWidth != windowedW {
		t.Fatalf("fullscreen frame 1: WindowedWidth=%d, want %d", ctx.WindowedWidth, windowedW)
	}

	// Stay in fullscreen for several frames.
	for i := 0; i < 5; i++ {
		ctx.SyncWindowState(monitorW, monitorH, true)
	}
	if ctx.WindowedWidth != windowedW {
		t.Fatalf("fullscreen stable: WindowedWidth=%d, want %d", ctx.WindowedWidth, windowedW)
	}

	// Exit fullscreen: guard set, one-frame lag (fullscreen=false but still monitor size).
	ctx.protectWindowedSize = true
	ctx.SyncWindowState(monitorW, monitorH, false)
	if ctx.WindowedWidth != windowedW {
		t.Fatalf("exit lag frame: WindowedWidth=%d, want %d", ctx.WindowedWidth, windowedW)
	}

	// After lag, windowed dimensions are restored.
	ctx.SyncWindowState(windowedW, windowedH, false)
	if ctx.WindowedWidth != windowedW || ctx.WindowedHeight != windowedH {
		t.Fatalf("after exit: got %d×%d, want %d×%d",
			ctx.WindowedWidth, ctx.WindowedHeight, windowedW, windowedH)
	}
}

// TestSyncWindowState_EnterFullscreenLagDoesNotCorruptSize verifies that a
// one-frame lag where IsWindowFullscreen() still reports false after
// ToggleFullscreen (macOS async) does not overwrite WindowedWidth/Height.
func TestSyncWindowState_EnterFullscreenLagDoesNotCorruptSize(t *testing.T) {
	const (
		windowedW int32 = 1280
		windowedH int32 = 720
		monitorW  int32 = 2560
		monitorH  int32 = 1600
	)
	ctx := &RuntimeContext{
		WindowedWidth:  windowedW,
		WindowedHeight: windowedH,
	}

	// Guard set on enter; first frame reports false with monitor dimensions.
	ctx.protectWindowedSize = true
	ctx.SyncWindowState(monitorW, monitorH, false)
	if ctx.WindowedWidth != windowedW || ctx.WindowedHeight != windowedH {
		t.Fatalf("enter lag: got %d×%d, want %d×%d",
			ctx.WindowedWidth, ctx.WindowedHeight, windowedW, windowedH)
	}
}
