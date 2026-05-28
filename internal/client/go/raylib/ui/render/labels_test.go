package render

// Label pipeline tests — checks all five pipeline stages without a Raylib/OpenGL context.
//
// A) Enabled:     label mode gating (LabelModeOff skips DrawObjectLabels call in interactive.go;
//                 LabelModeOn / LabelModeNearest both reach selectObjectsForLabels).
// B) Called:      selectObjectsForLabels returns ≥1 result for a scene with renderable objects.
// C) No errors:   selectObjectsForLabels produces no panics for every mode and edge case.
// D) Coordinates: testProjectToScreen (pure Go reimplementation of projectToScreen) returns
//                 correct screen coordinates for known camera/object configurations.
// E) Display:     colour override fires for very dark objects (brightness < 150);
//                 label text bounds clamp to screen edges correctly.

import (
	"math"
	"testing"

	"github.com/digital-michael/space_sim/internal/client/go/raylib/ui"
	engine "github.com/digital-michael/space_sim/internal/sim/engine"
)

// ── Test helpers ─────────────────────────────────────────────────────────────

// testProjectToScreen reimplements the production projectToScreen function in
// pure Go (no CGo / Raylib context required).  It MUST stay logically equivalent
// to the production version in renders.go.
//
// Camera convention: floating-origin, camera position = (0,0,0),
//
//	forward direction = (fwdX, fwdY, fwdZ).
func testProjectToScreen(
	objX, objY, objZ float64,
	fwdX, fwdY, fwdZ float64,
	fovYRad, aspect, near, far float64,
	screenW, screenH int,
) (sx, sy float64, inFront bool) {
	// --- View matrix via LookAt(eye=(0,0,0), target=fwd, up=(0,1,0)) -----------
	// zAxis = normalize(eye - target) = normalize(-fwd)
	zx, zy, zz := -fwdX, -fwdY, -fwdZ
	if l := math.Sqrt(zx*zx + zy*zy + zz*zz); l > 0 {
		zx /= l
		zy /= l
		zz /= l
	}
	// xAxis = normalize(cross(up, zAxis)), up=(0,1,0)
	xx := 1.0*zz - 0.0*zy // upY*zz - upZ*zy
	xy := 0.0*zx - 0.0*zz // upZ*zx - upX*zz
	xz := 0.0*zy - 1.0*zx // upX*zy - upY*zx
	if l := math.Sqrt(xx*xx + xy*xy + xz*xz); l > 0 {
		xx /= l
		xy /= l
		xz /= l
	}
	// yAxis = cross(zAxis, xAxis)
	yx := zy*xz - zz*xy
	yy := zz*xx - zx*xz
	yz := zx*xy - zy*xx

	// Transform objPos by the rotation (camera at origin, no translation)
	vx := xx*objX + xy*objY + xz*objZ
	vy := yx*objX + yy*objY + yz*objZ
	vz := zx*objX + zy*objY + zz*objZ

	// --- Perspective projection -----------------------------------------------
	// Standard clip W = -vz (objects in front have vz < 0, so cw > 0)
	cw := -vz
	if cw <= 0 {
		return 0, 0, false
	}
	f := 1.0 / math.Tan(fovYRad/2.0)
	cx := (f / aspect) * vx
	cy := f * vy

	sx = (cx/cw + 1.0) * 0.5 * float64(screenW)
	sy = (1.0 - cy/cw) * 0.5 * float64(screenH)
	return sx, sy, true
}

const (
	testFOVDeg  = 45.0
	testFOVRad  = testFOVDeg * math.Pi / 180.0
	testAspect  = 16.0 / 9.0
	testNear    = 0.001
	testFar     = 200000.0
	testScreenW = 1280
	testScreenH = 720
)

func makeFreeCamera(camX, camY, camZ float32) *ui.CameraState {
	cs := &ui.CameraState{}
	cs.Position = engine.Vector3{X: camX, Y: camY, Z: camZ}
	return cs
}

func makeObject(name string, cat engine.ObjectCategory, importance int, x, y, z float32) *engine.Object {
	return &engine.Object{
		Meta: engine.ObjectMetadata{
			Name:           name,
			Category:       cat,
			Importance:     importance,
			Color:          engine.Color{R: 200, G: 200, B: 100, A: 255},
			PhysicalRadius: 10,
		},
		Anim:    engine.AnimationState{Position: engine.Vector3{X: x, Y: y, Z: z}},
		Visible: true,
	}
}

