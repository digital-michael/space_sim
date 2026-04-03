package app

import (
	"fmt"
	"math"
	"sort"
	"time"

	engine "github.com/digital-michael/space_sim/internal/sim/engine"
	sim "github.com/digital-michael/space_sim/internal/sim/world"
	"github.com/digital-michael/space_sim/internal/client/go/raylib/ui"
	rl "github.com/gen2brain/raylib-go/raylib"
)

func currentScreenWidth() int {
	return rl.GetScreenWidth()
}

func currentScreenHeight() int {
	return rl.GetScreenHeight()
}

// InstanceBatch represents a group of objects with the same rendering properties
type InstanceBatch struct {
	rings      int32
	slices     int32
	radius     float32
	objects    []*engine.Object
	isPoint    bool
	pointSize  float32
	wireRings  int32
	wireSlices int32
}

// drawObjectsInstanced draws objects using batching to reduce draw calls
func drawObjectsInstanced(objects []*engine.Object, cameraPos engine.Vector3, pointRenderingEnabled bool, lodEnabled bool, importanceThreshold int) int {
	// Group objects into batches by their rendering properties
	batches := make(map[string]*InstanceBatch)

	for _, obj := range objects {
		// Skip objects below importance threshold
		if obj.Meta.Importance < importanceThreshold {
			continue
		}

		// Skip rings - they need individual rendering
		if obj.Meta.InnerRadius > 0 {
			drawObject(obj, cameraPos, pointRenderingEnabled, lodEnabled)
			continue
		}

		// Calculate distance and determine rendering properties
		dx := obj.Anim.Position.X - cameraPos.X
		dy := obj.Anim.Position.Y - cameraPos.Y
		dz := obj.Anim.Position.Z - cameraPos.Z
		distance := math.Sqrt(float64(dx*dx + dy*dy + dz*dz))

		// Determine if this should be a point
		isPoint := false
		pointSize := engine.PointSizeDefault
		if pointRenderingEnabled {
			pointThreshold := engine.PointThresholdDefault
			if obj.Meta.Category == engine.CategoryAsteroid {
				pointThreshold = engine.PointThresholdAsteroid
			} else if obj.Meta.Category == engine.CategoryPlanet {
				pointThreshold = engine.PointThresholdPlanet
			} else if obj.Meta.Category == engine.CategoryMoon {
				pointThreshold = engine.PointThresholdMoon
			}

			if distance > pointThreshold || obj.Meta.PhysicalRadius < 0.5 {
				isPoint = true
				if obj.Meta.Category == engine.CategoryPlanet {
					pointSize = engine.PointSizePlanet
				} else if obj.Meta.Category == engine.CategoryMoon {
					pointSize = engine.PointSizeMoon
				}
			}
		}

		// Determine LOD level
		rings := int32(16)
		slices := int32(16)
		wireRings := int32(8)
		wireSlices := int32(8)

		if lodEnabled && !isPoint {
			if distance < engine.LODVeryClose {
				rings, slices = 32, 32
			} else if distance < engine.LODClose {
				rings, slices = 24, 24
			} else if distance < engine.LODMedium {
				rings, slices = 16, 16
			} else if distance < engine.LODFar {
				rings, slices = 12, 12
			} else {
				rings, slices = 6, 6 // Reduced for better FPS
			}

			if obj.Meta.PhysicalRadius < 1.0 {
				rings, slices = rings/2, slices/2
				if rings < 4 {
					rings = 4
				}
				if slices < 4 {
					slices = 4
				}
			}

			if distance > 50.0 {
				wireRings, wireSlices = 4, 4
			}
		}

		// Create batch key based on rendering properties
		// Round radius to reduce batch fragmentation
		radiusBucket := float32(math.Round(float64(obj.Meta.PhysicalRadius)*10.0) / 10.0)

		batchKey := fmt.Sprintf("%d_%d_%.1f_%v_%.1f_%d_%d",
			rings, slices, radiusBucket, isPoint, pointSize, wireRings, wireSlices)

		batch, exists := batches[batchKey]
		if !exists {
			batch = &InstanceBatch{
				rings:      rings,
				slices:     slices,
				radius:     radiusBucket,
				objects:    make([]*engine.Object, 0, 100),
				isPoint:    isPoint,
				pointSize:  pointSize,
				wireRings:  wireRings,
				wireSlices: wireSlices,
			}
			batches[batchKey] = batch
		}

		batch.objects = append(batch.objects, obj)
	}

	// Draw each batch
	drawnCount := 0
	for _, batch := range batches {
		for _, obj := range batch.objects {
			pos := rl.Vector3{
				X: float32(obj.Anim.Position.X),
				Y: float32(obj.Anim.Position.Y),
				Z: float32(obj.Anim.Position.Z),
			}

			color := rl.Color{
				R: obj.Meta.Color.R,
				G: obj.Meta.Color.G,
				B: obj.Meta.Color.B,
				A: obj.Meta.Color.A,
			}

			if batch.isPoint {
				// Draw as point
				rl.DrawSphere(pos, batch.pointSize*0.1, color)
			} else {
				// Draw as sphere with batch properties
				rl.DrawSphereEx(pos, float32(obj.Meta.PhysicalRadius), batch.rings, batch.slices, color)

				// Wireframe
				if obj.Meta.Material != engine.MaterialEmissive {
					rl.DrawSphereWires(pos, float32(obj.Meta.PhysicalRadius), batch.wireRings, batch.wireSlices,
						rl.Color{R: 255, G: 255, B: 255, A: 100})
				}
			}

			drawnCount++
		}
	}

	return drawnCount
}

// drawObject renders a single object
func drawObject(obj *engine.Object, cameraPos engine.Vector3, pointRenderingEnabled bool, lodEnabled bool) {
	pos := rl.Vector3{
		X: float32(obj.Anim.Position.X),
		Y: float32(obj.Anim.Position.Y),
		Z: float32(obj.Anim.Position.Z),
	}

	color := rl.Color{
		R: obj.Meta.Color.R,
		G: obj.Meta.Color.G,
		B: obj.Meta.Color.B,
		A: obj.Meta.Color.A,
	}

	// Calculate distance from camera (used by both point rendering and LOD)
	dx := obj.Anim.Position.X - cameraPos.X
	dy := obj.Anim.Position.Y - cameraPos.Y
	dz := obj.Anim.Position.Z - cameraPos.Z
	distance := math.Sqrt(float64(dx*dx + dy*dy + dz*dz)) // Point rendering takes priority if enabled and distance threshold met
	if pointRenderingEnabled {
		// Use point rendering for distant small objects
		pointThreshold := engine.PointThresholdDefault
		if obj.Meta.Category == engine.CategoryAsteroid {
			pointThreshold = engine.PointThresholdAsteroid
		} else if obj.Meta.Category == engine.CategoryPlanet {
			pointThreshold = engine.PointThresholdPlanet
		} else if obj.Meta.Category == engine.CategoryMoon {
			pointThreshold = engine.PointThresholdMoon
		}

		if distance > pointThreshold || obj.Meta.PhysicalRadius < 0.5 {
			// Draw as a point with size based on object importance
			pointSize := engine.PointSizeDefault
			if obj.Meta.Category == engine.CategoryPlanet {
				pointSize = engine.PointSizePlanet
			} else if obj.Meta.Category == engine.CategoryMoon {
				pointSize = engine.PointSizeMoon
			}

			// Draw point using a small sphere (more visible than DrawPixel3D)
			rl.DrawSphere(pos, pointSize*0.1, color)
			return
		}
	} // Planetary rings are drawn as flat circles, not spheres
	// Check if this object has an InnerRadius (rings have inner radius > 0)
	if obj.Meta.InnerRadius > 0 {
		// This is a ring - draw as a flat disc in XZ plane with axial tilt
		segments := 64 // More segments for smoother rings
		outerRadius := float32(obj.Meta.PhysicalRadius)
		innerRadius := float32(obj.Meta.InnerRadius)

		// Apply axial tilt rotation if present
		rl.PushMatrix()
		// Translate to object position
		rl.Translatef(pos.X, pos.Y, pos.Z)
		// Rotate around X-axis by axial tilt angle
		if obj.Meta.AxialTilt != 0 {
			rl.Rotatef(obj.Meta.AxialTilt, 1, 0, 0)
		}

		// Draw both sides of the ring for visibility from any angle
		// (Ring is now at origin due to translation)
		for side := 0; side < 2; side++ {
			for i := 0; i < segments; i++ {
				angle1 := float32(i) * 2.0 * 3.14159 / float32(segments)
				angle2 := float32(i+1) * 2.0 * 3.14159 / float32(segments)

				// Outer arc (relative to origin)
				p1 := rl.Vector3{X: outerRadius * float32(math.Cos(float64(angle1))), Y: 0, Z: outerRadius * float32(math.Sin(float64(angle1)))}
				p2 := rl.Vector3{X: outerRadius * float32(math.Cos(float64(angle2))), Y: 0, Z: outerRadius * float32(math.Sin(float64(angle2)))}

				// Inner arc (relative to origin)
				p3 := rl.Vector3{X: innerRadius * float32(math.Cos(float64(angle1))), Y: 0, Z: innerRadius * float32(math.Sin(float64(angle1)))}
				p4 := rl.Vector3{X: innerRadius * float32(math.Cos(float64(angle2))), Y: 0, Z: innerRadius * float32(math.Sin(float64(angle2)))}

				// Draw quad (ring segment) - reverse winding for back side
				if side == 0 {
					rl.DrawTriangle3D(p1, p2, p3, color)
					rl.DrawTriangle3D(p2, p4, p3, color)
				} else {
					rl.DrawTriangle3D(p1, p3, p2, color)
					rl.DrawTriangle3D(p2, p3, p4, color)
				}
			}
		}

		rl.PopMatrix()
		return
	}

	// Draw sphere for planets and other objects with LOD support
	// Determine sphere quality based on distance
	rings := int32(16)
	slices := int32(16)

	if lodEnabled {
		// LOD levels based on distance
		if distance < engine.LODVeryClose {
			// Very close: High detail
			rings = 32
			slices = 32
		} else if distance < engine.LODClose {
			// Close: Medium-high detail
			rings = 24
			slices = 24
		} else if distance < engine.LODMedium {
			// Medium: Medium detail
			rings = 16
			slices = 16
		} else if distance < engine.LODFar {
			// Far: Low detail
			rings = 12
			slices = 12
		} else {
			// Very far: Minimal detail (reduced for better FPS with many objects)
			rings = 6
			slices = 6
		}

		// Smaller objects get simpler geometry
		if obj.Meta.PhysicalRadius < 1.0 {
			rings = rings / 2
			slices = slices / 2
			if rings < 4 {
				rings = 4
			}
			if slices < 4 {
				slices = 4
			}
		}
	}

	rl.DrawSphereEx(pos, float32(obj.Meta.PhysicalRadius), rings, slices, color)

	// Draw wireframe for better depth perception (skip for rings and sun)
	// Use simpler wireframe for distant objects when LOD is enabled
	if obj.Meta.Material != engine.MaterialEmissive {
		wireRings := int32(8)
		wireSlices := int32(8)
		if lodEnabled && distance > 50.0 {
			wireRings = 4
			wireSlices = 4
		}
		rl.DrawSphereWires(pos, float32(obj.Meta.PhysicalRadius), wireRings, wireSlices, rl.Color{R: 255, G: 255, B: 255, A: 100})
	}
}

