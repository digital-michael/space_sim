package render

import (
	"fmt"
	"math"

	engine "github.com/digital-michael/space_sim/internal/sim/engine"
	rl "github.com/gen2brain/raylib-go/raylib"
)

// lodScale returns a multiplier applied to the absolute LOD distance thresholds
// so that large bodies retain higher geometry quality at greater distances.
//
// Larger bodies subtend a bigger solid angle at the same camera distance — their
// triangulation is more visible. The multiplier is proportional to physical
// radius relative to an Earth-sized reference (0.5 sim units), clamped to
// [1, 10] so very small bodies get no degradation and very large bodies (Sol)
// are bounded to 10× the normal quality budget.
//
// Example results:
//
//	Sol (r=27.25): 10× → LODFar = 2000 instead of 200
//	Jupiter (r=2.8): 5.6× → LODFar = 1120
//	Earth (r=0.5): 1.0× (unchanged)
//	Asteroids (<0.5): 1.0× (clamped, no quality reduction)
const lodScaleRef = float32(0.5) // Earth-sized reference

func lodScale(physRadius float32) float64 {
	const maxScale = 10.0
	s := float64(physRadius / lodScaleRef)
	if s < 1.0 {
		s = 1.0
	}
	if s > maxScale {
		s = maxScale
	}
	return s
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
func drawObjectsInstanced(r *Renderer, objects []*engine.Object, cameraPos engine.Vector3, simTime float64, pointRenderingEnabled bool, lodEnabled bool, importanceThreshold int) int {
	// Build a name→radius map so ring shadow computation can look up the host
	// planet's physical radius without accessing the full object list per ring.
	parentRadius := make(map[string]float32, len(objects))
	for _, obj := range objects {
		if obj.Meta.InnerRadius == 0 {
			parentRadius[obj.Meta.Name] = obj.Meta.PhysicalRadius
		}
	}

	// Group objects into batches by their rendering properties
	batches := make(map[string]*InstanceBatch)
	// Rings are collected here and drawn after all planet batches (including their
	// atmosphere glows) to eliminate ring/atmosphere Z-fighting at the equatorial
	// boundary — both are transparent (DisableDepthMask) and share the same depth
	// range around the planet centre; drawing rings last gives consistent ordering.
	var deferredRings []*engine.Object
	drawnCount := 0

	for _, obj := range objects {
		// Skip objects below importance threshold
		if obj.Meta.Importance < importanceThreshold {
			continue
		}

		// Defer rings — see deferredRings comment above.
		if obj.Meta.InnerRadius > 0 {
			deferredRings = append(deferredRings, obj)
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
			} else if obj.Meta.Category == engine.CategoryStar {
				pointThreshold = engine.PointThresholdStar
				pointSize = engine.PointSizeStar
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
		rings := int32(64)
		slices := int32(64)
		wireRings := int32(8)
		wireSlices := int32(8)

		if lodEnabled && !isPoint {
			// Scale LOD thresholds by body size — larger bodies subtend a bigger solid
			// angle and show triangulation artefacts sooner. See lodScale().
			ls := lodScale(obj.Meta.PhysicalRadius)
			if distance < engine.LODVeryClose*ls {
				rings, slices = 128, 128
			} else if distance < engine.LODClose*ls {
				rings, slices = 64, 64
			} else if distance < engine.LODMedium*ls {
				rings, slices = 32, 32
			} else if distance < engine.LODFar*ls {
				rings, slices = 16, 16
			} else {
				rings, slices = 8, 8
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
				X: obj.Anim.Position.X - cameraPos.X,
				Y: obj.Anim.Position.Y - cameraPos.Y,
				Z: obj.Anim.Position.Z - cameraPos.Z,
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
				// Textured draw: use a cached Model when texture is available.
				texPath := obj.Meta.TexturePath
				if !r.noTextures && texPath != "" {
					if model, ok := r.getModel(texPath, batch.rings, batch.slices); ok {
						transform, drawScale := buildModelTransform(obj.Meta, simTime)
						model.Transform = transform
						var nightTex rl.Texture2D
						if obj.Meta.NightTexturePath != "" {
							nightTex = r.loadTexture(obj.Meta.NightTexturePath)
						}
						r.lighting.applyToModel(model, obj.Meta.SelfLuminous, nightTex)
						infraF := float32(0)
						if r.infraMode == 1 && !obj.Meta.SelfLuminous {
							infraF = spotlightFactor(obj.Anim.Position, cameraPos, r.cameraForward)
							if infraF > 0 {
								r.lighting.setAmbient(defaultAmbient + (infraSpotAmbient-defaultAmbient)*infraF)
							}
						}
						rl.DrawModel(model, pos, drawScale, rl.White)
						if infraF > 0 {
							r.lighting.setAmbient(defaultAmbient)
						}
						r.drawAtmosphereGlow(obj, pos)
						if obj.Meta.Category == engine.CategoryBlackHole {
							r.drawBlackHoleGlow(obj, pos)
						}
						drawnCount++
						continue
					}
				}
				// Fallback: untextured sphere with Phong lighting when available.
				if !r.noLighting && !obj.Meta.SelfLuminous {
					model := r.getUntexturedModel(batch.rings, batch.slices)
					transform, drawScale := buildModelTransform(obj.Meta, simTime)
					model.Transform = transform
					r.lighting.applyToModel(model, false, rl.Texture2D{})
					infraF := float32(0)
					if r.infraMode == 1 {
						infraF = spotlightFactor(obj.Anim.Position, cameraPos, r.cameraForward)
						if infraF > 0 {
							r.lighting.setAmbient(defaultAmbient + (infraSpotAmbient-defaultAmbient)*infraF)
						}
					}
					rl.DrawModel(model, pos, drawScale, color)
					if infraF > 0 {
						r.lighting.setAmbient(defaultAmbient)
					}
				} else {
					rl.DrawSphereEx(pos, float32(obj.Meta.PhysicalRadius), batch.rings, batch.slices, color)
				}
				r.drawAtmosphereGlow(obj, pos)
				if obj.Meta.Category == engine.CategoryBlackHole {
					r.drawBlackHoleGlow(obj, pos)
				}
				// Wireframe
				if obj.Meta.Material != engine.MaterialEmissive {
					rl.DrawSphereWires(pos, float32(obj.Meta.PhysicalRadius), batch.wireRings, batch.wireSlices,
						rl.Color{R: 255, G: 255, B: 255, A: 100})
				}
			}

			drawnCount++
		}
	}

	// Draw rings after all opaque bodies and their atmosphere glows.
	for _, obj := range deferredRings {
		pos := rl.Vector3{
			X: obj.Anim.Position.X - cameraPos.X,
			Y: obj.Anim.Position.Y - cameraPos.Y,
			Z: obj.Anim.Position.Z - cameraPos.Z,
		}
		pr := parentRadius[obj.Meta.ParentName]
		drawRingDisc(r, obj, pos, pr)
		drawnCount++
	}

	return drawnCount
}

// buildModelTransform returns the rotation+scale transform and the uniform draw
// scale for DrawModel. For oblate bodies (equatorial_radius ≠ polar_radius) the
// non-uniform scale is baked into the transform and drawScale is returned as 1.0
// so that DrawModel applies no additional uniform scaling. For all other bodies
// drawScale is PhysicalRadius and the transform is pure rotation.
func buildModelTransform(meta engine.ObjectMetadata, simTime float64) (rl.Matrix, float32) {
	tilt := meta.AxialTilt * rl.Deg2rad
	var spin float32
	if meta.RotationPeriod > 0 {
		periodSec := float64(meta.RotationPeriod) * 3600.0
		spin = float32(math.Mod(simTime/periodSec*360.0, 360.0)) * rl.Deg2rad
	}
	poleAndTilt := rl.MatrixMultiply(rl.MatrixRotateX(-90*rl.Deg2rad), rl.MatrixRotateZ(tilt))
	rotMat := rl.MatrixMultiply(poleAndTilt, rl.MatrixRotateY(-spin))

	eqR := meta.EquatorialRadius
	polR := meta.PolarRadius
	if eqR > 0 && polR > 0 {
		// Oblate: bake non-uniform scale — eqR on X/Z, polR on Y — into transform.
		// Matrix is column-major; M0/M5/M10/M15 are the diagonal of a scale matrix.
		scaleMat := rl.Matrix{
			M0: eqR, M5: polR, M10: eqR, M15: 1,
		}
		return rl.MatrixMultiply(rotMat, scaleMat), 1.0
	}
	return rotMat, meta.PhysicalRadius
}

// drawObject renders a single object
func drawObject(r *Renderer, obj *engine.Object, cameraPos engine.Vector3, simTime float64, pointRenderingEnabled bool, lodEnabled bool) {
	pos := rl.Vector3{
		X: obj.Anim.Position.X - cameraPos.X,
		Y: obj.Anim.Position.Y - cameraPos.Y,
		Z: obj.Anim.Position.Z - cameraPos.Z,
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
		} else if obj.Meta.Category == engine.CategoryStar {
			pointThreshold = engine.PointThresholdStar
		}

		if distance > pointThreshold || obj.Meta.PhysicalRadius < 0.5 {
			// Draw as a point with size based on object importance
			pointSize := engine.PointSizeDefault
			if obj.Meta.Category == engine.CategoryPlanet {
				pointSize = engine.PointSizePlanet
			} else if obj.Meta.Category == engine.CategoryMoon {
				pointSize = engine.PointSizeMoon
			} else if obj.Meta.Category == engine.CategoryStar {
				pointSize = engine.PointSizeStar
			}

			// Draw point using a small sphere (more visible than DrawPixel3D)
			rl.DrawSphere(pos, pointSize*0.1, color)
			return
		}
	} // Planetary rings are drawn after all opaque objects and their atmosphere glows
	// (see drawObjectsInstanced). This path handles the single-object draw case
	// (no parent radius available, so planet shadow is skipped).
	if obj.Meta.InnerRadius > 0 {
		drawRingDisc(r, obj, pos, 0)
		return
	}

	// Draw sphere for planets and other objects with LOD support
	// Determine sphere quality based on distance; scale thresholds for large bodies.
	rings := int32(64)
	slices := int32(64)

	if lodEnabled {
		ls := lodScale(obj.Meta.PhysicalRadius)
		if distance < engine.LODVeryClose*ls {
			rings, slices = 128, 128
		} else if distance < engine.LODClose*ls {
			rings, slices = 64, 64
		} else if distance < engine.LODMedium*ls {
			rings, slices = 32, 32
		} else if distance < engine.LODFar*ls {
			rings, slices = 16, 16
		} else {
			rings, slices = 8, 8
		}
	}

	// Textured draw: use a cached Model when texture is available.
	texPath := obj.Meta.TexturePath
	if !r.noTextures && texPath != "" {
		if model, ok := r.getModel(texPath, rings, slices); ok {
			transform, drawScale := buildModelTransform(obj.Meta, simTime)
			model.Transform = transform
			var nightTex rl.Texture2D
			if obj.Meta.NightTexturePath != "" {
				nightTex = r.loadTexture(obj.Meta.NightTexturePath)
			}
			r.lighting.applyToModel(model, obj.Meta.SelfLuminous, nightTex)
			infraF := float32(0)
			if r.infraMode == 1 && !obj.Meta.SelfLuminous {
				infraF = spotlightFactor(obj.Anim.Position, cameraPos, r.cameraForward)
				if infraF > 0 {
					r.lighting.setAmbient(defaultAmbient + (infraSpotAmbient-defaultAmbient)*infraF)
				}
			}
			rl.DrawModel(model, pos, drawScale, rl.White)
			if infraF > 0 {
				r.lighting.setAmbient(defaultAmbient)
			}
			r.drawAtmosphereGlow(obj, pos)
			return
		}
	}

	// Fallback: untextured sphere with Phong lighting when available.
	if !r.noLighting && !obj.Meta.SelfLuminous {
		model := r.getUntexturedModel(rings, slices)
		transform, drawScale := buildModelTransform(obj.Meta, simTime)
		model.Transform = transform
		r.lighting.applyToModel(model, false, rl.Texture2D{})
		infraF := float32(0)
		if r.infraMode == 1 {
			infraF = spotlightFactor(obj.Anim.Position, cameraPos, r.cameraForward)
			if infraF > 0 {
				r.lighting.setAmbient(defaultAmbient + (infraSpotAmbient-defaultAmbient)*infraF)
			}
		}
		rl.DrawModel(model, pos, drawScale, color)
		if infraF > 0 {
			r.lighting.setAmbient(defaultAmbient)
		}
	} else {
		rl.DrawSphereEx(pos, float32(obj.Meta.PhysicalRadius), rings, slices, color)
	}
	r.drawAtmosphereGlow(obj, pos)
	if obj.Meta.Category == engine.CategoryBlackHole {
		r.drawBlackHoleGlow(obj, pos)
	}

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

// drawRingDisc renders a planetary ring disc with Lambert star illumination and
// optional smooth planet-shadow darkening.
//
// Illumination: ring particles scatter light diffusely; the whole disc is scaled
// by |dot(ringNormal, toStar)| — brightest when starlight shines from above/below
// the ring plane, dimmest (ambient floor) when edge-on. Because the Lambert factor
// is the same for every point on the flat disc it is computed once and applied as
// a global multiplier before per-segment shadow blending.
//
// Shadow: when parentRadius > 0 each segment is tested against the planet's shadow
// cylinder with a smoothstep penumbra (see ringSegmentShadowFactor). Segments in
// shadow are further dimmed to ringAmbient regardless of the Lambert factor.
//
// Pass parentRadius 0 to skip shadow computation (single-object draw path).
//
// Rings are drawn with BlendAlpha + DisableDepthMask so they do not occlude
// objects behind them while still depth-testing against closer opaque geometry.
