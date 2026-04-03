package group

import (
	"sync"
	"testing"

	basepool "github.com/digital-michael/space_sim/internal/server/pool"
	"github.com/google/uuid"
)

func TestNewPoolType(t *testing.T) {
	groupPool := NewPool()
	if groupPool.GetType() != basepool.PoolTypeGroup {
		t.Fatalf("expected pool type %v", basepool.PoolTypeGroup)
	}
}

func TestPoolObjectCRUD(t *testing.T) {
	groupPool := NewPool()
	id := uuid.New()

	if err := groupPool.CreateObject(id, "sphere", map[string]interface{}{"mass": 12.5}); err != nil {
		t.Fatalf("CreateObject failed: %v", err)
	}

	definition, err := groupPool.GetObject(id)
	if err != nil {
		t.Fatalf("GetObject failed: %v", err)
	}
	if definition.Type != "sphere" {
		t.Fatalf("expected type sphere, got %s", definition.Type)
	}

	if err := groupPool.UpdateObject(id, map[string]interface{}{"mass": 13.0}); err != nil {
		t.Fatalf("UpdateObject failed: %v", err)
	}

	updated, err := groupPool.GetObject(id)
	if err != nil {
		t.Fatalf("GetObject after update failed: %v", err)
	}
	if updated.Properties["mass"] != 13.0 {
		t.Fatalf("expected updated mass 13.0, got %v", updated.Properties["mass"])
	}

	if err := groupPool.DeleteObject(id); err != nil {
		t.Fatalf("DeleteObject failed: %v", err)
	}
	if _, err := groupPool.GetObject(id); err == nil {
		t.Fatal("expected object not found after delete")
	}
}

func TestPoolGroupMembershipLocking(t *testing.T) {
	groupPool := NewPool()
	groupID := uuid.New()
	objectID := uuid.New()

	if err := groupPool.CreateObject(objectID, "cube", nil); err != nil {
		t.Fatalf("CreateObject failed: %v", err)
	}
	if err := groupPool.CreateGroup(groupID, "Root", nil, nil); err != nil {
		t.Fatalf("CreateGroup failed: %v", err)
	}

	if err := groupPool.AddGroupMember(groupID, objectID); err != nil {
		t.Fatalf("AddGroupMember failed: %v", err)
	}

	groupDefinition, err := groupPool.GetGroup(groupID)
	if err != nil {
		t.Fatalf("GetGroup failed: %v", err)
	}
	if len(groupDefinition.Members) != 1 || groupDefinition.Members[0] != objectID {
		t.Fatalf("expected one member %s", objectID)
	}

	if err := groupPool.LockGroup(groupID); err != nil {
		t.Fatalf("LockGroup failed: %v", err)
	}
	if err := groupPool.RemoveGroupMember(groupID, objectID); err == nil {
		t.Fatal("expected remove member to fail while locked")
	}

	if err := groupPool.UnlockGroup(groupID); err != nil {
		t.Fatalf("UnlockGroup failed: %v", err)
	}
	if err := groupPool.RemoveGroupMember(groupID, objectID); err != nil {
		t.Fatalf("RemoveGroupMember failed: %v", err)
	}
}

func TestPoolConcurrentCreateObject(t *testing.T) {
	groupPool := NewPool()

	const workers = 20
	const perWorker = 10

	var wg sync.WaitGroup
	for worker := 0; worker < workers; worker++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for index := 0; index < perWorker; index++ {
				id := uuid.New()
				if err := groupPool.CreateObject(id, "particle", nil); err != nil {
					t.Errorf("CreateObject failed: %v", err)
				}
			}
		}()
	}
	wg.Wait()

	ids := groupPool.ListObjects()
	if len(ids) != workers*perWorker {
		t.Fatalf("expected %d objects, got %d", workers*perWorker, len(ids))
	}
}

func TestObjectPoolInterfaceMethods(t *testing.T) {
	groupPool := NewPool()
	id := uuid.New()

	if err := groupPool.Create(id, "planet", nil); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	item, err := groupPool.Get(id)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	objectDefinition, ok := item.(*ObjectDefinition)
	if !ok {
		t.Fatalf("expected *ObjectDefinition, got %T", item)
	}
	if objectDefinition.ID != id {
		t.Fatalf("expected id %s, got %s", id, objectDefinition.ID)
	}

	if err := groupPool.Update(id, map[string]interface{}{"name": "earth"}); err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	list := groupPool.List()
	if len(list) != 1 || list[0] != id {
		t.Fatalf("expected list to contain %s", id)
	}

	if err := groupPool.Delete(id); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}
}

