package app

import (
	"fmt"
	"log"
	"path/filepath"

	"github.com/digital-michael/space_sim/internal/client/go/raylib/ui"
	"github.com/digital-michael/space_sim/internal/protocol"
	"github.com/digital-michael/space_sim/internal/server/ship"
	engine "github.com/digital-michael/space_sim/internal/sim/engine"
	sim "github.com/digital-michael/space_sim/internal/sim/world"
)

type runtimeSession struct {
	// sim is the local simulation world. Nil in remote-renderer mode; all
	// simulation commands in input.go must guard with `if session.sim != nil`.
	sim *sim.World
	// snapSrc is the snapshot source used by the render loop. Always non-nil;
	// set to sim for local mode and a grpcSnapshotSource for remote mode.
	snapSrc         protocol.SnapshotSource
	cameraState     *ui.CameraState
	inputState      *ui.InputState
	debugTracker    *DebugTracker
	navigationOrder []engine.ObjectCategory
	ship            *ship.ShipInstance
	driftMode       bool // move.drift_toggle: when true thrust keys are ignored

	// sessionID is the gRPC session ID registered with the server, used to
	// identify this client's own marker (F-021). Empty string means unknown —
	// all markers render at full opacity.
	sessionID string
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
	// Fall back to first black hole if no star present
	if solIndex < 0 {
		for i, obj := range initialState.Objects {
			if obj.Meta.Category == engine.CategoryBlackHole {
				solIndex = i
				starRadius = obj.Meta.PhysicalRadius
				break
			}
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
		// Position camera so the star fills ~50% of screen height, but never
		// closer than 0.75 AU (75 su) past the surface. For large stars this
		// prevents the initial view from being entirely filled by the star disk.
		cameraState.Tracking.Distance = max(float64(starRadius)+75.0, ui.CalculateAutoZoomDistance(starRadius, 0.40))
		log.Printf("Camera started tracking object (index %d), TrackDistance=%.2f", solIndex, cameraState.Tracking.Distance)
	} else {
		log.Printf("Warning: no star or black hole found in simulation, starting in free-fly mode")
	}

	if a.cfg.PerformanceMode {
		sim.SetWorkerCount(a.cfg.Threads)
		log.Printf("Physics worker threads set to: %d", a.cfg.Threads)
		if a.cfg.NoLocking {
			sim.DisableLocking()
			log.Println("WARNING: Double-buffer locking DISABLED - data races possible")
		}
	}

	shipInst := loadDefaultShip(a.cfg)
	if shipInst != nil {
		// Initialise ship position to match the camera start position so the
		// first free-fly frame doesn't snap the camera to the origin.
		shipInst.Position[0] = float64(cameraState.Position.X)
		shipInst.Position[1] = float64(cameraState.Position.Y)
		shipInst.Position[2] = float64(cameraState.Position.Z)
	}

	return &runtimeSession{
		sim:             sim,
		snapSrc:         sim, // world.World satisfies protocol.SnapshotSource
		cameraState:     cameraState,
		inputState:      inputState,
		debugTracker:    debugTracker,
		navigationOrder: navigationOrder,
		ship:            shipInst,
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
