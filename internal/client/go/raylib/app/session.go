package app

import (
	"fmt"
	"log"

	engine "github.com/digital-michael/space_sim/internal/sim/engine"
	sim "github.com/digital-michael/space_sim/internal/sim/world"
	"github.com/digital-michael/space_sim/internal/client/go/raylib/ui"
)

type runtimeSession struct {
	sim             *sim.World
	cameraState     *ui.CameraState
	inputState      *ui.InputState
	debugTracker    *DebugTracker
	navigationOrder []engine.ObjectCategory
}

func (a *App) newRuntimeSession(systemConfigPath string) (session *runtimeSession, err error) {
	if systemConfigPath == "" {
		systemConfigPath = a.cfg.SystemConfig
	}
	defer func() {
		if recovered := recover(); recovered != nil {
			err = fmt.Errorf("failed to create runtime session for %s: %v", systemConfigPath, recovered)
			session = nil
		}
	}()

	cameraState := ui.NewCameraState()
	cameraState.Position = engine.Vector3{X: 0, Y: 50, Z: -100}
	cameraState.UpdateForwardFromAngles()

	var debugTracker *DebugTracker
	if a.cfg.Debug {
		debugTracker = NewDebugTracker()
	}

	normalizedPath := normalizeSystemConfigPath(systemConfigPath)
	sim, err := sim.NewWorld(defaultSimHz, normalizedPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load system %s: %w", normalizedPath, err)
	}

	initialState := sim.GetState().LockFront()
	solIndex := -1
	for i, obj := range initialState.Objects {
		if obj.Meta.Name == "Sol" {
			solIndex = i
			break
		}
	}
	navigationOrder := initialState.NavigationOrder
	sim.GetState().UnlockFront()

	firstCategory := engine.CategoryStar
	if len(navigationOrder) > 0 {
		firstCategory = navigationOrder[0]
	}
	inputState := ui.NewInputState(firstCategory)
	inputState.ActiveSystemPath = normalizedPath

	if solIndex >= 0 {
		cameraState.StartTracking(solIndex)
		log.Printf("Camera started tracking Sol (index %d)", solIndex)
	} else {
		log.Printf("Warning: Sol not found in simulation, starting in free-fly mode")
	}

	if a.cfg.PerformanceMode {
		sim.SetWorkerCount(a.cfg.Threads)
		log.Printf("Physics worker threads set to: %d", a.cfg.Threads)
		if a.cfg.NoLocking {
			sim.DisableLocking()
			log.Println("WARNING: Double-buffer locking DISABLED - data races possible")
		}
	}

	return &runtimeSession{
		sim:             sim,
		cameraState:     cameraState,
		inputState:      inputState,
		debugTracker:    debugTracker,
		navigationOrder: navigationOrder,
	}, nil
}