// drawGroundPlane draws a grid for spatial reference
func drawGroundPlane() {
	gridSize := 30
	gridSpacing := float32(2.0)

	for i := -gridSize; i <= gridSize; i++ {
		// Lines parallel to X-axis
		start := rl.Vector3{X: float32(i) * gridSpacing, Y: -1, Z: float32(-gridSize) * gridSpacing}
		end := rl.Vector3{X: float32(i) * gridSpacing, Y: -1, Z: float32(gridSize) * gridSpacing}
		rl.DrawLine3D(start, end, rl.DarkGray)

		// Lines parallel to Z-axis
		start = rl.Vector3{X: float32(-gridSize) * gridSpacing, Y: -1, Z: float32(i) * gridSpacing}
		end = rl.Vector3{X: float32(gridSize) * gridSpacing, Y: -1, Z: float32(i) * gridSpacing}
		rl.DrawLine3D(start, end, rl.DarkGray)
	}
}

// drawObjectLabels draws labels for important/visible objects with connector lines
func drawObjectLabels(state *engine.SimulationState, cameraState *ui.CameraState, camera rl.Camera3D, objectsToRender []*engine.Object) {
	// Determine which objects should have labels based on priority
	labeledObjects := selectObjectsForLabels(state, cameraState, objectsToRender, 20) // Max 20 labels

	// Project object positions to screen space and draw labels
	for _, obj := range labeledObjects {
		// Get object position in 3D
		objPos := rl.Vector3{X: obj.Anim.Position.X, Y: obj.Anim.Position.Y, Z: obj.Anim.Position.Z}

		// Project to screen space
		screenPos := rl.GetWorldToScreen(objPos, camera)

		// Skip if behind camera (w < 0 in projection)
		if screenPos.X < -1000 || screenPos.X > float32(currentScreenWidth()+1000) || screenPos.Y < -1000 || screenPos.Y > float32(currentScreenHeight()+1000) {
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
		fontSize := int32(16)
		textSize := rl.MeasureTextEx(rl.GetFontDefault(), labelText, float32(fontSize), 1.0)

		// Position label offset from object (right and slightly up)
		labelX := screenPos.X + 15
		labelY := screenPos.Y - 10

		// Clamp label position to screen bounds
		if labelX+textSize.X > float32(currentScreenWidth()-10) {
			labelX = screenPos.X - textSize.X - 15
		}
		if labelY < 10 {
			labelY = 10
		}
		if labelY+textSize.Y > float32(currentScreenHeight()-10) {
			labelY = float32(currentScreenHeight()) - textSize.Y - 10
		}

		// Draw background box
		padding := float32(4)
		rl.DrawRectangle(int32(labelX-padding), int32(labelY-padding), int32(textSize.X+padding*2), int32(textSize.Y+padding*2), rl.Color{R: 0, G: 0, B: 0, A: 180})
		rl.DrawRectangleLines(int32(labelX-padding), int32(labelY-padding), int32(textSize.X+padding*2), int32(textSize.Y+padding*2), labelColor)

		// Draw text
		rl.DrawText(labelText, int32(labelX), int32(labelY), fontSize, labelColor)

		// Draw connector line from object to label
		rl.DrawLineEx(screenPos, rl.Vector2{X: labelX, Y: labelY + textSize.Y/2}, 1.5, rl.Color{R: labelColor.R, G: labelColor.G, B: labelColor.B, A: 150})
	}
}

// selectObjectsForLabels determines which objects should have labels based on priority
func selectObjectsForLabels(state *engine.SimulationState, cameraState *ui.CameraState, objectsToRender []*engine.Object, maxLabels int) []*engine.Object {
	type labelCandidate struct {
		obj      *engine.Object
		priority float32
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

		candidates = append(candidates, labelCandidate{obj: obj, priority: priority})
	}

	// Sort by priority (highest first)
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].priority > candidates[j].priority
	})

	// Take top N candidates
	result := make([]*engine.Object, 0, maxLabels)
	for i := 0; i < len(candidates) && i < maxLabels; i++ {
		result = append(result, candidates[i].obj)
	}

	return result
}