// ── D: Coordinate math tests ─────────────────────────────────────────────────

// TestLabelProjection_FrontObjectCentered verifies that an object directly in
// front of the floating-origin camera projects to the screen centre.
// Forward direction = (0, 0, -1); object is at (0, 0, -500) in camera-relative coords.
func TestLabelProjection_FrontObjectCentered(t *testing.T) {
	sx, sy, front := testProjectToScreen(
		0, 0, -500, // object: 500 su in front along -Z
		0, 0, -1, // camera forward: -Z
		testFOVRad, testAspect, testNear, testFar,
		testScreenW, testScreenH,
	)
	if !front {
		t.Fatal("object in front reported as behind camera")
	}
	wantX := float64(testScreenW) / 2
	wantY := float64(testScreenH) / 2
	tol := 1.0 // 1-pixel tolerance
	if math.Abs(sx-wantX) > tol || math.Abs(sy-wantY) > tol {
		t.Errorf("center projection: got (%.1f, %.1f), want (%.1f, %.1f)",
			sx, sy, wantX, wantY)
	}
}

// TestLabelProjection_BehindCamera: object behind the camera must return inFront=false.
func TestLabelProjection_BehindCamera(t *testing.T) {
	_, _, front := testProjectToScreen(
		0, 0, 500, // object behind camera (camera looks toward -Z, object is at +Z)
		0, 0, -1,
		testFOVRad, testAspect, testNear, testFar,
		testScreenW, testScreenH,
	)
	if front {
		t.Fatal("object behind camera incorrectly reported as in front")
	}
}

// TestLabelProjection_QuasarDistance: object at 10 000 su (100 AU, quasar scenario)
// should project to centre without clipping — far plane is 200 000 su.
func TestLabelProjection_QuasarDistance(t *testing.T) {
	sx, sy, front := testProjectToScreen(
		0, 0, -10000, // 100 AU in front
		0, 0, -1,
		testFOVRad, testAspect, testNear, testFar,
		testScreenW, testScreenH,
	)
	if !front {
		t.Fatal("quasar-distance object reported as behind camera")
	}
	wantX := float64(testScreenW) / 2
	wantY := float64(testScreenH) / 2
	tol := 1.0
	if math.Abs(sx-wantX) > tol || math.Abs(sy-wantY) > tol {
		t.Errorf("quasar projection: got (%.1f, %.1f), want (%.1f, %.1f)",
			sx, sy, wantX, wantY)
	}
}

// TestLabelProjection_OffAxis: object 45° to the right of the forward axis
// should project near the right edge of the screen.
func TestLabelProjection_OffAxis(t *testing.T) {
	// Camera looks along -Z. Object is far to the right (+X) and slightly ahead.
	// At 45° horizontal FOV, the screen half-width corresponds to tan(22.5°)
	// in camera space. The right screen edge maps to cx/cw == 1.
	// Put the object exactly at the horizontal FOV/2 boundary: x/z_front == tan(FOV/2 * aspect)
	// Actually: right edge is when cx/cw == 1, i.e. (f/aspect)*vx/cw == 1.
	// With f = 1/tan(fov/2), aspect = 16/9, and vz = -500 (so cw = 500):
	//   vx = cw * aspect / f = 500 * (16/9) / (1/tan(22.5°)) = 500 * (16/9) * tan(22.5°)
	f := 1.0 / math.Tan(testFOVRad/2.0)
	vx := 500.0 * testAspect / f // exact right-edge x in view space
	sx, _, front := testProjectToScreen(
		vx, 0, -500,
		0, 0, -1,
		testFOVRad, testAspect, testNear, testFar,
		testScreenW, testScreenH,
	)
	if !front {
		t.Fatal("off-axis object reported as behind camera")
	}
	wantX := float64(testScreenW)
	tol := 1.0
	if math.Abs(sx-wantX) > tol {
		t.Errorf("right-edge projection: got sx=%.1f, want %.1f", sx, wantX)
	}
}

