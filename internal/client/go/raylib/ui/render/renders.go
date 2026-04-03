package render

import (
	"fmt"
	"math"
	"path/filepath"
	"sort"
	"strings"
	"time"

	engine "github.com/digital-michael/space_sim/internal/sim/engine"
	simlib "github.com/digital-michael/space_sim/internal/sim/world"
	"github.com/digital-michael/space_sim/internal/client/go/raylib/ui"
	rl "github.com/gen2brain/raylib-go/raylib"
)

var layoutWidth int32
var layoutHeight int32

const hideDateAtOrAboveSecondsPerSecond = 86400.0

// Renderer owns Raylib-specific drawing behavior for the Space Sim application.
type Renderer struct {
	renderWidth  int32
	renderHeight int32
	target       rl.RenderTexture2D
	targetLoaded bool
}

// New creates a Raylib renderer.
func New() *Renderer {
	setLayoutSize(0, 0)
	return &Renderer{}
}

func setLayoutSize(width, height int32) {
	layoutWidth = width
	layoutHeight = height
}

func currentScreenWidth() int {
	if layoutWidth > 0 {
		return int(layoutWidth)
	}
	return rl.GetScreenWidth()
}

func currentScreenHeight() int {
	if layoutHeight > 0 {
		return int(layoutHeight)
	}
	return rl.GetScreenHeight()
}

func formatSimulationDateText(simSeconds float64, secondsPerSecond float32) string {
	j2000 := time.Date(2000, 1, 1, 12, 0, 0, 0, time.UTC)
	currentTime := j2000.Add(time.Duration(simSeconds * float64(time.Second)))
	localTime := currentTime.Local()

	year := localTime.Year()
	month := int(localTime.Month())
	day := localTime.Day()

	if secondsPerSecond >= hideDateAtOrAboveSecondsPerSecond {
		return fmt.Sprintf("Date: %04d/%02d/%02d", year, month, day)
	}

	hour := localTime.Hour()
	minute := localTime.Minute()
	second := localTime.Second()
	millisecond := localTime.Nanosecond() / 1000000

	return fmt.Sprintf("Date: %04d/%02d/%02d %02d:%02d:%02d.%03d",
		year, month, day, hour, minute, second, millisecond)
}

func uiScale() float32 {
	width := float32(currentScreenWidth())
	height := float32(currentScreenHeight())
	if width <= 0 || height <= 0 {
		return 1.0
	}

	scaleW := width / 1920.0
	scaleH := height / 1080.0
	scale := scaleW
	if scaleH < scale {
		scale = scaleH
	}

	if scale < 0.85 {
		return 0.85
	}
	if scale > 1.35 {
		return 1.35
	}
	return scale
}

func scaledInt32(base int32) int32 {
	scaled := int32(math.Round(float64(float32(base) * uiScale())))
	if scaled < 1 {
		return 1
	}
	return scaled
}

func (r *Renderer) HasRenderTarget() bool {
	return r.targetLoaded
}

func (r *Renderer) DisableRenderTarget() {
	if r.targetLoaded {
		rl.UnloadRenderTexture(r.target)
		r.targetLoaded = false
	}
	setLayoutSize(0, 0)
}

func (r *Renderer) ConfigureRenderTarget(width, height int32) {
	if width <= 0 || height <= 0 {
		return
	}
	if r.targetLoaded && r.renderWidth == width && r.renderHeight == height {
		setLayoutSize(width, height)
		return
	}
	if r.targetLoaded {
		rl.UnloadRenderTexture(r.target)
	}
	r.target = rl.LoadRenderTexture(width, height)
	r.targetLoaded = true
	r.renderWidth = width
	r.renderHeight = height
	rl.SetTextureFilter(r.target.Texture, rl.FilterBilinear)
	setLayoutSize(width, height)
}

func (r *Renderer) Close() {
	if r.targetLoaded {
		rl.UnloadRenderTexture(r.target)
		r.targetLoaded = false
	}
	setLayoutSize(0, 0)
}

func (r *Renderer) BeginFrame() {
	if r.targetLoaded {
		setLayoutSize(r.renderWidth, r.renderHeight)
		rl.BeginTextureMode(r.target)
	} else {
		setLayoutSize(int32(rl.GetScreenWidth()), int32(rl.GetScreenHeight()))
		rl.BeginDrawing()
	}
	rl.ClearBackground(rl.Black)
}

