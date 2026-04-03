package runtime

import (
	"testing"

	"github.com/digital-michael/space_sim/internal/server/pool/group"
	"github.com/google/uuid"
)

func TestInitializeGroupAndGetGroupState(t *testing.T) {
	definitions := group.NewPool()
	objectA := uuid.New()
	objectB := uuid.New()
	groupID := uuid.New()

	if err := definitions.CreateObject(objectA, "sphere", nil); err != nil {
		t.Fatalf("CreateObject A failed: %v", err)
	}
	if err := definitions.CreateObject(objectB, "cube", nil); err != nil {
		t.Fatalf("CreateObject B failed: %v", err)
	}
	if err := definitions.CreateGroup(groupID, "test", nil, nil); err != nil {
		t.Fatalf("CreateGroup failed: %v", err)
	}
	if err := definitions.AddGroupMember(groupID, objectA); err != nil {
		t.Fatalf("AddGroupMember A failed: %v", err)
	}
	if err := definitions.AddGroupMember(groupID, objectB); err != nil {
		t.Fatalf("AddGroupMember B failed: %v", err)
	}

	environment := NewRuntimeEnvironment(definitions)
	if err := environment.InitializeObject(objectA, func(index int) Vector3 {
		return Vector3{X: 10, Y: 0, Z: 0}
	}, Vector3{X: 2, Y: 0, Z: 0}); err != nil {
		t.Fatalf("InitializeObject A failed: %v", err)
	}
	if err := environment.InitializeObject(objectB, func(index int) Vector3 {
		return Vector3{X: 30, Y: 0, Z: 0}
	}, Vector3{X: 6, Y: 0, Z: 0}); err != nil {
		t.Fatalf("InitializeObject B failed: %v", err)
	}

	if err := environment.InitializeGroup(groupID); err != nil {
		t.Fatalf("InitializeGroup failed: %v", err)
	}

	state, err := environment.GetGroupState(groupID)
	if err != nil {
		t.Fatalf("GetGroupState failed: %v", err)
	}

	if state.MemberCount != 2 {
		t.Fatalf("expected member count 2, got %d", state.MemberCount)
	}
	if state.Center != (Vector3{X: 20, Y: 0, Z: 0}) {
		t.Fatalf("unexpected center: %+v", state.Center)
	}
	if state.AvgVelocity != (Vector3{X: 4, Y: 0, Z: 0}) {
		t.Fatalf("unexpected average velocity: %+v", state.AvgVelocity)
	}
	if state.BoundingMin != (Vector3{X: 10, Y: 0, Z: 0}) {
		t.Fatalf("unexpected bounding min: %+v", state.BoundingMin)
	}
	if state.BoundingMax != (Vector3{X: 30, Y: 0, Z: 0}) {
		t.Fatalf("unexpected bounding max: %+v", state.BoundingMax)
	}
}

func TestUpdateObjectStateInvalidatesGroupCache(t *testing.T) {
	definitions := group.NewPool()
	objectID := uuid.New()
	groupID := uuid.New()

	if err := definitions.CreateObject(objectID, "sphere", nil); err != nil {
		t.Fatalf("CreateObject failed: %v", err)
	}
	if err := definitions.CreateGroup(groupID, "test", nil, nil); err != nil {
		t.Fatalf("CreateGroup failed: %v", err)
	}
	if err := definitions.AddGroupMember(groupID, objectID); err != nil {
		t.Fatalf("AddGroupMember failed: %v", err)
	}

	environment := NewRuntimeEnvironment(definitions)
	if err := environment.InitializeObject(objectID, func(index int) Vector3 {
		return Vector3{X: 1, Y: 0, Z: 0}
	}, Vector3{}); err != nil {
		t.Fatalf("InitializeObject failed: %v", err)
	}
	if err := environment.InitializeGroup(groupID); err != nil {
		t.Fatalf("InitializeGroup failed: %v", err)
	}

	if _, err := environment.GetGroupState(groupID); err != nil {
		t.Fatalf("initial GetGroupState failed: %v", err)
	}
	if !environment.groups[groupID].CachedValid {
		t.Fatal("expected group cache to be valid after first get")
	}

	if err := environment.UpdateObjectState(objectID, func(state *ObjectState) {
		state.Position = Vector3{X: 9, Y: 0, Z: 0}
	}); err != nil {
		t.Fatalf("UpdateObjectState failed: %v", err)
	}
	if environment.groups[groupID].CachedValid {
		t.Fatal("expected group cache to be invalidated after object update")
	}

	updatedState, err := environment.GetGroupState(groupID)
	if err != nil {
		t.Fatalf("second GetGroupState failed: %v", err)
	}
	if updatedState.Center != (Vector3{X: 9, Y: 0, Z: 0}) {
		t.Fatalf("expected updated center, got %+v", updatedState.Center)
	}
}