func TestPoolPreventsHierarchyCycle(t *testing.T) {
	groupPool := NewPool()
	groupA := uuid.New()
	groupB := uuid.New()

	if err := groupPool.CreateGroup(groupA, "A", nil, nil); err != nil {
		t.Fatalf("CreateGroup A failed: %v", err)
	}
	if err := groupPool.CreateGroup(groupB, "B", nil, nil); err != nil {
		t.Fatalf("CreateGroup B failed: %v", err)
	}

	if err := groupPool.AddGroupMember(groupA, groupB); err != nil {
		t.Fatalf("expected A->B to succeed, got %v", err)
	}

	err := groupPool.AddGroupMember(groupB, groupA)
	if err == nil {
		t.Fatal("expected B->A to fail with cycle")
	}
	if err != ErrHierarchyCycle {
		t.Fatalf("expected ErrHierarchyCycle, got %v", err)
	}
}

func TestCreateGroupWithParentAddsMemberRelation(t *testing.T) {
	groupPool := NewPool()
	root := uuid.New()
	child := uuid.New()

	if err := groupPool.CreateGroup(root, "Root", nil, nil); err != nil {
		t.Fatalf("CreateGroup root failed: %v", err)
	}
	if err := groupPool.CreateGroup(child, "Child", &root, nil); err != nil {
		t.Fatalf("CreateGroup child with parent failed: %v", err)
	}

	rootDef, err := groupPool.GetGroup(root)
	if err != nil {
		t.Fatalf("GetGroup root failed: %v", err)
	}
	if len(rootDef.Members) != 1 || rootDef.Members[0] != child {
		t.Fatalf("expected root to contain child member %s", child)
	}
}

func TestMembershipReverseIndexLifecycle(t *testing.T) {
	groupPool := NewPool()
	groupA := uuid.New()
	groupB := uuid.New()
	member := uuid.New()

	if err := groupPool.CreateGroup(groupA, "A", nil, nil); err != nil {
		t.Fatalf("CreateGroup A failed: %v", err)
	}
	if err := groupPool.CreateGroup(groupB, "B", nil, nil); err != nil {
		t.Fatalf("CreateGroup B failed: %v", err)
	}
	if err := groupPool.CreateObject(member, "asteroid", nil); err != nil {
		t.Fatalf("CreateObject member failed: %v", err)
	}

	if err := groupPool.AddGroupMember(groupA, member); err != nil {
		t.Fatalf("AddGroupMember A failed: %v", err)
	}
	if err := groupPool.AddGroupMember(groupB, member); err != nil {
		t.Fatalf("AddGroupMember B failed: %v", err)
	}

	groups := groupPool.GroupsForMember(member)
	if len(groups) != 2 {
		t.Fatalf("expected member in 2 groups, got %d", len(groups))
	}

	if err := groupPool.RemoveGroupMember(groupA, member); err != nil {
		t.Fatalf("RemoveGroupMember A failed: %v", err)
	}
	groups = groupPool.GroupsForMember(member)
	if len(groups) != 1 {
		t.Fatalf("expected member in 1 group after removal, got %d", len(groups))
	}

	if err := groupPool.RemoveGroupMember(groupB, member); err != nil {
		t.Fatalf("RemoveGroupMember B failed: %v", err)
	}

	if err := groupPool.Delete(groupB); err != nil {
		t.Fatalf("Delete group B failed: %v", err)
	}
	groups = groupPool.GroupsForMember(member)
	if len(groups) != 0 {
		t.Fatalf("expected member index to be empty after deleting parent group, got %d", len(groups))
	}
}

