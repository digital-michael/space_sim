package app

import (
	"fmt"
	"log"
	"math"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/digital-michael/space_sim/internal/space"
	engine "github.com/digital-michael/space_sim/internal/space/engine"
	spatial "github.com/digital-michael/space_sim/internal/space/raylib/spatial"
	"github.com/digital-michael/space_sim/internal/space/ui"
	rl "github.com/gen2brain/raylib-go/raylib"
)

// Helper functions for float64 math
func cos64(x float64) float64 {
	return math.Cos(x)
}

func sin64(x float64) float64 {
	return math.Sin(x)
}

// PerformanceTestConfig defines a single test configuration
type PerformanceTestConfig struct {
	Dataset            engine.AsteroidDataset
	FrustumCulling     bool
	LODEnabled         bool
	PointRendering     bool
	SpatialPartition   bool
	InstancedRendering bool
	Description        string
}

// PerformanceResult stores the results of a single test
type PerformanceResult struct {
	Config     PerformanceTestConfig
	FPS        float64
	AvgDraw    time.Duration
	AvgCull    time.Duration
	MemAllocMB float64 // Memory allocated in MB at mid-test
	MemSysMB   float64 // System memory in MB at mid-test
	NumGC      uint32  // Number of GC runs
}

// getMemoryStats returns current memory statistics
func getMemoryStats() (allocMB, sysMB float64, numGC uint32) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	allocMB = float64(m.Alloc) / 1024 / 1024
	sysMB = float64(m.Sys) / 1024 / 1024
	numGC = m.NumGC
	return
}

// logMemoryStats logs current memory usage
func logMemoryStats(label string) {
	allocMB, sysMB, numGC := getMemoryStats()
	log.Printf("[MEMORY] %s: Alloc=%.2fMB, Sys=%.2fMB, NumGC=%d\n", label, allocMB, sysMB, numGC)
	fmt.Printf("  Memory: %.2fMB allocated, %.2fMB system, %d GC runs\n", allocMB, sysMB, numGC)
}

