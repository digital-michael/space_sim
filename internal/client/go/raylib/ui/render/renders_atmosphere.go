package render

import (
	"math"
	"unsafe"

	engine "github.com/digital-michael/space_sim/internal/sim/engine"
	rl "github.com/gen2brain/raylib-go/raylib"
)

// sampleRingTexture samples a colour from a horizontal ring colour-strip at
// normalised radial position u ∈ [0,1] (0 = inner edge, 1 = outer edge).
// The middle row of the image is used, which works for any image height.
func sampleRingTexture(img *rl.Image, u float32) rl.Color {
	if u < 0 {
		u = 0
	} else if u > 1 {
		u = 1
	}
	px := int32(u * float32(img.Width-1))
	py := img.Height / 2
	return rl.GetImageColor(*img, px, py)
}

func drawRingDisc(r *Renderer, obj *engine.Object, pos rl.Vector3, parentRadius float32) {
	const segments = 64   // angular resolution
	const radialBands = 8 // radial subdivisions for texture colour sampling

	outerRadius := float32(obj.Meta.PhysicalRadius)
	innerRadius := float32(obj.Meta.InnerRadius)
	ringRange := outerRadius - innerRadius

	// Load CPU-side colour strip for radial sampling (nil = use fallback color).
	var ringImg *rl.Image
	hasTexture := !r.noTextures && obj.Meta.TexturePath != ""
	if hasTexture {
		ringImg = r.loadRingImage(obj.Meta.TexturePath)
		hasTexture = ringImg != nil
	}

	// Axial tilt (same value used in rl.Rotatef below).
	tiltRad := float64(obj.Meta.AxialTilt) * math.Pi / 180.0
	cos_t := float32(math.Cos(tiltRad))
	sin_t := float32(math.Sin(tiltRad))

	// Lighting: Lambert factor and shadow axis, computed once per ring.
	// Ring normal in world space after Rx(tilt): (0, cos_t, sin_t).
	const ringAmbient = float32(0.05)
	var shadowAxisX, shadowAxisY, shadowAxisZ float32
	lambertFactor := float32(1.0) // full brightness when lighting is off
	hasLighting := !r.noLighting && r.lighting.loaded
	hasShadow := hasLighting && parentRadius > 0
	if hasLighting {
		dx := r.lighting.PrimaryLightPos[0] - pos.X
		dy := r.lighting.PrimaryLightPos[1] - pos.Y
		dz := r.lighting.PrimaryLightPos[2] - pos.Z
		dist := float32(math.Sqrt(float64(dx*dx + dy*dy + dz*dz)))
		if dist > 0 {
			toStarX := dx / dist
			toStarY := dy / dist
			toStarZ := dz / dist

			// |dot(ringNormal, toStar)| — two-sided disc.
			lf := cos_t*toStarY + sin_t*toStarZ
			if lf < 0 {
				lf = -lf
			}
			lambertFactor = ringAmbient + lf*(1.0-ringAmbient)

			if hasShadow {
				shadowAxisX = -toStarX
				shadowAxisY = -toStarY
				shadowAxisZ = -toStarZ
			}
		} else {
			hasShadow = false
		}
	}

	color := rl.Color{
		R: obj.Meta.Color.R,
		G: obj.Meta.Color.G,
		B: obj.Meta.Color.B,
		A: obj.Meta.Color.A,
	}

	rl.PushMatrix()
	rl.Translatef(pos.X, pos.Y, pos.Z)
	if obj.Meta.AxialTilt != 0 {
		rl.Rotatef(obj.Meta.AxialTilt, 1, 0, 0)
	}

	rl.BeginBlendMode(rl.BlendAlpha)
	rl.DisableDepthMask()

	for band := 0; band < radialBands; band++ {
		bandInner := innerRadius + float32(band)/float32(radialBands)*ringRange
		bandOuter := innerRadius + float32(band+1)/float32(radialBands)*ringRange
		bandMidU := (float32(band) + 0.5) / float32(radialBands)
		bandMidRadius := (bandInner + bandOuter) * 0.5

		baseColor := color
		if hasTexture {
			sampled := sampleRingTexture(ringImg, bandMidU)
			// Preserve the ring's configured alpha; use texture RGB for colour.
			baseColor = rl.Color{R: sampled.R, G: sampled.G, B: sampled.B, A: color.A}
		}

		for i := 0; i < segments; i++ {
			angle1 := float32(i) * 2.0 * math.Pi / float32(segments)
			angle2 := float32(i+1) * 2.0 * math.Pi / float32(segments)

			// Start from the global Lambert factor; attenuate further in shadow.
			litFactor := lambertFactor
			if hasShadow {
				midAngle := (angle1 + angle2) * 0.5
				shadowLit := ringSegmentShadowFactor(float64(bandMidRadius), float64(midAngle),
					tiltRad, shadowAxisX, shadowAxisY, shadowAxisZ, parentRadius)
				if shadowLit < 1.0 {
					// In shadow: darken to ringAmbient, smoothly blended at the penumbra.
					inShadowFactor := ringAmbient + (1.0-ringAmbient)*shadowLit
					if inShadowFactor < litFactor {
						litFactor = inShadowFactor
					}
				}
			}

			segColor := baseColor
			if litFactor < 1.0 {
				segColor = applyRingLightFactor(baseColor, litFactor)
			}

			cos1 := float32(math.Cos(float64(angle1)))
			sin1 := float32(math.Sin(float64(angle1)))
			cos2 := float32(math.Cos(float64(angle2)))
			sin2 := float32(math.Sin(float64(angle2)))

			p1 := rl.Vector3{X: bandOuter * cos1, Y: 0, Z: bandOuter * sin1}
			p2 := rl.Vector3{X: bandOuter * cos2, Y: 0, Z: bandOuter * sin2}
			p3 := rl.Vector3{X: bandInner * cos1, Y: 0, Z: bandInner * sin1}
			p4 := rl.Vector3{X: bandInner * cos2, Y: 0, Z: bandInner * sin2}

			// Front face
			rl.DrawTriangle3D(p1, p2, p3, segColor)
			rl.DrawTriangle3D(p2, p4, p3, segColor)
			// Back face
			rl.DrawTriangle3D(p1, p3, p2, segColor)
			rl.DrawTriangle3D(p2, p3, p4, segColor)
		}
	}

	rl.EnableDepthMask()
	rl.EndBlendMode()
	rl.PopMatrix()
}