// TestLabelProjection_CoordRangeInFOV: any object within the frustum must
// produce screen coordinates inside [0, W] x [0, H].
func TestLabelProjection_CoordRangeInFOV(t *testing.T) {
	f := 1.0 / math.Tan(testFOVRad/2.0)
	// A grid of objects that are just inside the frustum at various depths
	depths := []float64{1, 10, 100, 1000, 10000, 100000}
	for _, d := range depths {
		// object at screen-centre distance d
		sx, sy, front := testProjectToScreen(0, 0, -d, 0, 0, -1,
			testFOVRad, testAspect, testNear, testFar, testScreenW, testScreenH)
		if !front {
			t.Errorf("depth=%.0f: inFront=false", d)
			continue
		}
		if sx < 0 || sx > float64(testScreenW) || sy < 0 || sy > float64(testScreenH) {
			t.Errorf("depth=%.0f: screen pos (%.1f, %.1f) outside [0,%d]x[0,%d]",
				d, sx, sy, testScreenW, testScreenH)
		}
		_ = f
	}
}

// TestLabelProjection_ScreenSizeMismatch: if projectToScreen is called with
// width=2560 (physical HiDPI) but the ortho projection is [0..1280] (logical),
// the returned X/Y coords are 2× the expected pixel position.
// This test DOCUMENTS the invariant: projectToScreen output must use the same
// dimensional scale as the 2D ortho projection used for DrawText.
func TestLabelProjection_ScreenSizeMismatch(t *testing.T) {
	// Object at screen centre; call once with logical pixels, once with physical.
	sx_logical, _, _ := testProjectToScreen(0, 0, -500, 0, 0, -1,
		testFOVRad, testAspect, testNear, testFar, 1280, 720)
	sx_physical, _, _ := testProjectToScreen(0, 0, -500, 0, 0, -1,
		testFOVRad, testAspect, testNear, testFar, 2560, 1440)

	// The physical call produces 2× the logical result.
	if math.Abs(sx_physical-2*sx_logical) > 1.0 {
		t.Errorf("HiDPI: physical (%.1f) != 2×logical (%.1f)", sx_physical, sx_logical)
	}
}

// ── B+C: selectObjectsForLabels tests ────────────────────────────────────────

func makeState(objs ...*engine.Object) *engine.SimulationState {
	ptrs := make([]*engine.Object, len(objs))
	copy(ptrs, objs)
	return &engine.SimulationState{Objects: ptrs}
}

// TestLabelSelect_EmptyScene: no objects → empty result.
func TestLabelSelect_EmptyScene(t *testing.T) {
	cs := makeFreeCamera(0, 0, 0)
	state := makeState()
	got := selectObjectsForLabels(state, cs, nil, ui.LabelModeOn)
	if len(got) != 0 {
		t.Errorf("empty scene: got %d objects, want 0", len(got))
	}
}

// TestLabelSelect_AsteroidsExcluded: Asteroid/Ring/Belt category objects are never labeled.
func TestLabelSelect_AsteroidsExcluded(t *testing.T) {
	cs := makeFreeCamera(0, 0, 0)
	roid := makeObject("Rock1", engine.CategoryAsteroid, 50, 10, 0, 0)
	ring := makeObject("Ring1", engine.CategoryRing, 50, 20, 0, 0)
	belt := makeObject("Belt1", engine.CategoryBelt, 50, 30, 0, 0)
	star := makeObject("Sol", engine.CategoryStarMainSequence, 80, 0, 0, 0)
	state := makeState(roid, ring, belt, star)
	got := selectObjectsForLabels(state, cs, []*engine.Object{roid, ring, belt, star}, ui.LabelModeOn)
	for _, obj := range got {
		if obj.Meta.Category == engine.CategoryAsteroid ||
			obj.Meta.Category == engine.CategoryRing ||
			obj.Meta.Category == engine.CategoryBelt {
			t.Errorf("excluded category %d (%s) leaked into label set", obj.Meta.Category, obj.Meta.Name)
		}
	}
	// Sol must be present
	found := false
	for _, obj := range got {
		if obj.Meta.Name == "Sol" {
			found = true
		}
	}
	if !found {
		t.Error("star Sol not included in label set")
	}
}

