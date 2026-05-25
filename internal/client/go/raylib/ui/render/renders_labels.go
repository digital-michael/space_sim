package render

import (
	"math"
	"sort"

	engine "github.com/digital-michael/space_sim/internal/sim/engine"
	"github.com/digital-michael/space_sim/internal/client/go/raylib/ui"
	rl "github.com/gen2brain/raylib-go/raylib"
)

func projectToScreen(objPos rl.Vector3, camera rl.Camera3D, screenWidth, screenHeight int32) (rl.Vector2, bool) {
	matView := rl.GetCameraMatrix(camera)
	matProj := rl.MatrixPerspective(
		engine.CameraFOV*rl.Deg2rad,
		float32(screenWidth)/float32(screenHeight),
		engine.CameraNearPlane,
		engine.CameraFarPlane,
	)
	mvp := rl.MatrixMultiply(matProj, matView)
	// Transform point: clip = MVP * [x, y, z, 1] (column-major matrix layout)
	cx := mvp.M0*objPos.X + mvp.M4*objPos.Y + mvp.M8*objPos.Z + mvp.M12
	cy := mvp.M1*objPos.X + mvp.M5*objPos.Y + mvp.M9*objPos.Z + mvp.M13
	cw := mvp.M3*objPos.X + mvp.M7*objPos.Y + mvp.M11*objPos.Z + mvp.M15
	if cw <= 0 {
		return rl.Vector2{}, false // behind or on the camera plane
	}
	return rl.Vector2{
		X: (cx/cw + 1.0) * 0.5 * float32(screenWidth),
		Y: (1.0 - cy/cw) * 0.5 * float32(screenHeight),
	}, true
}

// drawObjectLabels draws labels for important/visible objects with connector lines
func drawObjectLabels(state *engine.SimulationState, cameraState *ui.CameraState, camera rl.Camera3D, objectsToRender []*engine.Object, mode ui.LabelMode, screenW, screenH int32) {
	fontSize := scaledInt32(16)
	labelOffsetX := float32(scaledInt32(15))
	labelOffsetY := float32(scaledInt32(10))
	edgePadding := float32(scaledInt32(10))
	padding := float32(scaledInt32(4))
	lineWidth := float32(scaledInt32(2))

	// Determine which objects should have labels based on priority
	labeledObjects := selectObjectsForLabels(state, cameraState, objectsToRender, mode)

	// Project object positions to screen space and draw labels
	for _, obj := range labeledObjects {
		// Camera-relative position keeps GetWorldToScreenEx numerically stable.
		objPos := rl.Vector3{
			X: obj.Anim.Position.X - cameraState.Position.X,
			Y: obj.Anim.Position.Y - cameraState.Position.Y,
			Z: obj.Anim.Position.Z - cameraState.Position.Z,
		}

		// Project to screen space using the same perspective as the scene render.
		// GetWorldToScreenEx has a hardcoded far plane of 1000 su; beyond ~10 AU
		// the projected w goes negative and labels appear to orbit the screen centre.
		screenPos, inFront := projectToScreen(objPos, camera, screenW, screenH)
		if !inFront {
			continue
		}
		if screenPos.X < -1000 || screenPos.X > float32(screenW+1000) || screenPos.Y < -1000 || screenPos.Y > float32(screenH+1000) {
			continue
		}

		// Get object color for label/line
		labelColor := rl.Color{R: obj.Meta.Color.R, G: obj.Meta.Color.G, B: obj.Meta.Color.B, A: 255}

		// For very dark objects, use a brighter version
		brightness := float32(labelColor.R) + float32(labelColor.G) + float32(labelColor.B)
		if brightness < 150 {
			labelColor = rl.Color{R: 200, G: 200, B: 200, A: 255}
		}

		// Draw label text with background
		labelText := obj.Meta.Name
		textSize := rl.MeasureTextEx(rl.GetFontDefault(), labelText, float32(fontSize), 1.0)

		// Position label offset from object (right and slightly up)
		labelX := screenPos.X + labelOffsetX
		labelY := screenPos.Y - labelOffsetY

		// Clamp label position to screen bounds
		if labelX+textSize.X > float32(screenW)-edgePadding {
			labelX = screenPos.X - textSize.X - labelOffsetX
		}
		if labelY < edgePadding {
			labelY = edgePadding
		}
		if labelY+textSize.Y > float32(screenH)-edgePadding {
			labelY = float32(screenH) - textSize.Y - edgePadding
		}

		// Draw background box
		rl.DrawRectangle(int32(labelX-padding), int32(labelY-padding), int32(textSize.X+padding*2), int32(textSize.Y+padding*2), rl.Color{R: 0, G: 0, B: 0, A: 180})
		rl.DrawRectangleLines(int32(labelX-padding), int32(labelY-padding), int32(textSize.X+padding*2), int32(textSize.Y+padding*2), labelColor)

		// Draw text
		rl.DrawText(labelText, int32(labelX), int32(labelY), fontSize, labelColor)

		// Draw connector line from object to label
		rl.DrawLineEx(screenPos, rl.Vector2{X: labelX, Y: labelY + textSize.Y/2}, lineWidth, rl.Color{R: labelColor.R, G: labelColor.G, B: labelColor.B, A: 150})
	}
}