// ringSegmentShadowFactor returns a [0,1] light factor for a ring point at the
// given polar coordinates (radius, angle) in the ring plane.
//
//	0 = fully in the planet's shadow (darkened to ringAmbient by caller)
//	1 = fully lit
//
// The shadow is modelled as a cylinder of radius parentRadius aligned with
// shadowAxis (the normalised anti-star direction in camera-relative world space).
// A smoothstep penumbra spanning ±ringPenumbraFrac of parentRadius around the
// cylinder edge removes the staircase artefact from the discrete segment grid.
//
// The ring point is transformed into world space by applying the axial tilt:
//
//	P = Rx(tiltRad) * (radius·cos(a), 0, radius·sin(a))
//	  = (radius·cos(a), −radius·sin(a)·sin(tilt), radius·sin(a)·cos(tilt))
const ringPenumbraFrac = float32(0.25) // shadow-edge transition = ±25% of planet radius

func ringSegmentShadowFactor(radius, angle, tiltRad float64, shadowAxisX, shadowAxisY, shadowAxisZ, parentRadius float32) float32 {
	cos_a := float32(math.Cos(angle))
	sin_a := float32(math.Sin(angle))
	cos_t := float32(math.Cos(tiltRad))
	sin_t := float32(math.Sin(tiltRad))
	rv := float32(radius)

	px := rv * cos_a
	py := -rv * sin_a * sin_t
	pz := rv * sin_a * cos_t

	// Project P onto the shadow axis (anti-star direction from planet centre).
	proj := px*shadowAxisX + py*shadowAxisY + pz*shadowAxisZ
	if proj <= 0 {
		return 1.0 // illuminated side
	}

	// Perpendicular distance from the shadow axis.
	perpX := px - proj*shadowAxisX
	perpY := py - proj*shadowAxisY
	perpZ := pz - proj*shadowAxisZ
	perpDist := float32(math.Sqrt(float64(perpX*perpX + perpY*perpY + perpZ*perpZ)))

	// Penumbra zone: innerEdge (full shadow) → outerEdge (full lit).
	innerEdge := parentRadius * (1.0 - ringPenumbraFrac)
	outerEdge := parentRadius * (1.0 + ringPenumbraFrac)

	if perpDist >= outerEdge {
		return 1.0 // fully lit
	}
	if perpDist <= innerEdge {
		return 0.0 // fully in shadow
	}
	// Smooth transition across the penumbra band.
	t := (perpDist - innerEdge) / (outerEdge - innerEdge)
	return t * t * (3 - 2*t) // smoothstep(0,1,t)
}