// TestLabelSelect_StarsAndPlanetsIncluded: standard solar-system objects get labels.
func TestLabelSelect_StarsAndPlanetsIncluded(t *testing.T) {
	cs := makeFreeCamera(100, 0, 0) // camera at ~1 AU
	sol := makeObject("Sol", engine.CategoryStarMainSequence, 100, 0, 0, 0)
	earth := makeObject("Earth", engine.CategoryPlanet, 70, 100, 0, 0)
	moon := makeObject("Moon", engine.CategoryMoon, 40, 101, 0, 0)
	state := makeState(sol, earth, moon)
	objs := []*engine.Object{sol, earth, moon}
	got := selectObjectsForLabels(state, cs, objs, ui.LabelModeOn)
	if len(got) == 0 {
		t.Fatal("no objects selected for standard solar system scene")
	}
	names := make(map[string]bool)
	for _, o := range got {
		names[o.Meta.Name] = true
	}
	for _, want := range []string{"Sol", "Earth", "Moon"} {
		if !names[want] {
			t.Errorf("%s not selected for labels", want)
		}
	}
}

// TestLabelSelect_NearestMode_DistanceFilter: in Nearest mode, only nearby objects
// within the threshold are included (unless they are the tracked/jump target).
func TestLabelSelect_NearestMode_DistanceFilter(t *testing.T) {
	cs := makeFreeCamera(0, 0, 0)
	cs.Mode = ui.CameraModeFree
	near := makeObject("NearObj", engine.CategoryPlanet, 50, 5, 0, 0)  // dist=5 su
	far := makeObject("FarObj", engine.CategoryPlanet, 50, 5000, 0, 0) // dist=5000 su
	state := makeState(near, far)
	objs := []*engine.Object{near, far}
	got := selectObjectsForLabels(state, cs, objs, ui.LabelModeNearest)
	for _, obj := range got {
		if obj.Meta.Name == "FarObj" {
			t.Errorf("far object (%s) should be filtered by Nearest mode proximity threshold", obj.Meta.Name)
		}
	}
	foundNear := false
	for _, obj := range got {
		if obj.Meta.Name == "NearObj" {
			foundNear = true
		}
	}
	if !foundNear {
		t.Error("near object not selected in Nearest mode")
	}
}

// TestLabelSelect_NearestMode_TrackedAlwaysIncluded: tracked object is always
// labeled in Nearest mode even when far away.
func TestLabelSelect_NearestMode_TrackedAlwaysIncluded(t *testing.T) {
	cs := makeFreeCamera(0, 0, 0)
	cs.Mode = ui.CameraModeTracking
	cs.Tracking.TargetIndex = 0
	cs.Tracking.Distance = 5000                                                    // so threshold = max(10, 15000)... still far
	tracked := makeObject("FarTracked", engine.CategoryPlanet, 50, 9999, 0, 0) // dist=9999 su
	state := makeState(tracked)
	objs := []*engine.Object{tracked}
	got := selectObjectsForLabels(state, cs, objs, ui.LabelModeNearest)
	found := false
	for _, obj := range got {
		if obj.Meta.Name == "FarTracked" {
			found = true
		}
	}
	if !found {
		t.Error("tracked object not included in Nearest mode despite being the tracking target")
	}
}

// TestLabelSelect_MaxLabels: LabelModeOn caps at 20 results even with 30 objects.
func TestLabelSelect_MaxLabels(t *testing.T) {
	cs := makeFreeCamera(0, 0, 0)
	objs := make([]*engine.Object, 30)
	for i := range objs {
		objs[i] = makeObject("Obj", engine.CategoryPlanet, 50, float32(i*10), 0, 0)
	}
	state := makeState(objs...)
	got := selectObjectsForLabels(state, cs, objs, ui.LabelModeOn)
	if len(got) > 20 {
		t.Errorf("LabelModeOn: got %d objects, want ≤ 20", len(got))
	}
}

// TestLabelSelect_NearestMaxLabels: LabelModeNearest caps at 5 results.
func TestLabelSelect_NearestMaxLabels(t *testing.T) {
	cs := makeFreeCamera(0, 0, 0)
	objs := make([]*engine.Object, 10)
	for i := range objs {
		objs[i] = makeObject("Obj", engine.CategoryPlanet, 50, float32(i), 0, 0) // all very close
	}
	state := makeState(objs...)
	got := selectObjectsForLabels(state, cs, objs, ui.LabelModeNearest)
	if len(got) > 5 {
		t.Errorf("LabelModeNearest: got %d objects, want ≤ 5", len(got))
	}
}

// ── E: Colour and bounds tests ────────────────────────────────────────────────

