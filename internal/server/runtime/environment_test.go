package runtime

import (
	"testing"

	"github.com/digital-michael/space_sim/internal/server/pool/group"
	"github.com/google/uuid"
)

func TestInitializeObjectAndGetState(t *testing.T) {
	definitions := group.NewPool()
	objectID := uuid.New()
	if err := definitions.CreateObject(objectID, "sphere", nil); err != nil {
		t.Fatalf("CreateObject in definitions failed: %v", err)
	}

	environment := NewRuntimeEnvironment(definitions)
	if err := environment.InitializeObject(objectID, OriginPosition(), Vector3{X: 1, Y: 2, Z: 3}); err != nil {
		t.Fatalf("InitializeObject failed: %v", err)
	}

	state, err := environment.GetObjectState(objectID)
	if err != nil {
		t.Fatalf("GetObjectState failed: %v", err)
	}
	if state.Velocity != (Vector3{X: 1, Y: 2, Z: 3}) {
		t.Fatalf("unexpected velocity: %+v", state.Velocity)
	}
}

func TestInitializeObjectRequiresDefinition(t *testing.T) {
	environment := NewRuntimeEnvironment(group.NewPool())
	missingID := uuid.New()

	if err := environment.InitializeObject(missingID, OriginPosition(), Vector3{}); err == nil {
		t.Fatal("expected initialization to fail for missing definition")
	}
}

func TestInitializeObjectDuplicateFails(t *testing.T) {
	definitions := group.NewPool()
	objectID := uuid.New()
	if err := definitions.CreateObject(objectID, "sphere", nil); err != nil {
		t.Fatalf("CreateObject in definitions failed: %v", err)
	}

	environment := NewRuntimeEnvironment(definitions)
	if err := environment.InitializeObject(objectID, OriginPosition(), Vector3{}); err != nil {
		t.Fatalf("first InitializeObject failed: %v", err)
	}
	if err := environment.InitializeObject(objectID, OriginPosition(), Vector3{}); err == nil {
		t.Fatal("expected duplicate initialization to fail")
	}
}

func TestUpdateAndRemoveObjectState(t *testing.T) {
	definitions := group.NewPool()
	objectID := uuid.New()
	if err := definitions.CreateObject(objectID, "sphere", nil); err != nil {
		t.Fatalf("CreateObject in definitions failed: %v", err)
	}

	environment := NewRuntimeEnvironment(definitions)
	if err := environment.InitializeObject(objectID, OriginPosition(), Vector3{}); err != nil {
		t.Fatalf("InitializeObject failed: %v", err)
	}

	err := environment.UpdateObjectState(objectID, func(state *ObjectState) {
		state.Position = Vector3{X: 10, Y: 20, Z: 30}
	})
	if err != nil {
		t.Fatalf("UpdateObjectState failed: %v", err)
	}

	state, err := environment.GetObjectState(objectID)
	if err != nil {
		t.Fatalf("GetObjectState failed: %v", err)
	}
	if state.Position != (Vector3{X: 10, Y: 20, Z: 30}) {
		t.Fatalf("unexpected position after update: %+v", state.Position)
	}

	if err := environment.RemoveObjectState(objectID); err != nil {
		t.Fatalf("RemoveObjectState failed: %v", err)
	}
	if _, err := environment.GetObjectState(objectID); err == nil {
		t.Fatal("expected GetObjectState to fail after remove")
	}
}

func TestInitializeObjectUsesSequentialPositionIndex(t *testing.T) {
	definitions := group.NewPool()
	objectA := uuid.New()
	objectB := uuid.New()
	if err := definitions.CreateObject(objectA, "a", nil); err != nil {
		t.Fatalf("CreateObject A failed: %v", err)
	}
	if err := definitions.CreateObject(objectB, "b", nil); err != nil {
		t.Fatalf("CreateObject B failed: %v", err)
	}

	environment := NewRuntimeEnvironment(definitions)
	grid := GridPosition(2, 5)
	if err := environment.InitializeObject(objectA, grid, Vector3{}); err != nil {
		t.Fatalf("InitializeObject A failed: %v", err)
	}
	if err := environment.InitializeObject(objectB, grid, Vector3{}); err != nil {
		t.Fatalf("InitializeObject B failed: %v", err)
	}

	stateA, _ := environment.GetObjectState(objectA)
	stateB, _ := environment.GetObjectState(objectB)
	if stateA.Position == stateB.Position {
		t.Fatalf("expected distinct positions from sequential index, both were %+v", stateA.Position)
	}
}
