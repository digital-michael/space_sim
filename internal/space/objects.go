package space

import "github.com/digital-michael/space_sim/internal/space/engine"

// Object factory functions for creating solar system objects.
// Real stellar masses from NASA Solar System data.

// NewSun creates the sun (emissive star).
func NewSun() *engine.Object {
	return &engine.Object{
		Meta: engine.ObjectMetadata{
			Name:           "Sun",
			Category:       engine.CategoryStar,
			Mass:           1.989e30,
			PhysicalRadius: 27.25,
			Color:          engine.Color{R: 245, G: 245, B: 255, A: 255},
			Material:       engine.MaterialEmissive,
			Importance:     100,
			OrbitRadius:    0,
			OrbitSpeed:     0,
			ParentName:     "",
		},
		Anim: engine.AnimationState{
			Position:    engine.Vector3{X: 0, Y: 0, Z: 0},
			Velocity:    engine.Vector3{},
			OrbitCenter: engine.Vector3{X: 0, Y: 0, Z: 0},
			OrbitAngle:  0,
			OrbitAxis:   engine.Vector3{X: 0, Y: 1, Z: 0},
		},
		Visible: true,
		Dataset: -1,
	}
}

// NewPlanet creates a planet orbiting the sun.
func NewPlanet(name string, distance, speed, radius float32, mass float64, color engine.Color, importance int) *engine.Object {
	period := float32(0.0)
	if speed > 0 {
		period = (2.0 * 3.14159) / speed
	}

	return &engine.Object{
		Meta: engine.ObjectMetadata{
			Name:               name,
			Category:           engine.CategoryPlanet,
			Mass:               mass,
			PhysicalRadius:     radius,
			Color:              color,
			Material:           engine.MaterialDiffuse,
			Importance:         importance,
			SemiMajorAxis:      distance,
			Eccentricity:       0.0,
			Inclination:        0.0,
			LongAscendingNode:  0.0,
			ArgPeriapsis:       0.0,
			MeanAnomalyAtEpoch: 0.0,
			OrbitalPeriod:      period,
			OrbitRadius:        distance,
			OrbitSpeed:         speed,
			ParentName:         "",
		},
		Anim: engine.AnimationState{
			Position:    engine.Vector3{X: distance, Y: 0, Z: 0},
			Velocity:    engine.Vector3{},
			OrbitCenter: engine.Vector3{X: 0, Y: 0, Z: 0},
			MeanAnomaly: 0.0,
			TrueAnomaly: 0.0,
			OrbitAngle:  0.0,
			OrbitAxis:   engine.Vector3{X: 0, Y: 1, Z: 0},
		},
		Visible: true,
		Dataset: -1,
	}
}

// NewAsteroid creates a small asteroid in the belt.
func NewAsteroid(name string, distance, speed float32) *engine.Object {
	period := float32(0.0)
	if speed > 0 {
		period = (2.0 * 3.14159) / speed
	}

	return &engine.Object{
		Meta: engine.ObjectMetadata{
			Name:               name,
			Category:           engine.CategoryAsteroid,
			Mass:               2.0e12,
			PhysicalRadius:     0.1,
			Color:              engine.ColorGray,
			Material:           engine.MaterialMetallic,
			Importance:         10,
			SemiMajorAxis:      distance,
			Eccentricity:       0.0,
			Inclination:        0.0,
			LongAscendingNode:  0.0,
			ArgPeriapsis:       0.0,
			MeanAnomalyAtEpoch: 0.0,
			OrbitalPeriod:      period,
			OrbitRadius:        distance,
			OrbitSpeed:         speed,
			ParentName:         "",
		},
		Anim: engine.AnimationState{
			Position:    engine.Vector3{X: distance, Y: 0, Z: 0},
			Velocity:    engine.Vector3{},
			OrbitCenter: engine.Vector3{X: 0, Y: 0, Z: 0},
			MeanAnomaly: 0.0,
			TrueAnomaly: 0.0,
			OrbitAngle:  0,
			OrbitAxis:   engine.Vector3{X: 0, Y: 1, Z: 0},
		},
		Visible: false,
		Dataset: -1,
	}
}

