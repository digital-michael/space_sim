package engine

// MetersPerSimUnit is the number of real metres in one simulation unit.
// Derivation: Earth semi-major-axis = 100 sim units = 1 AU = 1.496 × 10¹¹ m
// → 1 sim unit = 1.496 × 10⁹ m.
// Use this to convert SI accelerations (m/s²) to sim-unit accelerations (sim_units/s²).
const MetersPerSimUnit = 1.496e9

// G_sim is the gravitational constant in simulation units:
//
//	G_SI  = 6.674e-11 m³ / (kg·s²)
//	1 sim = 1 AU/100  = 1.496e9 m
//	G_sim = G_SI / (1.496e9)³ ≈ 1.991e-38  sim³ / (kg·s²)
//
// Sanity check: Earth orbital period via T = 2π√(a³/GM_sol) with
// a = 100 sim units, GM_sol ≈ 3.96e-8 → T ≈ 3.15e7 s (1 year). ✓
const G_sim = 1.991e-38

// Rendering distance thresholds (simulation units).
const (
	// Point rendering — when objects switch from 3D mesh to point.
	PointThresholdDefault  = 200.0
	PointThresholdAsteroid = 100.0
	PointThresholdPlanet   = 500.0
	PointThresholdMoon     = 300.0

	// Point sizes (multiplied by 0.1 for actual sphere radius).
	PointSizeDefault = float32(2.0)
	PointSizeMoon    = float32(4.0)
	PointSizePlanet  = float32(6.0)
	PointSizeStar    = float32(10.0) // minimum apparent size; stars rarely point-render

	// Stars never become points via the threshold — they are always drawn as
	// full spheres so their atmosphere/corona glow is never skipped.
	PointThresholdStar = 1e15

	// LOD distance thresholds for sphere geometry quality.
	LODVeryClose = 20.0
	LODClose     = 50.0
	LODMedium    = 100.0
	LODFar       = 200.0
	LODVeryFar   = 0.0

	// Spatial partitioning grid.
	SpatialGridCellSize       = 50.0
	SpatialViewDistMin        = 1000.0
	SpatialViewDistMax        = 5000.0
	SpatialViewDistMultiplier = 2.0

	// Camera configuration.
	CameraFOV              = 45.0
	CameraNearPlane        = 0.001
	CameraFarPlane         = 200000.0
	CameraTrackDistMin     = 1.0
	CameraTrackDistMax     = 100000.0
	CameraTrackDistClose   = 8.0
	CameraTrackDistSurface = 1.1
	CameraJumpDistance     = 5.0

	// Frustum culling.
	FrustumFOVMargin            = 1.5
	FrustumNearCheck            = 0.01
	FrustumNearObjectMultiplier = 3.0
)
