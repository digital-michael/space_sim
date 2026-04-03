package runtime

import (
	"testing"

	"github.com/digital-michael/space_sim/internal/server/pool/group"
	"github.com/google/uuid"
)

func TestListObjectStates(t *testing.T) {
	definitions := group.NewPool()
	objectA := uuid.New()
	objectB := uuid.New()
	objectC := uuid.New()

	if err := definitions.CreateObject(objectA, "sphere", nil); err != nil {
		t.Fatalf("CreateObject A failed: %v", err)
	}
	if err := definitions.CreateObject(objectB, "cube", nil); err != nil {
		t.Fatalf("CreateObject B failed: %v", err)
	}
	if err := definitions.CreateObject(objectC, "cone", nil); err != nil {
		t.Fatalf("CreateObject C failed: %v", err)
	}

	environment := NewRuntimeEnvironment(definitions)
	if err := environment.InitializeObject(objectA, OriginPosition(), Vector3{}); err != nil {
		t.Fatalf("InitializeObject A failed: %v", err)
	}
	if err := environment.InitializeObject(objectB, OriginPosition(), Vector3{}); err != nil {
		t.Fatalf("InitializeObject B failed: %v", err)
	}

	ids := environment.ListObjectStates()
	if len(ids) != 2 {
		t.Fatalf("expected 2 object states, got %d", len(ids))
	}

	idMap := make(map[uuid.UUID]struct{})
	for _, id := range ids {
		idMap[id] = struct{}{}
	}
	if _, hasA := idMap[objectA]; !hasA {
		t.Fatal("expected objectA in list")
	}
	if _, hasB := idMap[objectB]; !hasB {
		t.Fatal("expected objectB in list")
	}
	if _, hasC := idMap[objectC]; hasC {
		t.Fatal("expected objectC not in list (not initialized)")
	}
}

func TestListGroupStates(t *testing.T) {
	definitions := group.NewPool()
	groupA := uuid.New()
	groupB := uuid.New()
	groupC := uuid.New()

	if err := definitions.CreateGroup(groupA, "a", nil, nil); err != nil {
		t.Fatalf("CreateGroup A failed: %v", err)
	}
	if err := definitions.CreateGroup(groupB, "b", nil, nil); err != nil {
		t.Fatalf("CreateGroup B failed: %v", err)
	}
	if err := definitions.CreateGroup(groupC, "c", nil, nil); err != nil {
		t.Fatalf("CreateGroup C failed: %v", err)
	}

	environment := NewRuntimeEnvironment(definitions)
	if err := environment.InitializeGroup(groupA); err != nil {
		t.Fatalf("InitializeGroup A failed: %v", err)
	}
	if err := environment.InitializeGroup(groupB); err != nil {
		t.Fatalf("InitializeGroup B failed: %v", err)
	}

	ids := environment.ListGroupStates()
	if len(ids) != 2 {
		t.Fatalf("expected 2 group states, got %d", len(ids))
	}

	idMap := make(map[uuid.UUID]struct{})
	for _, id := range ids {
		idMap[id] = struct{}{}
	}
	if _, hasA := idMap[groupA]; !hasA {
		t.Fatal("expected groupA in list")
	}
	if _, hasB := idMap[groupB]; !hasB {
		t.Fatal("expected groupB in list")
	}
	if _, hasC := idMap[groupC]; hasC {
		t.Fatal("expected groupC not in list (not initialized)")
	}
}

func TestGetAggregatesByIDs(t *testing.T) {
	definitions := group.NewPool()
	objectA := uuid.New()
	objectB := uuid.New()
	groupID := uuid.New()

	if err := definitions.CreateObject(objectA, "a", nil); err != nil {
		t.Fatalf("CreateObject A failed: %v", err)
	}
	if err := definitions.CreateObject(objectB, "b", nil); err != nil {
		t.Fatalf("CreateObject B failed: %v", err)
	}
	if err := definitions.CreateGroup(groupID, "group", nil, nil); err != nil {
		t.Fatalf("CreateGroup failed: %v", err)
	}
	if err := definitions.AddGroupMember(groupID, objectA); err != nil {
		t.Fatalf("AddGroupMember A failed: %v", err)
	}
	if err := definitions.AddGroupMember(groupID, objectB); err != nil {
		t.Fatalf("AddGroupMember B failed: %v", err)
	}

	environment := NewRuntimeEnvironment(definitions)
	if err := environment.InitializeObject(objectA, func(i int) Vector3 {
		return Vector3{X: 5, Y: 0, Z: 0}
	}, Vector3{X: 1, Y: 0, Z: 0}); err != nil {
		t.Fatalf("InitializeObject A failed: %v", err)
	}
	if err := environment.InitializeObject(objectB, func(i int) Vector3 {
		return Vector3{X: 15, Y: 0, Z: 0}
	}, Vector3{X: 3, Y: 0, Z: 0}); err != nil {
		t.Fatalf("InitializeObject B failed: %v", err)
	}
	if err := environment.InitializeGroup(groupID); err != nil {
		t.Fatalf("InitializeGroup failed: %v", err)
	}

	aggs := environment.GetAggregatesByIDs([]uuid.UUID{groupID})
	if len(aggs) != 1 {
		t.Fatalf("expected 1 aggregate, got %d", len(aggs))
	}
	if aggs[0].ID != groupID {
		t.Fatalf("expected aggregate ID %s, got %s", groupID, aggs[0].ID)
	}
	if aggs[0].MemberCount != 2 {
		t.Fatalf("expected member count 2, got %d", aggs[0].MemberCount)
	}
	if aggs[0].Center != (Vector3{X: 10, Y: 0, Z: 0}) {
		t.Fatalf("expected center (10,0,0), got %+v", aggs[0].Center)
	}

	invalidID := uuid.New()
	aggs = environment.GetAggregatesByIDs([]uuid.UUID{invalidID})
	if len(aggs) != 0 {
		t.Fatalf("expected 0 aggregates for invalid ID, got %d", len(aggs))
	}
}

func TestGetObjectStatesByIDs(t *testing.T) {
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
	if err := environment.InitializeObject(objectA, func(i int) Vector3 {
		return Vector3{X: 1, Y: 2, Z: 3}
	}, Vector3{X: 10, Y: 20, Z: 30}); err != nil {
		t.Fatalf("InitializeObject A failed: %v", err)
	}
	if err := environment.InitializeObject(objectB, func(i int) Vector3 {
		return Vector3{X: 4, Y: 5, Z: 6}
	}, Vector3{X: 40, Y: 50, Z: 60}); err != nil {
		t.Fatalf("InitializeObject B failed: %v", err)
	}

	states := environment.GetObjectStatesByIDs([]uuid.UUID{objectA, objectB})
	if len(states) != 2 {
		t.Fatalf("expected 2 states, got %d", len(states))
	}

	idMap := make(map[uuid.UUID]*ObjectState)
	for _, state := range states {
		idMap[state.ID] = state
	}

	if stateA, ok := idMap[objectA]; !ok || stateA.Position != (Vector3{X: 1, Y: 2, Z: 3}) {
		t.Fatalf("unexpected state for A: %+v", stateA)
	}
	if stateB, ok := idMap[objectB]; !ok || stateB.Position != (Vector3{X: 4, Y: 5, Z: 6}) {
		t.Fatalf("unexpected state for B: %+v", stateB)
	}

	invalidID := uuid.New()
	states = environment.GetObjectStatesByIDs([]uuid.UUID{invalidID})
	if len(states) != 0 {
		t.Fatalf("expected 0 states for invalid ID, got %d", len(states))
	}
}
