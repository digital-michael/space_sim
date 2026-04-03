package runtime

import (
	"math"
	"math/rand"
	"testing"
)

func TestOriginPosition(t *testing.T) {
	pos := OriginPosition()(10)
	if pos != (Vector3{}) {
		t.Fatalf("expected origin position, got %+v", pos)
	}
}

func TestGridPosition(t *testing.T) {
	grid := GridPosition(3, 10)

	first := grid(0)
	if first != (Vector3{X: 0, Y: 0, Z: 0}) {
		t.Fatalf("expected first at origin, got %+v", first)
	}

	fifth := grid(4)
	if fifth != (Vector3{X: 10, Y: 0, Z: 10}) {
		t.Fatalf("expected index 4 at (10,0,10), got %+v", fifth)
	}
}

func TestCirclePosition(t *testing.T) {
	circle := CirclePosition(100)

	atZero := circle(0)
	if math.Abs(atZero.X-100) > 1e-9 || math.Abs(atZero.Z) > 1e-9 {
		t.Fatalf("expected first point on +X axis, got %+v", atZero)
	}
}

func TestRandomPosition(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	min := Vector3{X: -1, Y: -2, Z: -3}
	max := Vector3{X: 1, Y: 2, Z: 3}
	randPos := RandomPosition(min, max, rng)

	for i := 0; i < 100; i++ {
		position := randPos(i)
		if position.X < min.X || position.X > max.X {
			t.Fatalf("x out of bounds: %+v", position)
		}
		if position.Y < min.Y || position.Y > max.Y {
			t.Fatalf("y out of bounds: %+v", position)
		}
		if position.Z < min.Z || position.Z > max.Z {
			t.Fatalf("z out of bounds: %+v", position)
		}
	}
}
