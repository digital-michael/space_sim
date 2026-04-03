package sim

import (
	"fmt"
	"math"
	"math/rand"
	"sort"

	"github.com/digital-michael/space_sim/internal/sim/engine"
)

// CreateBelt generates a debris belt based on configuration.
// ObjectTypes are iterated in sorted key order so that two calls with the same
// seed and config always produce an identical object sequence — required for
// front/back double-buffer parity.
func CreateBelt(state *engine.SimulationState, config BeltConfig, dataset engine.AsteroidDataset, rng *rand.Rand) {
	typeNames := make([]string, 0, len(config.ObjectTypes))
	for typeName := range config.ObjectTypes {
		typeNames = append(typeNames, typeName)
	}
	sort.Strings(typeNames)

	objectIndex := 0
	for _, typeName := range typeNames {
		typeConfig := config.ObjectTypes[typeName]
		if typeConfig.Count == 0 {
			continue
		}

		for i := 0; i < typeConfig.Count; i++ {
			var distanceAU float64
			if config.ClassicalBeltProbability > 0 && rng.Float64() < float64(config.ClassicalBeltProbability) {
				distanceAU = float64(config.ClassicalBeltMin)/float64(config.DistanceToAURatio) +
					rng.Float64()*(float64(config.ClassicalBeltMax-config.ClassicalBeltMin)/float64(config.DistanceToAURatio))
			} else {
				distanceAU = float64(config.InnerRadius)/float64(config.DistanceToAURatio) +
					rng.Float64()*(float64(config.OuterRadius-config.InnerRadius)/float64(config.DistanceToAURatio))
			}
			distance := float32(distanceAU * float64(config.DistanceToAURatio))

			var periodSeconds float32
			if config.UseKeplerLaw {
				periodYears := math.Pow(distanceAU, 1.5)
				periodSeconds = float32(periodYears * 365.256 * 86400.0)
			} else {
				periodSeconds = 365.256 * 86400.0 * float32(math.Sqrt(math.Pow(distanceAU, 3)))
			}

			objectName := fmt.Sprintf("%s%s%05d", config.NamePrefix, typeName[:1], objectIndex)
			objectIndex++

			eccentricity := config.EccentricityMin + float32(rng.Float64())*(config.EccentricityMax-config.EccentricityMin)
			inclination := config.InclinationMin + float32(rng.Float64())*(config.InclinationMax-config.InclinationMin)
			orbitAngle := float32(rng.Float64()) * 2 * math.Pi
			posX := distance * float32(math.Cos(float64(orbitAngle)))
			posZ := distance * float32(math.Sin(float64(orbitAngle)))
			orbitYOffset := float32((rng.Float64() - 0.5) * float64(config.Thickness))

			radius := typeConfig.SizeMin + float32(rng.Float64())*(typeConfig.SizeMax-typeConfig.SizeMin)

			var color engine.Color
			if len(config.ColorPalette) > 0 {
				baseColor := config.ColorPalette[rng.Intn(len(config.ColorPalette))]
				variation := config.ColorVariation
				color = engine.Color{
					R: clampUint8(int(baseColor[0]) + int((rng.Float64()-0.5)*float64(variation)*255)),
					G: clampUint8(int(baseColor[1]) + int((rng.Float64()-0.5)*float64(variation)*255)),
					B: clampUint8(int(baseColor[2]) + int((rng.Float64()-0.5)*float64(variation)*255)),
					A: 255,
				}
			} else {
				color = engine.Color{R: 128, G: 128, B: 128, A: 255}
			}

			obj := &engine.Object{
				Meta: engine.ObjectMetadata{
					Name:               objectName,
					Category:           engine.CategoryAsteroid,
					Mass:               5.0e12,
					PhysicalRadius:     radius,
					Color:              color,
					Material:           engine.MaterialMetallic,
					Importance:         typeConfig.Importance,
					SemiMajorAxis:      distance,
					Eccentricity:       eccentricity,
					Inclination:        inclination,
					LongAscendingNode:  float32(rng.Float64() * 2 * math.Pi),
					ArgPeriapsis:       float32(rng.Float64() * 2 * math.Pi),
					MeanAnomalyAtEpoch: 0.0,
					OrbitalPeriod:      periodSeconds,
					OrbitRadius:        distance,
					OrbitSpeed:         0,
					ParentName:         "",
				},
				Anim: engine.AnimationState{
					Position:     engine.Vector3{X: posX, Y: orbitYOffset, Z: posZ},
					Velocity:     engine.Vector3{},
					OrbitCenter:  engine.Vector3{X: 0, Y: 0, Z: 0},
					OrbitAngle:   orbitAngle,
					MeanAnomaly:  orbitAngle,
					TrueAnomaly:  orbitAngle,
					OrbitYOffset: orbitYOffset,
					OrbitAxis:    engine.Vector3{X: 0, Y: 1, Z: 0},
				},
				Visible: true,
				Dataset: dataset,
			}

			state.AddObject(obj)
		}
	}

	totalCount := 0
	for _, typeConfig := range config.ObjectTypes {
		totalCount += typeConfig.Count
	}
	fmt.Printf("Created %d %s objects for dataset %d\n", totalCount, config.Name, dataset)
}

// clampUint8 clamps an integer to uint8 range [0, 255].
func clampUint8(val int) uint8 {
	if val < 0 {
		return 0
	}
	if val > 255 {
		return 255
	}
	return uint8(val)
}
