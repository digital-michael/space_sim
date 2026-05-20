package app

import (
	"fmt"
	"log"
	"path/filepath"

	"github.com/digital-michael/space_sim/internal/client/go/raylib/ui"
	"github.com/digital-michael/space_sim/internal/server/ship"
	engine "github.com/digital-michael/space_sim/internal/sim/engine"
	sim "github.com/digital-michael/space_sim/internal/sim/world"
)

type runtimeSession struct {
	sim             *sim.World
	cameraState     *ui.CameraState
	inputState      *ui.InputState
	debugTracker    *DebugTracker
	navigationOrder []engine.ObjectCategory
	ship            *ship.ShipInstance
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
	if a.cfg.SimTimeScale > 0 {
		back := sim.GetState().GetBack()
		back.SecondsPerSecond = float32(a.cfg.SimTimeScale)
	}

	initialState := sim.GetState().LockFront()
	solIndex := -1
	var starRadius float32
	for i, obj := range initialState.Objects {
		if obj.Meta.Category == engine.CategoryStar {
			solIndex = i
			starRadius = obj.Meta.PhysicalRadius
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
	// Restore persisted performance options.
	pc := a.runtime.PerfConfig
	inputState.PerfOptions.FrustumCulling = pc.FrustumCulling
	inputState.PerfOptions.LODEnabled = pc.LODEnabled
	inputState.PerfOptions.InstancedRendering = pc.InstancedRendering
	inputState.PerfOptions.SpatialPartition = pc.SpatialPartition
	inputState.PerfOptions.PointRendering = pc.PointRendering
	inputState.PerfOptions.ImportanceThreshold = pc.ImportanceThreshold
	inputState.PerfOptions.UseInPlaceSwap = pc.UseInPlaceSwap

	if solIndex >= 0 {
		cameraState.StartTracking(solIndex)
		// Position 0.75 AU beyond the star's surface.
		// 1 AU = 100 simulation units (Earth semi_major_axis).
		cameraState.TrackDistance = float64(starRadius) + 75.0
		log.Printf("Camera started tracking star (index %d), TrackDistance=%.2f", solIndex, cameraState.TrackDistance)
	} else {
		log.Printf("Warning: no star found in simulation, starting in free-fly mode")
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
		ship:            loadDefaultShip(a.cfg),
	}, nil
}

// loadDefaultShip loads the ship catalog from the directory adjacent to the
// app config file and returns a ShipInstance for the default ship. If the
// catalog cannot be loaded or is empty, it returns nil (the session runs
// without a ship; F-022 movement will be a no-op).
func loadDefaultShip(cfg Config) *ship.ShipInstance {
	// Locate data/ships/ relative to the config file directory, or fall back
	// to a path relative to the working directory.
	baseDir := "."
	if cfg.AppConfigPath != "" {
		baseDir = filepath.Dir(cfg.AppConfigPath)
	}
	shipsDir := filepath.Join(baseDir, "data", "ships")

	defaultID := cfg.AppConfig.DefaultShipID
	if defaultID == "" {
		defaultID = "scout_mk1"
	}

	cat, err := ship.LoadCatalog(shipsDir, defaultID)
	if err != nil {
		log.Printf("ship catalog: load error: %v (running without ship)", err)
		return nil
	}
	def := cat.Default()
	if def == nil {
		log.Printf("ship catalog: empty catalog at %s (running without ship)", shipsDir)
		return nil
	}

	inst := ship.NewInstance(def, "local-session", "Player")
	log.Printf("ship: assigned %s (%s) transponder %s", def.Name, def.ID, inst.TransponderID)
	return inst
}