// drawHUD draws the on-screen display
func drawHUD(state *engine.SimulationState, cameraState *ui.CameraState, inputState *ui.InputState, asteroidDataset engine.AsteroidDataset, mouseModeEnabled bool, s *sim.World) {
	fps := rl.GetFPS()
	rl.DrawText(fmt.Sprintf("FPS: %3d / %d threads", fps, state.NumWorkers), 10, 10, 20, rl.Green)

	// Display object counts: total / visible / rendered
	totalObjects := len(state.Objects)
	visibleObjects := 0
	for _, obj := range state.Objects {
		if obj.Visible {
			visibleObjects++
		}
	}
	datasetName := sim.GetDatasetName(asteroidDataset)
	rl.DrawText(fmt.Sprintf("Objects: %d total / %d visible (Dataset: %s)", totalObjects, visibleObjects, datasetName), 10, 35, 20, rl.White)

	// Simulation time display - show date since J2000.0 epoch (Jan 1, 2000, 12:00 TT)
	simSeconds := state.Time

	// J2000.0 epoch: January 1, 2000, 12:00:00 TT (approximately 12:00 UTC)
	// Using 12:00 UTC as base to match the astronomical J2000.0 standard
	j2000 := time.Date(2000, 1, 1, 12, 0, 0, 0, time.UTC)
	currentTime := j2000.Add(time.Duration(simSeconds * float64(time.Second)))

	// Convert to local timezone
	localTime := currentTime.Local()

	year := localTime.Year()
	month := int(localTime.Month())
	day := localTime.Day()

	timeText := fmt.Sprintf("Date: %04d/%02d/%02d", year, month, day)
	if state.SecondsPerSecond < 86400.0 {
		hour := localTime.Hour()
		minute := localTime.Minute()
		second := localTime.Second()
		millisecond := localTime.Nanosecond() / 1000000
		timeText = fmt.Sprintf("Date: %04d/%02d/%02d %02d:%02d:%02d.%03d",
			year, month, day, hour, minute, second, millisecond)
	}
	rl.DrawText(timeText, 10, 60, 20, rl.White)

	// Time rate indicator (simulation seconds per real second)
	var timeRateText string
	timeRateColor := rl.Gray

	sps := state.SecondsPerSecond
	if sps == 0.0 {
		timeRateText = "Time Rate: PAUSED"
		timeRateColor = rl.Red
	} else if sps == 1.0 {
		timeRateText = "Time Rate: real-time"
		timeRateColor = rl.White
	} else if sps == 3600.0 {
		timeRateText = "Time Rate: 1 hour/sec"
		timeRateColor = rl.Green
	} else if sps == 86400.0 {
		timeRateText = "Time Rate: 1 day/sec"
		timeRateColor = rl.Green
	} else if sps == 604800.0 {
		timeRateText = "Time Rate: 1 week/sec"
		timeRateColor = rl.Green
	} else if sps == 2628000.0 {
		timeRateText = "Time Rate: 1 month/sec"
		timeRateColor = rl.Yellow
	} else if sps == 31557600.0 {
		timeRateText = "Time Rate: 1 year/sec"
		timeRateColor = rl.Yellow
	} else {
		timeRateText = fmt.Sprintf("Time Rate: %.0f sec/sec", sps)
		timeRateColor = rl.Gray
	}
	rl.DrawText(timeRateText, 10, 85, 18, timeRateColor)

	// Anim speed indicator (physics tick rate as % of full 60Hz)
	animSpeed := s.GetSpeed()
	var animSpeedText string
	var animSpeedColor rl.Color
	if animSpeed == 0.0 {
		animSpeedText = "Anim Speed: PAUSED"
		animSpeedColor = rl.Red
	} else if animSpeed >= 1.0 {
		animSpeedText = "Anim Speed: 100%"
		animSpeedColor = rl.White
	} else {
		animSpeedText = fmt.Sprintf("Anim Speed: %d%%", int(animSpeed*100))
		animSpeedColor = rl.Color{R: 255, G: 165, B: 0, A: 255} // orange — below full speed
	}
	rl.DrawText(animSpeedText, 10, 108, 18, animSpeedColor)

	// Camera mode indicator
	var modeText string
	switch cameraState.Mode {
	case ui.CameraModeFree:
		modeText = "FREE-FLY"
	case ui.CameraModeJumping:
		modeText = "JUMPING"
	case ui.CameraModeTracking:
		modeText = fmt.Sprintf("TRACKING: %s", state.Objects[cameraState.TrackTargetIndex].Meta.Name)
	}
	rl.DrawText(fmt.Sprintf("Mode: %s", modeText), 10, 133, 20, rl.Yellow)

	// Camera position
	posText := fmt.Sprintf("Camera Position: X:%.1f Y:%.1f Z:%.1f", cameraState.Position.X, cameraState.Position.Y, cameraState.Position.Z)
	rl.DrawText(posText, 10, 158, 18, rl.Color{R: 0, G: 255, B: 255, A: 255})

	// Help hint (only shows when HUD is visible)
	rl.DrawText("Ctrl+/ for help | Ctrl+Q to quit", 10, int32(currentScreenHeight())-30, 20, rl.Gray)

	// Object selection UI
	if inputState.SelectionActive {
		drawSelectionUI(state, inputState)
	}

	// Tracking info HUD (lower right)
	if cameraState.Mode == ui.CameraModeTracking {
		drawTrackingInfo(state, cameraState)
	}
}

// drawZoomIndicator draws a visual indicator when zooming
func drawZoomIndicator(zoomValue float32) {
	centerX := int32(currentScreenWidth() / 2)
	centerY := int32(currentScreenHeight() / 2)

	// Determine zoom direction and text
	var text string
	var color rl.Color
	var barLength int32

	if zoomValue > 0 {
		text = "ZOOM IN"
		color = rl.Color{R: 100, G: 200, B: 255, A: 220}
		barLength = int32(zoomValue * 50)
	} else {
		text = "ZOOM OUT"
		color = rl.Color{R: 255, G: 150, B: 100, A: 220}
		barLength = int32(-zoomValue * 50)
	}

	if barLength > 200 {
		barLength = 200
	}

	// Draw semi-transparent background
	bgWidth := int32(260)
	bgHeight := int32(60)
	rl.DrawRectangle(centerX-bgWidth/2, centerY+150, bgWidth, bgHeight, rl.Color{R: 0, G: 0, B: 0, A: 150})

	// Draw text
	textWidth := rl.MeasureText(text, 20)
	rl.DrawText(text, centerX-textWidth/2, centerY+160, 20, color)

	// Draw zoom bar
	barY := centerY + 185
	rl.DrawRectangle(centerX-100, barY, 200, 8, rl.Color{R: 50, G: 50, B: 50, A: 200})

	if zoomValue > 0 {
		// Zoom in - bar grows from center to right
		rl.DrawRectangle(centerX, barY, barLength, 8, color)
	} else {
		// Zoom out - bar grows from center to left
		rl.DrawRectangle(centerX-barLength, barY, barLength, 8, color)
	}
}

