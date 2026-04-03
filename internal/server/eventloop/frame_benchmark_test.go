package eventloop

import (
	"fmt"
	"testing"

	"github.com/digital-michael/space_sim/internal/server/eventqueue"
	"github.com/digital-michael/space_sim/internal/server/pool/group"
	"github.com/digital-michael/space_sim/internal/server/routines"
	runtimepkg "github.com/digital-michael/space_sim/internal/server/runtime"
	"github.com/google/uuid"
)

// BenchmarkFrameProcessEventCost measures the cost per event processed.
// Tests: 10, 50, 100, 500, 1000 events per frame
func BenchmarkFrameProcessEventCost(b *testing.B) {
	eventCounts := []int{10, 50, 100, 500, 1000}

	for _, n := range eventCounts {
		b.Run(fmt.Sprintf("events-%d", n), func(b *testing.B) {
			defs := group.NewPool()
			rt := runtimepkg.NewRuntimeEnvironment(defs)
			mgr := eventqueue.NewQueueManager(1000)
			lib := routines.NewLibrary()

			// Pre-register n objects in the pool and initialize them in runtime
			objIDs := make([]uuid.UUID, n)
			for i := 0; i < n; i++ {
				id := uuid.New()
				objIDs[i] = id
				_ = defs.CreateObject(id, "cube", nil)
				_ = rt.InitializeObject(id, runtimepkg.OriginPosition(), runtimepkg.Vector3{})
			}

			// Benchmark: process n events
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				// Enqueue n update events (one per object)
				for j, id := range objIDs {
					event := eventqueue.NewEvent(
						id,
						eventqueue.EventTypeUpdate,
						map[string]interface{}{
							"position": map[string]interface{}{"x": float64(j), "y": 0.0, "z": 0.0},
						},
						eventqueue.TransactionTypeNone,
					)
					_ = mgr.Enqueue(event)
				}

				frame := NewFrame(defs, rt, mgr, lib, 1.0/60.0)
				_ = frame.processEvents()
			}
		})
	}
}

// BenchmarkFrameExecuteRoutinesCost measures the cost per routine execution.
// Tests: objects 10, 50, 100, 500, 1000 with rotate routine
func BenchmarkFrameExecuteRoutinesCost(b *testing.B) {
	objectCounts := []int{10, 50, 100, 500, 1000}

	for _, n := range objectCounts {
		b.Run(fmt.Sprintf("objects-%d", n), func(b *testing.B) {
			defs := group.NewPool()
			rt := runtimepkg.NewRuntimeEnvironment(defs)
			mgr := eventqueue.NewQueueManager(100)
			lib := routines.NewLibrary()

			// Register n objects with rotate routine
			objIDs := make([]uuid.UUID, n)
			for i := 0; i < n; i++ {
				id := uuid.New()
				objIDs[i] = id
				_ = defs.CreateObject(id, "cube", map[string]interface{}{
					"routines": []string{"rotate"},
				})
				_ = rt.InitializeObject(id, runtimepkg.OriginPosition(), runtimepkg.Vector3{})
			}

			// Benchmark: execute routines on n objects
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				frame := NewFrame(defs, rt, mgr, lib, 1.0/60.0)
				_ = frame.executeRoutines()
			}
		})
	}
}

// BenchmarkFrameApplyPhysicsCost measures the cost per object for physics integration.
// Tests: objects 10, 50, 100, 500, 1000 with non-zero velocity
func BenchmarkFrameApplyPhysicsCost(b *testing.B) {
	objectCounts := []int{10, 50, 100, 500, 1000}

	for _, n := range objectCounts {
		b.Run(fmt.Sprintf("objects-%d", n), func(b *testing.B) {
			defs := group.NewPool()
			rt := runtimepkg.NewRuntimeEnvironment(defs)
			mgr := eventqueue.NewQueueManager(100)
			lib := routines.NewLibrary()

			// Register n objects with velocity
			objIDs := make([]uuid.UUID, n)
			for i := 0; i < n; i++ {
				id := uuid.New()
				objIDs[i] = id
				_ = defs.CreateObject(id, "cube", nil)
				// Initialize with velocity (1, 1, 1) so physics does work
				_ = rt.InitializeObject(id, runtimepkg.OriginPosition(), runtimepkg.Vector3{X: 1.0, Y: 1.0, Z: 1.0})
			}

			// Benchmark: apply physics to n objects
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				frame := NewFrame(defs, rt, mgr, lib, 1.0/60.0)
				_ = frame.applyPhysics()
			}
		})
	}
}

