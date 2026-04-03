package runtime

import (
	"math"
	"math/rand"
	"time"
)

type PositionFunc func(index int) Vector3

func OriginPosition() PositionFunc {
	return func(index int) Vector3 {
		return Vector3{}
	}
}

func GridPosition(columns int, spacing float64) PositionFunc {
	if columns <= 0 {
		columns = 1
	}
	return func(index int) Vector3 {
		row := index / columns
		column := index % columns
		return Vector3{
			X: float64(column) * spacing,
			Y: 0,
			Z: float64(row) * spacing,
		}
	}
}

func CirclePosition(radius float64) PositionFunc {
	return func(index int) Vector3 {
		angle := float64(index) * 0.1
		return Vector3{
			X: math.Cos(angle) * radius,
			Y: 0,
			Z: math.Sin(angle) * radius,
		}
	}
}

func RandomPosition(min Vector3, max Vector3, random *rand.Rand) PositionFunc {
	rng := random
	if rng == nil {
		rng = rand.New(rand.NewSource(time.Now().UnixNano()))
	}

	return func(index int) Vector3 {
		return Vector3{
			X: min.X + rng.Float64()*(max.X-min.X),
			Y: min.Y + rng.Float64()*(max.Y-min.Y),
			Z: min.Z + rng.Float64()*(max.Z-min.Z),
		}
	}
}