// drawTrackingInfo displays comprehensive information about the tracked object in the lower right
func drawTrackingInfo(state *engine.SimulationState, cameraState *ui.CameraState) {
	if cameraState.TrackTargetIndex < 0 || cameraState.TrackTargetIndex >= len(state.Objects) {
		return
	}

	obj := state.Objects[cameraState.TrackTargetIndex]

	// Calculate display metrics
	const auToSimUnits = 100.0 // 100 units = 1 AU

	// Distance from Sol (distance from origin in AUs)
	distFromSol := obj.Anim.Position.Sub(engine.Vector3{X: 0, Y: 0, Z: 0}).Length() / auToSimUnits

	// Camera distance to object
	cameraDistUnits := cameraState.Position.Sub(obj.Anim.Position).Length()
	cameraDistAU := cameraDistUnits / auToSimUnits
	cameraDistKm := cameraDistAU * 149597870.7 // 1 AU in km

	// Determine parent and count siblings
	parentName := "Sol"
	siblingCount := 0
	siblingIndex := 0

	if obj.Meta.ParentName != "" {
		parentName = obj.Meta.ParentName
		// Count objects with same parent
		for i, otherObj := range state.Objects {
			if otherObj.Meta.ParentName == obj.Meta.ParentName && otherObj.Meta.Category == obj.Meta.Category {
				siblingCount++
				if i <= cameraState.TrackTargetIndex && otherObj.Meta.Name == obj.Meta.Name {
					siblingIndex = siblingCount
				}
			}
		}
	} else if obj.Meta.Category == engine.CategoryPlanet {
		// Count planets (objects orbiting Sol with Category=Planet)
		for i, otherObj := range state.Objects {
			if otherObj.Meta.Category == engine.CategoryPlanet && otherObj.Meta.ParentName == "" {
				siblingCount++
				if i <= cameraState.TrackTargetIndex && otherObj.Meta.Name == obj.Meta.Name {
					siblingIndex = siblingCount
				}
			}
		}
	} else if obj.Meta.Category == engine.CategoryDwarfPlanet {
		// Count dwarf planets
		for i, otherObj := range state.Objects {
			if otherObj.Meta.Category == engine.CategoryDwarfPlanet {
				siblingCount++
				if i <= cameraState.TrackTargetIndex && otherObj.Meta.Name == obj.Meta.Name {
					siblingIndex = siblingCount
				}
			}
		}
	} else if obj.Meta.Category == engine.CategoryAsteroid {
		// For asteroids, show belt membership
		if len(obj.Meta.Name) >= 9 && obj.Meta.Name[0:9] == "Asteroid-" {
			parentName = "Asteroid Belt"
		} else {
			parentName = "Kuiper Belt"
		}
		// Count visible asteroids in same belt
		for i, otherObj := range state.Objects {
			if otherObj.Visible && otherObj.Meta.Category == engine.CategoryAsteroid {
				namePrefix := ""
				if len(otherObj.Meta.Name) >= 9 {
					namePrefix = otherObj.Meta.Name[0:9]
				}
				objPrefix := ""
				if len(obj.Meta.Name) >= 9 {
					objPrefix = obj.Meta.Name[0:9]
				}
				if namePrefix == objPrefix {
					siblingCount++
					if i <= cameraState.TrackTargetIndex && otherObj.Meta.Name == obj.Meta.Name {
						siblingIndex = siblingCount
					}
				}
			}
		}
	}

	// Format category name
	categoryName := ""
	switch obj.Meta.Category {
	case engine.CategoryPlanet:
		categoryName = "Planet"
	case engine.CategoryDwarfPlanet:
		categoryName = "Dwarf Planet"
	case engine.CategoryMoon:
		categoryName = "Moon"
	case engine.CategoryAsteroid:
		categoryName = "Asteroid"
	case engine.CategoryStar:
		categoryName = "Star"
	case engine.CategoryRing:
		categoryName = "Ring System"
	}

	// Format mass (scientific notation)
	massStr := fmt.Sprintf("%.2e kg", obj.Meta.Mass)

	// Format rotation period
	rotationStr := "Unknown"
	if obj.Meta.RotationPeriod > 0 {
		if obj.Meta.RotationPeriod < 48 {
			rotationStr = fmt.Sprintf("%.1f hours", obj.Meta.RotationPeriod)
		} else {
			days := obj.Meta.RotationPeriod / 24.0
			rotationStr = fmt.Sprintf("%.1f days", days)
		}

		// Check for tidal locking (rotation ~= orbital period)
		if obj.Meta.ParentName != "" && obj.Meta.OrbitalPeriod > 0 {
			orbitalDays := obj.Meta.OrbitalPeriod / 86400.0 // Convert seconds to days
			rotationalDays := obj.Meta.RotationPeriod / 24.0
			if math.Abs(float64(orbitalDays-rotationalDays)) < 0.5 {
				rotationStr += " (tidally locked)"
			}
		}
	}

	// Format orbital period
	orbitalStr := "N/A"
	if obj.Meta.OrbitalPeriod > 0 {
		days := obj.Meta.OrbitalPeriod / 86400.0 // Convert seconds to Earth days
		if days < 1 {
			hours := obj.Meta.OrbitalPeriod / 3600.0
			orbitalStr = fmt.Sprintf("%.2f hours", hours)
		} else if days < 730 {
			orbitalStr = fmt.Sprintf("%.1f days", days)
		} else {
			years := days / 365.256
			orbitalStr = fmt.Sprintf("%.1f years", years)
		}
	}

	// Format camera distance (auto-scale)
	cameraDistStr := ""
	if cameraDistKm < 1000 {
		cameraDistStr = fmt.Sprintf("%.1f km", cameraDistKm)
	} else if cameraDistAU < 0.1 {
		cameraDistStr = fmt.Sprintf("%.0f km", cameraDistKm)
	} else if cameraDistAU < 10 {
		cameraDistStr = fmt.Sprintf("%.3f AU", cameraDistAU)
	} else {
		cameraDistStr = fmt.Sprintf("%.1f AU", cameraDistAU)
	}

	// Format eccentricity
	eccentricityStr := fmt.Sprintf("%.3f", obj.Meta.Eccentricity)
	if obj.Meta.Eccentricity < 0.01 {
		eccentricityStr += " (circular)"
	} else if obj.Meta.Eccentricity < 0.1 {
		eccentricityStr += " (low)"
	} else if obj.Meta.Eccentricity < 0.5 {
		eccentricityStr += " (moderate)"
	} else {
		eccentricityStr += " (high)"
	}

	// Calculate orbital phase (percentage through orbit)
	orbitalPhase := 0.0
	if obj.Meta.OrbitalPeriod > 0 {
		// Mean anomaly is in radians, convert to percentage
		phase := float64(obj.Anim.MeanAnomaly) / (2.0 * math.Pi)
		// Normalize to 0-100%
		for phase < 0 {
			phase += 1.0
		}
		for phase > 1 {
			phase -= 1.0
		}
		orbitalPhase = phase * 100.0
	}

	// Build info lines with labels and values separated for color formatting
	type InfoLine struct {
		label string
		value string
	}

	infoLines := []InfoLine{
		{"Object:", obj.Meta.Name},
		{"Type:", categoryName},
		{"Distance from Sol:", fmt.Sprintf("%.2f AU", distFromSol)},
	}

	// Add orbit info if applicable
	if siblingCount > 1 {
		// Only show count if more than one sibling
		infoLines = append(infoLines, InfoLine{"Orbits:", fmt.Sprintf("%s (%d of %d)", parentName, siblingIndex, siblingCount)})
	} else {
		infoLines = append(infoLines, InfoLine{"Orbits:", parentName})
	}

	infoLines = append(infoLines,
		InfoLine{"Mass:", massStr},
		InfoLine{"Rotation:", rotationStr},
	)

	// Add axial tilt if present
	if obj.Meta.AxialTilt != 0 {
		infoLines = append(infoLines, InfoLine{"Axial Tilt:", fmt.Sprintf("%.1f°", obj.Meta.AxialTilt)})
	}

	infoLines = append(infoLines,
		InfoLine{"Orbital Period:", orbitalStr},
	)

	// Add orbital metrics if object orbits something
	if obj.Meta.OrbitalPeriod > 0 {
		infoLines = append(infoLines,
			InfoLine{"Eccentricity:", eccentricityStr},
			InfoLine{"Orbital Phase:", fmt.Sprintf("%.1f%%", orbitalPhase)},
		)
	}

	// Calculate and display orbital velocity
	velocityMagnitude := obj.Anim.Velocity.Length()
	velocityKmPerSec := velocityMagnitude * 1495978.707 // Convert sim units to km/s (1 AU = 149597870.7 km, 1 sim unit = 0.01 AU)
	if velocityKmPerSec > 0.01 {
		infoLines = append(infoLines, InfoLine{"Orbital Velocity:", fmt.Sprintf("%8.2f km/s", velocityKmPerSec)})
	}

	// Calculate and display rotational velocity
	if obj.Meta.RotationPeriod > 0 && obj.Meta.PhysicalRadius > 0 {
		// Rotational velocity at equator: v = 2πr / T
		radiusKm := obj.Meta.PhysicalRadius / 1000.0              // Convert meters to km
		rotationPeriodSeconds := obj.Meta.RotationPeriod * 3600.0 // Convert hours to seconds
		rotationalVelocityKmPerSec := (2.0 * math.Pi * radiusKm) / rotationPeriodSeconds
		infoLines = append(infoLines, InfoLine{"Rotational Velocity:", fmt.Sprintf("%8.2f km/s (at equator)", rotationalVelocityKmPerSec)})
	}

	infoLines = append(infoLines, InfoLine{"Camera Distance:", cameraDistStr})

	// Calculate dimensions
	fontSize := int32(16)
	lineHeight := int32(22)
	padding := int32(15)

	maxWidth := int32(0)
	for _, info := range infoLines {
		fullLine := info.label + " " + info.value
		width := rl.MeasureText(fullLine, fontSize)
		if width > maxWidth {
			maxWidth = width
		}
	}

	boxWidth := maxWidth + padding*2
	boxHeight := int32(len(infoLines))*lineHeight + padding*2

	// Position in lower right corner
	boxX := int32(currentScreenWidth()) - boxWidth - 20
	boxY := int32(currentScreenHeight()) - boxHeight - 20

	// Draw semi-transparent background
	rl.DrawRectangle(boxX, boxY, boxWidth, boxHeight, rl.Color{R: 0, G: 0, B: 0, A: 180})

	// Draw border
	rl.DrawRectangleLines(boxX, boxY, boxWidth, boxHeight, rl.Color{R: 100, G: 150, B: 200, A: 255})

	// Draw lines with colored labels and values
	labelColor := rl.Color{R: 150, G: 200, B: 255, A: 255} // Light blue for labels
	valueColor := rl.Color{R: 255, G: 255, B: 255, A: 255} // White for values

	textY := boxY + padding
	for _, info := range infoLines {
		textX := boxX + padding
		// Draw label in light blue
		rl.DrawText(info.label, textX, textY, fontSize, labelColor)
		labelWidth := rl.MeasureText(info.label, fontSize)

		// Draw value in white, offset by label width plus a space
		rl.DrawText(info.value, textX+labelWidth+rl.MeasureText(" ", fontSize), textY, fontSize, valueColor)
		textY += lineHeight
	}
}