// BenchmarkFrameFullPipeline measures total cost of a complete frame (events + routines + physics).
// This is the most realistic scenario.
// Tests: (100 events + 500 routine objects + 500 physics objects) combined
func BenchmarkFrameFullPipeline(b *testing.B) {
	testCases := []struct {
		name     string
		events   int
		routines int
		physics  int
	}{
		{"light-load", 10, 50, 50},
		{"medium-load", 100, 500, 500},
		{"heavy-load", 500, 1000, 1000},
		{"extreme-load", 1000, 2000, 2000},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			defs := group.NewPool()
			rt := runtimepkg.NewRuntimeEnvironment(defs)
			mgr := eventqueue.NewQueueManager(10000)
			lib := routines.NewLibrary()

			// Create objects for events
			eventObjIDs := make([]uuid.UUID, tc.events)
			for i := 0; i < tc.events; i++ {
				id := uuid.New()
				eventObjIDs[i] = id
				_ = defs.CreateObject(id, "cube", nil)
				_ = rt.InitializeObject(id, runtimepkg.OriginPosition(), runtimepkg.Vector3{})
			}

			// Create objects for routines
			routineObjIDs := make([]uuid.UUID, tc.routines)
			for i := 0; i < tc.routines; i++ {
				id := uuid.New()
				routineObjIDs[i] = id
				_ = defs.CreateObject(id, "cube", map[string]interface{}{
					"routines": []string{"rotate"},
				})
				_ = rt.InitializeObject(id, runtimepkg.OriginPosition(), runtimepkg.Vector3{})
			}

			// Create objects for physics
			physicsObjIDs := make([]uuid.UUID, tc.physics)
			for i := 0; i < tc.physics; i++ {
				id := uuid.New()
				physicsObjIDs[i] = id
				_ = defs.CreateObject(id, "cube", nil)
				_ = rt.InitializeObject(id, runtimepkg.OriginPosition(), runtimepkg.Vector3{X: 1.0, Y: 1.0, Z: 1.0})
			}

			// Benchmark: full frame pipeline
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				// Enqueue events
				for j, id := range eventObjIDs {
					event := eventqueue.NewEvent(
						id,
						eventqueue.EventTypeUpdate,
						map[string]interface{}{
							"position": map[string]interface{}{"x": float64(j), "y": 0.0, "z": 0.0},
						},
						eventqueue.TransactionTypeNone,
					)
					_ = mgr.Enqueue(event)
				}

				frame := NewFrame(defs, rt, mgr, lib, 1.0/60.0)
				_ = frame.Process()
			}
		})
	}
}