// applyRingLightFactor scales the RGB channels of c by factor, leaving alpha unchanged.
func applyRingLightFactor(c rl.Color, factor float32) rl.Color {
	if factor >= 1.0 {
		return c
	}
	if factor < 0 {
		factor = 0
	}
	return rl.Color{
		R: uint8(float32(c.R) * factor),
		G: uint8(float32(c.G) * factor),
		B: uint8(float32(c.B) * factor),
		A: c.A,
	}
}

// infraSpotAmbient is the ambient floor at the centre of the infra spotlight (mode 1).
// 0.90 makes unlit sides of planets clearly visible — like a fill light.
const infraSpotAmbient = float32(0.90)

// infraSpotInnerCos = cos(20°): objects within 20° of camera centre are fully lit.
// infraSpotOuterCos = cos(40°): objects beyond 40° off-axis receive no boost.
const (
	infraSpotInnerCos = float32(0.9397) // cos(20°)
	infraSpotOuterCos = float32(0.7660) // cos(40°)
)

// spotlightFactor returns a [0,1] blend factor indicating how centred objPos is
// in the camera view. 1 = inside the inner cone, 0 = outside the outer cone.
// cameraForward must be normalised.
func spotlightFactor(objPos, cameraPos, cameraForward engine.Vector3) float32 {
	dx := objPos.X - cameraPos.X
	dy := objPos.Y - cameraPos.Y
	dz := objPos.Z - cameraPos.Z
	dist := float32(math.Sqrt(float64(dx*dx + dy*dy + dz*dz)))
	if dist < 0.001 {
		return 1.0
	}
	cosAngle := (dx*cameraForward.X + dy*cameraForward.Y + dz*cameraForward.Z) / dist
	// smoothstep from outer edge to inner edge
	t := (cosAngle - infraSpotOuterCos) / (infraSpotInnerCos - infraSpotOuterCos)
	if t <= 0 {
		return 0
	}
	if t >= 1 {
		return 1
	}
	return t * t * (3 - 2*t)
}

// skyTexPath is the default path to the Milky Way equirectangular panorama used
// as the background skysphere. The file is downloaded by scripts/download_textures.sh
// ("Downloading starfield background" step: 8k_stars_milky_way.jpg → starfield_8k.jpg).
const skyTexPath = "data/assets/textures/starfield_8k.jpg"

// skyRadius is the radius of the background skysphere in sim units. The sphere
// is always centred at the floating-origin camera (0,0,0); any radius larger
// than the near plane (0.001) covers the entire field of view. A small radius
// is critical: with a large radius (e.g. 180 000) half the sphere vertices have
// negative z in view-space, and OpenGL's near-plane homogeneous clip arithmetic
// has float32 precision ratio ~1.8e8:1 at that scale — the clipping becomes
// numerically degenerate and triangles are silently discarded.
// The sky uses DisableDepthTest + DisableDepthMask, so the 5-unit sphere never
// occludes or is occluded by scene geometry regardless of its actual position.
const skyRadius = float32(5.0)