func TestCreateGroupHierarchyDepthLimit(t *testing.T) {
	groupPool := NewPool()

	rootID := uuid.New()
	if err := groupPool.CreateGroup(rootID, "Level-1", nil, nil); err != nil {
		t.Fatalf("failed creating root group: %v", err)
	}

	parentID := rootID
	for depth := 2; depth <= maxGroupHierarchyDepth; depth++ {
		groupID := uuid.New()
		if err := groupPool.CreateGroup(groupID, "Level", &parentID, nil); err != nil {
			t.Fatalf("failed creating depth %d group: %v", depth, err)
		}
		parentID = groupID
	}

	tooDeepID := uuid.New()
	err := groupPool.CreateGroup(tooDeepID, "TooDeep", &parentID, nil)
	if err == nil {
		t.Fatal("expected depth limit error when creating group beyond max depth")
	}
	if err != ErrHierarchyDepthExceeded {
		t.Fatalf("expected ErrHierarchyDepthExceeded, got %v", err)
	}
}

func TestAddGroupMemberHierarchyDepthLimit(t *testing.T) {
	groupPool := NewPool()

	rootID := uuid.New()
	if err := groupPool.CreateGroup(rootID, "Level-1", nil, nil); err != nil {
		t.Fatalf("failed creating root group: %v", err)
	}

	deepestParent := rootID
	for depth := 2; depth <= maxGroupHierarchyDepth; depth++ {
		groupID := uuid.New()
		if err := groupPool.CreateGroup(groupID, "Level", &deepestParent, nil); err != nil {
			t.Fatalf("failed creating depth %d group: %v", depth, err)
		}
		deepestParent = groupID
	}

	detachedChild := uuid.New()
	if err := groupPool.CreateGroup(detachedChild, "Detached", nil, nil); err != nil {
		t.Fatalf("failed creating detached child: %v", err)
	}

	err := groupPool.AddGroupMember(deepestParent, detachedChild)
	if err == nil {
		t.Fatal("expected depth limit error when attaching child under max-depth parent")
	}
	if err != ErrHierarchyDepthExceeded {
		t.Fatalf("expected ErrHierarchyDepthExceeded, got %v", err)
	}
}

func TestDeleteGroupFailsWhenNonEmpty(t *testing.T) {
	groupPool := NewPool()
	groupID := uuid.New()
	memberID := uuid.New()

	if err := groupPool.CreateGroup(groupID, "Parent", nil, nil); err != nil {
		t.Fatalf("CreateGroup failed: %v", err)
	}
	if err := groupPool.CreateObject(memberID, "particle", nil); err != nil {
		t.Fatalf("CreateObject failed: %v", err)
	}
	if err := groupPool.AddGroupMember(groupID, memberID); err != nil {
		t.Fatalf("AddGroupMember failed: %v", err)
	}

	err := groupPool.Delete(groupID)
	if err == nil {
		t.Fatal("expected delete to fail for non-empty group")
	}
	if err != ErrGroupNotEmpty {
		t.Fatalf("expected ErrGroupNotEmpty, got %v", err)
	}
}

func TestAddGroupMemberSetsChildParentID(t *testing.T) {
	groupPool := NewPool()
	parentID := uuid.New()
	childID := uuid.New()

	if err := groupPool.CreateGroup(parentID, "Parent", nil, nil); err != nil {
		t.Fatalf("CreateGroup parent failed: %v", err)
	}
	if err := groupPool.CreateGroup(childID, "Child", nil, nil); err != nil {
		t.Fatalf("CreateGroup child failed: %v", err)
	}

	if err := groupPool.AddGroupMember(parentID, childID); err != nil {
		t.Fatalf("AddGroupMember failed: %v", err)
	}

	child, err := groupPool.GetGroup(childID)
	if err != nil {
		t.Fatalf("GetGroup child failed: %v", err)
	}
	if child.ParentID == nil || *child.ParentID != parentID {
		t.Fatalf("expected child parent to be %s, got %+v", parentID, child.ParentID)
	}
}

func TestRemoveGroupMemberClearsChildParentID(t *testing.T) {
	groupPool := NewPool()
	parentID := uuid.New()
	childID := uuid.New()

	if err := groupPool.CreateGroup(parentID, "Parent", nil, nil); err != nil {
		t.Fatalf("CreateGroup parent failed: %v", err)
	}
	if err := groupPool.CreateGroup(childID, "Child", nil, nil); err != nil {
		t.Fatalf("CreateGroup child failed: %v", err)
	}
	if err := groupPool.AddGroupMember(parentID, childID); err != nil {
		t.Fatalf("AddGroupMember failed: %v", err)
	}

	if err := groupPool.RemoveGroupMember(parentID, childID); err != nil {
		t.Fatalf("RemoveGroupMember failed: %v", err)
	}

	child, err := groupPool.GetGroup(childID)
	if err != nil {
		t.Fatalf("GetGroup child failed: %v", err)
	}
	if child.ParentID != nil {
		t.Fatalf("expected child parent to be nil after unlink, got %s", *child.ParentID)
	}
}