// runPerformanceTest executes automated performance testing
func (a *App) runPerformanceTest(sim *space.Simulation, cameraState *ui.CameraState, inputState *ui.InputState, profile string, threads int, noLocking bool) {
	log.Printf("=== Starting Performance Test (profile=%s, threads=%d) ===", profile, threads)
	fmt.Printf("\n=== AUTOMATED PERFORMANCE TESTING MODE ===\n")
	fmt.Printf("Profile: %s\n", profile)
	fmt.Printf("Physics Threads: %d\n", threads)
	fmt.Println("This will test all datasets with various optimization combinations")
	fmt.Println("Results will be saved to performance_results.txt")
	fmt.Println("Debug log saved to performance_debug.log")

	logMemoryStats("Initial")

	// Enable cursor for performance mode
	rl.EnableCursor()
	log.Println("Cursor enabled")

	// Setup camera based on profile
	var targetName string
	switch profile {
	case "worst":
		// Overview position: high above, looking at sun (worst case - sees everything)
		cameraState.Position = engine.Vector3{X: 0, Y: 800, Z: -400}
		targetName = "Sun"
		log.Println("Profile 'worst': Overview position (0, 800, -400) looking at Sun")
	case "better":
		// In middle-outer belt (195-240 range, use ~215), tracking Sun at origin
		// Position in outer quarter of belt, above orbital plane
		cameraState.Position = engine.Vector3{X: 215, Y: 60, Z: 0}
		targetName = "Sun"
		log.Println("Profile 'better': Outer belt position (215, 60, 0) tracking Sun")
	default:
		// Fallback to worst
		cameraState.Position = engine.Vector3{X: 0, Y: 800, Z: -400}
		targetName = "Sun"
	}
	log.Printf("Camera position set to: (%.1f, %.1f, %.1f)", cameraState.Position.X, cameraState.Position.Y, cameraState.Position.Z)

	// Calculate forward vector based on target
	var targetPos engine.Vector3
	if targetName == "Sun" {
		targetPos = engine.Vector3{X: 0, Y: 0, Z: 0}
	} else {
		// Find Jupiter's current position
		state := sim.GetState().LockFront()
		if jupiterObj, exists := state.ObjectMap["Jupiter"]; exists {
			targetPos = jupiterObj.Anim.Position
			log.Printf("Jupiter found at: (%.1f, %.1f, %.1f)", targetPos.X, targetPos.Y, targetPos.Z)
		} else {
			// Fallback if Jupiter not found yet
			targetPos = engine.Vector3{X: 260, Y: 0, Z: 0}
			log.Println("Jupiter not found, using estimated position")
		}
		sim.GetState().UnlockFront()
	}

	// Calculate forward vector to look at target
	toTarget := targetPos.Sub(cameraState.Position)
	cameraState.Forward = toTarget.Normalize()
	log.Printf("Forward vector calculated: (%.3f, %.3f, %.3f)", cameraState.Forward.X, cameraState.Forward.Y, cameraState.Forward.Z)

	// Update yaw and pitch from the forward vector
	cameraState.Yaw = math.Atan2(float64(cameraState.Forward.X), float64(cameraState.Forward.Z))
	cameraState.Pitch = math.Asin(float64(cameraState.Forward.Y))

	fmt.Printf("Camera positioned at: (%.1f, %.1f, %.1f)\n", cameraState.Position.X, cameraState.Position.Y, cameraState.Position.Z)
	fmt.Printf("Looking toward (forward): (%.3f, %.3f, %.3f)\n", cameraState.Forward.X, cameraState.Forward.Y, cameraState.Forward.Z)
	fmt.Printf("Target: %s at (%.1f, %.1f, %.1f)\n", targetName, targetPos.X, targetPos.Y, targetPos.Z)

	// Wait for simulation to stabilize and render initial frames
	fmt.Println("Initializing rendering system...")
	log.Println("Starting initialization render loop (60 frames)")
	for i := 0; i < 60; i++ {
		if rl.WindowShouldClose() {
			log.Println("Window close requested during initialization")
			return
		}

		a.syncWindowState()
		a.syncRenderState()
		renderWidth, renderHeight := a.performanceRenderSize()

		state := sim.GetState().LockFront()

		a.renderer.BeginFrame()

		// Set custom projection matrix
		aspect := float32(renderWidth) / float32(renderHeight)
		rl.SetMatrixProjection(rl.MatrixPerspective(45.0*rl.Deg2rad, aspect, 0.001, 200000.0))

		camera := rl.Camera3D{
			Position:   rl.Vector3{X: cameraState.Position.X, Y: cameraState.Position.Y, Z: cameraState.Position.Z},
			Target:     rl.Vector3{X: cameraState.Position.X + cameraState.Forward.X, Y: cameraState.Position.Y + cameraState.Forward.Y, Z: cameraState.Position.Z + cameraState.Forward.Z},
			Up:         rl.Vector3{X: 0, Y: 1, Z: 0},
			Fovy:       45.0,
			Projection: rl.CameraPerspective,
		}

		rl.BeginMode3D(camera)

		// Draw all objects to initialize rendering
		for _, obj := range state.Objects {
			a.renderer.DrawObject(obj, cameraState.Position, false, false)
		}

		rl.EndMode3D()
		a.renderer.EndFrame(int32(rl.GetScreenWidth()), int32(rl.GetScreenHeight()))

		sim.GetState().UnlockFront()

		if i%20 == 0 {
			log.Printf("Initialization frame %d/60", i)
		}
	}

	log.Println("Initialization complete")
	fmt.Println("System initialized. Starting tests...")
	logMemoryStats("After Init")
	time.Sleep(1 * time.Second)

	datasets := []engine.AsteroidDataset{
		engine.AsteroidDatasetSmall,
		engine.AsteroidDatasetMedium,
		engine.AsteroidDatasetLarge,
		engine.AsteroidDatasetHuge,
	}

	// Define test configurations
	var testConfigs []PerformanceTestConfig

	for _, dataset := range datasets {
		// Baseline (no optimizations)
		testConfigs = append(testConfigs, PerformanceTestConfig{
			Dataset:            dataset,
			FrustumCulling:     false,
			LODEnabled:         false,
			PointRendering:     false,
			SpatialPartition:   false,
			InstancedRendering: false,
			Description:        "Baseline",
		})

		// Individual optimizations
		testConfigs = append(testConfigs, PerformanceTestConfig{
			Dataset:            dataset,
			FrustumCulling:     true,
			LODEnabled:         false,
			PointRendering:     false,
			SpatialPartition:   false,
			InstancedRendering: false,
			Description:        "Frustum Only",
		})

		testConfigs = append(testConfigs, PerformanceTestConfig{
			Dataset:            dataset,
			FrustumCulling:     false,
			LODEnabled:         true,
			PointRendering:     false,
			SpatialPartition:   false,
			InstancedRendering: false,
			Description:        "LOD Only",
		})

		testConfigs = append(testConfigs, PerformanceTestConfig{
			Dataset:            dataset,
			FrustumCulling:     false,
			LODEnabled:         false,
			PointRendering:     true,
			SpatialPartition:   false,
			InstancedRendering: false,
			Description:        "Point Only",
		})

		testConfigs = append(testConfigs, PerformanceTestConfig{
			Dataset:            dataset,
			FrustumCulling:     false,
			LODEnabled:         false,
			PointRendering:     false,
			SpatialPartition:   true,
			InstancedRendering: false,
			Description:        "Spatial Only",
		})

		testConfigs = append(testConfigs, PerformanceTestConfig{
			Dataset:            dataset,
			FrustumCulling:     false,
			LODEnabled:         false,
			PointRendering:     false,
			SpatialPartition:   false,
			InstancedRendering: true,
			Description:        "Instanced Only",
		})

		// All optimizations combined
		testConfigs = append(testConfigs, PerformanceTestConfig{
			Dataset:            dataset,
			FrustumCulling:     true,
			LODEnabled:         true,
			PointRendering:     true,
			SpatialPartition:   true,
			InstancedRendering: true,
			Description:        "All Combined",
		})
	}

	fmt.Printf("Total tests to run: %d\n\n", len(testConfigs))
	log.Printf("Total test configurations: %d", len(testConfigs))

	results := make([]PerformanceResult, 0, len(testConfigs))
	currentDataset := engine.AsteroidDatasetSmall

	// Run each test
	for i, config := range testConfigs {
		objectCount := space.GetAsteroidCount(config.Dataset) + 314
		testDuration := 20 // 8 sec warmup + 12 sec measurement
		log.Printf("[TEST %d/%d] Starting: %s - %s (%d objects)", i+1, len(testConfigs), space.GetDatasetName(config.Dataset), config.Description, objectCount)
		fmt.Printf("[%d/%d] Testing %s - %s (%d objects, ~%d seconds)...\n",
			i+1, len(testConfigs), space.GetDatasetName(config.Dataset), config.Description, objectCount, testDuration)

		logMemoryStats(fmt.Sprintf("Before Test %d", i+1))

		// Reload simulation if dataset changed
		if config.Dataset != currentDataset {
			log.Printf("Dataset change requested: %s -> %s", space.GetDatasetName(currentDataset), space.GetDatasetName(config.Dataset))
			fmt.Printf("  Loading dataset: %s (%d objects)\n", space.GetDatasetName(config.Dataset), space.GetAsteroidCount(config.Dataset)+314)

			// Switch to the new dataset using lazy allocation
			sim.SetAsteroidDataset(config.Dataset)
			currentDataset = config.Dataset

			// Wait for changes to propagate
			time.Sleep(100 * time.Millisecond)
		}

		// Run test with this configuration
		log.Printf("Calling runSingleTest for config: %s", config.Description)
		result := a.runSingleTest(sim, cameraState, inputState, config)
		log.Printf("Test completed: %.1f FPS", result.FPS)
		results = append(results, result)

		logMemoryStats(fmt.Sprintf("After Test %d", i+1))

		fmt.Printf("  Result: %.1f FPS (Draw: %.2fms, Cull: %.2fms)\n\n",
			result.FPS,
			float64(result.AvgDraw.Microseconds())/1000.0,
			float64(result.AvgCull.Microseconds())/1000.0)
	}

	// Output results
	log.Printf("All %d tests completed successfully", len(results))
	printPerformanceResults(results)
	savePerformanceResults(results, profile, threads, noLocking)

	fmt.Println("\n=== PERFORMANCE TESTING COMPLETE ===")
	fmt.Println("Results saved to performance_results.txt")
	fmt.Println("Shutting down...")
	log.Println("Performance testing completed successfully")
}