// DrawSky renders the Milky Way background skysphere before scene geometry.
// Must be called inside rl.BeginMode3D and before any other 3D draw calls so
// the sky is always occluded by foreground objects.
//
// The sphere is always centred at the floating-origin camera position (0,0,0),
// so it rotates with camera orientation but never translates — producing the
// correct parallax-free backdrop for a solar-system-scale scene.
// If noTextures is set or the texture file is absent, DrawSky is a no-op and
// the background remains black.
func (r *Renderer) DrawSky() {
	if r.noTextures {
		return
	}
	if !r.skyLoaded {
		r.skyLoaded = true
		r.skyTex = r.loadTexture(skyTexPath)
		if r.skyTex.ID > 0 {
			mesh := rl.GenMeshSphere(1.0, 32, 16)
			// UV fix for inside-sphere (sky-dome) viewing.
			// par_shapes stores: uv[0]=latitude (0=south, 1=north),
			//                    uv[1]=longitude (0→1 left-to-right from outside).
			// Standard equirectangular wants U=longitude, V=1-latitude
			// (stb_image is top-first; OpenGL V=0 is bottom, so V must be flipped).
			// Unlike planet models (outside view, needs extra U-flip), the sky is
			// viewed from inside — longitude direction is already correct.
			n := int(mesh.VertexCount)
			uv := unsafe.Slice(mesh.Texcoords, n*2)
			for i := 0; i < n; i++ {
				uv[i*2], uv[i*2+1] = uv[i*2+1], 1.0-uv[i*2]
			}
			rl.UpdateMeshBuffer(mesh, 1, unsafe.Slice((*byte)(unsafe.Pointer(mesh.Texcoords)), n*8), 0)
			r.skyModel = rl.LoadModelFromMesh(mesh)
			mats := r.skyModel.GetMaterials()
			if len(mats) > 0 {
				mats[0].GetMap(rl.MapDiffuse).Texture = r.skyTex
				// LoadModelFromMesh initialises the material with Raylib's built-in
				// default flat shader (texture * diffuse color, no lighting). We leave
				// that shader in place — DrawSky never calls applyToModel, so the Phong
				// star shader is never bound to the sky material.
			}
			// Align sphere: par_shapes north pole is +Z; rotate to world +Y (up).
			// Scale to sky radius after rotation — order: scale then rotate.
			r.skyModel.Transform = rl.MatrixMultiply(
				rl.MatrixRotateX(-90*rl.Deg2rad),
				rl.MatrixScale(skyRadius, skyRadius, skyRadius),
			)
			r.skyModelLoaded = true
		}
	}
	if !r.skyModelLoaded {
		return // texture not found or failed to load
	}
	rl.DisableDepthMask()       // sky must never write to the depth buffer
	rl.DisableBackfaceCulling() // we are inside the sphere; render its inner surface
	rl.DrawModel(r.skyModel, rl.Vector3{}, 1.0, rl.White)
	rl.EnableBackfaceCulling()
	rl.EnableDepthMask()
}

