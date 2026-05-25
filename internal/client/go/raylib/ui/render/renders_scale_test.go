package render

import (
	"testing"
)

func TestScaledInt32Default(t *testing.T) {
	SetUIScale(1.0)
	if got := scaledInt32(100); got != 100 {
		t.Errorf("scaledInt32(100) scale=1.0 = %d, want 100", got)
	}
}

func TestScaledInt32DoubleScale(t *testing.T) {
	SetUIScale(2.0)
	if got := scaledInt32(100); got != 200 {
		t.Errorf("scaledInt32(100) scale=2.0 = %d, want 200", got)
	}
	SetUIScale(1.0)
}

func TestSetUIScaleZeroClamp(t *testing.T) {
	SetUIScale(0.0)
	if got := uiScale(); got != 1.0 {
		t.Errorf("uiScale() after SetUIScale(0) = %f, want 1.0", got)
	}
}

func TestSetUIScaleNegativeClamp(t *testing.T) {
	SetUIScale(-3.0)
	if got := uiScale(); got != 1.0 {
		t.Errorf("uiScale() after SetUIScale(-3) = %f, want 1.0", got)
	}
}

func TestSetUIScaleRoundtrip(t *testing.T) {
	SetUIScale(1.5)
	if got := uiScale(); got != 1.5 {
		t.Errorf("uiScale() = %f, want 1.5", got)
	}
	SetUIScale(1.0)
}

func TestScaledInt32HalfScale(t *testing.T) {
	SetUIScale(0.5)
	if got := scaledInt32(80); got != 40 {
		t.Errorf("scaledInt32(80) scale=0.5 = %d, want 40", got)
	}
	SetUIScale(1.0)
}

func TestScaledInt32MinClamp(t *testing.T) {
	// base=0 → math.Round(0) = 0 < 1 → returns 1
	SetUIScale(1.0)
	if got := scaledInt32(0); got != 1 {
		t.Errorf("scaledInt32(0) = %d, want 1 (min clamp)", got)
	}
}

func TestNewRendererHasNoTarget(t *testing.T) {
	r := New(true, true)
	if r.HasRenderTarget() {
		t.Error("new Renderer should have no render target")
	}
}

func TestDisableRenderTargetOnNewRenderer(t *testing.T) {
	r := New(true, true)
	r.DisableRenderTarget() // no-op when no target
	if r.HasRenderTarget() {
		t.Error("render target should remain disabled")
	}
}