// runSingleTest executes a single performance test configuration
func (a *App) runSingleTest(sim *space.Simulation, cameraState *ui.CameraState, inputState *ui.InputState, config PerformanceTestConfig) PerformanceResult {
	log.Printf("runSingleTest started for: %s", config.Description)

	// Set performance options according to config
	inputState.PerfOptions.FrustumCulling = config.FrustumCulling
	inputState.PerfOptions.LODEnabled = config.LODEnabled
	inputState.PerfOptions.PointRendering = config.PointRendering
	inputState.PerfOptions.SpatialPartition = config.SpatialPartition
	inputState.PerfOptions.InstancedRendering = config.InstancedRendering

	// Warmup period (let FPS stabilize)
	warmupFrames := 480 // 8 seconds at 60 FPS (4x longer for equalization)
	log.Printf("Starting warmup: %d frames", warmupFrames)
	for i := 0; i < warmupFrames; i++ {
		a.syncWindowState()
		a.syncRenderState()
		renderWidth, renderHeight := a.performanceRenderSize()

		state := sim.GetState().LockFront()

		a.renderer.BeginFrame()

		// Set custom projection matrix with extended far plane
		aspect := float32(renderWidth) / float32(renderHeight)
		rl.SetMatrixProjection(rl.MatrixPerspective(45.0*rl.Deg2rad, aspect, 0.001, 200000.0))

		camera := rl.Camera3D{
			Position:   rl.Vector3{X: cameraState.Position.X, Y: cameraState.Position.Y, Z: cameraState.Position.Z},
			Target:     rl.Vector3{X: cameraState.Position.X + cameraState.Forward.X, Y: cameraState.Position.Y + cameraState.Forward.Y, Z: cameraState.Position.Z + cameraState.Forward.Z},
			Up:         rl.Vector3{X: 0, Y: 1, Z: 0},
			Fovy:       45.0,
			Projection: rl.CameraPerspective,
		}

		rl.BeginMode3D(camera)

		// Filter by visibility first
		visibleObjects := make([]*engine.Object, 0, len(state.Objects))
		for _, obj := range state.Objects {
			if obj.Visible {
				visibleObjects = append(visibleObjects, obj)
			}
		}

		// Apply culling if enabled
		if inputState.PerfOptions.FrustumCulling {
			if inputState.PerfOptions.SpatialPartition {
				visibleObjects = spatial.SpatialFrustumCull(visibleObjects, camera)
			} else {
				visibleObjects = spatial.SimpleFrustumCull(visibleObjects, cameraState)
			}
		}

		if inputState.PerfOptions.InstancedRendering {
			a.renderer.DrawObjectsInstanced(visibleObjects, cameraState.Position, inputState.PerfOptions.PointRendering, inputState.PerfOptions.LODEnabled, inputState.PerfOptions.ImportanceThreshold)
		} else {
			for _, obj := range visibleObjects {
				a.renderer.DrawObject(obj, cameraState.Position, inputState.PerfOptions.PointRendering, inputState.PerfOptions.LODEnabled)
			}
		}

		rl.EndMode3D()
		a.renderer.EndFrame(int32(rl.GetScreenWidth()), int32(rl.GetScreenHeight()))

		sim.GetState().UnlockFront()

		// Display progress every second during warmup
		if i > 0 && i%60 == 0 {
			currentFPS := rl.GetFPS()
			fmt.Printf("  Warmup: %d/%d frames (%.1f FPS, %d objects)\r", i, warmupFrames, float32(currentFPS), len(state.Objects))
			log.Printf("Warmup progress: frame %d/%d, FPS %.1f, objects %d", i, warmupFrames, float32(currentFPS), len(state.Objects))
		}

		if rl.WindowShouldClose() {
			log.Println("Window close detected during warmup")
			break
		}
	}
	log.Printf("Warmup completed: %d frames", warmupFrames)
	fmt.Println() // Newline after warmup progress

	// Measurement period
	measureFrames := 720 // 12 seconds at 60 FPS (4x longer for accuracy)
	log.Printf("Starting measurement: %d frames", measureFrames)
	var totalDrawTime, totalCullTime time.Duration
	frameCount := 0

	for i := 0; i < measureFrames; i++ {
		// Log every 60 frames (once per second) to reduce log volume
		// Debug logging disabled - only errors will be logged
		a.syncWindowState()
		a.syncRenderState()
		renderWidth, renderHeight := a.performanceRenderSize()

		state := sim.GetState().LockFront()

		a.renderer.BeginFrame()

		// Set custom projection matrix with extended far plane
		aspect := float32(renderWidth) / float32(renderHeight)
		rl.SetMatrixProjection(rl.MatrixPerspective(45.0*rl.Deg2rad, aspect, 0.001, 200000.0))

		camera := rl.Camera3D{
			Position:   rl.Vector3{X: cameraState.Position.X, Y: cameraState.Position.Y, Z: cameraState.Position.Z},
			Target:     rl.Vector3{X: cameraState.Position.X + cameraState.Forward.X, Y: cameraState.Position.Y + cameraState.Forward.Y, Z: cameraState.Position.Z + cameraState.Forward.Z},
			Up:         rl.Vector3{X: 0, Y: 1, Z: 0},
			Fovy:       45.0,
			Projection: rl.CameraPerspective,
		}

		rl.BeginMode3D(camera)

		// Measure culling time
		cullStart := time.Now()
		visibleObjects := state.Objects
		if inputState.PerfOptions.FrustumCulling {
			if inputState.PerfOptions.SpatialPartition {
				visibleObjects = spatial.SpatialFrustumCull(visibleObjects, camera)
			} else {
				visibleObjects = spatial.SimpleFrustumCull(visibleObjects, cameraState)
			}
		}
		cullTime := time.Since(cullStart)
		totalCullTime += cullTime

		// Measure draw time
		drawStart := time.Now()
		if inputState.PerfOptions.InstancedRendering {
			a.renderer.DrawObjectsInstanced(visibleObjects, cameraState.Position, inputState.PerfOptions.PointRendering, inputState.PerfOptions.LODEnabled, inputState.PerfOptions.ImportanceThreshold)
		} else {
			for _, obj := range visibleObjects {
				a.renderer.DrawObject(obj, cameraState.Position, inputState.PerfOptions.PointRendering, inputState.PerfOptions.LODEnabled)
			}
		}
		drawTime := time.Since(drawStart)
		totalDrawTime += drawTime

		rl.EndMode3D()
		a.renderer.EndFrame(int32(rl.GetScreenWidth()), int32(rl.GetScreenHeight()))

		sim.GetState().UnlockFront()
		frameCount++

		// Display progress every second during measurement
		if i > 0 && i%60 == 0 {
			currentFPS := rl.GetFPS()
			visibleCount := len(visibleObjects)
			fmt.Printf("  Measuring: %d/%d frames (%.1f FPS, %d/%d visible)\r", i, measureFrames, float32(currentFPS), visibleCount, len(state.Objects))
			log.Printf("Measurement progress: frame %d/%d, FPS %.1f, visible %d/%d", i, measureFrames, float32(currentFPS), visibleCount, len(state.Objects))

			// Log memory mid-test
			if i == measureFrames/2 {
				allocMB, sysMB, numGC := getMemoryStats()
				log.Printf("Mid-test memory: Alloc=%.2f MB, Sys=%.2f MB, NumGC=%d", allocMB, sysMB, numGC)
			}
		}

		if rl.WindowShouldClose() {
			log.Println("Window close detected during measurement")
			break
		}
	}
	log.Printf("Measurement completed: %d frames", frameCount)
	fmt.Println() // Newline after measurement progress

	if frameCount == 0 {
		frameCount = 1
	}

	avgDraw := totalDrawTime / time.Duration(frameCount)
	avgCull := totalCullTime / time.Duration(frameCount)
	totalAvg := avgDraw + avgCull
	fps := 1000.0 / float64(totalAvg.Milliseconds())

	// Capture memory stats at end of test
	allocMB, sysMB, numGC := getMemoryStats()
	log.Printf("Test result: %.1f FPS, avgDraw %v, avgCull %v, memory: %.2f/%.2f MB, GC %d", fps, avgDraw, avgCull, allocMB, sysMB, numGC)

	return PerformanceResult{
		Config:     config,
		FPS:        fps,
		AvgDraw:    avgDraw,
		AvgCull:    avgCull,
		MemAllocMB: allocMB,
		MemSysMB:   sysMB,
		NumGC:      numGC,
	}
}