// drawAtmosphereGlow renders a Fresnel rim-glow atmosphere with day/night lighting.
// Uses a custom GLSL shader (atmosphereState) that applies:
//   - a Fresnel term: transparent face-on, bright at the limb (grazing angle)
//   - a Lambert sun-facing term: day side lit, night side 8% ambient (secondary scatter)
//
// Glow radius is physics-calibrated from AtmosphereThicknessKm:
//
//	km_per_sim_unit = 12742  (calibrated to Earth: 6371 km / 0.5 sim = 12742 km/su)
//	frac = (AtmosphereThicknessKm / (bodyRadius * km_per_sim_unit)) * 4
//	glowRadius = bodyRadius * (1 + clamp(frac, 0.04, 0.15))
func (r *Renderer) drawAtmosphereGlow(obj *engine.Object, pos rl.Vector3) {
	if obj.Meta.AtmosphereThicknessKm <= 0 || obj.Meta.AtmosphereColorHint.A == 0 {
		return
	}
	// Lazy-load atmosphere shader and unit-sphere model.
	if !r.atmosphere.loaded {
		r.atmosphere.load()
	}
	if !r.atmoSphereLoaded {
		mesh := rl.GenMeshSphere(1.0, 64, 32)
		r.atmoSphere = rl.LoadModelFromMesh(mesh)
		r.atmoSphereLoaded = true
	}

	// Base radius: equatorial for oblate bodies, physical radius otherwise.
	baseRadius := obj.Meta.PhysicalRadius
	if obj.Meta.EquatorialRadius > 0 {
		baseRadius = obj.Meta.EquatorialRadius
	}
	// Physics-calibrated glow fraction.
	const (
		// kmPerSimUnit is the Earth-calibrated fallback: 6371 km / 0.5 sim units.
		// Gas giants have a ~2× larger real km/su ratio; bodies with PhysicalRadiusKm
		// set will use the accurate per-body conversion instead.
		kmPerSimUnit = float32(12742)
		atmoBoost    = float32(4)    // visual boost; 100km/6371km*4 ≈ 6% for Earth
		atmoFloor    = float32(0.04) // minimum 4% — ensures any atmosphere is visible
		atmoCap      = float32(0.15) // maximum 15% — keeps Sol's corona as a tight rim halo
	)
	// Use the stored real-world radius when available; fall back to Earth-calibrated.
	bodyRadiusKm := baseRadius * kmPerSimUnit
	if obj.Meta.PhysicalRadiusKm > 0 {
		bodyRadiusKm = obj.Meta.PhysicalRadiusKm
	}
	frac := (obj.Meta.AtmosphereThicknessKm / bodyRadiusKm) * atmoBoost
	if frac < atmoFloor {
		frac = atmoFloor
	}
	if frac > atmoCap {
		frac = atmoCap
	}
	glowRadius := baseRadius * (1.0 + frac)

	// Scale the unit sphere to the glow radius.
	r.atmoSphere.Transform = rl.MatrixScale(glowRadius, glowRadius, glowRadius)

	// Bind atmosphere shader to the model's material.
	mats := r.atmoSphere.GetMaterials()
	if len(mats) > 0 {
		mats[0].Shader = r.atmosphere.shader
	}

	// Upload per-draw uniforms.
	ch := obj.Meta.AtmosphereColorHint
	glowColor := []float32{
		float32(ch.R) / 255.0,
		float32(ch.G) / 255.0,
		float32(ch.B) / 255.0,
		float32(ch.A) / 255.0 * 0.6, // intensity weight
	}
	rl.SetShaderValue(r.atmosphere.shader, r.atmosphere.locGlowColor, glowColor, rl.ShaderUniformVec4)
	// Copy PrimaryLightPos to a fresh local slice — slicing a struct field produces an
	// interior pointer into Renderer (which contains Go pointers), and CGo rejects that.
	lightPos := []float32{r.lighting.PrimaryLightPos[0], r.lighting.PrimaryLightPos[1], r.lighting.PrimaryLightPos[2]}
	rl.SetShaderValue(r.atmosphere.shader, r.atmosphere.locLightPos, lightPos, rl.ShaderUniformVec3)
	const glowEdge = float32(3.5) // Fresnel exponent: narrower > 5, broader < 3
	rl.SetShaderValue(r.atmosphere.shader, r.atmosphere.locGlowEdge, []float32{glowEdge}, rl.ShaderUniformFloat)
	// Self-luminous bodies (stars) illuminate their own corona from within; bypass
	// the Lambert day/night term so the glow is at full intensity all around the limb.
	selfLum := int32(0)
	if obj.Meta.SelfLuminous {
		selfLum = 1
	}
	selfLumF := *(*float32)(unsafe.Pointer(&selfLum))
	rl.SetShaderValue(r.atmosphere.shader, r.atmosphere.locSelfLuminous, []float32{selfLumF}, rl.ShaderUniformInt)

	rl.BeginBlendMode(rl.BlendAddColors)
	// Disable depth writes so the atmosphere sphere does not overwrite the depth
	// buffer at glowRadius. Without this, opaque objects between physicalRadius
	// and glowRadius fail the depth test on subsequent draw calls and flicker.
	// The atmosphere contributes colour only — it must never occlude geometry.
	rl.DisableDepthMask()
	rl.DrawModel(r.atmoSphere, pos, 1.0, rl.White)
	rl.EnableDepthMask()
	rl.EndBlendMode()
}