func TestDeleteObjectRemovesGroupMembershipReferences(t *testing.T) {
	groupPool := NewPool()
	groupID := uuid.New()
	objectID := uuid.New()

	if err := groupPool.CreateGroup(groupID, "Group", nil, nil); err != nil {
		t.Fatalf("CreateGroup failed: %v", err)
	}
	if err := groupPool.CreateObject(objectID, "particle", nil); err != nil {
		t.Fatalf("CreateObject failed: %v", err)
	}
	if err := groupPool.AddGroupMember(groupID, objectID); err != nil {
		t.Fatalf("AddGroupMember failed: %v", err)
	}

	if err := groupPool.DeleteObject(objectID); err != nil {
		t.Fatalf("DeleteObject failed: %v", err)
	}

	group, err := groupPool.GetGroup(groupID)
	if err != nil {
		t.Fatalf("GetGroup failed: %v", err)
	}
	for _, memberID := range group.Members {
		if memberID == objectID {
			t.Fatalf("expected deleted object %s to be removed from group members", objectID)
		}
	}

	if groups := groupPool.GroupsForMember(objectID); len(groups) != 0 {
		t.Fatalf("expected reverse index for deleted object to be empty, got %d entries", len(groups))
	}
}

func TestDeleteObjectViaGenericDeleteRemovesReferences(t *testing.T) {
	groupPool := NewPool()
	groupID := uuid.New()
	objectID := uuid.New()

	if err := groupPool.CreateGroup(groupID, "Group", nil, nil); err != nil {
		t.Fatalf("CreateGroup failed: %v", err)
	}
	if err := groupPool.CreateObject(objectID, "particle", nil); err != nil {
		t.Fatalf("CreateObject failed: %v", err)
	}
	if err := groupPool.AddGroupMember(groupID, objectID); err != nil {
		t.Fatalf("AddGroupMember failed: %v", err)
	}

	if err := groupPool.Delete(objectID); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	group, err := groupPool.GetGroup(groupID)
	if err != nil {
		t.Fatalf("GetGroup failed: %v", err)
	}
	for _, memberID := range group.Members {
		if memberID == objectID {
			t.Fatalf("expected deleted object %s to be removed from group members", objectID)
		}
	}
}

func TestUpdateGroupProperties(t *testing.T) {
	groupPool := NewPool()
	groupID := uuid.New()

	if err := groupPool.CreateGroup(groupID, "Group", nil, map[string]interface{}{"color": "blue"}); err != nil {
		t.Fatalf("CreateGroup failed: %v", err)
	}

	err := groupPool.UpdateGroup(groupID, map[string]interface{}{"color": "red", "mass": 42.0})
	if err != nil {
		t.Fatalf("UpdateGroup failed: %v", err)
	}

	group, err := groupPool.GetGroup(groupID)
	if err != nil {
		t.Fatalf("GetGroup failed: %v", err)
	}
	if group.Properties["color"] != "red" {
		t.Fatalf("expected updated color red, got %v", group.Properties["color"])
	}
	if group.Properties["mass"] != 42.0 {
		t.Fatalf("expected mass 42.0, got %v", group.Properties["mass"])
	}
}

func TestListGroups(t *testing.T) {
	groupPool := NewPool()
	groupA := uuid.New()
	groupB := uuid.New()

	if err := groupPool.CreateGroup(groupA, "A", nil, nil); err != nil {
		t.Fatalf("CreateGroup A failed: %v", err)
	}
	if err := groupPool.CreateGroup(groupB, "B", nil, nil); err != nil {
		t.Fatalf("CreateGroup B failed: %v", err)
	}

	groups := groupPool.ListGroups()
	if len(groups) != 2 {
		t.Fatalf("expected 2 groups, got %d", len(groups))
	}

	foundA := false
	foundB := false
	for _, id := range groups {
		if id == groupA {
			foundA = true
		}
		if id == groupB {
			foundB = true
		}
	}
	if !foundA || !foundB {
		t.Fatalf("expected group list to contain both %s and %s", groupA, groupB)
	}
}