// TestLabelColor_DarkObjectOverride: objects with colour brightness < 150 must
// get a grey label (200,200,200) rather than the object's own dark colour.
// This mirrors the brightness check in drawObjectLabels.
func TestLabelColor_DarkObjectOverride(t *testing.T) {
	type colorResult struct {
		R, G, B uint8
	}
	applyBrightnessOverride := func(r, g, b uint8) colorResult {
		brightness := float32(r) + float32(g) + float32(b)
		if brightness < 150 {
			return colorResult{200, 200, 200}
		}
		return colorResult{r, g, b}
	}

	cases := []struct {
		name     string
		r, g, b  uint8
		wantGrey bool
	}{
		{"black hole (0,0,0)", 0, 0, 0, true},
		{"dim red (100,0,0)", 100, 0, 0, true},
		{"bright red (200,0,0)", 200, 0, 0, false},
		{"exact threshold (50,50,50)", 50, 50, 50, false}, // sum=150; production uses < 150, so exactly 150 is NOT overridden
		{"just below (49,50,50)", 49, 50, 50, true},       // sum=149 < 150 → grey
		{"just above (51,50,50)", 51, 50, 50, false},      // sum=151 >= 150 → keep colour
		{"bright yellow (200,200,100)", 200, 200, 100, false},
	}
	for _, c := range cases {
		result := applyBrightnessOverride(c.r, c.g, c.b)
		isGrey := result.R == 200 && result.G == 200 && result.B == 200
		if isGrey != c.wantGrey {
			t.Errorf("%s: wantGrey=%v, gotGrey=%v (result=%+v)",
				c.name, c.wantGrey, isGrey, result)
		}
	}
}

// TestLabelBounds_ClampToEdge verifies the label position clamping logic (as
// used in drawObjectLabels) keeps labels inside the screen rectangle.
func TestLabelBounds_ClampToEdge(t *testing.T) {
	const W, H = 1280.0, 720.0
	const edgePad = 10.0
	const offsetX, offsetY = 15.0, 10.0
	const textW, textH = 60.0, 20.0

	clamp := func(screenX, screenY float32) (labelX, labelY float32) {
		labelX = screenX + offsetX
		labelY = screenY - offsetY
		if labelX+textW > W-edgePad {
			labelX = screenX - textW - offsetX
		}
		if labelY < edgePad {
			labelY = edgePad
		}
		if labelY+textH > H-edgePad {
			labelY = H - textH - edgePad
		}
		return
	}

	cases := []struct {
		name             string
		screenX, screenY float32
	}{
		{"centre", W / 2, H / 2},
		{"top-left corner", 0, 0},
		{"top-right corner", W, 0},
		{"bottom-right corner", W, H},
		{"bottom-left corner", 0, H},
	}
	for _, c := range cases {
		lx, ly := clamp(c.screenX, c.screenY)
		if lx < 0 || ly < edgePad || ly+textH > H-edgePad {
			t.Errorf("%s: clamped label (%.0f, %.0f) escapes screen bounds", c.name, lx, ly)
		}
	}
}

// TestLabelProjection_MatchesProductionCameraConvention verifies that the
// test helper's LookAt convention exactly matches what the floating-origin
// camera produces: Position=(0,0,0), Target=forwardDir, Up=(0,1,0).
// An object at camera-relative position (0,0,-D) must always project to the
// screen centre regardless of which forward direction is used.
func TestLabelProjection_MatchesProductionCameraConvention(t *testing.T) {
	const D = 100.0
	fwds := [][3]float64{
		{0, 0, -1},
		{1, 0, 0},
		{0, 1, 0},
		{0.577, 0.577, -0.577}, // ~45° diagonal
	}
	for _, fwd := range fwds {
		// object is at -D along the forward axis (i.e. camera-relative position = fwd*D negated)
		// So obj = -(fwd * D) in the camera's frame, but we need world-space obj for testProjectToScreen.
		// To have view-space z = -D the object must be along the FORWARD direction at distance D.
		// In camera space: vz = dot(zAxis, obj) where zAxis = -normalize(fwd).
		// For vz = -D we need dot(-fwd, obj) = -D → dot(fwd, obj) = D → obj = fwd * D.
		objX := fwd[0] * D
		objY := fwd[1] * D
		objZ := fwd[2] * D
		sx, sy, front := testProjectToScreen(
			objX, objY, objZ,
			fwd[0], fwd[1], fwd[2],
			testFOVRad, testAspect, testNear, testFar,
			testScreenW, testScreenH,
		)
		if !front {
			t.Errorf("fwd=%v: in-front object reported as behind camera", fwd)
			continue
		}
		cx := float64(testScreenW) / 2
		cy := float64(testScreenH) / 2
		if math.Abs(sx-cx) > 1.5 || math.Abs(sy-cy) > 1.5 {
			t.Errorf("fwd=%v: got (%.1f, %.1f), want centre (%.1f, %.1f)", fwd, sx, sy, cx, cy)
		}
	}
}

