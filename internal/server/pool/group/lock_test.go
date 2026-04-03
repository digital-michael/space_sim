package group

import (
	"testing"

	"github.com/google/uuid"
)

func TestLockUnlockGroupHierarchy(t *testing.T) {
	groupPool := NewPool()
	rootID := uuid.New()
	childID := uuid.New()
	grandchildID := uuid.New()
	objectID := uuid.New()

	if err := groupPool.CreateGroup(rootID, "Root", nil, nil); err != nil {
		t.Fatalf("CreateGroup root failed: %v", err)
	}
	if err := groupPool.CreateGroup(childID, "Child", &rootID, nil); err != nil {
		t.Fatalf("CreateGroup child failed: %v", err)
	}
	if err := groupPool.CreateGroup(grandchildID, "Grandchild", &childID, nil); err != nil {
		t.Fatalf("CreateGroup grandchild failed: %v", err)
	}
	if err := groupPool.CreateObject(objectID, "particle", nil); err != nil {
		t.Fatalf("CreateObject failed: %v", err)
	}

	if err := groupPool.LockGroupHierarchy(rootID); err != nil {
		t.Fatalf("LockGroupHierarchy failed: %v", err)
	}

	root, err := groupPool.GetGroup(rootID)
	if err != nil {
		t.Fatalf("GetGroup root failed: %v", err)
	}
	child, err := groupPool.GetGroup(childID)
	if err != nil {
		t.Fatalf("GetGroup child failed: %v", err)
	}
	grandchild, err := groupPool.GetGroup(grandchildID)
	if err != nil {
		t.Fatalf("GetGroup grandchild failed: %v", err)
	}
	if !root.Locked || !child.Locked || !grandchild.Locked {
		t.Fatal("expected root, child, and grandchild groups to all be locked")
	}

	if err := groupPool.AddGroupMember(childID, objectID); err == nil {
		t.Fatal("expected AddGroupMember to fail while child is locked")
	}

	if err := groupPool.UnlockGroupHierarchy(rootID); err != nil {
		t.Fatalf("UnlockGroupHierarchy failed: %v", err)
	}

	root, _ = groupPool.GetGroup(rootID)
	child, _ = groupPool.GetGroup(childID)
	grandchild, _ = groupPool.GetGroup(grandchildID)
	if root.Locked || child.Locked || grandchild.Locked {
		t.Fatal("expected root, child, and grandchild groups to all be unlocked")
	}

	if err := groupPool.AddGroupMember(childID, objectID); err != nil {
		t.Fatalf("expected AddGroupMember after unlock to succeed, got %v", err)
	}
}

func TestLockGroupHierarchyMissingGroup(t *testing.T) {
	groupPool := NewPool()
	missingID := uuid.New()

	if err := groupPool.LockGroupHierarchy(missingID); err != ErrGroupNotFound {
		t.Fatalf("expected ErrGroupNotFound, got %v", err)
	}
	if err := groupPool.UnlockGroupHierarchy(missingID); err != ErrGroupNotFound {
		t.Fatalf("expected ErrGroupNotFound, got %v", err)
	}
}
