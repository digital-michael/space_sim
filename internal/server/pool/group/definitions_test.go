package group

import (
"testing"
"time"

"github.com/google/uuid"
)

func TestObjectDefinitionValidate(t *testing.T) {
id := uuid.New()
valid := &ObjectDefinition{ID: id, Type: "sphere"}
if err := valid.Validate(); err != nil {
t.Fatalf("expected valid definition, got error: %v", err)
}

invalidID := &ObjectDefinition{Type: "sphere"}
if err := invalidID.Validate(); err == nil {
t.Fatal("expected nil ID validation error")
}

invalidType := &ObjectDefinition{ID: id}
if err := invalidType.Validate(); err == nil {
t.Fatal("expected empty type validation error")
}
}

func TestObjectDefinitionClone(t *testing.T) {
id := uuid.New()
createdAt := time.Now().Add(-time.Minute)
updatedAt := time.Now()

original := &ObjectDefinition{
ID:         id,
Type:       "cube",
Properties: map[string]interface{}{"color": "red"},
CreatedAt:  createdAt,
UpdatedAt:  updatedAt,
}

clone := original.Clone()
if clone == original {
t.Fatal("expected clone to be a distinct pointer")
}
if clone.ID != original.ID || clone.Type != original.Type {
t.Fatal("expected cloned core fields to match original")
}
if clone.Properties["color"] != "red" {
t.Fatal("expected cloned properties to match original")
}

clone.Properties["color"] = "blue"
if original.Properties["color"] != "red" {
t.Fatal("expected property map to be copied")
}
}

func TestGroupDefinitionValidate(t *testing.T) {
id := uuid.New()
valid := &GroupDefinition{ID: id, Name: "Planets"}
if err := valid.Validate(); err != nil {
t.Fatalf("expected valid group, got error: %v", err)
}

invalidID := &GroupDefinition{Name: "Planets"}
if err := invalidID.Validate(); err == nil {
t.Fatal("expected nil ID validation error")
}

invalidName := &GroupDefinition{ID: id}
if err := invalidName.Validate(); err == nil {
t.Fatal("expected empty name validation error")
}
}

func TestGroupDefinitionMemberLifecycle(t *testing.T) {
group := &GroupDefinition{ID: uuid.New(), Name: "Group A"}
member := uuid.New()

if group.HasMember(member) {
t.Fatal("expected no member before add")
}

if err := group.AddMember(member); err != nil {
t.Fatalf("expected add to succeed, got %v", err)
}
if !group.HasMember(member) {
t.Fatal("expected member to exist after add")
}

if err := group.AddMember(member); err == nil {
t.Fatal("expected duplicate add error")
}

if err := group.RemoveMember(member); err != nil {
t.Fatalf("expected remove to succeed, got %v", err)
}
if group.HasMember(member) {
t.Fatal("expected member to be removed")
}

if err := group.RemoveMember(member); err == nil {
t.Fatal("expected remove missing member error")
}
}

func TestGroupDefinitionLocking(t *testing.T) {
group := &GroupDefinition{ID: uuid.New(), Name: "Group A"}
member := uuid.New()

group.Lock()
if !group.Locked {
t.Fatal("expected group to be locked")
}
if err := group.AddMember(member); err == nil {
t.Fatal("expected add while locked to fail")
}

group.Unlock()
if group.Locked {
t.Fatal("expected group to be unlocked")
}
if err := group.AddMember(member); err != nil {
t.Fatalf("expected add after unlock to succeed, got %v", err)
}
}