// drawSelectionUI draws the object selection menu with category tabs
func drawSelectionUI(state *engine.SimulationState, inputState *ui.InputState) {
	// Check if we're in performance mode
	if inputState.SelectionMode == ui.SelectionModePerformance {
		drawPerformanceUI(inputState)
		return
	}

	// Semi-transparent background
	bgX := int32(currentScreenWidth()/2 - 250)
	bgY := int32(currentScreenHeight()/2 - 250)
	bgWidth := int32(500)
	bgHeight := int32(500)
	rl.DrawRectangle(bgX, bgY, bgWidth, bgHeight, rl.Color{R: 0, G: 0, B: 0, A: 200})
	rl.DrawRectangleLines(bgX, bgY, bgWidth, bgHeight, rl.White)

	// Title - show different text based on mode
	titleText := "SELECT OBJECT"
	if inputState.SelectionMode == ui.SelectionModeJump {
		titleText = "SELECT OBJECT TO JUMP TO"
	} else if inputState.SelectionMode == ui.SelectionModeTrack {
		titleText = "SELECT OBJECT TO TRACK"
	} else if inputState.SelectionMode == ui.SelectionModeTrackEquatorial {
		titleText = "SELECT OBJECT (EQUATORIAL)"
	}
	rl.DrawText(titleText, bgX+50, bgY+10, 20, rl.White)
	rl.DrawText("UP/DOWN: select, LEFT/RIGHT: category, ENTER: confirm, ESC: cancel", bgX+10, bgY+40, 12, rl.LightGray)
	rl.DrawText("PgUp/PgDn: page, HOME/END: jump to start/end", bgX+10, bgY+55, 12, rl.Gray)

	// Filter text box
	filterY := bgY + 75
	filterBoxHeight := int32(25)
	if inputState.FilterText != "" {
		rl.DrawRectangle(bgX+10, filterY, bgWidth-20, filterBoxHeight, rl.Color{R: 40, G: 40, B: 40, A: 255})
		rl.DrawRectangleLines(bgX+10, filterY, bgWidth-20, filterBoxHeight, rl.Color{R: 100, G: 150, B: 200, A: 255})
		filterDisplay := "Filter: " + inputState.FilterText + "_"
		rl.DrawText(filterDisplay, bgX+15, filterY+5, 16, rl.Green)
		filterY += filterBoxHeight + 5
	} else {
		rl.DrawText("Type to filter...", bgX+15, filterY+5, 14, rl.Gray)
		filterY += filterBoxHeight + 5
	}

	// Category tabs - map display order to ObjectCategory enum values
	type categoryTab struct {
		name     string
		category engine.ObjectCategory
	}
	categories := []categoryTab{
		{"Stars", engine.CategoryStar}, // First tab
		{"Planets", engine.CategoryPlanet},
		{"Dwarf Planets", engine.CategoryDwarfPlanet},
		{"Moons", engine.CategoryMoon},
		{"Belts", engine.CategoryBelt}, // Asteroid Belt and Kuiper Belt
	}
	tabWidth := int32(95)
	tabHeight := int32(30)
	tabY := filterY

	for i, cat := range categories {
		tabX := bgX + 10 + int32(i)*tabWidth
		tabColor := rl.Color{R: 50, G: 50, B: 50, A: 255}
		textColor := rl.LightGray

		// Highlight active category
		if cat.category == inputState.SelectedCategory {
			tabColor = rl.Color{R: 80, G: 120, B: 160, A: 255}
			textColor = rl.White
		}

		rl.DrawRectangle(tabX, tabY, tabWidth-5, tabHeight, tabColor)
		rl.DrawRectangleLines(tabX, tabY, tabWidth-5, tabHeight, rl.White)
		rl.DrawText(cat.name, tabX+5, tabY+8, 14, textColor)
	}

	// Filter objects by category
	if len(inputState.FilteredIndices) == 0 {
		inputState.FilteredIndices = filterObjectsByCategory(state.Objects, inputState.SelectedCategory)
	}

	// Object list (filtered by category) - start below the tabs
	startY := tabY + tabHeight + 10
	lineHeight := int32(30)
	listAreaHeight := bgHeight - (startY - bgY) - 10 // Available height for list
	visibleItems := int(listAreaHeight / lineHeight)
	totalItems := len(inputState.FilteredIndices)

	// Calculate scroll bounds
	maxScroll := totalItems - visibleItems
	if maxScroll < 0 {
		maxScroll = 0
	}
	if inputState.ScrollOffset > maxScroll {
		inputState.ScrollOffset = maxScroll
	}
	if inputState.ScrollOffset < 0 {
		inputState.ScrollOffset = 0
	}

	// Update distance cache every 5 seconds
	currentTime := rl.GetTime()
	if currentTime-inputState.LastDistanceUpdate > 5.0 {
		inputState.DistanceCache = make(map[int]string)
		for _, idx := range inputState.FilteredIndices {
			// Skip virtual belt indices
			if idx < 0 {
				continue
			}
			obj := state.Objects[idx]
			dist := obj.Anim.Position.Sub(engine.Vector3{}).Length()
			inputState.DistanceCache[idx] = fmt.Sprintf("%.0f units", dist)
		}
		inputState.LastDistanceUpdate = currentTime
	}

	// Render only visible items
	for i := inputState.ScrollOffset; i < inputState.ScrollOffset+visibleItems && i < totalItems; i++ {
		actualIndex := inputState.FilteredIndices[i]
		y := startY + int32(i-inputState.ScrollOffset)*lineHeight

		// Highlight selected
		if i == inputState.SelectedIndex {
			rl.DrawRectangle(bgX+5, y-2, bgWidth-10, lineHeight-2, rl.Color{R: 50, G: 100, B: 150, A: 255})
			rl.DrawText(">", bgX+15, y+5, 20, rl.Yellow)
		}

		// Handle virtual belt indices
		if actualIndex == -1 {
			// Asteroid Belt
			rl.DrawText("Asteroid Belt", bgX+40, y+5, 20, rl.White)
			rl.DrawText("195-240 AU", bgX+250, y+5, 16, rl.LightGray)
			rl.DrawRectangleRec(rl.Rectangle{X: float32(bgX + 350), Y: float32(y + 5), Width: 20, Height: 20}, rl.Color{R: 150, G: 150, B: 150, A: 255})
		} else if actualIndex == -2 {
			// Kuiper Belt
			rl.DrawText("Kuiper Belt", bgX+40, y+5, 20, rl.White)
			rl.DrawText("3000-5000 AU", bgX+250, y+5, 16, rl.LightGray)
			rl.DrawRectangleRec(rl.Rectangle{X: float32(bgX + 350), Y: float32(y + 5), Width: 20, Height: 20}, rl.Color{R: 200, G: 150, B: 130, A: 255})
		} else {
			// Normal object
			obj := state.Objects[actualIndex]

			// Object name and info
			nameText := fmt.Sprintf("%s", obj.Meta.Name)
			distText := inputState.DistanceCache[actualIndex]
			if distText == "" {
				distText = "--- units" // Placeholder until first update
			}

			rl.DrawText(nameText, bgX+40, y+5, 20, rl.White)
			rl.DrawText(distText, bgX+250, y+5, 16, rl.LightGray)

			// Color indicator
			colorBox := rl.Rectangle{X: float32(bgX + 350), Y: float32(y + 5), Width: 20, Height: 20}
			rl.DrawRectangleRec(colorBox, rl.Color{R: obj.Meta.Color.R, G: obj.Meta.Color.G, B: obj.Meta.Color.B, A: 255})
		}
	}

	// Draw scroll bar if needed
	if totalItems > visibleItems {
		scrollBarX := bgX + bgWidth - 15
		scrollBarY := startY
		scrollBarHeight := listAreaHeight
		scrollBarWidth := int32(10)

		// Scroll bar background
		rl.DrawRectangle(scrollBarX, scrollBarY, scrollBarWidth, scrollBarHeight, rl.Color{R: 30, G: 30, B: 30, A: 200})

		// Scroll thumb
		thumbHeight := int32(float32(visibleItems) / float32(totalItems) * float32(scrollBarHeight))
		if thumbHeight < 20 {
			thumbHeight = 20 // Minimum thumb size
		}
		thumbY := scrollBarY + int32(float32(inputState.ScrollOffset)/float32(maxScroll)*float32(scrollBarHeight-thumbHeight))
		rl.DrawRectangle(scrollBarX, thumbY, scrollBarWidth, thumbHeight, rl.Color{R: 100, G: 150, B: 200, A: 255})
		rl.DrawRectangleLines(scrollBarX, thumbY, scrollBarWidth, thumbHeight, rl.White)
	}
}