// drawBlackHoleVisual renders a black hole using a flat accretion disk mesh plus
// two Fresnel shell halos:
//
//   - Flat annular disk (1.5 R → 7 R): temperature gradient from near-white inner
//     edge (ISCO) to transparent outer rim; angle-dependent (thin line edge-on, full
//     ellipse face-on).  Skipped when accretion energy is effectively zero.
//
//   - Shell 1 (1.6 R): tight photon-ring halo, white-orange Fresnel rim. Always
//     present — it is a gravitational effect, not an accretion effect.
//
//   - Shell 2 (2.2 R): lamppost-corona haze, blue-purple. Scales with accretion energy.
//
// Disk tilt priority: BHDiskTilt (rendering JSON) → AxialTilt (physical JSON) → 30° default.
// Accretion energy priority: BHAccretionEnergy (0 = absent → 1.0 default).
func (r *Renderer) drawBlackHoleVisual(obj *engine.Object, pos rl.Vector3) {
	radius := float32(obj.Meta.PhysicalRadius)
	if radius <= 0 {
		return
	}

	if !r.atmosphere.loaded {
		r.atmosphere.load()
	}
	if !r.atmoSphereLoaded {
		mesh := rl.GenMeshSphere(1.0, 64, 32)
		r.atmoSphere = rl.LoadModelFromMesh(mesh)
		r.atmoSphereLoaded = true
	}

	mats := r.atmoSphere.GetMaterials()
	if len(mats) > 0 {
		mats[0].Shader = r.atmosphere.shader
	}

	lightPos := []float32{r.lighting.PrimaryLightPos[0], r.lighting.PrimaryLightPos[1], r.lighting.PrimaryLightPos[2]}
	rl.SetShaderValue(r.atmosphere.shader, r.atmosphere.locLightPos, lightPos, rl.ShaderUniformVec3)
	selfLum := int32(1)
	selfLumF := *(*float32)(unsafe.Pointer(&selfLum))
	rl.SetShaderValue(r.atmosphere.shader, r.atmosphere.locSelfLuminous, []float32{selfLumF}, rl.ShaderUniformInt)

	// Mass scale: sqrt(radius/5); reference R=5 → ms=1.0. Clamped [0.5, 2.0].
	ms := float32(math.Sqrt(float64(radius) / 5.0))
	if ms < 0.5 {
		ms = 0.5
	}
	if ms > 2.0 {
		ms = 2.0
	}

	// Accretion energy: 0 in meta means "not set" → default 1.0.
	ae := obj.Meta.BHAccretionEnergy
	if ae <= 0 {
		ae = 1.0
	}

	// Disk tilt: explicit rendering override → physical axial_tilt → 30° fallback.
	diskTilt := obj.Meta.BHDiskTilt
	if diskTilt == 0 {
		diskTilt = obj.Meta.AxialTilt
	}
	if diskTilt == 0 {
		diskTilt = 30.0
	}

	// Fresnel shell halos — 3D presence visible from all viewing angles.
	type shell struct {
		mult               float32
		r, g, b, intensity float32
		edge               float32
	}
	shells := []shell{
		{1.6, 1.00, 0.72, 0.22, ms * 0.90, 7.5},       // photon ring: always present
		{2.2, 0.18, 0.28, 0.82, ms * 0.22 * ae, 3.8},  // lamppost corona: scales with accretion
	}

	rl.BeginBlendMode(rl.BlendAddColors)
	rl.DisableDepthMask()
	for _, s := range shells {
		sr := radius * s.mult
		r.atmoSphere.Transform = rl.MatrixScale(sr, sr, sr)
		glowColor := []float32{s.r, s.g, s.b, s.intensity}
		rl.SetShaderValue(r.atmosphere.shader, r.atmosphere.locGlowColor, glowColor, rl.ShaderUniformVec4)
		rl.SetShaderValue(r.atmosphere.shader, r.atmosphere.locGlowEdge, []float32{s.edge}, rl.ShaderUniformFloat)
		rl.DrawModel(r.atmoSphere, pos, 1.0, rl.White)
	}
	rl.EnableDepthMask()
	rl.EndBlendMode()

	diskIntensity := ms * 1.2 * ae
	if diskIntensity > 0.02 {
		r.drawBHDisk(pos, radius, diskTilt, diskIntensity)
	}
}

