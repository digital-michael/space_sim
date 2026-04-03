package eventloop

import (
"testing"
"time"
)

func TestFPSControllerBasic(t *testing.T) {
fc := NewFPSController(60.0)

expectedFPS := 60.0
if fc.GetTargetFPS() != expectedFPS {
t.Fatalf("expected FPS %f, got %f", expectedFPS, fc.GetTargetFPS())
}

frameDuration := fc.FrameDuration()
expectedDuration := time.Second / 60
if frameDuration != expectedDuration {
t.Fatalf("expected duration %v, got %v", expectedDuration, frameDuration)
}

deltaTime := fc.DeltaTime()
expectedDelta := 1.0 / 60.0
if deltaTime < expectedDelta-0.001 || deltaTime > expectedDelta+0.001 {
t.Fatalf("expected delta %f, got %f", expectedDelta, deltaTime)
}
}

func TestFPSControllerSetFPS(t *testing.T) {
fc := NewFPSController(60.0)

fc.SetFPS(120.0)

if fc.GetTargetFPS() != 120.0 {
t.Fatalf("expected FPS 120, got %f", fc.GetTargetFPS())
}

if !fc.HasChanged() {
t.Fatal("expected HasChanged to return true")
}

if fc.HasChanged() {
t.Fatal("expected HasChanged to return false on second call")
}
}

func TestFPSControllerDefaults(t *testing.T) {
fc := NewFPSController(-1) // Invalid FPS

if fc.GetTargetFPS() != 60.0 {
t.Fatalf("expected default FPS 60, got %f", fc.GetTargetFPS())
}

fc2 := NewFPSController(2000) // Above max

if fc2.GetTargetFPS() != 1000 {
t.Fatalf("expected max FPS 1000, got %f", fc2.GetTargetFPS())
}
}

func TestFPSControllerInvalidSetFPS(t *testing.T) {
fc := NewFPSController(60.0)

originalFPS := fc.GetTargetFPS()

fc.SetFPS(-1)
if fc.GetTargetFPS() != originalFPS {
t.Fatal("expected FPS to remain unchanged for invalid value")
}

fc.SetFPS(2000)
if fc.GetTargetFPS() != originalFPS {
t.Fatal("expected FPS to remain unchanged for value above max")
}
}
