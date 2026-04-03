package eventloop

import (
	"context"
	"testing"
	"time"

	"github.com/digital-michael/space_sim/internal/server/eventqueue"
	"github.com/digital-michael/space_sim/internal/server/pool/group"
	"github.com/digital-michael/space_sim/internal/server/routines"
	runtimepkg "github.com/digital-michael/space_sim/internal/server/runtime"
	"github.com/google/uuid"
)

// testObjID is a fixed UUID used across loop integration tests.
var testObjID = uuid.MustParse("11111111-1111-1111-1111-111111111111")

// newTestLoop is a helper that creates an EventLoop with sensible test defaults.
func newTestLoop(workerCount int) *EventLoop {
	defs := group.NewPool()
	rt := runtimepkg.NewRuntimeEnvironment(defs)
	mgr := eventqueue.NewQueueManager(100)
	lib := routines.NewLibrary()
	return NewEventLoop(defs, rt, mgr, lib, 60.0, workerCount)
}

func TestEventLoopCreation(t *testing.T) {
	loop := newTestLoop(1)

	if loop.IsRunning() {
		t.Fatal("expected loop to not be running initially")
	}

	target, actual := loop.GetFPS()
	if target != 60.0 {
		t.Fatalf("expected target FPS 60, got %f", target)
	}
	if actual != 0 {
		t.Fatalf("expected initial actual FPS 0, got %f", actual)
	}
}

func TestEventLoopStartStop(t *testing.T) {
	loop := newTestLoop(1)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := loop.Start(ctx); err != nil {
		t.Fatalf("start failed: %v", err)
	}
	if !loop.IsRunning() {
		t.Fatal("expected loop to be running")
	}
	if loop.workerPool.IsRunning() {
		t.Fatal("expected worker pool to remain stopped; event processing is frame-owned")
	}

	if err := loop.Stop(); err != nil {
		t.Fatalf("stop failed: %v", err)
	}
	if loop.IsRunning() {
		t.Fatal("expected loop to be stopped after Stop()")
	}
}