// drawPerformanceUI draws the performance options menu with tabs
func drawPerformanceUI(inputState *ui.InputState) {
	// Semi-transparent background
	bgX := int32(currentScreenWidth()/2 - 250)
	bgY := int32(currentScreenHeight()/2 - 250)
	bgWidth := int32(500)
	bgHeight := int32(500)
	rl.DrawRectangle(bgX, bgY, bgWidth, bgHeight, rl.Color{R: 0, G: 0, B: 0, A: 200})
	rl.DrawRectangleLines(bgX, bgY, bgWidth, bgHeight, rl.White)

	// Title
	rl.DrawText("PERFORMANCE & CONFIGURATION", bgX+80, bgY+10, 24, rl.White)
	rl.DrawText("UP/DOWN: select, SPACE: toggle, LEFT/RIGHT: tab/adjust, ESC: close", bgX+20, bgY+40, 12, rl.LightGray)

	// Draw tabs
	tabWidth := int32(250)
	tabHeight := int32(35)
	tabY := bgY + 65

	// Performance tab
	perfTabColor := rl.Color{R: 50, G: 50, B: 50, A: 255}
	if inputState.PerformanceTab == 0 {
		perfTabColor = rl.Color{R: 80, G: 120, B: 160, A: 255}
	}
	rl.DrawRectangle(bgX, tabY, tabWidth, tabHeight, perfTabColor)
	rl.DrawRectangleLines(bgX, tabY, tabWidth, tabHeight, rl.White)
	rl.DrawText("Performance", bgX+70, tabY+8, 18, rl.White)

	// Configuration tab
	confTabColor := rl.Color{R: 50, G: 50, B: 50, A: 255}
	if inputState.PerformanceTab == 1 {
		confTabColor = rl.Color{R: 80, G: 120, B: 160, A: 255}
	}
	rl.DrawRectangle(bgX+tabWidth, tabY, tabWidth, tabHeight, confTabColor)
	rl.DrawRectangleLines(bgX+tabWidth, tabY, tabWidth, tabHeight, rl.White)
	rl.DrawText("Configuration", bgX+tabWidth+60, tabY+8, 18, rl.White)

	startY := tabY + tabHeight + 20
	lineHeight := int32(60)

	if inputState.PerformanceTab == 0 {
		// Performance tab options
		options := []struct {
			name    string
			desc    string
			enabled *bool
		}{
			{"Frustum Culling", "Cull objects outside camera view", &inputState.PerfOptions.FrustumCulling},
			{"Level of Detail", "Use simpler models for distant objects", &inputState.PerfOptions.LODEnabled},
			{"Instanced Rendering", "Batch objects by rendering properties", &inputState.PerfOptions.InstancedRendering},
			{"Spatial Partitioning", "Use grid-based spatial culling", &inputState.PerfOptions.SpatialPartition},
			{"Point Rendering", "Render distant objects as points", &inputState.PerfOptions.PointRendering},
		}

		for i, opt := range options {
			y := startY + int32(i)*lineHeight

			// Highlight selected
			if i == inputState.SelectedIndex {
				rl.DrawRectangle(bgX+5, y-2, bgWidth-10, lineHeight-2, rl.Color{R: 50, G: 100, B: 150, A: 255})
				rl.DrawText(">", bgX+15, y+10, 20, rl.Yellow)
			}

			// Checkbox
			checkX := bgX + 40
			checkY := y + 8
			checkSize := int32(24)
			rl.DrawRectangle(checkX, checkY, checkSize, checkSize, rl.Color{R: 40, G: 40, B: 40, A: 255})
			rl.DrawRectangleLines(checkX, checkY, checkSize, checkSize, rl.White)

			if *opt.enabled {
				rl.DrawText("X", checkX+5, checkY+2, 20, rl.Green)
			}

			// Option name and description
			rl.DrawText(opt.name, checkX+35, checkY, 18, rl.White)
			rl.DrawText(opt.desc, checkX+35, checkY+22, 12, rl.Gray)
		}

		// Stats
		statsY := bgY + bgHeight - 40
		culledText := "Objects will be culled based on selected optimizations"
		rl.DrawText(culledText, bgX+50, statsY, 14, rl.LightGray)
	} else {
		// Configuration tab options
		// Option 0: Importance Threshold
		y := startY

		// Highlight selected
		if 0 == inputState.SelectedIndex {
			rl.DrawRectangle(bgX+5, y-2, bgWidth-10, lineHeight-2, rl.Color{R: 50, G: 100, B: 150, A: 255})
			rl.DrawText(">", bgX+15, y+10, 20, rl.Yellow)
		}

		checkX := bgX + 40
		checkY := y + 8
		rl.DrawText("Importance Threshold", checkX, checkY, 18, rl.White)

		// Value and description
		thresholdDesc := ""
		switch inputState.PerfOptions.ImportanceThreshold {
		case 0:
			thresholdDesc = "0 (All objects)"
		case 5:
			thresholdDesc = "5 (Hide tiny asteroids)"
		case 10:
			thresholdDesc = "10 (Hide medium asteroids)"
		case 15:
			thresholdDesc = "15 (Hide all asteroids)"
		case 30:
			thresholdDesc = "30 (Hide rings)"
		case 40:
			thresholdDesc = "40 (Hide small moons)"
		case 50:
			thresholdDesc = "50 (Hide dwarf planets)"
		case 60:
			thresholdDesc = "60 (Hide large moons)"
		case 70:
			thresholdDesc = "70 (Hide ice giants)"
		case 80:
			thresholdDesc = "80 (Hide rocky planets)"
		case 90:
			thresholdDesc = "90 (Hide gas giants)"
		default:
			thresholdDesc = fmt.Sprintf("%d", inputState.PerfOptions.ImportanceThreshold)
		}

		rl.DrawText(fmt.Sprintf("Value: %s", thresholdDesc), checkX, checkY+22, 12, rl.Gray)
		rl.DrawText("LEFT/RIGHT: adjust", checkX+250, checkY+22, 12, rl.LightGray)

		// Option 1: Zero-Allocation In-Place Swap
		y = startY + lineHeight

		// Highlight selected
		if 1 == inputState.SelectedIndex {
			rl.DrawRectangle(bgX+5, y-2, bgWidth-10, lineHeight-2, rl.Color{R: 50, G: 100, B: 150, A: 255})
			rl.DrawText(">", bgX+15, y+10, 20, rl.Yellow)
		}

		checkY = y + 8
		checkSize := int32(24)
		rl.DrawRectangle(checkX, checkY, checkSize, checkSize, rl.Color{R: 40, G: 40, B: 40, A: 255})
		rl.DrawRectangleLines(checkX, checkY, checkSize, checkSize, rl.White)

		if inputState.PerfOptions.UseInPlaceSwap {
			rl.DrawText("X", checkX+5, checkY+2, 20, rl.Green)
		}

		rl.DrawText("Zero-Allocation In-Place Swap", checkX+35, checkY, 18, rl.White)
		rl.DrawText("Eliminates buffer allocations (disables dynamic adds)", checkX+35, checkY+22, 12, rl.Gray)

		// Stats
		statsY := bgY + bgHeight - 40
		rl.DrawText("Configuration affects memory usage and performance", bgX+50, statsY, 14, rl.LightGray)
	}
}

// simpleFrustumCull is a wrapper for frustumCullObjects that takes CameraState
func simpleFrustumCull(objects []*engine.Object, cameraState *ui.CameraState) []*engine.Object {
	camera := rl.Camera3D{
		Position:   rl.Vector3{X: cameraState.Position.X, Y: cameraState.Position.Y, Z: cameraState.Position.Z},
		Target:     rl.Vector3{X: cameraState.Position.X + cameraState.Forward.X, Y: cameraState.Position.Y + cameraState.Forward.Y, Z: cameraState.Position.Z + cameraState.Forward.Z},
		Up:         rl.Vector3{X: 0, Y: 1, Z: 0},
		Fovy:       45.0,
		Projection: rl.CameraPerspective,
	}
	return frustumCullObjects(objects, camera)
}

