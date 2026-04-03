package eventloop

import (
"testing"
"time"
)

func TestWorkerMetricsBasic(t *testing.T) {
m := NewWorkerMetrics(0)

stats := m.GetStats()
if stats.WorkerID != 0 {
t.Fatalf("expected worker ID 0, got %d", stats.WorkerID)
}
if stats.EventsProcessed != 0 {
t.Fatalf("expected 0 events, got %d", stats.EventsProcessed)
}
if stats.Errors != 0 {
t.Fatalf("expected 0 errors, got %d", stats.Errors)
}
}

func TestWorkerMetricsRecordEvent(t *testing.T) {
m := NewWorkerMetrics(0)

m.RecordEvent(10 * time.Millisecond)
m.RecordEvent(20 * time.Millisecond)
m.RecordEvent(30 * time.Millisecond)

stats := m.GetStats()
if stats.EventsProcessed != 3 {
t.Fatalf("expected 3 events, got %d", stats.EventsProcessed)
}

expectedAvg := time.Duration(20 * time.Millisecond)
if stats.AverageTimePerEvent != expectedAvg {
t.Fatalf("expected avg %v, got %v", expectedAvg, stats.AverageTimePerEvent)
}
}

func TestWorkerMetricsErrors(t *testing.T) {
m := NewWorkerMetrics(1)

m.RecordError(nil)
m.RecordError(nil)

stats := m.GetStats()
if stats.Errors != 2 {
t.Fatalf("expected 2 errors, got %d", stats.Errors)
}
}

func TestLoopMetricsFrameTiming(t *testing.T) {
m := NewLoopMetrics()

m.RecordFrameTime(16 * time.Millisecond)
m.RecordFrameTime(17 * time.Millisecond)
m.RecordFrameTime(15 * time.Millisecond)

m.RecordFrameSuccess()
m.RecordFrameSuccess()
m.RecordFrameSuccess()

stats := m.GetStats()
if stats.FramesProcessed != 3 {
t.Fatalf("expected 3 frames, got %d", stats.FramesProcessed)
}
if stats.MinFrameTime != 15*time.Millisecond {
t.Fatalf("expected min 15ms, got %v", stats.MinFrameTime)
}
if stats.MaxFrameTime != 17*time.Millisecond {
t.Fatalf("expected max 17ms, got %v", stats.MaxFrameTime)
}
if stats.LastFrameTime != 15*time.Millisecond {
t.Fatalf("expected last 15ms, got %v", stats.LastFrameTime)
}
}

func TestLoopMetricsGetActualFPS(t *testing.T) {
m := NewLoopMetrics()

// Record frames at 60 FPS (16.67ms per frame)
for i := 0; i < 10; i++ {
m.RecordFrameSuccess()
m.RecordFrameTime(time.Second / 60)
}

fps := m.GetActualFPS()
expectedFPS := 60.0
tolerance := 1.0
if fps < expectedFPS-tolerance || fps > expectedFPS+tolerance {
t.Fatalf("expected FPS ~%f, got %f", expectedFPS, fps)
}
}

func TestLoopMetricsErrors(t *testing.T) {
m := NewLoopMetrics()

m.RecordFrameSuccess()
m.RecordFrameError()
m.RecordFrameSuccess()
m.RecordFrameError()

stats := m.GetStats()
if stats.FramesProcessed != 2 {
t.Fatalf("expected 2 successful frames, got %d", stats.FramesProcessed)
}
if stats.FrameErrors != 2 {
t.Fatalf("expected 2 errors, got %d", stats.FrameErrors)
}
}