func TestInitializeGroupValidation(t *testing.T) {
	definitions := group.NewPool()
	groupID := uuid.New()
	if err := definitions.CreateGroup(groupID, "test", nil, nil); err != nil {
		t.Fatalf("CreateGroup failed: %v", err)
	}

	environment := NewRuntimeEnvironment(definitions)
	if err := environment.InitializeGroup(groupID); err != nil {
		t.Fatalf("InitializeGroup failed: %v", err)
	}
	if err := environment.InitializeGroup(groupID); err == nil {
		t.Fatal("expected duplicate InitializeGroup to fail")
	}
	if err := environment.InitializeGroup(uuid.New()); err == nil {
		t.Fatal("expected InitializeGroup to fail for missing definition")
	}
}

func TestRecursiveGroupAggregationAndCacheInvalidation(t *testing.T) {
	definitions := group.NewPool()
	objectID := uuid.New()
	childGroupID := uuid.New()
	parentGroupID := uuid.New()

	if err := definitions.CreateObject(objectID, "sphere", nil); err != nil {
		t.Fatalf("CreateObject failed: %v", err)
	}
	if err := definitions.CreateGroup(parentGroupID, "parent", nil, nil); err != nil {
		t.Fatalf("CreateGroup parent failed: %v", err)
	}
	if err := definitions.CreateGroup(childGroupID, "child", &parentGroupID, nil); err != nil {
		t.Fatalf("CreateGroup child failed: %v", err)
	}
	if err := definitions.AddGroupMember(childGroupID, objectID); err != nil {
		t.Fatalf("AddGroupMember object->child failed: %v", err)
	}

	environment := NewRuntimeEnvironment(definitions)
	if err := environment.InitializeObject(objectID, func(index int) Vector3 {
		return Vector3{X: 2, Y: 0, Z: 0}
	}, Vector3{X: 1, Y: 0, Z: 0}); err != nil {
		t.Fatalf("InitializeObject failed: %v", err)
	}
	if err := environment.InitializeGroup(childGroupID); err != nil {
		t.Fatalf("InitializeGroup child failed: %v", err)
	}
	if err := environment.InitializeGroup(parentGroupID); err != nil {
		t.Fatalf("InitializeGroup parent failed: %v", err)
	}

	state, err := environment.GetGroupState(parentGroupID)
	if err != nil {
		t.Fatalf("GetGroupState parent failed: %v", err)
	}
	if state.MemberCount != 1 || state.Center != (Vector3{X: 2, Y: 0, Z: 0}) {
		t.Fatalf("unexpected parent aggregate: %+v", state)
	}
	if !environment.groups[childGroupID].CachedValid || !environment.groups[parentGroupID].CachedValid {
		t.Fatal("expected child and parent caches to be valid after read")
	}

	if err := environment.UpdateObjectState(objectID, func(state *ObjectState) {
		state.Position = Vector3{X: 8, Y: 0, Z: 0}
	}); err != nil {
		t.Fatalf("UpdateObjectState failed: %v", err)
	}

	if environment.groups[childGroupID].CachedValid {
		t.Fatal("expected child group cache invalidation")
	}
	if environment.groups[parentGroupID].CachedValid {
		t.Fatal("expected parent group cache invalidation")
	}

	updatedState, err := environment.GetGroupState(parentGroupID)
	if err != nil {
		t.Fatalf("GetGroupState parent after update failed: %v", err)
	}
	if updatedState.Center != (Vector3{X: 8, Y: 0, Z: 0}) {
		t.Fatalf("expected updated parent center, got %+v", updatedState.Center)
	}
}