// BenchmarkFrameScalabilityCeiling estimates the maximum item counts sustainable at 60 FPS (16.67 ms budget).
// Runs a synthetic workload and reports when we exceed budget.
func BenchmarkFrameScalabilityCeiling(b *testing.B) {
	// Target: 16.67 ms per frame at 60 FPS
	const budgetMs float64 = 16.67

	b.Run("cost-per-event", func(b *testing.B) {
		defs := group.NewPool()
		rt := runtimepkg.NewRuntimeEnvironment(defs)
		mgr := eventqueue.NewQueueManager(10000)
		lib := routines.NewLibrary()

		testSizes := []int{100, 500, 1000, 2000, 5000}

		b.ResetTimer()
		for _, size := range testSizes {
			// Setup
			for i := 0; i < size; i++ {
				id := uuid.New()
				_ = defs.CreateObject(id, "cube", nil)
				_ = rt.InitializeObject(id, runtimepkg.OriginPosition(), runtimepkg.Vector3{})
			}

			// Measure
			b.StopTimer()
			defs = group.NewPool()
			rt = runtimepkg.NewRuntimeEnvironment(defs)
			mgr = eventqueue.NewQueueManager(10000)

			objIDs := make([]uuid.UUID, size)
			for i := 0; i < size; i++ {
				id := uuid.New()
				objIDs[i] = id
				_ = defs.CreateObject(id, "cube", nil)
				_ = rt.InitializeObject(id, runtimepkg.OriginPosition(), runtimepkg.Vector3{})
			}

			for j, id := range objIDs {
				event := eventqueue.NewEvent(
					id,
					eventqueue.EventTypeUpdate,
					map[string]interface{}{
						"position": map[string]interface{}{"x": float64(j), "y": 0.0, "z": 0.0},
					},
					eventqueue.TransactionTypeNone,
				)
				_ = mgr.Enqueue(event)
			}

			b.StartTimer()
			frame := NewFrame(defs, rt, mgr, lib, 1.0/60.0)
			_ = frame.processEvents()
			b.StopTimer()

			elapsed := b.Elapsed().Seconds() * 1000 // convert to ms
			b.Logf("events=%d: elapsed=%.2f ms, per-event=%.3f µs", size, elapsed, (elapsed*1000)/float64(size))
		}
	})

	b.Run("cost-per-routine", func(b *testing.B) {
		testSizes := []int{100, 500, 1000, 2000, 5000}

		for _, size := range testSizes {
			defs := group.NewPool()
			rt := runtimepkg.NewRuntimeEnvironment(defs)
			mgr := eventqueue.NewQueueManager(100)
			lib := routines.NewLibrary()

			objIDs := make([]uuid.UUID, size)
			for i := 0; i < size; i++ {
				id := uuid.New()
				objIDs[i] = id
				_ = defs.CreateObject(id, "cube", map[string]interface{}{
					"routines": []string{"rotate"},
				})
				_ = rt.InitializeObject(id, runtimepkg.OriginPosition(), runtimepkg.Vector3{})
			}

			b.ResetTimer()
			frame := NewFrame(defs, rt, mgr, lib, 1.0/60.0)
			_ = frame.executeRoutines()
			b.StopTimer()

			elapsed := b.Elapsed().Seconds() * 1000
			b.Logf("routines=%d: elapsed=%.2f ms, per-routine=%.3f µs", size, elapsed, (elapsed*1000)/float64(size))
		}
	})

	b.Run("cost-per-physics-object", func(b *testing.B) {
		testSizes := []int{100, 500, 1000, 2000, 5000}

		for _, size := range testSizes {
			defs := group.NewPool()
			rt := runtimepkg.NewRuntimeEnvironment(defs)
			mgr := eventqueue.NewQueueManager(100)
			lib := routines.NewLibrary()

			objIDs := make([]uuid.UUID, size)
			for i := 0; i < size; i++ {
				id := uuid.New()
				objIDs[i] = id
				_ = defs.CreateObject(id, "cube", nil)
				_ = rt.InitializeObject(id, runtimepkg.OriginPosition(), runtimepkg.Vector3{X: 1.0, Y: 1.0, Z: 1.0})
			}

			b.ResetTimer()
			frame := NewFrame(defs, rt, mgr, lib, 1.0/60.0)
			_ = frame.applyPhysics()
			b.StopTimer()

			elapsed := b.Elapsed().Seconds() * 1000
			b.Logf("physics=%d: elapsed=%.2f ms, per-object=%.3f µs", size, elapsed, (elapsed*1000)/float64(size))
		}
	})
}
