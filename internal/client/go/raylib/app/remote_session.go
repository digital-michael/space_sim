package app

import (
	"context"
	"time"

	ui "github.com/digital-michael/space_sim/internal/client/go/raylib/ui"
	"github.com/digital-michael/space_sim/internal/protocol"
	engine "github.com/digital-michael/space_sim/internal/sim/engine"
)

// newRemoteRuntimeSession constructs a runtimeSession for a remote renderer.
// sim is nil; snapSrc must be non-nil and will be called on every render frame.
// navOrder is the category tab ordering derived from the first received snapshot.
func (a *App) newRemoteRuntimeSession(snapSrc protocol.SnapshotSource, navOrder []engine.ObjectCategory) *runtimeSession {
	cameraState := ui.NewCameraState()
	cameraState.Position = engine.Vector3{X: 0, Y: 50, Z: -100}
	cameraState.UpdateForwardFromAngles()

	firstCategory := engine.CategoryStar
	if len(navOrder) > 0 {
		firstCategory = navOrder[0]
	}

	return &runtimeSession{
		sim:             nil,
		snapSrc:         snapSrc,
		cameraState:     cameraState,
		inputState:      ui.NewInputState(firstCategory),
		navigationOrder: navOrder,
	}
}

// RunWithSnapshot initialises the Raylib window and enters the interactive
// render loop using snapSrc as the sole snapshot source.
// Used by cmd/space-sim-client; no local simulation is started.
// Blocks on the main OS thread (Raylib requirement) until the window is closed
// or ctx is cancelled.
func (a *App) RunWithSnapshot(ctx context.Context, snapSrc protocol.SnapshotSource) error {
	if err := a.initWindow(); err != nil {
		return err
	}
	defer a.closeWindow()
	a.syncRenderState()

	// Wait for the first non-empty snapshot to derive navigation order and
	// position the camera on a star.
	var firstSnap protocol.WorldSnapshot
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
waitLoop:
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			s := snapSrc.LatestSnapshot()
			if s.State != nil && len(s.State.Objects) > 0 {
				firstSnap = s
				break waitLoop
			}
		}
	}

	// Derive navigation order from the snapshot. The server populates
	// State.NavigationOrder from the loaded system; fall back to deriving
	// it from the object set if the field is empty (forward-compat guard).
	navOrder := firstSnap.State.NavigationOrder
	if len(navOrder) == 0 {
		navOrder = deriveNavigationOrder(firstSnap.State.Objects)
	}

	session := a.newRemoteRuntimeSession(snapSrc, navOrder)

	// Point camera at the first star if one is present in the snapshot.
	for i, obj := range firstSnap.State.Objects {
		if obj.Meta.Category == engine.CategoryStar {
			session.cameraState.StartTracking(i)
			session.cameraState.Tracking.Distance = float64(obj.Meta.PhysicalRadius) + 75.0
			break
		}
	}

	a.startKeybindingsWatcher(ctx, defaultProfilesDir, a.cfg.keybindingsPath())
	return a.runInteractive(ctx, session)
}

// deriveNavigationOrder builds a stable category-tab order from the objects
// present in a snapshot. Uses the canonical ordering; includes only categories
// that have at least one object.
func deriveNavigationOrder(objects []*engine.Object) []engine.ObjectCategory {
	canonical := []engine.ObjectCategory{
		engine.CategoryBlackHole,
		engine.CategoryStar,
		engine.CategoryPlanet,
		engine.CategoryDwarfPlanet,
		engine.CategoryMoon,
		engine.CategoryAsteroid,
		engine.CategoryBelt,
		engine.CategoryRing,
	}
	present := make(map[engine.ObjectCategory]bool, len(objects))
	for _, obj := range objects {
		present[obj.Meta.Category] = true
	}
	order := make([]engine.ObjectCategory, 0, len(canonical))
	for _, cat := range canonical {
		if present[cat] {
			order = append(order, cat)
		}
	}
	return order
}