// selectObjectsForLabels determines which objects should have labels based on priority
func selectObjectsForLabels(state *engine.SimulationState, cameraState *ui.CameraState, objectsToRender []*engine.Object, mode ui.LabelMode) []*engine.Object {
	type labelCandidate struct {
		obj      *engine.Object
		priority float32
		dist     float32
	}

	candidates := make([]labelCandidate, 0, len(objectsToRender))

	// Calculate camera position
	camPos := cameraState.Position

	for _, obj := range objectsToRender {
		// Skip asteroids, rings, and belts (too numerous/not interesting for labels)
		if obj.Meta.Category == engine.CategoryAsteroid || obj.Meta.Category == engine.CategoryRing || obj.Meta.Category == engine.CategoryBelt {
			continue
		}

		// Calculate distance to camera
		dx := obj.Anim.Position.X - camPos.X
		dy := obj.Anim.Position.Y - camPos.Y
		dz := obj.Anim.Position.Z - camPos.Z
		distToCam := float32(math.Sqrt(float64(dx*dx + dy*dy + dz*dz)))

		// Priority calculation:
		// - Higher importance = higher priority
		// - Currently tracked object = highest priority
		// - Closer objects = higher priority
		// - Stars always high priority

		priority := float32(obj.Meta.Importance)

		// Boost priority for tracked object
		if cameraState.Mode == ui.CameraModeTracking && cameraState.TrackTargetIndex >= 0 {
			if cameraState.TrackTargetIndex < len(state.Objects) && state.Objects[cameraState.TrackTargetIndex] == obj {
				priority += 1000.0 // Tracked object always gets a label
			}
		}

		// Boost priority for stars
		if obj.Meta.Category == engine.CategoryStar {
			priority += 500.0
		}

		// Boost priority for major planets
		if obj.Meta.Category == engine.CategoryPlanet {
			priority += 200.0
		}

		// Closer objects get priority boost (inverse distance with much stronger weight)
		// This ensures nearby moons are prioritized over distant planets
		if distToCam > 0 {
			priority += 5000.0 / (distToCam + 1.0)
		}

		candidates = append(candidates, labelCandidate{obj: obj, priority: priority, dist: distToCam})
	}

	// Sort by priority (highest first)
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].priority > candidates[j].priority
	})

	// Determine maxLabels and apply nearest-mode proximity filter
	maxLabels := 20
	if mode == ui.LabelModeNearest {
		maxLabels = 5
		threshold := float32(10.0)
		if cameraState.Mode == ui.CameraModeTracking && cameraState.TrackDistance > 0 {
			threshold = float32(math.Max(10.0, cameraState.TrackDistance*3.0))
		}
		targetIdx := -1
		if cameraState.Mode == ui.CameraModeTracking {
			targetIdx = cameraState.TrackTargetIndex
		} else if cameraState.Mode == ui.CameraModeJumping {
			targetIdx = cameraState.JumpTargetIndex
		}
		filtered := candidates[:0]
		for _, c := range candidates {
			isTarget := targetIdx >= 0 && targetIdx < len(state.Objects) && state.Objects[targetIdx] == c.obj
			if isTarget || c.dist <= threshold {
				filtered = append(filtered, c)
			}
		}
		candidates = filtered
	}

	// Take top N candidates
	result := make([]*engine.Object, 0, maxLabels)
	for i := 0; i < len(candidates) && i < maxLabels; i++ {
		result = append(result, candidates[i].obj)
	}

	return result
}

// drawHUDDebug draws the upper-left statistics block and the lower-left
// screen/render diagnostic lines.
