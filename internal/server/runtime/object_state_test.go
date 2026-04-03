package runtime

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestObjectStateClone(t *testing.T) {
	id := uuid.New()
	createdAt := time.Now().Add(-time.Minute)
	updatedAt := time.Now()

	original := &ObjectState{
		ID:       id,
		Position: Vector3{X: 1, Y: 2, Z: 3},
		Rotation: Vector3{X: 0.1, Y: 0.2, Z: 0.3},
		Scale:    Vector3{X: 1, Y: 1, Z: 1},
		Velocity: Vector3{X: 5, Y: 0, Z: -1},
		RoutineStates: map[string]interface{}{
			"orbit": "enabled",
		},
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
	}

	clone := original.Clone()
	if clone == original {
		t.Fatal("expected clone to be a different pointer")
	}
	if clone.ID != original.ID || clone.Position != original.Position || clone.Velocity != original.Velocity {
		t.Fatal("expected clone fields to match original")
	}

	clone.RoutineStates["orbit"] = "disabled"
	if original.RoutineStates["orbit"] != "enabled" {
		t.Fatal("expected routine states map to be copied")
	}
}