func (r *Renderer) EndFrame(windowWidth, windowHeight int32) {
	if !r.targetLoaded {
		rl.EndDrawing()
		return
	}

	rl.EndTextureMode()
	rl.BeginDrawing()
	rl.ClearBackground(rl.Black)

	scaleX := float32(windowWidth) / float32(r.renderWidth)
	scaleY := float32(windowHeight) / float32(r.renderHeight)
	scale := scaleX
	if scaleY < scale {
		scale = scaleY
	}
	destWidth := float32(r.renderWidth) * scale
	destHeight := float32(r.renderHeight) * scale
	destX := (float32(windowWidth) - destWidth) * 0.5
	destY := (float32(windowHeight) - destHeight) * 0.5

	source := rl.Rectangle{X: 0, Y: 0, Width: float32(r.renderWidth), Height: -float32(r.renderHeight)}
	dest := rl.Rectangle{X: destX, Y: destY, Width: destWidth, Height: destHeight}
	rl.DrawTexturePro(r.target.Texture, source, dest, rl.Vector2{}, 0, rl.White)
	rl.EndDrawing()
}

func (r *Renderer) DrawObjectsInstanced(objects []*engine.Object, cameraPos engine.Vector3, pointRenderingEnabled bool, lodEnabled bool, importanceThreshold int) int {
	return drawObjectsInstanced(objects, cameraPos, pointRenderingEnabled, lodEnabled, importanceThreshold)
}

func (r *Renderer) DrawObject(obj *engine.Object, cameraPos engine.Vector3, pointRenderingEnabled bool, lodEnabled bool) {
	drawObject(obj, cameraPos, pointRenderingEnabled, lodEnabled)
}

func (r *Renderer) DrawGroundPlane() {
	drawGroundPlane()
}

func (r *Renderer) DrawObjectLabels(state *engine.SimulationState, cameraState *ui.CameraState, camera rl.Camera3D, objectsToRender []*engine.Object) {
	drawObjectLabels(state, cameraState, camera, objectsToRender)
}

func (r *Renderer) DrawHUD(state *engine.SimulationState, cameraState *ui.CameraState, inputState *ui.InputState, asteroidDataset engine.AsteroidDataset, mouseModeEnabled bool, sim *simlib.World, inViewCount int, eligibleInViewCount int, renderedCount int) {
	drawHUD(state, cameraState, inputState, asteroidDataset, mouseModeEnabled, sim, inViewCount, eligibleInViewCount, renderedCount)
}

func (r *Renderer) DrawZoomIndicator(zoomValue float32) {
	drawZoomIndicator(zoomValue)
}