func TestEventLoopUsesFrameOwnedEventProcessing(t *testing.T) {
	defs := group.NewPool()
	if err := defs.CreateObject(testObjID, "cube", nil); err != nil {
		t.Fatalf("CreateObject: %v", err)
	}

	rt := runtimepkg.NewRuntimeEnvironment(defs)
	if err := rt.InitializeObject(testObjID, runtimepkg.OriginPosition(), runtimepkg.Vector3{}); err != nil {
		t.Fatalf("InitializeObject: %v", err)
	}

	mgr := eventqueue.NewQueueManager(100)
	lib := routines.NewLibrary()
	loop := NewEventLoop(defs, rt, mgr, lib, 60.0, 2)

	event := eventqueue.NewEvent(
		testObjID,
		eventqueue.EventTypeUpdate,
		map[string]interface{}{
			"position": map[string]interface{}{"x": 9.0, "y": 1.0, "z": 0.0},
		},
		eventqueue.TransactionTypeNone,
	)
	if err := mgr.Enqueue(event); err != nil {
		t.Fatalf("Enqueue: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := loop.Start(ctx); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer loop.Stop()

	if loop.workerPool.IsRunning() {
		t.Fatal("expected worker pool to remain stopped while loop is running")
	}

	time.Sleep(100 * time.Millisecond)

	state, err := rt.GetObjectState(testObjID)
	if err != nil {
		t.Fatalf("GetObjectState: %v", err)
	}
	if state.Position.X != 9.0 || state.Position.Y != 1.0 {
		t.Fatalf("expected frame pipeline to apply queued event, got %+v", state.Position)
	}
	if loop.workerPool.IsRunning() {
		t.Fatal("expected worker pool to remain stopped after frame processing")
	}
}

func TestEventLoopDoubleStart(t *testing.T) {
	loop := newTestLoop(1)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := loop.Start(ctx); err != nil {
		t.Fatalf("first start failed: %v", err)
	}
	if err := loop.Start(ctx); err == nil {
		t.Fatal("expected second start to fail")
	}

	_ = loop.Stop()
}

func TestEventLoopSetFPS(t *testing.T) {
	loop := newTestLoop(1)

	if err := loop.SetFPS(120.0); err != nil {
		t.Fatalf("setFPS failed: %v", err)
	}

	target, _ := loop.GetFPS()
	if target != 120.0 {
		t.Fatalf("expected FPS 120, got %f", target)
	}

	if err := loop.SetFPS(-1); err == nil {
		t.Fatal("expected invalid FPS to fail")
	}
}

func TestEventLoopMetrics(t *testing.T) {
	loop := newTestLoop(1)

	stats := loop.GetMetrics()
	if stats.FramesProcessed != 0 {
		t.Fatalf("expected 0 frames, got %d", stats.FramesProcessed)
	}
	if stats.FrameErrors != 0 {
		t.Fatalf("expected 0 errors, got %d", stats.FrameErrors)
	}
}

func TestEventLoopGetWorkerMetrics(t *testing.T) {
	loop := newTestLoop(2)

	metrics := loop.GetWorkerMetrics()
	if len(metrics) != 2 {
		t.Fatalf("expected 2 worker metrics, got %d", len(metrics))
	}
}

func TestEventLoopGracefulShutdown(t *testing.T) {
	loop := newTestLoop(1)

	ctx, cancel := context.WithCancel(context.Background())

	if err := loop.Start(ctx); err != nil {
		t.Fatalf("start failed: %v", err)
	}

	time.Sleep(50 * time.Millisecond)
	cancel()

	if err := loop.Stop(); err != nil {
		t.Fatalf("stop failed: %v", err)
	}
}

// TestEventLoopFramesAccumulate verifies frames are counted while running.
func TestEventLoopFramesAccumulate(t *testing.T) {
	loop := newTestLoop(1)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := loop.Start(ctx); err != nil {
		t.Fatalf("start failed: %v", err)
	}
	defer loop.Stop()

	time.Sleep(150 * time.Millisecond)

	stats := loop.GetMetrics()
	if stats.FramesProcessed == 0 {
		t.Fatalf("expected frames to have been processed, got 0")
	}
}

// TestEventLoopIntegration enqueues a real update event, starts the loop,
// and verifies the runtime state reflects the applied change.
func TestEventLoopIntegration(t *testing.T) {
	// Setup: pool with a registered object definition.
	defs := group.NewPool()
	if err := defs.CreateObject(testObjID, "cube", nil); err != nil {
		t.Fatalf("CreateObject: %v", err)
	}

	rt := runtimepkg.NewRuntimeEnvironment(defs)
	if err := rt.InitializeObject(testObjID, runtimepkg.OriginPosition(), runtimepkg.Vector3{}); err != nil {
		t.Fatalf("InitializeObject: %v", err)
	}

	mgr := eventqueue.NewQueueManager(100)
	lib := routines.NewLibrary()

	// Enqueue an update event that sets position X = 7.
	event := eventqueue.NewEvent(
		testObjID,
		eventqueue.EventTypeUpdate,
		map[string]interface{}{
			"position": map[string]interface{}{"x": 7.0, "y": 0.0, "z": 0.0},
		},
		eventqueue.TransactionTypeNone,
	)
	if err := mgr.Enqueue(event); err != nil {
		t.Fatalf("Enqueue: %v", err)
	}

	loop := NewEventLoop(defs, rt, mgr, lib, 60.0, 1)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := loop.Start(ctx); err != nil {
		t.Fatalf("Start: %v", err)
	}

	// Give the loop at least two frames to process the event.
	time.Sleep(100 * time.Millisecond)

	if err := loop.Stop(); err != nil {
		t.Fatalf("Stop: %v", err)
	}

	state, err := rt.GetObjectState(testObjID)
	if err != nil {
		t.Fatalf("GetObjectState: %v", err)
	}
	// Position.X should be 7 after the update event was processed.
	if state.Position.X != 7.0 {
		t.Errorf("expected Position.X = 7.0 after update event, got %f", state.Position.X)
	}
}
