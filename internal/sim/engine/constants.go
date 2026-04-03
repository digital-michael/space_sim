package engine

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