func (r *Renderer) DrawHelpScreen() {
	drawHelpScreen()
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
	drawnCount := 0

	for _, obj := range objects {
		// Skip objects below importance threshold
		if obj.Meta.Importance < importanceThreshold {
			continue
		}

		// Skip rings - they need individual rendering
		if obj.Meta.InnerRadius > 0 {
			drawObject(obj, cameraPos, pointRenderingEnabled, lodEnabled)
			drawnCount++
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
	fontSize := scaledInt32(16)
	labelOffsetX := float32(scaledInt32(15))
	labelOffsetY := float32(scaledInt32(10))
	edgePadding := float32(scaledInt32(10))
	padding := float32(scaledInt32(4))
	lineWidth := float32(scaledInt32(2))

	// Determine which objects should have labels based on priority
	labeledObjects := selectObjectsForLabels(state, cameraState, objectsToRender, 20) // Max 20 labels

	// Project object positions to screen space and draw labels
	for _, obj := range labeledObjects {
		// Get object position in 3D
		objPos := rl.Vector3{X: obj.Anim.Position.X, Y: obj.Anim.Position.Y, Z: obj.Anim.Position.Z}

		// Project to screen space
		screenPos := rl.GetWorldToScreenEx(objPos, camera, int32(currentScreenWidth()), int32(currentScreenHeight()))

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
		textSize := rl.MeasureTextEx(rl.GetFontDefault(), labelText, float32(fontSize), 1.0)

		// Position label offset from object (right and slightly up)
		labelX := screenPos.X + labelOffsetX
		labelY := screenPos.Y - labelOffsetY

		// Clamp label position to screen bounds
		if labelX+textSize.X > float32(currentScreenWidth())-edgePadding {
			labelX = screenPos.X - textSize.X - labelOffsetX
		}
		if labelY < edgePadding {
			labelY = edgePadding
		}
		if labelY+textSize.Y > float32(currentScreenHeight())-edgePadding {
			labelY = float32(currentScreenHeight()) - textSize.Y - edgePadding
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
func drawHUD(state *engine.SimulationState, cameraState *ui.CameraState, inputState *ui.InputState, asteroidDataset engine.AsteroidDataset, mouseModeEnabled bool, sim *simlib.World, inViewCount int, eligibleInViewCount int, renderedCount int) {
	leftPad := scaledInt32(10)
	fontLarge := scaledInt32(20)
	fontMedium := scaledInt32(18)
	line1Y := scaledInt32(10)
	line2Y := scaledInt32(35)
	line3Y := scaledInt32(60)
	line4Y := scaledInt32(85)
	line5Y := scaledInt32(108)
	line6Y := scaledInt32(133)
	line7Y := scaledInt32(158)
	helpY := int32(currentScreenHeight()) - scaledInt32(30)

	fps := rl.GetFPS()
	rl.DrawText(fmt.Sprintf("FPS: %3d / %d threads", fps, state.NumWorkers), leftPad, line1Y, fontLarge, rl.Green)

	// Display object counts: total / visible / rendered
	totalObjects := len(state.Objects)
	visibleObjects := 0
	for _, obj := range state.Objects {
		if obj.Visible {
			visibleObjects++
		}
	}
	datasetName := simlib.GetDatasetName(asteroidDataset)
	rl.DrawText(fmt.Sprintf("Objects: %d total / %d visible (Dataset: %s)", totalObjects, visibleObjects, datasetName), leftPad, line2Y, fontLarge, rl.White)

	dateText := formatSimulationDateText(state.Time, state.SecondsPerSecond)
	rl.DrawText(dateText, leftPad, line3Y, fontLarge, rl.White)

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
	rl.DrawText(timeRateText, leftPad, line4Y, fontMedium, timeRateColor)

	// Anim speed indicator (physics tick rate as % of full 60Hz)
	animSpeed := sim.GetSpeed()
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
	rl.DrawText(animSpeedText, leftPad, line5Y, fontMedium, animSpeedColor)

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
	rl.DrawText(fmt.Sprintf("Mode: %s", modeText), leftPad, line6Y, fontLarge, rl.Yellow)

	// Camera position
	posText := fmt.Sprintf("Camera Position: X:%.1f Y:%.1f Z:%.1f", cameraState.Position.X, cameraState.Position.Y, cameraState.Position.Z)
	rl.DrawText(posText, leftPad, line7Y, fontMedium, rl.Color{R: 0, G: 255, B: 255, A: 255})

	// Debug info: screen/monitor/render dimensions and processing capacity
	screenW := rl.GetScreenWidth()
	screenH := rl.GetScreenHeight()
	monitorW := rl.GetMonitorWidth(0)
	monitorH := rl.GetMonitorHeight(0)
	isFullscreen := rl.IsWindowFullscreen()

	debugLine1Y := int32(currentScreenHeight()) - scaledInt32(85)
	debugLine2Y := int32(currentScreenHeight()) - scaledInt32(60)
	debugFontSize := scaledInt32(14)
	debugColor := rl.Color{R: 200, G: 200, B: 200, A: 200}

	// Display screen/monitor info
	fsText := "windowed"
	if isFullscreen {
		fsText = "fullscreen"
	}
	debugDimensionsText := fmt.Sprintf("Screen: %dx%d | Monitor: %dx%d | %s", screenW, screenH, monitorW, monitorH, fsText)
	rl.DrawText(debugDimensionsText, leftPad, debugLine1Y, debugFontSize, debugColor)

	// Display element processing capacity: show in-view vs visible, and how many we're rendering
	var visiblePct float32 = 0.0
	if totalObjects > 0 {
		visiblePct = float32(visibleObjects) / float32(totalObjects) * 100.0
	}

	var renderPct float32 = 0.0
	if eligibleInViewCount > 0 {
		renderPct = float32(renderedCount) / float32(eligibleInViewCount) * 100.0
	}

	processingColor := rl.Gray
	if eligibleInViewCount > 0 {
		processingColor = rl.Red
		if renderPct >= 60.0 {
			processingColor = rl.Yellow
		}
		if renderPct >= 90.0 {
			processingColor = rl.Green
		}
	}
	processingText := fmt.Sprintf("Render: %d/%d eligible (%.1f%%) | In-view: %d/%d visible | Visible: %.1f%% of %d total", renderedCount, eligibleInViewCount, renderPct, inViewCount, visibleObjects, visiblePct, totalObjects)
	rl.DrawText(processingText, leftPad, debugLine2Y, debugFontSize, processingColor)

	// Help hint (only shows when HUD is visible)
	rl.DrawText("Ctrl+/ for help | Ctrl+Q to quit", leftPad, helpY, fontLarge, rl.Gray)

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
	fontSize := scaledInt32(16)
	lineHeight := scaledInt32(22)
	padding := scaledInt32(15)
	panelMargin := scaledInt32(20)

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
	boxX := int32(currentScreenWidth()) - boxWidth - panelMargin
	boxY := int32(currentScreenHeight()) - boxHeight - panelMargin

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
	if inputState.SelectionMode == ui.SelectionModeSystemSelector {
		drawSystemSelectorUI(inputState)
		return
	}

	// Check if we're in performance mode
	if inputState.SelectionMode == ui.SelectionModePerformance {
		drawPerformanceUI(inputState)
		return
	}

	// Semi-transparent background - responsive to screen size
	sw := int32(currentScreenWidth())
	sh := int32(currentScreenHeight())
	titleFont := scaledInt32(20)
	hintFont := scaledInt32(12)
	filterFont := scaledInt32(16)
	filterHintFont := scaledInt32(14)
	tabFont := scaledInt32(14)
	itemFont := scaledInt32(20)
	itemSubFont := scaledInt32(16)
	arrowFont := scaledInt32(20)
	// Panel is 40% of screen width, clamped to reasonable bounds (400-700)
	bgWidth := sw * 40 / 100
	if bgWidth < scaledInt32(400) {
		bgWidth = scaledInt32(400)
	}
	if bgWidth > scaledInt32(700) {
		bgWidth = scaledInt32(700)
	}
	// Panel height matches width for square aspect
	bgHeight := bgWidth
	// Center on screen
	bgX := (sw - bgWidth) / 2
	bgY := (sh - bgHeight) / 2
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
	rl.DrawText(titleText, bgX+scaledInt32(50), bgY+scaledInt32(10), titleFont, rl.White)
	rl.DrawText("UP/DOWN: select, LEFT/RIGHT: category, ENTER: confirm, ESC: cancel", bgX+scaledInt32(10), bgY+scaledInt32(40), hintFont, rl.LightGray)
	rl.DrawText("PgUp/PgDn: page, HOME/END: jump to start/end", bgX+scaledInt32(10), bgY+scaledInt32(55), hintFont, rl.Gray)

	// Filter text box
	filterY := bgY + scaledInt32(75)
	filterBoxHeight := scaledInt32(25)
	if inputState.FilterText != "" {
		rl.DrawRectangle(bgX+scaledInt32(10), filterY, bgWidth-scaledInt32(20), filterBoxHeight, rl.Color{R: 40, G: 40, B: 40, A: 255})
		rl.DrawRectangleLines(bgX+scaledInt32(10), filterY, bgWidth-scaledInt32(20), filterBoxHeight, rl.Color{R: 100, G: 150, B: 200, A: 255})
		filterDisplay := "Filter: " + inputState.FilterText + "_"
		rl.DrawText(filterDisplay, bgX+scaledInt32(15), filterY+scaledInt32(5), filterFont, rl.Green)
		filterY += filterBoxHeight + scaledInt32(5)
	} else {
		rl.DrawText("Type to filter...", bgX+scaledInt32(15), filterY+scaledInt32(5), filterHintFont, rl.Gray)
		filterY += filterBoxHeight + scaledInt32(5)
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
	// Calculate tab width based on panel width and number of tabs
	numTabs := int32(len(categories))
	tabWidth := (bgWidth - scaledInt32(20)) / numTabs // 20 pixels margin
	if tabWidth < scaledInt32(60) {
		tabWidth = scaledInt32(60)
	}
	tabHeight := scaledInt32(30)
	tabY := filterY

	for i, cat := range categories {
		tabX := bgX + scaledInt32(10) + int32(i)*tabWidth
		tabColor := rl.Color{R: 50, G: 50, B: 50, A: 255}
		textColor := rl.LightGray

		// Highlight active category
		if cat.category == inputState.SelectedCategory {
			tabColor = rl.Color{R: 80, G: 120, B: 160, A: 255}
			textColor = rl.White
		}

		rl.DrawRectangle(tabX, tabY, tabWidth-scaledInt32(5), tabHeight, tabColor)
		rl.DrawRectangleLines(tabX, tabY, tabWidth-scaledInt32(5), tabHeight, rl.White)
		rl.DrawText(cat.name, tabX+scaledInt32(5), tabY+scaledInt32(8), tabFont, textColor)
	}

	// Filter objects by category
	if len(inputState.FilteredIndices) == 0 {
		inputState.FilteredIndices = filterObjectsByCategory(state.Objects, inputState.SelectedCategory)
	}

	// Object list (filtered by category) - start below the tabs
	startY := tabY + tabHeight + scaledInt32(10)
	lineHeight := scaledInt32(30)
	listAreaHeight := bgHeight - (startY - bgY) - scaledInt32(10) // Available height for list
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
			rl.DrawRectangle(bgX+scaledInt32(5), y-scaledInt32(2), bgWidth-scaledInt32(10), lineHeight-scaledInt32(2), rl.Color{R: 50, G: 100, B: 150, A: 255})
			rl.DrawText(">", bgX+scaledInt32(15), y+scaledInt32(5), arrowFont, rl.Yellow)
		}

		// Handle virtual belt indices
		if actualIndex == -1 {
			// Asteroid Belt
			rl.DrawText("Asteroid Belt", bgX+scaledInt32(40), y+scaledInt32(5), itemFont, rl.White)
			rl.DrawText("195-240 AU", bgX+scaledInt32(250), y+scaledInt32(5), itemSubFont, rl.LightGray)
			rl.DrawRectangleRec(rl.Rectangle{X: float32(bgX + scaledInt32(350)), Y: float32(y + scaledInt32(5)), Width: float32(scaledInt32(20)), Height: float32(scaledInt32(20))}, rl.Color{R: 150, G: 150, B: 150, A: 255})
		} else if actualIndex == -2 {
			// Kuiper Belt
			rl.DrawText("Kuiper Belt", bgX+scaledInt32(40), y+scaledInt32(5), itemFont, rl.White)
			rl.DrawText("3000-5000 AU", bgX+scaledInt32(250), y+scaledInt32(5), itemSubFont, rl.LightGray)
			rl.DrawRectangleRec(rl.Rectangle{X: float32(bgX + scaledInt32(350)), Y: float32(y + scaledInt32(5)), Width: float32(scaledInt32(20)), Height: float32(scaledInt32(20))}, rl.Color{R: 200, G: 150, B: 130, A: 255})
		} else {
			// Normal object
			obj := state.Objects[actualIndex]

			// Object name and info
			nameText := fmt.Sprintf("%s", obj.Meta.Name)
			distText := inputState.DistanceCache[actualIndex]
			if distText == "" {
				distText = "--- units" // Placeholder until first update
			}

			rl.DrawText(nameText, bgX+scaledInt32(40), y+scaledInt32(5), itemFont, rl.White)
			rl.DrawText(distText, bgX+scaledInt32(250), y+scaledInt32(5), itemSubFont, rl.LightGray)

			// Color indicator
			colorBox := rl.Rectangle{X: float32(bgX + scaledInt32(350)), Y: float32(y + scaledInt32(5)), Width: float32(scaledInt32(20)), Height: float32(scaledInt32(20))}
			rl.DrawRectangleRec(colorBox, rl.Color{R: obj.Meta.Color.R, G: obj.Meta.Color.G, B: obj.Meta.Color.B, A: 255})
		}
	}

	// Draw scroll bar if needed
	if totalItems > visibleItems {
		scrollBarX := bgX + bgWidth - scaledInt32(15)
		scrollBarY := startY
		scrollBarHeight := listAreaHeight
		scrollBarWidth := scaledInt32(10)

		// Scroll bar background
		rl.DrawRectangle(scrollBarX, scrollBarY, scrollBarWidth, scrollBarHeight, rl.Color{R: 30, G: 30, B: 30, A: 200})

		// Scroll thumb
		thumbHeight := int32(float32(visibleItems) / float32(totalItems) * float32(scrollBarHeight))
		if thumbHeight < scaledInt32(20) {
			thumbHeight = scaledInt32(20) // Minimum thumb size
		}
		thumbY := scrollBarY + int32(float32(inputState.ScrollOffset)/float32(maxScroll)*float32(scrollBarHeight-thumbHeight))
		rl.DrawRectangle(scrollBarX, thumbY, scrollBarWidth, thumbHeight, rl.Color{R: 100, G: 150, B: 200, A: 255})
		rl.DrawRectangleLines(scrollBarX, thumbY, scrollBarWidth, thumbHeight, rl.White)
	}
}

func drawSystemSelectorUI(inputState *ui.InputState) {
	sw := int32(currentScreenWidth())
	sh := int32(currentScreenHeight())
	titleFont := scaledInt32(20)
	hintFont := scaledInt32(12)
	itemFont := scaledInt32(18)
	statusFont := scaledInt32(14)
	arrowFont := scaledInt32(20)

	bgWidth := sw * 40 / 100
	if bgWidth < scaledInt32(420) {
		bgWidth = scaledInt32(420)
	}
	if bgWidth > scaledInt32(760) {
		bgWidth = scaledInt32(760)
	}
	bgHeight := bgWidth
	bgX := (sw - bgWidth) / 2
	bgY := (sh - bgHeight) / 2

	rl.DrawRectangle(bgX, bgY, bgWidth, bgHeight, rl.Color{R: 0, G: 0, B: 0, A: 210})
	rl.DrawRectangleLines(bgX, bgY, bgWidth, bgHeight, rl.White)

	titleText := "SELECT RUNTIME SYSTEM"
	rl.DrawText(titleText, bgX+scaledInt32(18), bgY+scaledInt32(12), titleFont, rl.White)
	rl.DrawText("UP/DOWN: select, ENTER: load, ESC: cancel", bgX+scaledInt32(18), bgY+scaledInt32(42), hintFont, rl.LightGray)
	rl.DrawText("Cmd+S opens this selector", bgX+scaledInt32(18), bgY+scaledInt32(58), hintFont, rl.Gray)

	activeLabel := filepath.Base(inputState.ActiveSystemPath)
	if activeLabel == "." || activeLabel == string(filepath.Separator) || activeLabel == "" {
		activeLabel = inputState.ActiveSystemPath
	}
	rl.DrawText("Current: "+activeLabel, bgX+scaledInt32(18), bgY+scaledInt32(86), statusFont, rl.Color{R: 140, G: 210, B: 255, A: 255})

	listStartY := bgY + scaledInt32(116)
	lineHeight := scaledInt32(32)
	listHeight := bgHeight - scaledInt32(176)
	visibleItems := int(listHeight / lineHeight)
	if visibleItems < 1 {
		visibleItems = 1
	}

	totalItems := len(inputState.SystemOptions)
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

	if totalItems == 0 {
		rl.DrawText("No runtime systems available.", bgX+scaledInt32(18), listStartY, itemFont, rl.LightGray)
	} else {
		for idx := inputState.ScrollOffset; idx < inputState.ScrollOffset+visibleItems && idx < totalItems; idx++ {
			option := inputState.SystemOptions[idx]
			y := listStartY + int32(idx-inputState.ScrollOffset)*lineHeight

			if idx == inputState.SelectedIndex {
				rl.DrawRectangle(bgX+scaledInt32(10), y-scaledInt32(2), bgWidth-scaledInt32(20), lineHeight-scaledInt32(2), rl.Color{R: 50, G: 100, B: 150, A: 255})
				rl.DrawText(">", bgX+scaledInt32(18), y+scaledInt32(4), arrowFont, rl.Yellow)
			}

			labelColor := rl.White
			suffix := ""
			if option.Path == inputState.ActiveSystemPath {
				labelColor = rl.Color{R: 140, G: 210, B: 255, A: 255}
				suffix = " [current]"
			}

			rl.DrawText(option.Label+suffix, bgX+scaledInt32(44), y+scaledInt32(4), itemFont, labelColor)
		}
	}

	if totalItems > visibleItems {
		scrollBarX := bgX + bgWidth - scaledInt32(15)
		scrollBarY := listStartY
		scrollBarWidth := scaledInt32(10)
		scrollBarHeight := listHeight
		rl.DrawRectangle(scrollBarX, scrollBarY, scrollBarWidth, scrollBarHeight, rl.Color{R: 30, G: 30, B: 30, A: 200})

		thumbHeight := int32(float32(visibleItems) / float32(totalItems) * float32(scrollBarHeight))
		if thumbHeight < scaledInt32(20) {
			thumbHeight = scaledInt32(20)
		}
		thumbY := scrollBarY
		if maxScroll > 0 {
			thumbY += int32(float32(inputState.ScrollOffset) / float32(maxScroll) * float32(scrollBarHeight-thumbHeight))
		}
		rl.DrawRectangle(scrollBarX, thumbY, scrollBarWidth, thumbHeight, rl.Color{R: 100, G: 150, B: 200, A: 255})
		rl.DrawRectangleLines(scrollBarX, thumbY, scrollBarWidth, thumbHeight, rl.White)
	}

	statusText := inputState.SystemStatusMessage
	if statusText == "" && inputState.SelectedIndex >= 0 && inputState.SelectedIndex < len(inputState.SystemOptions) {
		selected := inputState.SystemOptions[inputState.SelectedIndex]
		if selected.Path == inputState.ActiveSystemPath {
			statusText = "Press ENTER to close without reloading the current system."
		} else {
			statusText = "Press ENTER to load the highlighted system."
		}
	}
	if statusText != "" {
		rl.DrawText(statusText, bgX+scaledInt32(18), bgY+bgHeight-scaledInt32(34), statusFont, rl.LightGray)
	}
}

// filterObjectsByCategory returns indices of objects matching the given category
func filterObjectsByCategory(objects []*engine.Object, category engine.ObjectCategory) []int {
	var indices []int

	// Special handling for belt category - return virtual entries for belts
	if category == engine.CategoryBelt {
		// Check if there are any asteroids (Asteroid Belt)
		hasAsteroids := false
		for _, obj := range objects {
			if strings.HasPrefix(obj.Meta.Name, "Asteroid-") {
				hasAsteroids = true
				break
			}
		}

		// Check if there are any Kuiper Belt objects
		hasKuiper := false
		for _, obj := range objects {
			if strings.HasPrefix(obj.Meta.Name, "KBO-") {
				hasKuiper = true
				break
			}
		}

		// Return virtual indices: -1 for Asteroid Belt, -2 for Kuiper Belt
		if hasAsteroids {
			indices = append(indices, -1)
		}
		if hasKuiper {
			indices = append(indices, -2)
		}
		return indices
	}

	// Normal category filtering
	for i, obj := range objects {
		if obj.Meta.Category == category {
			indices = append(indices, i)
		}
	}
	return indices
}

// filterObjectsByCategoryAndText filters objects by category and optional text search
func filterObjectsByCategoryAndText(objects []*engine.Object, category engine.ObjectCategory, filterText string) []int {
	var indices []int
	lowerFilter := strings.ToLower(filterText)

	// Special handling for belt category
	if category == engine.CategoryBelt {
		// Check if there are any asteroids (Asteroid Belt)
		hasAsteroids := false
		for _, obj := range objects {
			if strings.HasPrefix(obj.Meta.Name, "Asteroid-") {
				hasAsteroids = true
				break
			}
		}

		// Check if there are any Kuiper Belt objects
		hasKuiper := false
		for _, obj := range objects {
			if strings.HasPrefix(obj.Meta.Name, "KBO-") {
				hasKuiper = true
				break
			}
		}

		// Filter by text if provided
		if filterText == "" {
			if hasAsteroids {
				indices = append(indices, -1)
			}
			if hasKuiper {
				indices = append(indices, -2)
			}
		} else {
			if hasAsteroids && strings.Contains("asteroid belt", lowerFilter) {
				indices = append(indices, -1)
			}
			if hasKuiper && strings.Contains("kuiper belt", lowerFilter) {
				indices = append(indices, -2)
			}
		}
		return indices
	}

	// Normal category filtering
	for i, obj := range objects {
		// First check category match
		if obj.Meta.Category != category {
			continue
		}

		// If no filter text, include all objects in category
		if filterText == "" {
			indices = append(indices, i)
			continue
		}

		// Check if object name contains the filter text (case-insensitive)
		lowerName := strings.ToLower(obj.Meta.Name)
		if strings.Contains(lowerName, lowerFilter) {
			indices = append(indices, i)
		}
	}
	return indices
}

// drawPerformanceUI draws the performance options menu with tabs
func drawPerformanceUI(inputState *ui.InputState) {
	// Semi-transparent background - responsive to screen size
	sw := int32(currentScreenWidth())
	sh := int32(currentScreenHeight())
	titleFont := scaledInt32(24)
	hintFont := scaledInt32(12)
	tabFont := scaledInt32(18)
	optionFont := scaledInt32(18)
	optionDescFont := scaledInt32(12)
	arrowFont := scaledInt32(20)
	statsFont := scaledInt32(14)
	// Panel is 40% of screen width, clamped to reasonable bounds (400-700)
	bgWidth := sw * 40 / 100
	if bgWidth < scaledInt32(400) {
		bgWidth = scaledInt32(400)
	}
	if bgWidth > scaledInt32(700) {
		bgWidth = scaledInt32(700)
	}
	// Panel height matches width for square aspect
	bgHeight := bgWidth
	// Center on screen
	bgX := (sw - bgWidth) / 2
	bgY := (sh - bgHeight) / 2
	rl.DrawRectangle(bgX, bgY, bgWidth, bgHeight, rl.Color{R: 0, G: 0, B: 0, A: 200})
	rl.DrawRectangleLines(bgX, bgY, bgWidth, bgHeight, rl.White)

	// Title
	titleText := "PERFORMANCE & CONFIGURATION"
	titleWidth := rl.MeasureText(titleText, titleFont)
	titleX := bgX + (bgWidth-int32(titleWidth))/2
	rl.DrawText(titleText, titleX, bgY+scaledInt32(10), titleFont, rl.White)
	rl.DrawText("UP/DOWN: select, SPACE: toggle, LEFT/RIGHT: tab/adjust, ESC: close", bgX+scaledInt32(20), bgY+scaledInt32(40), hintFont, rl.LightGray)

	// Draw tabs - divide panel width evenly between 2 tabs
	tabWidth := (bgWidth - scaledInt32(20)) / 2 // 20 pixels margin
	if tabWidth < scaledInt32(120) {
		tabWidth = scaledInt32(120)
	}
	tabHeight := scaledInt32(35)
	tabY := bgY + scaledInt32(65)

	// Performance tab
	perfTabColor := rl.Color{R: 50, G: 50, B: 50, A: 255}
	if inputState.PerformanceTab == 0 {
		perfTabColor = rl.Color{R: 80, G: 120, B: 160, A: 255}
	}
	rl.DrawRectangle(bgX+scaledInt32(10), tabY, tabWidth, tabHeight, perfTabColor)
	rl.DrawRectangleLines(bgX+scaledInt32(10), tabY, tabWidth, tabHeight, rl.White)
	perfText := "Performance"
	perfWidth := rl.MeasureText(perfText, tabFont)
	perfX := bgX + scaledInt32(10) + (tabWidth-int32(perfWidth))/2
	rl.DrawText(perfText, perfX, tabY+scaledInt32(8), tabFont, rl.White)

	// Configuration tab
	confTabColor := rl.Color{R: 50, G: 50, B: 50, A: 255}
	if inputState.PerformanceTab == 1 {
		confTabColor = rl.Color{R: 80, G: 120, B: 160, A: 255}
	}
	rl.DrawRectangle(bgX+scaledInt32(10)+tabWidth+scaledInt32(10), tabY, tabWidth, tabHeight, confTabColor)
	rl.DrawRectangleLines(bgX+scaledInt32(10)+tabWidth+scaledInt32(10), tabY, tabWidth, tabHeight, rl.White)
	confText := "Configuration"
	confWidth := rl.MeasureText(confText, tabFont)
	confX := bgX + scaledInt32(10) + tabWidth + scaledInt32(10) + (tabWidth-int32(confWidth))/2
	rl.DrawText(confText, confX, tabY+scaledInt32(8), tabFont, rl.White)

	startY := tabY + tabHeight + scaledInt32(20)
	lineHeight := scaledInt32(60)

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
				rl.DrawRectangle(bgX+scaledInt32(5), y-scaledInt32(2), bgWidth-scaledInt32(10), lineHeight-scaledInt32(2), rl.Color{R: 50, G: 100, B: 150, A: 255})
				rl.DrawText(">", bgX+scaledInt32(15), y+scaledInt32(10), arrowFont, rl.Yellow)
			}

			// Checkbox
			checkX := bgX + scaledInt32(40)
			checkY := y + scaledInt32(8)
			checkSize := scaledInt32(24)
			rl.DrawRectangle(checkX, checkY, checkSize, checkSize, rl.Color{R: 40, G: 40, B: 40, A: 255})
			rl.DrawRectangleLines(checkX, checkY, checkSize, checkSize, rl.White)

			if *opt.enabled {
				rl.DrawText("X", checkX+scaledInt32(5), checkY+scaledInt32(2), arrowFont, rl.Green)
			}

			// Option name and description
			rl.DrawText(opt.name, checkX+scaledInt32(35), checkY, optionFont, rl.White)
			rl.DrawText(opt.desc, checkX+scaledInt32(35), checkY+scaledInt32(22), optionDescFont, rl.Gray)
		}

		// Stats
		statsY := bgY + bgHeight - scaledInt32(40)
		culledText := "Objects will be culled based on selected optimizations"
		rl.DrawText(culledText, bgX+scaledInt32(50), statsY, statsFont, rl.LightGray)
	} else {
		// Configuration tab options
		// Option 0: Importance Threshold
		y := startY

		// Highlight selected
		if 0 == inputState.SelectedIndex {
			rl.DrawRectangle(bgX+scaledInt32(5), y-scaledInt32(2), bgWidth-scaledInt32(10), lineHeight-scaledInt32(2), rl.Color{R: 50, G: 100, B: 150, A: 255})
			rl.DrawText(">", bgX+scaledInt32(15), y+scaledInt32(10), arrowFont, rl.Yellow)
		}

		checkX := bgX + scaledInt32(40)
		checkY := y + scaledInt32(8)
		rl.DrawText("Importance Threshold", checkX, checkY, optionFont, rl.White)

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

		rl.DrawText(fmt.Sprintf("Value: %s", thresholdDesc), checkX, checkY+scaledInt32(22), optionDescFont, rl.Gray)
		rl.DrawText("LEFT/RIGHT: adjust", checkX+scaledInt32(250), checkY+scaledInt32(22), optionDescFont, rl.LightGray)

		// Option 1: Zero-Allocation In-Place Swap
		y = startY + lineHeight

		// Highlight selected
		if 1 == inputState.SelectedIndex {
			rl.DrawRectangle(bgX+scaledInt32(5), y-scaledInt32(2), bgWidth-scaledInt32(10), lineHeight-scaledInt32(2), rl.Color{R: 50, G: 100, B: 150, A: 255})
			rl.DrawText(">", bgX+scaledInt32(15), y+scaledInt32(10), arrowFont, rl.Yellow)
		}

		checkY = y + scaledInt32(8)
		checkSize := scaledInt32(24)
		rl.DrawRectangle(checkX, checkY, checkSize, checkSize, rl.Color{R: 40, G: 40, B: 40, A: 255})
		rl.DrawRectangleLines(checkX, checkY, checkSize, checkSize, rl.White)

		if inputState.PerfOptions.UseInPlaceSwap {
			rl.DrawText("X", checkX+scaledInt32(5), checkY+scaledInt32(2), arrowFont, rl.Green)
		}

		rl.DrawText("Zero-Allocation In-Place Swap", checkX+scaledInt32(35), checkY, optionFont, rl.White)
		rl.DrawText("Eliminates buffer allocations (disables dynamic adds)", checkX+scaledInt32(35), checkY+scaledInt32(22), optionDescFont, rl.Gray)

		// Stats
		statsY := bgY + bgHeight - scaledInt32(40)
		rl.DrawText("Configuration affects memory usage and performance", bgX+scaledInt32(50), statsY, statsFont, rl.LightGray)
	}
}

// drawHelpScreen displays comprehensive keyboard and mouse controls
func drawHelpScreen() {
	sw := int32(currentScreenWidth())
	sh := int32(currentScreenHeight())
	bgWidth := sw * 70 / 100
	if bgWidth < scaledInt32(800) {
		bgWidth = scaledInt32(800)
	}
	if bgWidth > scaledInt32(1400) {
		bgWidth = scaledInt32(1400)
	}
	bgHeight := sh * 78 / 100
	if bgHeight < scaledInt32(600) {
		bgHeight = scaledInt32(600)
	}
	if bgHeight > scaledInt32(1000) {
		bgHeight = scaledInt32(1000)
	}
	margin := scaledInt32(20)
	valueGap := scaledInt32(150)
	lineHeight := scaledInt32(25)
	titleSize := scaledInt32(28)
	headerSize := scaledInt32(20)
	bodySize := scaledInt32(16)
	hintSize := scaledInt32(16)

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