// NewSatellite creates a satellite orbiting a planet.
func NewSatellite(name, parentName string, parentDistance, localDistance, speed float32) *engine.Object {
	period := float32(0.0)
	if speed > 0 {
		period = (2.0 * 3.14159) / speed
	}

	return &engine.Object{
		Meta: engine.ObjectMetadata{
			Name:               name,
			Category:           engine.CategoryMoon,
			Mass:               1.0e20,
			PhysicalRadius:     0.15,
			Color:              engine.ColorCyan,
			Material:           engine.MaterialMirror,
			Importance:         40,
			SemiMajorAxis:      localDistance,
			Eccentricity:       0.0,
			Inclination:        0.0,
			LongAscendingNode:  0.0,
			ArgPeriapsis:       0.0,
			MeanAnomalyAtEpoch: 0.0,
			OrbitalPeriod:      period,
			OrbitRadius:        localDistance,
			OrbitSpeed:         speed,
			ParentName:         parentName,
		},
		Anim: engine.AnimationState{
			Position:    engine.Vector3{X: parentDistance + localDistance, Y: 0, Z: 0},
			Velocity:    engine.Vector3{},
			OrbitCenter: engine.Vector3{X: parentDistance, Y: 0, Z: 0},
			MeanAnomaly: 0.0,
			TrueAnomaly: 0.0,
			OrbitAngle:  0,
			OrbitAxis:   engine.Vector3{X: 0, Y: 1, Z: 0},
		},
		Visible: true,
		Dataset: -1,
	}
}

// NewDwarfPlanet creates a dwarf planet orbiting the sun.
func NewDwarfPlanet(name string, distance, speed, radius float32, mass float64, color engine.Color) *engine.Object {
	period := float32(0.0)
	if speed > 0 {
		period = (2.0 * 3.14159) / speed
	}

	return &engine.Object{
		Meta: engine.ObjectMetadata{
			Name:               name,
			Category:           engine.CategoryDwarfPlanet,
			Mass:               mass,
			PhysicalRadius:     radius,
			Color:              color,
			Material:           engine.MaterialDiffuse,
			Importance:         50,
			SemiMajorAxis:      distance,
			Eccentricity:       0.0,
			Inclination:        0.0,
			LongAscendingNode:  0.0,
			ArgPeriapsis:       0.0,
			MeanAnomalyAtEpoch: 0.0,
			OrbitalPeriod:      period,
			OrbitRadius:        distance,
			OrbitSpeed:         speed,
			ParentName:         "",
		},
		Anim: engine.AnimationState{
			Position:    engine.Vector3{X: distance, Y: 0, Z: 0},
			Velocity:    engine.Vector3{},
			OrbitCenter: engine.Vector3{X: 0, Y: 0, Z: 0},
			MeanAnomaly: 0.0,
			TrueAnomaly: 0.0,
			OrbitAngle:  0,
			OrbitAxis:   engine.Vector3{X: 0, Y: 1, Z: 0},
		},
		Visible: true,
		Dataset: -1,
	}
}

// NewMoon creates a moon orbiting a planet.
func NewMoon(name, parentName string, parentDistance, localDistance, speed, radius float32, mass float64, color engine.Color, importance int) *engine.Object {
	period := float32(0.0)
	if speed > 0 {
		period = (2.0 * 3.14159) / speed
	}

	return &engine.Object{
		Meta: engine.ObjectMetadata{
			Name:               name,
			Category:           engine.CategoryMoon,
			Mass:               mass,
			PhysicalRadius:     radius,
			Color:              color,
			Material:           engine.MaterialDiffuse,
			Importance:         importance,
			SemiMajorAxis:      localDistance,
			Eccentricity:       0.0,
			Inclination:        0.0,
			LongAscendingNode:  0.0,
			ArgPeriapsis:       0.0,
			MeanAnomalyAtEpoch: 0.0,
			OrbitalPeriod:      period,
			OrbitRadius:        localDistance,
			OrbitSpeed:         speed,
			ParentName:         parentName,
		},
		Anim: engine.AnimationState{
			Position:    engine.Vector3{X: parentDistance + localDistance, Y: 0, Z: 0},
			Velocity:    engine.Vector3{},
			OrbitCenter: engine.Vector3{X: parentDistance, Y: 0, Z: 0},
			MeanAnomaly: 0.0,
			TrueAnomaly: 0.0,
			OrbitAngle:  0,
			OrbitAxis:   engine.Vector3{X: 0, Y: 1, Z: 0},
		},
		Visible: true,
		Dataset: -1,
	}
}

// PlanetColors provides predefined planet colors.
var PlanetColors = struct {
	Mercury engine.Color
	Venus   engine.Color
	Earth   engine.Color
	Mars    engine.Color
	Jupiter engine.Color
	Saturn  engine.Color
	Uranus  engine.Color
	Neptune engine.Color
}{
	Mercury: engine.Color{R: 167, G: 167, B: 167, A: 255},
	Venus:   engine.Color{R: 255, G: 227, B: 132, A: 255},
	Earth:   engine.Color{R: 100, G: 149, B: 237, A: 255},
	Mars:    engine.Color{R: 193, G: 68, B: 14, A: 255},
	Jupiter: engine.Color{R: 201, G: 176, B: 129, A: 255},
	Saturn:  engine.Color{R: 249, G: 219, B: 144, A: 255},
	Uranus:  engine.Color{R: 175, G: 238, B: 238, A: 255},
	Neptune: engine.Color{R: 99, G: 133, B: 255, A: 255},
}
