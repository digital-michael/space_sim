package app

import (
	"context"
	"fmt"
	"log"
	"math"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/digital-michael/space_sim/internal/client/go/raylib/input"
	spatial "github.com/digital-michael/space_sim/internal/client/go/raylib/spatial"
	"github.com/digital-michael/space_sim/internal/client/go/raylib/ui"
	"github.com/digital-michael/space_sim/internal/protocol"
	engine "github.com/digital-michael/space_sim/internal/sim/engine"
	rl "github.com/gen2brain/raylib-go/raylib"
)

func (a *App) runInteractive(ctx context.Context, session *runtimeSession) error {
	startSession := func(activeSession *runtimeSession) context.CancelFunc {
		simCtx, cancel := context.WithCancel(ctx)
		if activeSession.sim != nil {
			go activeSession.sim.Start(simCtx)
		}
		return cancel
	}

	sessionCancel := startSession(session)
	defer func() {
		sessionCancel()
		if session.sim != nil {
			session.sim.Stop()
		}
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
		snap := session.snapSrc.LatestSnapshot()
		a.broadcaster.Push(snap)
		a.drainCmds(session, snap)
		state := snap.State

		if session.debugTracker != nil {
			session.debugTracker.CheckVisibility(state.Objects, "after Snapshot()")
		}

		shouldQuit = a.handleInput(session, state)
		zoomIndicator := a.updateCameraState(session, state, dt)

		renderWidth := a.runtime.RenderWidth
		renderHeight := a.runtime.RenderHeight
		if renderWidth <= 0 || renderHeight <= 0 {
			renderWidth = int32(rl.GetScreenWidth())
			renderHeight = int32(rl.GetScreenHeight())
		}

		a.renderer.BeginFrame()

		aspect := float32(renderWidth) / float32(renderHeight)
		rl.SetMatrixProjection(rl.MatrixPerspective(engine.CameraFOV*rl.Deg2rad, aspect, engine.CameraNearPlane, engine.CameraFarPlane))

		// worldCam: absolute world-space positions — used only for frustum culling.
		// Culling math must operate in world space to correctly eliminate out-of-view objects.
		worldCam := rl.Camera3D{
			Position:   rl.Vector3{X: session.cameraState.Position.X, Y: session.cameraState.Position.Y, Z: session.cameraState.Position.Z},
			Target:     rl.Vector3Add(rl.Vector3{X: session.cameraState.Position.X, Y: session.cameraState.Position.Y, Z: session.cameraState.Position.Z}, rl.Vector3{X: session.cameraState.Forward.X, Y: session.cameraState.Forward.Y, Z: session.cameraState.Forward.Z}),
			Up:         rl.Vector3{X: 0, Y: 1, Z: 0},
			Fovy:       engine.CameraFOV,
			Projection: rl.CameraPerspective,
		}

		// camera: floating-origin — Raylib always works near (0,0,0).
		// All object positions are shifted by -cameraPos before being passed to Draw calls,
		// eliminating float32 catastrophic cancellation at large camera distances (DEF-001).
		camera := rl.Camera3D{
			Position:   rl.Vector3{},
			Target:     rl.Vector3{X: session.cameraState.Forward.X, Y: session.cameraState.Forward.Y, Z: session.cameraState.Forward.Z},
			Up:         rl.Vector3{X: 0, Y: 1, Z: 0},
			Fovy:       engine.CameraFOV,
			Projection: rl.CameraPerspective,
		}

		rl.BeginMode3D(camera)

		// Sky must be drawn first inside BeginMode3D — before any scene geometry —
		// so that all foreground objects occlude it. The skysphere is centred at the
		// floating-origin camera (0,0,0) and only rotates with the camera.
		a.renderer.DrawSky()

		objectsToRender := make([]*engine.Object, 0, len(state.Objects))
		for _, obj := range state.Objects {
			if obj.Visible {
				objectsToRender = append(objectsToRender, obj)
			} else if session.debugTracker != nil && (obj.Meta.Name == "Earth" || obj.Meta.Name == "Moon") {
				session.debugTracker.LogRenderDecision(obj, false, "obj.Visible=false")
			}
		}

		if a.runtime.PerfConfig.FrustumCulling {
			preCullCount := len(objectsToRender)
			if a.runtime.PerfConfig.SpatialPartition {
				objectsToRender = spatial.SpatialFrustumCull(objectsToRender, worldCam)
			} else {
				objectsToRender = spatial.FrustumCullObjects(objectsToRender, worldCam)
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
							if a.runtime.PerfConfig.SpatialPartition {
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
			if obj.Meta.Importance >= a.runtime.PerfConfig.ImportanceThreshold {
				eligibleInViewCount++
			}
		}
		renderedCount := 0

		// Upload current star light positions/colours to the Phong shader before
		// any draw calls. Uses the full state.Objects (not frustum-culled) so the
		// sun illuminates the scene even when it is behind the camera.
		a.renderer.UpdateLights(state.Objects, session.cameraState.Position)
		a.renderer.SetInfraState(a.runtime.InfraMode, session.cameraState.Forward)

		if a.runtime.PerfConfig.InstancedRendering {
			renderedCount = a.renderer.DrawObjectsInstanced(objectsToRender, session.cameraState.Position, state.Time, a.runtime.PerfConfig.PointRendering, a.runtime.PerfConfig.LODEnabled, a.runtime.PerfConfig.ImportanceThreshold)
		} else {
			for _, obj := range objectsToRender {
				if obj.Meta.Importance < a.runtime.PerfConfig.ImportanceThreshold {
					continue
				}
				a.renderer.DrawObject(obj, session.cameraState.Position, state.Time, a.runtime.PerfConfig.PointRendering, a.runtime.PerfConfig.LODEnabled)
				renderedCount++
			}
		}

		// F-021 Phase 1: blinking sphere markers for remote client sessions.
		a.renderer.DrawClientMarkers(snap.ClientSessions, session.cameraState.Position, session.sessionID, float64(rl.GetTime()))

		rl.EndMode3D()

		rl.SetMatrixProjection(rl.MatrixOrtho(0.0, float32(renderWidth), float32(renderHeight), 0.0, 0.0, 1.0))
		rl.SetMatrixModelview(rl.MatrixIdentity())

		// F-021 Phase 1: 2D text labels above nearby client markers.
		a.renderer.DrawClientMarkerLabels(snap.ClientSessions, session.cameraState.Position, session.sessionID, camera)

		if a.runtime.HUDVisible {
			a.renderer.DrawHUD(state, session.cameraState, session.inputState, a.runtime.AsteroidDataset, a.runtime.MouseModeEnabled, snap.Speed, inViewCount, eligibleInViewCount, renderedCount, a.runtime.HUD.Debug, a.runtime.HUD.Info, a.runtime.HUD.Help)
		}
		if a.runtime.LabelMode != ui.LabelModeOff {
			a.renderer.DrawObjectLabels(state, session.cameraState, camera, objectsToRender, a.runtime.LabelMode)
		}
		if zoomIndicator != 0 {
			a.renderer.DrawZoomIndicator(zoomIndicator)
		}
		if a.runtime.SettingsVisible {
			a.renderer.DrawSettingsDialog(&a.runtime.Settings, a.keyMap.Load())
		}
		if a.runtime.RecordingActive {
			a.renderer.DrawRecordingIndicator(a.runtime.RecordingPaused)
		}
		if a.runtime.WelcomeBannerText != "" {
			elapsed := time.Since(a.runtime.WelcomeBannerAt)
			if elapsed >= 2*time.Second {
				a.runtime.WelcomeBannerText = ""
			} else {
				a.renderer.DrawWelcomeBanner(a.runtime.WelcomeBannerText, elapsed)
			}
		}

		a.renderer.EndFrame(int32(rl.GetScreenWidth()), int32(rl.GetScreenHeight()))

		// Capture after EndFrame: render texture content is stable and the
		// temporary-FBO approach in CaptureRenderTexture does not depend on
		// Raylib's framebuffer binding state.
		if a.runtime.RecordingActive && a.recorder != nil {
			var frame []byte
			if !a.runtime.RecordingPaused {
				frame = a.renderer.CaptureRenderTexture()
			}
			if err := a.recorder.WriteFrame(frame); err != nil {
				// Pipe broken — ffmpeg died. Abort cleanly.
				a.stopRecording()
			}
		}

		if pendingSystemPath := session.inputState.ConsumePendingSystemPath(); pendingSystemPath != "" {
			newSession, err := a.newRuntimeSession(pendingSystemPath)
			if err != nil {
				log.Printf("Failed to reload runtime session for %s: %v", pendingSystemPath, err)
				session.inputState.SetSystemSelectorStatus(err.Error())
				continue
			}

			sessionCancel()
			if session.sim != nil {
				session.sim.Stop()
			}

			a.cfg.SystemConfig = pendingSystemPath
			session = newSession
			a.worldPtr.Store(newSession.sim)
			a.runtime.SettingsVisible = false
			systemName := readSystemDisplayName(pendingSystemPath)
			if systemName == "" {
				systemName = filepath.Base(pendingSystemPath)
			}
			a.runtime.WelcomeBannerText = "Welcome to " + systemName
			a.runtime.WelcomeBannerAt = time.Now()
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
			a.runtime.PerfConfig.FrustumCulling = c.Options.FrustumCulling
			a.runtime.Settings.Perf.FrustumCulling = c.Options.FrustumCulling
		}
		if c.SetLODEnabled {
			is.PerfOptions.LODEnabled = c.Options.LODEnabled
			a.runtime.PerfConfig.LODEnabled = c.Options.LODEnabled
			a.runtime.Settings.Perf.LODEnabled = c.Options.LODEnabled
		}
		if c.SetInstancedRendering {
			is.PerfOptions.InstancedRendering = c.Options.InstancedRendering
			a.runtime.PerfConfig.InstancedRendering = c.Options.InstancedRendering
			a.runtime.Settings.Perf.InstancedRendering = c.Options.InstancedRendering
		}
		if c.SetSpatialPartition {
			is.PerfOptions.SpatialPartition = c.Options.SpatialPartition
			a.runtime.PerfConfig.SpatialPartition = c.Options.SpatialPartition
			a.runtime.Settings.Perf.SpatialPartition = c.Options.SpatialPartition
		}
		if c.SetPointRendering {
			is.PerfOptions.PointRendering = c.Options.PointRendering
			a.runtime.PerfConfig.PointRendering = c.Options.PointRendering
			a.runtime.Settings.Perf.PointRendering = c.Options.PointRendering
		}
		if c.SetImportanceThreshold {
			is.PerfOptions.ImportanceThreshold = c.Options.ImportanceThreshold
			a.runtime.PerfConfig.ImportanceThreshold = c.Options.ImportanceThreshold
			a.runtime.Settings.Perf.ImportanceThreshold = c.Options.ImportanceThreshold
		}
		if c.SetUseInPlaceSwap {
			is.PerfOptions.UseInPlaceSwap = c.Options.UseInPlaceSwap
			a.runtime.PerfConfig.UseInPlaceSwap = c.Options.UseInPlaceSwap
			a.runtime.Settings.Perf.UseInPlaceSwap = c.Options.UseInPlaceSwap
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
		if c.SetHUDDebug {
			a.runtime.HUD.Debug = c.HUDCategory.Debug
		}
		if c.SetHUDInfo {
			a.runtime.HUD.Info = c.HUDCategory.Info
		}
		if c.SetHUDHelp {
			a.runtime.HUD.Help = c.HUDCategory.Help
		}
		if c.SetHUDPlayer {
			a.runtime.HUD.Player = c.HUDCategory.Player
		}
		if c.SetLabelMode {
			a.runtime.LabelMode = c.LabelMode
		}
		if c.SetInfraMode {
			a.runtime.InfraMode = c.InfraMode
		}

	case GetPerfCmd:
		numWorkers := snap.State.NumWorkers
		c.RespCh <- PerfSnapshot{
			Options:     *is.PerfOptions,
			CameraSpeed: a.runtime.CameraSpeed,
			NumWorkers:  numWorkers,
			HUDVisible:  a.runtime.HUDVisible,
			HUD:         a.runtime.HUD,
			LabelMode:   a.runtime.LabelMode,
			InfraMode:   a.runtime.InfraMode,
		}

	// ── System ─────────────────────────────────────────────────────────────

	case LoadSystemCmd:
		is.PendingSystemPath = c.Path

	case GetActiveSystemCmd:
		c.RespCh <- is.ActiveSystemPath

	// ── HUD ─────────────────────────────────────────────────────────────────

	case SetHUDCmd:
		switch c.Category {
		case "":
			a.runtime.HUDVisible = c.Visible
		case "debug":
			a.runtime.HUD.Debug = c.Visible
		case "info":
			a.runtime.HUD.Info = c.Visible
		case "help":
			a.runtime.HUD.Help = c.Visible
		case "player":
			a.runtime.HUD.Player = c.Visible
		}

	// ── Orbit ───────────────────────────────────────────────────────────────

	case OrbitCmd:
		idx := findBodyByName(state, c.Name)
		if idx < 0 {
			log.Printf("OrbitCmd: body %q not found", c.Name)
			return
		}
		orbitSpeed := c.SpeedDegPerSec * (math.Pi / 180.0)
		orbitRadians := c.Orbits * 2 * math.Pi
		if cs.Mode == ui.CameraModeJumping {
			// Jump is in flight — defer orbit to run on arrival.
			cs.PendingOrbitSpeed = orbitSpeed
			cs.PendingOrbitRadians = orbitRadians
			return
		}
		obj := state.Objects[idx]
		cs.StartTracking(idx)
		cs.TrackDistance = ui.CalculateAutoZoomDistance(obj.Meta.PhysicalRadius, 0.24)
		cs.TrackOffset = engine.Vector3{}
		cs.UpdateTracking(state)
		cs.OrbitSpeed = orbitSpeed
		cs.OrbitRadiansRemaining = orbitRadians

	// ── Recording ──────────────────────────────────────────────────────────

	case RecordStartCmd:
		if !a.runtime.RecordingActive {
			// Native mode has no render texture — capture would always return nil.
			// Force fixed mode so syncRenderState creates a render texture this frame.
			if a.runtime.RenderMode == RenderModeNative {
				a.runtime.RenderMode = RenderModeFixed
				a.syncRenderState()
				a.recordingForcedFixed = true
			}
			a.startRecording(c.Path)
		}

	case RecordPauseCmd:
		if a.runtime.RecordingActive {
			a.runtime.RecordingPaused = !a.runtime.RecordingPaused
			if a.runtime.RecordingPaused {
				fmt.Println("[REC] Paused")
			} else {
				fmt.Println("[REC] Resumed")
			}
		}

	case RecordStopCmd:
		if a.runtime.RecordingActive {
			a.stopRecording()
			// Restore native mode if we forced a switch on record start.
			if a.recordingForcedFixed {
				a.recordingForcedFixed = false
				a.runtime.RenderMode = RenderModeNative
				a.syncRenderState()
			}
		}

	// ── Keybindings ─────────────────────────────────────────────────────────

	case ReloadKeymapCmd:
		newKm, err := input.LoadKeyMap(defaultProfilesDir, c.Path)
		if err != nil {
			log.Printf("ReloadKeymapCmd: %v", err)
			return
		}
		a.keyMap.Store(newKm)
		a.runtime.KeybindingsPath = c.Path
		a.runtime.Settings.KeybindingsPath = c.Path
		a.runtime.Settings.SaveAsPath = c.Path
		a.runtime.Settings.KeybindsDirty = false
		a.runtime.Settings.AvailableFiles = nil
		a.cfg.AppConfig.KeybindingsPath = c.Path

	case SaveKeybindingsCmd:
		km := a.keyMap.Load()
		if err := input.WriteKeybindingsFile(c.Path, km, "laptop"); err != nil {
			log.Printf("SaveKeybindingsCmd: %v", err)
			return
		}
		a.runtime.KeybindingsPath = c.Path
		a.runtime.Settings.KeybindingsPath = c.Path
		a.runtime.Settings.KeybindsDirty = false
		a.cfg.AppConfig.KeybindingsPath = c.Path

	case ScanKeybindFilesCmd:
		files, err := input.ScanKeybindingsDir(c.Dir)
		if err != nil {
			log.Printf("ScanKeybindFilesCmd: %v", err)
			files = nil
		}
		c.RespCh <- files

	case GetKeymapCmd:
		km := a.keyMap.Load()
		all := input.AllActions()
		entries := make([]KeyBindingEntry, 0, len(all))
		for _, action := range all {
			key := km.BoundKey(action)
			if key == 0 {
				continue
			}
			mods := input.ModsToStrings(km.BoundMods(action))
			entries = append(entries, KeyBindingEntry{
				Action: action.String(),
				Key:    input.KeyName(key),
				Mods:   mods,
			})
		}
		c.RespCh <- entries
	}
}

// bodyAliases maps common alternative names to their canonical data-file names
// (lower-case). Add entries here as new aliases are needed.
var bodyAliases = map[string]string{
	"sun": "sol",
}

// findBodyByName returns the index of the first body whose name matches
// (case-insensitive) in state.Objects, or -1 if not found.
// Common aliases (e.g. "sun" → "Sol") are also resolved.
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
