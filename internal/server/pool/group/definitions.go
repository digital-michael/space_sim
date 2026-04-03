package group

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)









































































































type ObjectDefinition struct {
	ID         uuid.UUID              `json:"id"`
	Type       string                 `json:"type"`
	Properties map[string]interface{} `json:"properties"`
	CreatedAt  time.Time              `json:"created_at"`
	UpdatedAt  time.Time              `json:"updated_at"`
}

func (definition *ObjectDefinition) Validate() error {
	if definition.ID == uuid.Nil {
		return fmt.Errorf("object ID cannot be nil")
	}
	if definition.Type == "" {
		return fmt.Errorf("object type cannot be empty")
	}
	return nil
}

func (definition *ObjectDefinition) Clone() *ObjectDefinition {
	properties := make(map[string]interface{}, len(definition.Properties))
	for key, value := range definition.Properties {
		properties[key] = value
	}
	return &ObjectDefinition{
		ID:         definition.ID,
		Type:       definition.Type,
		Properties: properties,
		CreatedAt:  definition.CreatedAt,
		UpdatedAt:  definition.UpdatedAt,
	}
}

type GroupDefinition struct {
	ID         uuid.UUID              `json:"id"`
	Name       string                 `json:"name"`
	Members    []uuid.UUID            `json:"members"`
	ParentID   *uuid.UUID             `json:"parent_id"`
	Properties map[string]interface{} `json:"properties"`
	Locked     bool                   `json:"locked"`
	CreatedAt  time.Time              `json:"created_at"`
	UpdatedAt  time.Time              `json:"updated_at"`
}

func (definition *GroupDefinition) Validate() error {
	if definition.ID == uuid.Nil {
		return fmt.Errorf("group ID cannot be nil")
	}
	if definition.Name == "" {
		return fmt.Errorf("group name cannot be empty")
	}
	return nil
}

func (definition *GroupDefinition) HasMember(id uuid.UUID) bool {
	for _, memberID := range definition.Members {
		if memberID == id {
			return true
		}
	}
	return false
}

func (definition *GroupDefinition) AddMember(id uuid.UUID) error {
	if definition.Locked {
		return fmt.Errorf("group %s is locked", definition.ID)
	}
	if definition.HasMember(id) {
		return fmt.Errorf("member %s already exists", id)
	}
	definition.Members = append(definition.Members, id)
	definition.UpdatedAt = time.Now()
	return nil
}

func (definition *GroupDefinition) RemoveMember(id uuid.UUID) error {
	if definition.Locked {
		return fmt.Errorf("group %s is locked", definition.ID)
	}
	for index, memberID := range definition.Members {
		if memberID == id {
			definition.Members = append(definition.Members[:index], definition.Members[index+1:]...)
			definition.UpdatedAt = time.Now()
			return nil
		}
	}
	return fmt.Errorf("member %s not found", id)
}

func (definition *GroupDefinition) Lock() {
	definition.Locked = true
	definition.UpdatedAt = time.Now()
}

func (definition *GroupDefinition) Unlock() {
	definition.Locked = false
	definition.UpdatedAt = time.Now()
}