// frustumCullObjects performs view frustum culling on objects
func frustumCullObjects(objects []*engine.Object, camera rl.Camera3D) []*engine.Object {
	culled := make([]*engine.Object, 0, len(objects))

	// Calculate view frustum planes
	// For simplicity, use a cone-based approach: check if object is within FOV
	camPos := engine.Vector3{X: camera.Position.X, Y: camera.Position.Y, Z: camera.Position.Z}
	camTarget := engine.Vector3{X: camera.Target.X, Y: camera.Target.Y, Z: camera.Target.Z}
	viewDir := camTarget.Sub(camPos).Normalize()

	// FOV cone half-angle with margin for object radius
	fovCosHalfAngle := float32(math.Cos(float64((camera.Fovy / 2.0) * rl.Deg2rad * engine.FrustumFOVMargin)))

	for _, obj := range objects {
		// Vector from camera to object
		toObj := obj.Anim.Position.Sub(camPos)
		distance := toObj.Length()

		// Handle case where camera is very close or inside object
		if distance < engine.FrustumNearCheck {
			culled = append(culled, obj)
			continue
		}

		toObjNorm := toObj.Scale(1.0 / distance)

		// Dot product gives cos of angle between view direction and object direction
		cosAngle := viewDir.X*toObjNorm.X + viewDir.Y*toObjNorm.Y + viewDir.Z*toObjNorm.Z

		// Account for object radius - objects partially in view should be included
		radiusAngle := float32(math.Atan(float64(obj.Meta.PhysicalRadius / distance)))
		adjustedCos := float32(math.Cos(float64(fovCosHalfAngle) + float64(radiusAngle)))

		// Include object if within FOV cone OR if we're very close to it
		// The radius check ensures large objects don't disappear when we're near them
		if cosAngle >= adjustedCos || distance < obj.Meta.PhysicalRadius*engine.FrustumNearObjectMultiplier {
			culled = append(culled, obj)
		}
	}

	return culled
}

// SpatialGrid is a simple spatial hash grid for accelerated culling
type SpatialGrid struct {
	cellSize float32
	cells    map[int64][]*engine.Object
}

// hashPosition converts a 3D position to a grid cell hash
func (g *SpatialGrid) hashPosition(x, y, z float32) int64 {
	// Quantize position to grid cell coordinates
	cx := int64(x / g.cellSize)
	cy := int64(y / g.cellSize)
	cz := int64(z / g.cellSize)

	// Combine into a single hash (simple interleaving)
	return (cx << 42) | (cy << 21) | cz
}

// buildGrid constructs the spatial grid from objects
func (g *SpatialGrid) buildGrid(objects []*engine.Object) {
	g.cells = make(map[int64][]*engine.Object)

	for _, obj := range objects {
		// Add object to all cells it overlaps based on its radius
		radius := obj.Meta.PhysicalRadius

		// Calculate bounding box in cell coordinates
		minCellX := int64((obj.Anim.Position.X - radius) / g.cellSize)
		maxCellX := int64((obj.Anim.Position.X + radius) / g.cellSize)
		minCellY := int64((obj.Anim.Position.Y - radius) / g.cellSize)
		maxCellY := int64((obj.Anim.Position.Y + radius) / g.cellSize)
		minCellZ := int64((obj.Anim.Position.Z - radius) / g.cellSize)
		maxCellZ := int64((obj.Anim.Position.Z + radius) / g.cellSize)

		// Add to all overlapping cells (limit to 3x3x3 to avoid excessive duplication)
		for cx := minCellX; cx <= maxCellX && (cx-minCellX) < 3; cx++ {
			for cy := minCellY; cy <= maxCellY && (cy-minCellY) < 3; cy++ {
				for cz := minCellZ; cz <= maxCellZ && (cz-minCellZ) < 3; cz++ {
					hash := (cx << 42) | (cy << 21) | cz
					g.cells[hash] = append(g.cells[hash], obj)
				}
			}
		}
	}
}

// getCellsInFrustum returns all cells potentially visible in the frustum
func (g *SpatialGrid) getCellsInFrustum(camera rl.Camera3D) []*engine.Object {
	// Get camera position and view direction
	camPos := engine.Vector3{X: camera.Position.X, Y: camera.Position.Y, Z: camera.Position.Z}
	camTarget := engine.Vector3{X: camera.Target.X, Y: camera.Target.Y, Z: camera.Target.Z}
	viewDir := camTarget.Sub(camPos).Normalize()

	// Use adaptive view distance based on camera distance from origin
	// For tracking mode at various distances, this scales appropriately
	camDistFromOrigin := float32(math.Sqrt(float64(camPos.X*camPos.X + camPos.Y*camPos.Y + camPos.Z*camPos.Z)))
	maxViewDist := float32(math.Max(engine.SpatialViewDistMin, float64(camDistFromOrigin*engine.SpatialViewDistMultiplier)))
	if maxViewDist > engine.SpatialViewDistMax {
		maxViewDist = engine.SpatialViewDistMax // Cap to avoid excessive cell checks
	}
	cellRadius := int(maxViewDist / g.cellSize)

	// Get camera's cell position
	camCellX := int(camPos.X / g.cellSize)
	camCellY := int(camPos.Y / g.cellSize)
	camCellZ := int(camPos.Z / g.cellSize)

	// Collect objects from cells in view direction
	candidates := make([]*engine.Object, 0, len(g.cells)*2)
	seen := make(map[*engine.Object]bool)

	// ALWAYS include camera's own cell - objects very close to camera should never be culled
	// This fixes the bug where tracked objects in the same cell as the camera get missed
	camCellHash := (int64(camCellX) << 42) | (int64(camCellY) << 21) | int64(camCellZ)
	if cellObjects, exists := g.cells[camCellHash]; exists {
		for _, obj := range cellObjects {
			candidates = append(candidates, obj)
			seen[obj] = true
		}
	}

	// Check cells in a cone around view direction
	for dx := -cellRadius; dx <= cellRadius; dx++ {
		for dy := -cellRadius; dy <= cellRadius; dy++ {
			for dz := -cellRadius; dz <= cellRadius; dz++ {
				// Cell position
				cellX := camCellX + dx
				cellY := camCellY + dy
				cellZ := camCellZ + dz

				// Compute cell center
				cellCenterX := float32(cellX)*g.cellSize + g.cellSize*0.5
				cellCenterY := float32(cellY)*g.cellSize + g.cellSize*0.5
				cellCenterZ := float32(cellZ)*g.cellSize + g.cellSize*0.5

				// Vector from camera to cell center
				toCellX := cellCenterX - camPos.X
				toCellY := cellCenterY - camPos.Y
				toCellZ := cellCenterZ - camPos.Z
				cellDist := float32(math.Sqrt(float64(toCellX*toCellX + toCellY*toCellY + toCellZ*toCellZ)))

				if cellDist > maxViewDist {
					continue
				}

				// Rough frustum check: dot product with view direction
				if cellDist > 0.01 {
					toCellNormX := toCellX / cellDist
					toCellNormY := toCellY / cellDist
					toCellNormZ := toCellZ / cellDist

					dotProduct := viewDir.X*toCellNormX + viewDir.Y*toCellNormY + viewDir.Z*toCellNormZ

					// Skip cells behind camera (allow wide cone to avoid missing objects at frustum edges)
					if dotProduct < -0.2 { // Only cull cells clearly behind camera
						continue
					}
				}

				// Get objects from this cell
				hash := (int64(cellX) << 42) | (int64(cellY) << 21) | int64(cellZ)
				if cellObjects, exists := g.cells[hash]; exists {
					for _, obj := range cellObjects {
						if !seen[obj] {
							candidates = append(candidates, obj)
							seen[obj] = true
						}
					}
				}
			}
		}
	}

	return candidates
}

// spatialFrustumCull performs frustum culling using spatial partitioning
func spatialFrustumCull(objects []*engine.Object, camera rl.Camera3D) []*engine.Object {
	// Build spatial grid
	grid := &SpatialGrid{cellSize: engine.SpatialGridCellSize}
	grid.buildGrid(objects)

	// Get candidate objects from visible cells
	candidates := grid.getCellsInFrustum(camera)

	// Perform precise frustum culling on candidates
	culled := frustumCullObjects(candidates, camera)

	return culled
}