func (a *App) performanceRenderSize() (int32, int32) {
	renderWidth := a.runtime.RenderWidth
	renderHeight := a.runtime.RenderHeight
	if renderWidth <= 0 || renderHeight <= 0 {
		renderWidth = int32(rl.GetScreenWidth())
		renderHeight = int32(rl.GetScreenHeight())
	}
	return renderWidth, renderHeight
}

// printPerformanceResults displays results in console
func printPerformanceResults(results []PerformanceResult) {
	fmt.Println("\n" + strings.Repeat("=", 100))
	fmt.Println("PERFORMANCE TEST RESULTS")
	fmt.Println(strings.Repeat("=", 100) + "\n")

	currentDataset := engine.AsteroidDataset(-1)

	for _, result := range results {
		if result.Config.Dataset != currentDataset {
			currentDataset = result.Config.Dataset
			totalObjects := space.GetAsteroidCount(currentDataset) + 314
			fmt.Printf("\n--- %s (%d objects) ---\n", space.GetDatasetName(currentDataset), totalObjects)
			fmt.Printf("%-20s %10s %12s %12s\n", "Configuration", "FPS", "Draw (ms)", "Cull (ms)")
			fmt.Println(strings.Repeat("-", 60))
		}

		fmt.Printf("%-20s %10.1f %12.2f %12.2f\n",
			result.Config.Description,
			result.FPS,
			float64(result.AvgDraw.Microseconds())/1000.0,
			float64(result.AvgCull.Microseconds())/1000.0)
	}

	fmt.Println("\n" + strings.Repeat("=", 100))
}

