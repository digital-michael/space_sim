package eventloop

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/digital-michael/space_sim/internal/server/eventqueue"
	grouppool "github.com/digital-michael/space_sim/internal/server/pool/group"
	"github.com/digital-michael/space_sim/internal/server/routines"
	runtimepkg "github.com/digital-michael/space_sim/internal/server/runtime"
)

// EventLoop is the main simulation loop coordinating frame timing.
//
// Event processing within the loop is frame-owned: each tick constructs a
// Frame and executes the pipeline in order (events -> routines -> physics).
// The WorkerPool remains available as a standalone component, but it is not
// started by EventLoop because background queue draining would violate the
// frame boundary guarantees.
type EventLoop struct {
	mu            sync.RWMutex
	running       bool
	targetFPS     float64
	actualFPS     float64
	definitions   *grouppool.Pool
	runtime       *runtimepkg.RuntimeEnvironment
	eventManager  *eventqueue.QueueManager
	routineLib    *routines.Library
	workerPool    *WorkerPool
	fpsController *FPSController
	metrics       *LoopMetrics
	stopCh        chan struct{}
	doneCh        chan struct{}
}

// NewEventLoop creates a new event loop with FPS control.
//
// A WorkerPool instance is retained for metrics and standalone worker-pool
// tests, but the event loop itself uses frame-owned event processing.
func NewEventLoop(
	definitions *grouppool.Pool,
	runtime *runtimepkg.RuntimeEnvironment,
	eventManager *eventqueue.QueueManager,
	routineLib *routines.Library,
	targetFPS float64,
	workerCount int,
) *EventLoop {
	if targetFPS <= 0 {
		targetFPS = 60.0
	}

	workerPool := NewWorkerPool(workerCount, eventManager, runtime)

	return &EventLoop{
		targetFPS:     targetFPS,
		definitions:   definitions,
		runtime:       runtime,
		eventManager:  eventManager,
		routineLib:    routineLib,
		workerPool:    workerPool,
		fpsController: NewFPSController(targetFPS),
		metrics:       NewLoopMetrics(),
		stopCh:        make(chan struct{}),
		doneCh:        make(chan struct{}),
	}
}

// Start begins the event loop and worker pool.
func (el *EventLoop) Start(ctx context.Context) error {
	el.mu.Lock()
	if el.running {
		el.mu.Unlock()
		return fmt.Errorf("event loop already running")
	}
	el.running = true
	el.mu.Unlock()

	// Start main loop goroutine
	go el.run(ctx)
	return nil
}

// Stop gracefully shuts down the event loop.
func (el *EventLoop) Stop() error {
	el.mu.Lock()
	if !el.running {
		el.mu.Unlock()
		return fmt.Errorf("event loop not running")
	}
	el.mu.Unlock()

	close(el.stopCh)
	<-el.doneCh // Wait for loop to finish

	el.mu.Lock()
	el.running = false
	el.mu.Unlock()

	return nil
}

// run is the main loop execution goroutine.
func (el *EventLoop) run(ctx context.Context) {
	defer close(el.doneCh)

	ticker := time.NewTicker(el.fpsController.FrameDuration())
	defer ticker.Stop()

	for {
		select {
		case <-el.stopCh:
			return
		case <-ctx.Done():
			return
		case <-ticker.C:
			frameStart := time.Now()

			// Process one frame
			if err := el.processFrame(); err != nil {
				el.metrics.RecordFrameError()
			} else {
				el.metrics.RecordFrameSuccess()
			}

			frameTime := time.Since(frameStart)
			el.metrics.RecordFrameTime(frameTime)

			// Update actual FPS
			el.mu.Lock()
			el.actualFPS = el.metrics.GetActualFPS()
			el.mu.Unlock()

			// Adjust ticker if FPS changed
			if el.fpsController.HasChanged() {
				ticker.Reset(el.fpsController.FrameDuration())
			}
		}
	}
}

// processFrame processes one simulation frame using a Frame processor.
// Phase 1: Process events; Phase 2: Execute routines; Phase 3: Apply physics.
// Event processing is intentionally owned by Frame.Process rather than the
// WorkerPool so state mutation is bounded to the frame tick.
func (el *EventLoop) processFrame() error {
	frame := NewFrame(
		el.definitions,
		el.runtime,
		el.eventManager,
		el.routineLib,
		el.fpsController.DeltaTime(),
	)
	return frame.Process()
}

// SetFPS changes the target FPS at runtime.
func (el *EventLoop) SetFPS(fps float64) error {
	if fps <= 0 || fps > 1000 {
		return fmt.Errorf("invalid FPS: %f (must be 0 < fps <= 1000)", fps)
	}

	el.mu.Lock()
	defer el.mu.Unlock()

	el.targetFPS = fps
	el.fpsController.SetFPS(fps)
	return nil
}

// GetFPS returns the current target and actual FPS.
func (el *EventLoop) GetFPS() (target, actual float64) {
	el.mu.RLock()
	defer el.mu.RUnlock()
	return el.targetFPS, el.actualFPS
}

// GetMetrics returns the event loop metrics.
func (el *EventLoop) GetMetrics() LoopStats {
	return el.metrics.GetStats()
}

// IsRunning checks if the event loop is active.
func (el *EventLoop) IsRunning() bool {
	el.mu.RLock()
	defer el.mu.RUnlock()
	return el.running
}

// GetWorkerMetrics returns metrics for all workers.
func (el *EventLoop) GetWorkerMetrics() []WorkerStats {
	return el.workerPool.GetMetrics()
}
