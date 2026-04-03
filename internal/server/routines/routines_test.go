package routines

import (
	"testing"

	"github.com/digital-michael/space_sim/internal/server/pool"
)

func TestLibrary_Register(t *testing.T) {
	lib := NewLibrary()

	// Check built-ins are registered
	names := lib.List()
	if len(names) < 3 {
		t.Errorf("Expected at least 3 built-in routines, got %d", len(names))
	}
}

func TestLibrary_Get(t *testing.T) {
	lib := NewLibrary()

	routine, err := lib.Get("rotate")
	if err != nil {
		t.Fatalf("Failed to get rotate routine: %v", err)
	}

	if routine.Name() != "rotate" {
		t.Errorf("Expected name 'rotate', got %s", routine.Name())
	}
}

func TestLibrary_Execute(t *testing.T) {
	lib := NewLibrary()
	obj := pool.NewObject("test")

	err := lib.Execute("rotate", obj, 1.0)
	if err != nil {
		t.Fatalf("Failed to execute routine: %v", err)
	}

	// Check rotation changed
	if obj.Rotation.Z == 0 {
		t.Error("Expected rotation to change after execute")
	}
}

func TestRotateRoutine(t *testing.T) {
	routine := &RotateRoutine{}
	obj := pool.NewObject("test")
	obj.Properties["rotationSpeed"] = 45.0

	err := routine.Execute(obj, 1.0)
	if err != nil {
		t.Fatalf("Execution failed: %v", err)
	}

	if obj.Rotation.Z != 45.0 {
		t.Errorf("Expected rotation 45.0, got %f", obj.Rotation.Z)
	}
}

func TestMoveRoutine(t *testing.T) {
	routine := &MoveRoutine{}
	obj := pool.NewObject("test")
	obj.Properties["velocityX"] = 10.0
	obj.Properties["velocityY"] = 5.0

	err := routine.Execute(obj, 1.0)
	if err != nil {
		t.Fatalf("Execution failed: %v", err)
	}

	if obj.Position.X != 0.0 || obj.Position.Y != 0.0 || obj.Position.Z != 0.0 {
		t.Errorf("Expected position to remain unchanged, got %+v", obj.Position)
	}
	vx, _ := obj.Properties["velocityX"].(float64)
	vy, _ := obj.Properties["velocityY"].(float64)
	if vx != 10.0 || vy != 5.0 {
		t.Errorf("Expected velocity properties to remain configured, got velocityX=%v velocityY=%v", vx, vy)
	}
}
