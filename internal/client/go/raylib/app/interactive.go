package app

import (
	"context"
	"log"
	"math"
	"strconv"
	"strings"

	spatial "github.com/digital-michael/space_sim/internal/client/go/raylib/spatial"
	"github.com/digital-michael/space_sim/internal/client/go/raylib/ui"
	"github.com/digital-michael/space_sim/internal/protocol"
	engine "github.com/digital-michael/space_sim/internal/sim/engine"
	rl "github.com/gen2brain/raylib-go/raylib"
)

func (a *App) runInteractive(ctx context.Context, session *runtimeSession) error {
	startSession := func(activeSession *runtimeSession) context.CancelFunc {
		simCtx, cancel := context.WithCancel(ctx)
		go activeSession.sim.Start(simCtx)
		return cancel
	}

	sessionCancel := startSession(session)
	defer func() {
		sessionCancel()
		session.sim.Stop()
	}()

	shouldQuit := false
	for !rl.WindowShouldClose() && !shouldQuit {
		select {
		case <-ctx.Done():
			log.Println("Application context cancelled")
			return nil
		default:
		}

		a.syncWindowState()
		a.syncRenderState()
		dt := rl.GetFrameTime()
		snap := session.sim.Snapshot()
		a.broadcaster.Push(snap)
		a.drainCmds(session, snap)
		state := snap.State

		if session.debugTracker != nil {
			session.debugTracker.CheckVisibility(state.Objects, "after Snapshot()")
		}

		shouldQuit, a.runtime.GridVisible, a.runtime.AsteroidDataset, a.runtime.HUDVisible, a.runtime.HelpVisible, a.runtime.MouseModeEnabled, a.runtime.LabelsVisible = handleInput(
			a,
			session.sim,
			session.cameraState,
			session.inputState,
			state,
			session.navigationOrder,
			a.runtime.GridVisible,
			a.runtime.AsteroidDataset,
			a.runtime.HUDVisible,
			a.runtime.HelpVisible,
			a.runtime.MouseModeEnabled,
			a.runtime.LabelsVisible,
			a.cfg.Debug,
		)

		zoomIndicator := updateCameraState(session.cameraState, session.inputState, state, dt, a.runtime.CameraSpeed, a.runtime.MouseSensitivity, a.runtime.MouseModeEnabled, a.runtime.HelpVisible)

		renderWidth := a.runtime.RenderWidth
		renderHeight := a.runtime.RenderHeight
		if renderWidth <= 0 || renderHeight <= 0 {
			renderWidth = int32(rl.GetScreenWidth())
			renderHeight = int32(rl.GetScreenHeight())
		}

		a.renderer.BeginFrame()

		aspect := float32(renderWidth) / float32(renderHeight)
		rl.SetMatrixProjection(rl.MatrixPerspective(engine.CameraFOV*rl.Deg2rad, aspect, engine.CameraNearPlane, engine.CameraFarPlane))

		camera := rl.Camera3D{
			Position:   rl.Vector3{X: session.cameraState.Position.X, Y: session.cameraState.Position.Y, Z: session.cameraState.Position.Z},
			Target:     rl.Vector3Add(rl.Vector3{X: session.cameraState.Position.X, Y: session.cameraState.Position.Y, Z: session.cameraState.Position.Z}, rl.Vector3{X: session.cameraState.Forward.X, Y: session.cameraState.Forward.Y, Z: session.cameraState.Forward.Z}),
			Up:         rl.Vector3{X: 0, Y: 1, Z: 0},
			Fovy:       engine.CameraFOV,
			Projection: rl.CameraPerspective,
		}

		rl.BeginMode3D(camera)

		objectsToRender := make([]*engine.Object, 0, len(state.Objects))
		for _, obj := range state.Objects {
			if obj.Visible {
				objectsToRender = append(objectsToRender, obj)
			} else if session.debugTracker != nil && (obj.Meta.Name == "Earth" || obj.Meta.Name == "Moon") {
				session.debugTracker.LogRenderDecision(obj, false, "obj.Visible=false")
			}
		}

		if session.inputState.PerfOptions.FrustumCulling {
			preCullCount := len(objectsToRender)
			if session.inputState.PerfOptions.SpatialPartition {
				objectsToRender = spatial.SpatialFrustumCull(objectsToRender, camera)
			} else {
				objectsToRender = spatial.FrustumCullObjects(objectsToRender, camera)
			}

			postCullCount := len(objectsToRender)
			if session.debugTracker != nil && preCullCount != postCullCount {
				for _, obj := range state.Objects {
					if (obj.Meta.Name == "Earth" || obj.Meta.Name == "Moon") && obj.Visible {
						found := false
						for _, renderObj := range objectsToRender {
							if renderObj == obj {
								found = true
								break
							}
						}
						if !found {
							if session.inputState.PerfOptions.SpatialPartition {
								session.debugTracker.LogRenderDecision(obj, true, "spatial frustum culling")
							} else {
								session.debugTracker.LogRenderDecision(obj, true, "frustum culling")
							}
						}
					}
				}
			}
		}

		inViewCount := len(objectsToRender)
		eligibleInViewCount := 0
		for _, obj := range objectsToRender {
			if obj.Meta.Importance >= session.inputState.PerfOptions.ImportanceThreshold {
				eligibleInViewCount++
			}
		}
		renderedCount := 0

		if session.inputState.PerfOptions.InstancedRendering {
			renderedCount = a.renderer.DrawObjectsInstanced(objectsToRender, session.cameraState.Position, session.inputState.PerfOptions.PointRendering, session.inputState.PerfOptions.LODEnabled, session.inputState.PerfOptions.ImportanceThreshold)
		} else {
			for _, obj := range objectsToRender {
				if obj.Meta.Importance < session.inputState.PerfOptions.ImportanceThreshold {
					continue
				}
				a.renderer.DrawObject(obj, session.cameraState.Position, session.inputState.PerfOptions.PointRendering, session.inputState.PerfOptions.LODEnabled)
				renderedCount++
			}
		}

		if a.runtime.GridVisible {
			a.renderer.DrawGroundPlane()
		}

		rl.EndMode3D()

		rl.SetMatrixProjection(rl.MatrixOrtho(0.0, float32(renderWidth), float32(renderHeight), 0.0, 0.0, 1.0))
		rl.SetMatrixModelview(rl.MatrixIdentity())

		if a.runtime.HUDVisible {
			a.renderer.DrawHUD(state, session.cameraState, session.inputState, a.runtime.AsteroidDataset, a.runtime.MouseModeEnabled, snap.Speed, inViewCount, eligibleInViewCount, renderedCount)
		}
		if a.runtime.LabelsVisible {
			a.renderer.DrawObjectLabels(state, session.cameraState, camera, objectsToRender)
		}
		if zoomIndicator != 0 {
			a.renderer.DrawZoomIndicator(zoomIndicator)
		}
		if a.runtime.HelpVisible {
			a.renderer.DrawHelpScreen()
		}

		a.renderer.EndFrame(int32(rl.GetScreenWidth()), int32(rl.GetScreenHeight()))

		if pendingSystemPath := session.inputState.ConsumePendingSystemPath(); pendingSystemPath != "" {
			newSession, err := a.newRuntimeSession(pendingSystemPath)
			if err != nil {
				log.Printf("Failed to reload runtime session for %s: %v", pendingSystemPath, err)
				session.inputState.SetSystemSelectorStatus(err.Error())
				continue
			}

			sessionCancel()
			session.sim.Stop()

			a.cfg.SystemConfig = pendingSystemPath
			session = newSession
			a.worldPtr.Store(newSession.sim)
			a.runtime.HelpVisible = false
			sessionCancel = startSession(session)
			log.Printf("Reloaded runtime session using %s", pendingSystemPath)
		}
	}

	log.Println("Exiting app loop")
	return nil
}

