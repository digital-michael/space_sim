package app

import (
	"context"
	"math"
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

	firstCategory := engine.CategoryStarMainSequence
	if len(navOrder) > 0 {
		firstCategory = navOrder[0]
	}

	return &runtimeSession{
		sim:             nil,
		snapSrc:         snapSrc,
		cameraState:     cameraState,
		inputState:      ui.NewInputState(firstCategory),
		navigationOrder: navOrder,
		sessionID:       a.cfg.SessionID,
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

	// Open looking at where this client actually is, falling back to the star.
	//
	// Tracking the star meant every client opened pointing at the same place
	// regardless of where it spawned — and with spawns scattered across the
	// system, a peer ~100 degrees away was outside the frustum and appeared to be
	// missing entirely.
	if !a.placeCameraAtOwnSpawn(session, firstSnap) {
		for i, obj := range firstSnap.State.Objects {
			if engine.IsStarLike(obj.Meta.Category) {
				session.cameraState.StartTracking(i)
				session.cameraState.Tracking.Distance = float64(obj.Meta.PhysicalRadius) + 75.0
				break
			}
		}
	}

	a.startKeybindingsWatcher(ctx, a.cfg.profilesDir(), a.cfg.keybindingsPath())
	return a.runInteractive(ctx, session)
}

// deriveNavigationOrder builds a stable category-tab order from the objects
// present in a snapshot. Uses the canonical ordering; includes only categories
// that have at least one object.
func deriveNavigationOrder(objects []*engine.Object) []engine.ObjectCategory {
	canonical := []engine.ObjectCategory{
		engine.CategoryStarPreMain,
		engine.CategoryStarMainSequence,
		engine.CategoryStarEvolved,
		engine.CategorySubstellar,
		engine.CategoryStellarRemnant,
		engine.CategoryPlanet,
		engine.CategoryDwarfPlanet,
		engine.CategoryMoon,
		engine.CategoryAsteroid,
		engine.CategoryBelt,
		engine.CategoryRogue,
		engine.CategoryArtifact,
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

// placeCameraAtOwnSpawn positions the camera just outside this client's own spawn
// point, looking inward through it toward the system origin. Reports whether it
// could — it needs a registered session that has reached the snapshot.
//
// Looking inward is deliberate: the spawn point, anything clustered near it, and
// the star all fall ahead of the camera, so a new client sees its surroundings
// rather than empty space.
func (a *App) placeCameraAtOwnSpawn(session *runtimeSession, snap protocol.WorldSnapshot) bool {
	if a.cfg.SessionID == "" {
		return false
	}

	var own *protocol.ClientSessionSnapshot
	for i := range snap.ClientSessions {
		if snap.ClientSessions[i].SessionID == a.cfg.SessionID {
			own = &snap.ClientSessions[i]
			break
		}
	}
	if own == nil {
		return false
	}

	px, py, pz := own.Position[0], own.Position[1], own.Position[2]
	r := math.Sqrt(px*px + py*py + pz*pz)
	if r == 0 {
		return false // at the origin, which is inside the star; the star fallback is better
	}

	const standoffSU = 25.0
	nx, ny, nz := px/r, py/r, pz/r

	session.cameraState.Mode = ui.CameraModeFree
	session.cameraState.Position = engine.Vector3{
		X: float32(px + nx*standoffSU),
		Y: float32(py + ny*standoffSU),
		Z: float32(pz + nz*standoffSU),
	}
	// Face back down the radial, i.e. toward the origin.
	session.cameraState.Pitch = math.Asin(-ny)
	session.cameraState.Yaw = math.Atan2(-nx, -nz)
	session.cameraState.UpdateForwardFromAngles()
	return true
}