// savePerformanceResults saves results to a file
func savePerformanceResults(results []PerformanceResult, profile string, threads int, noLocking bool) {
	f, err := os.Create("performance_results.txt")
	if err != nil {
		fmt.Printf("Error creating results file: %v\n", err)
		return
	}
	defer f.Close()

	lockingStatus := "enabled"
	if noLocking {
		lockingStatus = "DISABLED"
	}
	fmt.Fprintln(f, "PERFORMANCE TEST RESULTS")
	fmt.Fprintf(f, "Profile: %s | Physics Threads: %d | Locking: %s\n", profile, threads, lockingStatus)
	fmt.Fprintln(f, "Generated:", time.Now().Format("2006-01-02 15:04:05"))
	fmt.Fprintln(f, strings.Repeat("=", 100))

	currentDataset := engine.AsteroidDataset(-1)

	for _, result := range results {
		if result.Config.Dataset != currentDataset {
			currentDataset = result.Config.Dataset
			totalObjects := space.GetAsteroidCount(currentDataset) + 314
			fmt.Fprintf(f, "\n%s (%d objects)\n", space.GetDatasetName(currentDataset), totalObjects)
			fmt.Fprintf(f, "%-20s %10s %12s %12s\n", "Configuration", "FPS", "Draw (ms)", "Cull (ms)")
			fmt.Fprintln(f, strings.Repeat("-", 60))
		}

		fmt.Fprintf(f, "%-20s %10.1f %12.2f %12.2f\n",
			result.Config.Description,
			result.FPS,
			float64(result.AvgDraw.Microseconds())/1000.0,
			float64(result.AvgCull.Microseconds())/1000.0)
	}

	fmt.Fprintln(f, "\n"+strings.Repeat("=", 100))
}