// drainCmds processes all pending AppCmds in the channel non-blockingly.
// Called once per frame on the OS main thread after the latest snapshot is taken.
func (a *App) drainCmds(session *runtimeSession, snap protocol.WorldSnapshot) {
	for {
		select {
		case cmd := <-a.cmdCh:
			a.dispatchCmd(session, snap, cmd)
		default:
			return
		}
	}
}

// dispatchCmd executes one AppCmd on the OS main thread.
func (a *App) dispatchCmd(session *runtimeSession, snap protocol.WorldSnapshot, cmd AppCmd) {
	cs := session.cameraState
	is := session.inputState
	state := snap.State

	switch c := cmd.(type) {

	// ── Window ─────────────────────────────────────────────────────────────

	case WindowSizeCmd:
		if c.Width > 0 && c.Height > 0 {
			rl.SetWindowSize(int(c.Width), int(c.Height))
			a.runtime.WindowedWidth = c.Width
			a.runtime.WindowedHeight = c.Height
			a.syncWindowState()
		}

	case WindowMaximizeCmd:
		rl.MaximizeWindow()
		a.syncWindowState()

	case WindowRestoreCmd:
		rl.RestoreWindow()
		ww := a.runtime.WindowedWidth
		wh := a.runtime.WindowedHeight
		if ww <= 0 {
			ww = defaultScreenWidth
		}
		if wh <= 0 {
			wh = defaultScreenHeight
		}
		rl.SetWindowSize(int(ww), int(wh))
		a.syncWindowState()

	case WindowFullscreenCmd:
		if c.On != rl.IsWindowFullscreen() {
			a.toggleFullscreen()
		}

	case GetWindowCmd:
		c.RespCh <- WindowSnapshot{
			Width:      a.runtime.ScreenWidth,
			Height:     a.runtime.ScreenHeight,
			Fullscreen: a.runtime.Fullscreen,
			Maximized:  rl.IsWindowMaximized(),
		}

	// ── Camera ─────────────────────────────────────────────────────────────

	case CameraOrientCmd:
		const degToRad = math.Pi / 180.0
		yaw := c.YawDeg * degToRad
		pitch := c.PitchDeg * degToRad
		// Clamp pitch to ±85°
		maxPitch := 85.0 * degToRad
		if pitch > maxPitch {
			pitch = maxPitch
		}
		if pitch < -maxPitch {
			pitch = -maxPitch
		}
		cs.Yaw = yaw
		cs.Pitch = pitch
		cs.UpdateForwardFromAngles()

	case CameraPositionCmd:
		cs.Position = c.Pos

	case CameraTrackCmd:
		if c.Name == "" {
			cs.Mode = ui.CameraModeFree
		} else {
			target := findBodyByName(state, c.Name)
			if target >= 0 {
				cs.StartTracking(target)
				cs.TrackDistance = ui.CalculateAutoZoomDistance(state.Objects[target].Meta.PhysicalRadius, 0.24)
			} else {
				log.Printf("CameraTrackCmd: body %q not found in snapshot", c.Name)
			}
		}

	case GetCameraCmd:
		trackName := ""
		if cs.Mode == ui.CameraModeTracking &&
			cs.TrackTargetIndex >= 0 &&
			cs.TrackTargetIndex < len(state.Objects) {
			trackName = state.Objects[cs.TrackTargetIndex].Meta.Name
		}
		const radToDeg = 180.0 / math.Pi
		c.RespCh <- CameraSnapshot{
			YawDeg:      cs.Yaw * radToDeg,
			PitchDeg:    cs.Pitch * radToDeg,
			Position:    cs.Position,
			Mode:        cs.Mode,
			TrackTarget: trackName,
		}

	// ── Navigation ─────────────────────────────────────────────────────────

	case SetVelocityCmd:
		cs.Velocity = c.Velocity

	case JumpToCmd:
		if len(c.Names) == 0 {
			return
		}
		// Resolve names against current snapshot. Numeric tokens following a name
		// are treated as dwell seconds for that stop (e.g. "Earth" "2.5" "Mars").
		var targets []ui.JumpTarget
		for _, name := range c.Names {
			if d, err := strconv.ParseFloat(name, 64); err == nil {
				// Dwell time for the preceding target.
				if len(targets) > 0 {
					targets[len(targets)-1].DwellSeconds = d
				}
				continue
			}
			if idx := findBodyByName(state, name); idx >= 0 {
				obj := state.Objects[idx]
				targets = append(targets, ui.JumpTarget{
					TargetIndex: idx,
					TargetPos:   obj.Anim.Position,
					ViewDist:    ui.CalculateAutoZoomDistance(obj.Meta.PhysicalRadius, 0.24),
				})
			} else {
				log.Printf("JumpToCmd: body %q not found, skipping", name)
			}
		}
		if len(targets) == 0 {
			return
		}
		// Start the first jump immediately; enqueue the rest.
		first := targets[0]
		cs.JumpCurrentDwell = first.DwellSeconds
		cs.StartJumpTo(first.TargetIndex, first.TargetPos, first.ViewDist)
		cs.JumpQueue = append(cs.JumpQueue[:0], targets[1:]...)

	case GetVelocityCmd:
		c.RespCh <- cs.Velocity

	// ── Performance ────────────────────────────────────────────────────────

	case PerfSetCmd:
		if c.SetFrustumCulling {
			is.PerfOptions.FrustumCulling = c.Options.FrustumCulling
		}
		if c.SetLODEnabled {
			is.PerfOptions.LODEnabled = c.Options.LODEnabled
		}
		if c.SetInstancedRendering {
			is.PerfOptions.InstancedRendering = c.Options.InstancedRendering
		}
		if c.SetSpatialPartition {
			is.PerfOptions.SpatialPartition = c.Options.SpatialPartition
		}
		if c.SetPointRendering {
			is.PerfOptions.PointRendering = c.Options.PointRendering
		}
		if c.SetImportanceThreshold {
			is.PerfOptions.ImportanceThreshold = c.Options.ImportanceThreshold
		}
		if c.SetUseInPlaceSwap {
			is.PerfOptions.UseInPlaceSwap = c.Options.UseInPlaceSwap
			if c.Options.UseInPlaceSwap {
				session.sim.GetState().EnableInPlaceSwap()
			} else {
				session.sim.GetState().DisableInPlaceSwap()
			}
		}
		if c.SetCameraSpeed {
			a.runtime.CameraSpeed = c.CameraSpeed
		}
		if c.SetNumWorkers && c.NumWorkers > 0 {
			session.sim.SetWorkerCount(c.NumWorkers)
		}
		if c.SetHUDVisible {
			a.runtime.HUDVisible = c.HUDVisible
		}

	case GetPerfCmd:
		numWorkers := snap.State.NumWorkers
		c.RespCh <- PerfSnapshot{
			Options:     *is.PerfOptions,
			CameraSpeed: a.runtime.CameraSpeed,
			NumWorkers:  numWorkers,
			HUDVisible:  a.runtime.HUDVisible,
		}

	// ── System ─────────────────────────────────────────────────────────────

	case LoadSystemCmd:
		is.PendingSystemPath = c.Path

	case GetActiveSystemCmd:
		c.RespCh <- is.ActiveSystemPath

	// ── HUD ─────────────────────────────────────────────────────────────────

	case SetHUDCmd:
		a.runtime.HUDVisible = c.Visible

	// ── Orbit ───────────────────────────────────────────────────────────────

	case OrbitCmd:
		idx := findBodyByName(state, c.Name)
		if idx < 0 {
			log.Printf("OrbitCmd: body %q not found", c.Name)
			return
		}
		obj := state.Objects[idx]
		cs.StartTracking(idx)
		cs.TrackDistance = ui.CalculateAutoZoomDistance(obj.Meta.PhysicalRadius, 0.24)
		cs.TrackOffset = engine.Vector3{}
		cs.UpdateTracking(state)
		cs.OrbitSpeed = c.SpeedDegPerSec * (math.Pi / 180.0)
		cs.OrbitRadiansRemaining = c.Orbits * 2 * math.Pi
	}
}

// bodyAliases maps common alternative names to their canonical data-file names
// (lower-case). Add entries here as new aliases are needed.
var bodyAliases = map[string]string{
	"sol": "sun",
}

// findBodyByName returns the index of the first body whose name matches
// (case-insensitive) in state.Objects, or -1 if not found.
// Common aliases (e.g. "sol" → "Sun") are also resolved.
func findBodyByName(state *engine.SimulationState, name string) int {
	for i, obj := range state.Objects {
		if strings.EqualFold(obj.Meta.Name, name) {
			return i
		}
	}
	if canonical, ok := bodyAliases[strings.ToLower(name)]; ok {
		for i, obj := range state.Objects {
			if strings.EqualFold(obj.Meta.Name, canonical) {
				return i
			}
		}
	}
	return -1
}