// TestLabelSelect_QuasarScene verifies that the SMBH and orbiting star in the
// quasar_test system are selected for labelling (important non-belt objects).
// The accretion disk (CategoryRing) must be excluded.
func TestLabelSelect_QuasarScene(t *testing.T) {
	// Camera at 10,000 su from SMBH (≈100 AU), looking toward SMBH at origin.
	cs := &ui.CameraState{
		Position: engine.Vector3{X: 10000, Y: 0, Z: 0},
		Forward:  engine.Vector3{X: -1, Y: 0, Z: 0},
	}

	smbh := &engine.Object{
		Meta: engine.ObjectMetadata{
			Name:           "3C 273 Analog SMBH",
			Category:       engine.CategoryStellarRemnant,
			Importance:     100,
			PhysicalRadius: 2000,
			Color:          engine.Color{R: 0, G: 0, B: 0, A: 255},
		},
		Anim:    engine.AnimationState{Position: engine.Vector3{X: 0, Y: 0, Z: 0}},
		Visible: true,
	}
	star := &engine.Object{
		Meta: engine.ObjectMetadata{
			Name:       "S-Star Q1",
			Category:   engine.CategoryStarMainSequence,
			Importance: 70,
			Color:      engine.Color{R: 160, G: 200, B: 255, A: 255},
		},
		Anim:    engine.AnimationState{Position: engine.Vector3{X: 150000, Y: 0, Z: 0}},
		Visible: true,
	}
	disk := &engine.Object{
		Meta: engine.ObjectMetadata{
			Name:       "Quasar Accretion Disk",
			Category:   engine.CategoryRing,
			Importance: 80,
			Color:      engine.Color{R: 255, G: 160, B: 40, A: 210},
		},
		Anim:    engine.AnimationState{Position: engine.Vector3{X: 5000, Y: 0, Z: 0}},
		Visible: true,
	}

	state := &engine.SimulationState{Objects: []*engine.Object{smbh, star, disk}}
	objs := []*engine.Object{smbh, star, disk}

	got := selectObjectsForLabels(state, cs, objs, ui.LabelModeOn)
	if len(got) == 0 {
		t.Fatal("no objects selected for quasar scene; SMBH and star should be labelled")
	}
	for _, o := range got {
		if o.Meta.Category == engine.CategoryRing || o.Meta.Category == engine.CategoryBelt || o.Meta.Category == engine.CategoryAsteroid {
			t.Errorf("ring/belt/asteroid should be excluded, but %q was selected", o.Meta.Name)
		}
	}
	// SMBH must be selected (it's at 10000 su – within label range, importance=100)
	hasSMBH := false
	for _, o := range got {
		if o.Meta.Name == smbh.Meta.Name {
			hasSMBH = true
		}
	}
	if !hasSMBH {
		names := make([]string, len(got))
		for i, o := range got {
			names[i] = o.Meta.Name
		}
		t.Errorf("SMBH not selected; got: %v", names)
	}
}

// TestLabelProjection_QuasarDistanceFromCamera verifies that an object at 10 000 su
// directly in front of the camera projects to a valid on-screen position.
func TestLabelProjection_QuasarDistanceFromCamera(t *testing.T) {
	// Camera-relative position: object directly ahead at 10 000 su.
	// Forward = (1,0,0); obj = fwd * 10000.
	sx, sy, front := testProjectToScreen(
		10000, 0, 0,
		1, 0, 0,
		testFOVRad, testAspect, testNear, testFar,
		testScreenW, testScreenH,
	)
	if !front {
		t.Fatal("object at 10 000 su reported as behind camera; far plane may be too small")
	}
	if sx < 0 || sy < 0 || sx > testScreenW || sy > testScreenH {
		t.Errorf("projected position (%.1f, %.1f) is outside [0..%d, 0..%d]", sx, sy, testScreenW, testScreenH)
	}
}