// drawBHDisk renders a flat annular accretion disk for a black hole.
// The disk spans innerMult..outerMult of radius in the XZ plane, tilted by
// tiltDeg degrees around the X axis (matching the AxialTilt convention used by
// drawRingDisc). A temperature gradient from near-white inner edge (ISCO) to deep
// red outer rim is applied; intensity is modulated by intensityScale.
// Drawn with additive blending; does not write to the depth buffer.
func (r *Renderer) drawBHDisk(pos rl.Vector3, radius, tiltDeg, intensityScale float32) {
	const (
		segments    = 72
		radialBands = 14
	)

	innerR := radius * 1.5 // ISCO ≈ 3 R_s for Schwarzschild; visual inner edge
	outerR := radius * 7.0
	ringRange := outerR - innerR

	// Temperature gradient: innermost band is near-white (~10^7 K), outer bands
	// cool through orange into dim red.  Values are linear RGB at unit intensity.
	type bandRGB struct{ r, g, b float32 }
	gradient := [radialBands]bandRGB{
		{1.00, 0.97, 0.88}, // 0  innermost — blazing near-white
		{1.00, 0.90, 0.65}, // 1
		{1.00, 0.80, 0.42}, // 2
		{1.00, 0.68, 0.22}, // 3
		{1.00, 0.55, 0.10}, // 4  orange
		{1.00, 0.42, 0.05}, // 5
		{0.90, 0.30, 0.02}, // 6
		{0.72, 0.19, 0.01}, // 7
		{0.54, 0.11, 0.00}, // 8
		{0.38, 0.06, 0.00}, // 9
		{0.24, 0.03, 0.00}, // 10
		{0.14, 0.01, 0.00}, // 11
		{0.07, 0.00, 0.00}, // 12
		{0.03, 0.00, 0.00}, // 13 outermost — near-black
	}

	rl.PushMatrix()
	rl.Translatef(pos.X, pos.Y, pos.Z)
	if tiltDeg != 0 {
		rl.Rotatef(tiltDeg, 1, 0, 0)
	}

	rl.BeginBlendMode(rl.BlendAddColors)
	rl.DisableDepthMask()

	for band := 0; band < radialBands; band++ {
		bandInner := innerR + float32(band)/float32(radialBands)*ringRange
		bandOuter := innerR + float32(band+1)/float32(radialBands)*ringRange

		g := gradient[band]
		scale := intensityScale * 255.0
		// Alpha fades quadratically from inner to outer edge so the geometric boundary
		// of the disk dissolves rather than cutting off as a visible dark ring.
		t := float32(band) / float32(radialBands)
		alpha := bhColorByte((1.0 - t) * (1.0 - t) * 255.0)
		col := rl.Color{
			R: bhColorByte(g.r * scale),
			G: bhColorByte(g.g * scale),
			B: bhColorByte(g.b * scale),
			A: alpha,
		}

		for i := 0; i < segments; i++ {
			angle1 := float32(i) * 2.0 * math.Pi / float32(segments)
			angle2 := float32(i+1) * 2.0 * math.Pi / float32(segments)

			cos1 := float32(math.Cos(float64(angle1)))
			sin1 := float32(math.Sin(float64(angle1)))
			cos2 := float32(math.Cos(float64(angle2)))
			sin2 := float32(math.Sin(float64(angle2)))

			p1 := rl.Vector3{X: bandOuter * cos1, Y: 0, Z: bandOuter * sin1}
			p2 := rl.Vector3{X: bandOuter * cos2, Y: 0, Z: bandOuter * sin2}
			p3 := rl.Vector3{X: bandInner * cos1, Y: 0, Z: bandInner * sin1}
			p4 := rl.Vector3{X: bandInner * cos2, Y: 0, Z: bandInner * sin2}

			// Front and back faces so the disk is visible from both sides.
			rl.DrawTriangle3D(p1, p2, p3, col)
			rl.DrawTriangle3D(p2, p4, p3, col)
			rl.DrawTriangle3D(p1, p3, p2, col)
			rl.DrawTriangle3D(p2, p3, p4, col)
		}
	}

	rl.EnableDepthMask()
	rl.EndBlendMode()
	rl.PopMatrix()
}

// bhColorByte clamps a float32 colour value to the [0, 255] uint8 range.
func bhColorByte(v float32) uint8 {
	if v <= 0 {
		return 0
	}
	if v >= 255 {
		return 255
	}
	return uint8(v)
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

// projectToScreen projects a camera-relative world position to screen coordinates
// using the same perspective projection as the scene render (engine.CameraFarPlane).
// Returns (screenPos, true) when the point is in front of the camera; (zero, false) otherwise.
// This replaces rl.GetWorldToScreenEx which uses a hardcoded far plane of 1000 su,
// causing labels to wander for objects farther than ~10 AU.
