package app

import (
	"context"
	"log"

	engine "github.com/digital-michael/space_sim/internal/space/engine"
	spatial "github.com/digital-michael/space_sim/internal/space/raylib/spatial"
	rl "github.com/gen2brain/raylib-go/raylib"
)

func (a *App) runInteractive(ctx context.Context, session *runtimeSession) error {
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
		state := session.sim.GetState().LockFront()

		if session.debugTracker != nil {
			session.debugTracker.CheckVisibility(state.Objects, "after LockFront()")
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

		zoomIndicator := updateCameraState(session.cameraState, session.inputState, state, dt, a.runtime.CameraSpeed, a.runtime.MouseSensitivity, a.runtime.MouseModeEnabled)

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
			a.renderer.DrawHUD(state, session.cameraState, session.inputState, a.runtime.AsteroidDataset, a.runtime.MouseModeEnabled, session.sim, inViewCount, eligibleInViewCount, renderedCount)
		}
		if a.runtime.LabelsVisible {
			a.renderer.DrawObjectLabels(state, session.cameraState, camera, objectsToRender)
			rl.DrawText("[Labels: ON]", 10, renderHeight-55, 16, rl.Color{R: 100, G: 255, B: 100, A: 200})
		}
		if zoomIndicator != 0 {
			a.renderer.DrawZoomIndicator(zoomIndicator)
		}
		if a.runtime.HelpVisible {
			a.renderer.DrawHelpScreen()
		}

		a.renderer.EndFrame(int32(rl.GetScreenWidth()), int32(rl.GetScreenHeight()))
		session.sim.GetState().UnlockFront()
	}

	log.Println("Exiting app loop")
	return nil
}
