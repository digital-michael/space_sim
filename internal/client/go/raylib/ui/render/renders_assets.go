package render

import (
	"path/filepath"
	"unsafe"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// loadTexture returns a cached GPU texture for path, loading it on first use.
// Returns a zero Texture2D (ID=0) if the path is empty or the file cannot be
// loaded — callers must check tex.ID > 0 before using.
func (r *Renderer) loadTexture(path string) rl.Texture2D {
	if path == "" {
		return rl.Texture2D{}
	}
	if tex, ok := r.textureCache[path]; ok {
		return tex
	}
	abs := filepath.Clean(path)
	tex := rl.LoadTexture(abs)
	if tex.ID == 0 {
		// File missing or format unsupported — cache the zero value so we do not
		// retry every frame.
		r.textureCache[path] = rl.Texture2D{}
		return rl.Texture2D{}
	}
	rl.SetTextureFilter(tex, rl.FilterBilinear)
	r.textureCache[path] = tex
	return tex
}

// getModel returns a cached textured sphere model for the given texture path
// and LOD quality. The model uses a unit sphere scaled at draw time.
// Returns (model, true) on success; (zero, false) if the texture failed to load.
func (r *Renderer) getModel(path string, rings, slices int32) (rl.Model, bool) {
	key := modelKey{path: path, rings: rings, slices: slices}
	if m, ok := r.modelCache[key]; ok {
		return m, true
	}
	tex := r.loadTexture(path)
	if tex.ID == 0 {
		return rl.Model{}, false
	}
	mesh := rl.GenMeshSphere(1.0, int(rings), int(slices))
	// par_shapes maps uv[0]→latitude (stored as U/horizontal) and uv[1]→longitude
	// (stored as V/vertical), which is transposed from standard equirectangular
	// convention (U=longitude, V=latitude, top=north). Swap to correct it.
	n := int(mesh.VertexCount)
	uv := unsafe.Slice(mesh.Texcoords, n*2)
	for i := 0; i < n; i++ {
		// par_shapes UV fix — four independent corrections stacked:
		//   1. Swap axes: par_shapes uv[0]=latitude, uv[1]=longitude; standard equirectangular is opposite.
		//   2. Flip V: stb_image loads top-first; OpenGL V=0 is bottom. Without flip, north pole samples south of image.
		//   3. Flip U: sphere UVs are wound for viewing from inside (sky-dome convention). Viewed from outside, longitude increases left instead of right — mirror east/west.
		//   4. Axis fix (draw time): par_shapes north pole is +Z; corrected to world +Y via RotX(-90°) in model.Transform.
		uv[i*2], uv[i*2+1] = 1.0-uv[i*2+1], 1.0-uv[i*2]
	}
	rl.UpdateMeshBuffer(mesh, 1, unsafe.Slice((*byte)(unsafe.Pointer(mesh.Texcoords)), n*8), 0)
	model := rl.LoadModelFromMesh(mesh)
	model.GetMaterials()[0].GetMap(rl.MapDiffuse).Texture = tex
	r.modelCache[key] = model
	return model, true
}

// getUntexturedModel returns a cached unit-sphere model with no texture bound.
// Used to apply the Phong shader with a solid-colour tint to untextured bodies
// (moons, dwarf planets, asteroids). The default Raylib texture for an unbound
// MAP_DIFFUSE slot is a 1×1 white pixel, so the fragment colour becomes
// colDiffuse × lighting — exactly the object's configured colour shaded by stars.
func (r *Renderer) getUntexturedModel(rings, slices int32) rl.Model {
	key := modelKey{path: "", rings: rings, slices: slices}
	if m, ok := r.modelCache[key]; ok {
		return m
	}
	mesh := rl.GenMeshSphere(1.0, int(rings), int(slices))
	model := rl.LoadModelFromMesh(mesh)
	r.modelCache[key] = model
	return model
}

// loadRingImage returns a cached CPU-side image for the given ring colour-strip
// path.  Returns nil if the path is empty or unreadable.  The image is kept in
// CPU memory so per-segment colours can be sampled each frame without a GPU
// readback.
func (r *Renderer) loadRingImage(path string) *rl.Image {
	if path == "" {
		return nil
	}
	if img, ok := r.ringImageCache[path]; ok {
		return img
	}
	abs := filepath.Clean(path)
	img := rl.LoadImage(abs)
	if img == nil || img.Width == 0 || img.Height == 0 {
		r.ringImageCache[path] = nil
		return nil
	}
	r.ringImageCache[path] = img
	return img
}

func (r *Renderer) Close() {
	if r.targetLoaded {
		rl.UnloadRenderTexture(r.target)
		r.targetLoaded = false
	}
	// Unload cached models first (they hold their own mesh+material, but NOT the
	// texture — the model's diffuse texture slot is cleared before unloading so
	// Raylib does not double-free the cached texture we still own).
	for key, model := range r.modelCache {
		model.GetMaterials()[0].GetMap(rl.MapDiffuse).Texture = rl.Texture2D{}
		rl.UnloadModel(model)
		delete(r.modelCache, key)
	}
	// Unload cached textures.
	for path, tex := range r.textureCache {
		if tex.ID > 0 {
			rl.UnloadTexture(tex)
		}
		delete(r.textureCache, path)
	}
	// Unload cached ring CPU images.
	for path, img := range r.ringImageCache {
		if img != nil && img.Width > 0 {
			rl.UnloadImage(img)
		}
		delete(r.ringImageCache, path)
	}
	setLayoutSize(0, 0)
	r.lighting.unload()
	if r.atmosphere.loaded {
		r.atmosphere.unload()
	}
	if r.atmoSphereLoaded {
		// Clear diffuse slot before unloading so Raylib does not double-free the
		// atmosphere texture (it is not cached separately — it has no path).
		mats := r.atmoSphere.GetMaterials()
		if len(mats) > 0 {
			mats[0].GetMap(rl.MapDiffuse).Texture = rl.Texture2D{}
		}
		rl.UnloadModel(r.atmoSphere)
		r.atmoSphereLoaded = false
	}
	if r.skyModelLoaded {
		// Clear diffuse slot — sky texture lives in textureCache and will be
		// freed by the texture-cache loop above; avoid a double-free here.
		mats := r.skyModel.GetMaterials()
		if len(mats) > 0 {
			mats[0].GetMap(rl.MapDiffuse).Texture = rl.Texture2D{}
		}
		rl.UnloadModel(r.skyModel)
		r.skyModelLoaded = false
	}
}

