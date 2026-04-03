package spatial

import (
	"math"

	engine "github.com/digital-michael/space_sim/internal/sim/engine"
	"github.com/digital-michael/space_sim/internal/client/go/raylib/ui"
	rl "github.com/gen2brain/raylib-go/raylib"
)

// SimpleFrustumCull is a wrapper for FrustumCullObjects that takes CameraState.
func SimpleFrustumCull(objects []*engine.Object, cameraState *ui.CameraState) []*engine.Object {
	camera := rl.Camera3D{
		Position:   rl.Vector3{X: cameraState.Position.X, Y: cameraState.Position.Y, Z: cameraState.Position.Z},
		Target:     rl.Vector3{X: cameraState.Position.X + cameraState.Forward.X, Y: cameraState.Position.Y + cameraState.Forward.Y, Z: cameraState.Position.Z + cameraState.Forward.Z},
		Up:         rl.Vector3{X: 0, Y: 1, Z: 0},
		Fovy:       45.0,
		Projection: rl.CameraPerspective,
	}
	return FrustumCullObjects(objects, camera)
}

// FrustumCullObjects performs view frustum culling on objects.
func FrustumCullObjects(objects []*engine.Object, camera rl.Camera3D) []*engine.Object {
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

// SpatialFrustumCull performs frustum culling using spatial partitioning.
func SpatialFrustumCull(objects []*engine.Object, camera rl.Camera3D) []*engine.Object {
	// Build spatial grid
	grid := &SpatialGrid{cellSize: engine.SpatialGridCellSize}
	grid.buildGrid(objects)

	// Get candidate objects from visible cells
	candidates := grid.getCellsInFrustum(camera)

	// Perform precise frustum culling on candidates
	culled := FrustumCullObjects(candidates, camera)

	return culled
}