// drawHelpScreen displays comprehensive keyboard and mouse controls
func drawHelpScreen() {
	// Layout constants — all geometry is derived from these four values so that
	// adjusting panel size only requires changing bgWidth / bgHeight.
	const (
		bgWidth    = int32(800)
		bgHeight   = int32(600)
		margin     = int32(20)  // inner padding from panel edge
		valueGap   = int32(150) // horizontal gap between key label and description
		lineHeight = int32(25)
		titleSize  = int32(28)
		headerSize = int32(20)
		bodySize   = int32(16)
		hintSize   = int32(16)
	)
	bgX := int32(currentScreenWidth()/2) - bgWidth/2
	bgY := int32(currentScreenHeight()/2) - bgHeight/2
	leftCol := bgX + margin
	rightCol := bgX + bgWidth/2 + margin

	// Semi-transparent background
	rl.DrawRectangle(bgX, bgY, bgWidth, bgHeight, rl.Color{R: 0, G: 0, B: 0, A: 230})
	rl.DrawRectangleLines(bgX, bgY, bgWidth, bgHeight, rl.White)

	// Title
	titleText := "KEYBOARD & MOUSE CONTROLS"
	titleX := bgX + (bgWidth-rl.MeasureText(titleText, titleSize))/2
	rl.DrawText(titleText, titleX, bgY+10, titleSize, rl.White)
	hintText := "Press Ctrl+/ or ESC to close"
	hintX := bgX + (bgWidth-rl.MeasureText(hintText, hintSize))/2
	rl.DrawText(hintText, hintX, bgY+45, hintSize, rl.Gray)

	y := bgY + 80

	// Left column - Camera Controls
	rl.DrawText("CAMERA CONTROLS", leftCol, y, headerSize, rl.Yellow)
	y += lineHeight + 5

	rl.DrawText("Mouse Move", leftCol, y, bodySize, rl.White)
	rl.DrawText("Look around", leftCol+valueGap, y, bodySize, rl.LightGray)
	y += lineHeight

	rl.DrawText("Mouse Wheel", leftCol, y, bodySize, rl.White)
	rl.DrawText("Zoom / adjust tracking distance", leftCol+valueGap, y, bodySize, rl.LightGray)
	y += lineHeight

	rl.DrawText("W / S", leftCol, y, bodySize, rl.White)
	rl.DrawText("Move forward/backward", leftCol+valueGap, y, bodySize, rl.LightGray)
	y += lineHeight

	rl.DrawText("A / D", leftCol, y, bodySize, rl.White)
	rl.DrawText("Strafe left/right", leftCol+valueGap, y, bodySize, rl.LightGray)
	y += lineHeight

	rl.DrawText("Shift", leftCol, y, bodySize, rl.White)
	rl.DrawText("Hold for 2x speed boost", leftCol+valueGap, y, bodySize, rl.LightGray)
	y += lineHeight

	rl.DrawText("Arrow Keys", leftCol, y, bodySize, rl.White)
	rl.DrawText("Move in system plane", leftCol+valueGap, y, bodySize, rl.LightGray)
	y += lineHeight + 10

	// Navigation
	rl.DrawText("NAVIGATION", leftCol, y, headerSize, rl.Yellow)
	y += lineHeight + 5

	rl.DrawText("C", leftCol, y, bodySize, rl.White)
	rl.DrawText("Center view on Sun / Reset zoom", leftCol+valueGap, y, bodySize, rl.LightGray)
	y += lineHeight

	rl.DrawText("J", leftCol, y, bodySize, rl.White)
	rl.DrawText("Open jump dialog", leftCol+valueGap, y, bodySize, rl.LightGray)
	y += lineHeight

	rl.DrawText("T", leftCol, y, bodySize, rl.White)
	rl.DrawText("Open track dialog", leftCol+valueGap, y, bodySize, rl.LightGray)
	y += lineHeight

	rl.DrawText("TAB", leftCol, y, bodySize, rl.White)
	rl.DrawText("Next sibling (when tracking)", leftCol+valueGap, y, bodySize, rl.LightGray)
	y += lineHeight

	rl.DrawText("Shift+TAB", leftCol, y, bodySize, rl.White)
	rl.DrawText("Previous sibling (tracking)", leftCol+valueGap, y, bodySize, rl.LightGray)
	y += lineHeight

	rl.DrawText("F", leftCol, y, bodySize, rl.White)
	rl.DrawText("Forward to child (tracking)", leftCol+valueGap, y, bodySize, rl.LightGray)
	y += lineHeight

	rl.DrawText("B", leftCol, y, bodySize, rl.White)
	rl.DrawText("Back to parent (tracking)", leftCol+valueGap, y, bodySize, rl.LightGray)
	y += lineHeight

	rl.DrawText("R", leftCol, y, bodySize, rl.White)
	rl.DrawText("Reset camera offset", leftCol+valueGap, y, bodySize, rl.LightGray)
	y += lineHeight

	rl.DrawText("ESC", leftCol, y, bodySize, rl.White)
	rl.DrawText("Exit tracking/mouse mode", leftCol+valueGap, y, bodySize, rl.LightGray)
	y += lineHeight

	rl.DrawText("Ctrl+Q", leftCol, y, bodySize, rl.White)
	rl.DrawText("Quit application", leftCol+valueGap, y, bodySize, rl.LightGray)

	// Right column - Display & Options
	y = bgY + 80
	rl.DrawText("SYSTEM & DISPLAY", rightCol, y, headerSize, rl.Yellow)
	y += lineHeight + 5

	rl.DrawText("Ctrl+G", rightCol, y, bodySize, rl.White)
	rl.DrawText("Toggle grid", rightCol+valueGap, y, bodySize, rl.LightGray)
	y += lineHeight

	rl.DrawText("Ctrl+H", rightCol, y, bodySize, rl.White)
	rl.DrawText("Toggle HUD", rightCol+valueGap, y, bodySize, rl.LightGray)
	y += lineHeight

	rl.DrawText("Ctrl+L", rightCol, y, bodySize, rl.White)
	rl.DrawText("Toggle object labels", rightCol+valueGap, y, bodySize, rl.LightGray)
	y += lineHeight

	rl.DrawText("Ctrl+M", rightCol, y, bodySize, rl.White)
	rl.DrawText("Toggle mouse mode", rightCol+valueGap, y, bodySize, rl.LightGray)
	y += lineHeight

	rl.DrawText("Ctrl+F", rightCol, y, bodySize, rl.White)
	rl.DrawText("Toggle fullscreen", rightCol+valueGap, y, bodySize, rl.LightGray)
	y += lineHeight

	rl.DrawText("+  /  -", rightCol, y, bodySize, rl.White)
	rl.DrawText("Asteroids (200->24K)", rightCol+valueGap, y, bodySize, rl.LightGray)
	y += lineHeight

	rl.DrawText(",  /  .", rightCol, y, bodySize, rl.White)
	rl.DrawText("Time scale (PAUSED->1yr/sec)", rightCol+valueGap, y, bodySize, rl.LightGray)
	y += lineHeight

	rl.DrawText("<  /  >", rightCol, y, bodySize, rl.White)
	rl.DrawText("Anim speed (0%->100% of 60Hz)", rightCol+valueGap, y, bodySize, rl.LightGray)
	y += lineHeight

	rl.DrawText("Ctrl+P", rightCol, y, bodySize, rl.White)
	rl.DrawText("Open performance dialog", rightCol+valueGap, y, bodySize, rl.LightGray)
	y += lineHeight

	rl.DrawText("Ctrl+/", rightCol, y, bodySize, rl.White)
	rl.DrawText("Open help dialog", rightCol+valueGap, y, bodySize, rl.LightGray)
	y += lineHeight + 10

	// Dialog controls
	rl.DrawText("DIALOG CONTROLS", rightCol, y, headerSize, rl.Yellow)
	y += lineHeight + 5

	rl.DrawText("Up / Down", rightCol, y, bodySize, rl.White)
	rl.DrawText("Navigate list", rightCol+valueGap, y, bodySize, rl.LightGray)
	y += lineHeight

	rl.DrawText("Left / Right", rightCol, y, bodySize, rl.White)
	rl.DrawText("Change category", rightCol+valueGap, y, bodySize, rl.LightGray)
	y += lineHeight

	rl.DrawText("Enter", rightCol, y, bodySize, rl.White)
	rl.DrawText("Confirm selection", rightCol+valueGap, y, bodySize, rl.LightGray)
	y += lineHeight

	rl.DrawText("Space", rightCol, y, bodySize, rl.White)
	rl.DrawText("Toggle option (perf menu)", rightCol+valueGap, y, bodySize, rl.LightGray)
	y += lineHeight

	rl.DrawText("ESC", rightCol, y, bodySize, rl.White)
	rl.DrawText("Close help or active dialog", rightCol+valueGap, y, bodySize, rl.LightGray)
	y += lineHeight

	rl.DrawText("Modal", rightCol, y, bodySize, rl.White)
	rl.DrawText("Dialogs suspend main controls", rightCol+valueGap, y, bodySize, rl.LightGray)
	y += lineHeight + 10

	// Camera Modes
	rl.DrawText("CAMERA MODES", rightCol, y, headerSize, rl.Yellow)
	y += lineHeight + 5

	rl.DrawText("Free-Fly", rightCol, y, bodySize, rl.Lime)
	rl.DrawText("Full manual control", rightCol+valueGap, y, bodySize, rl.LightGray)
	y += lineHeight

	rl.DrawText("Jumping", rightCol, y, bodySize, rl.Lime)
	rl.DrawText("Animated fly-to", rightCol+valueGap, y, bodySize, rl.LightGray)
	y += lineHeight

	rl.DrawText("Tracking", rightCol, y, bodySize, rl.Lime)
	rl.DrawText("Follow target object", rightCol+valueGap, y, bodySize, rl.LightGray)
}
