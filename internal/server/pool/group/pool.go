package group

import (
	"errors"
	"fmt"
	"sync"
	"time"

	basepool "github.com/digital-michael/space_sim/internal/server/pool"
	"github.com/google/uuid"
)

var (
	ErrObjectNotFound         = errors.New("object definition not found")
	ErrGroupNotFound          = errors.New("group definition not found")
	ErrGroupNotEmpty          = errors.New("group is not empty")
	ErrHierarchyCycle         = errors.New("group hierarchy cycle detected")
	ErrHierarchyDepthExceeded = errors.New("group hierarchy depth limit exceeded")
)

const maxGroupHierarchyDepth = 20

type Pool struct {
	mu             sync.RWMutex
	objects        map[uuid.UUID]*ObjectDefinition
	groups         map[uuid.UUID]*GroupDefinition
	dag            *DAG
	memberToGroups map[uuid.UUID]map[uuid.UUID]struct{}
}

func NewPool() *Pool {
	return &Pool{
		objects:        make(map[uuid.UUID]*ObjectDefinition),
		groups:         make(map[uuid.UUID]*GroupDefinition),
		dag:            NewDAG(),
		memberToGroups: make(map[uuid.UUID]map[uuid.UUID]struct{}),
	}
}

func (pool *Pool) GetType() basepool.PoolType {
	return basepool.PoolTypeGroup
}

func (pool *Pool) Create(id uuid.UUID, objType string, properties map[string]interface{}) error {
	return pool.CreateObject(id, objType, properties)
}

func (pool *Pool) Get(id uuid.UUID) (interface{}, error) {
	object, err := pool.GetObject(id)
	if err == nil {
		return object, nil
	}

	group, groupErr := pool.GetGroup(id)
	if groupErr == nil {
		return group, nil
	}

	return nil, ErrObjectNotFound
}

func (pool *Pool) Update(id uuid.UUID, properties map[string]interface{}) error {
	return pool.UpdateObject(id, properties)
}

func (pool *Pool) Delete(id uuid.UUID) error {
	pool.mu.Lock()
	defer pool.mu.Unlock()

	if _, exists := pool.objects[id]; exists {
		pool.removeMemberFromAllGroups(id)
		delete(pool.objects, id)
		return nil
	}
	if group, exists := pool.groups[id]; exists {
		if len(group.Members) > 0 {
			return ErrGroupNotEmpty
		}
		for _, memberID := range group.Members {
			pool.removeMembershipIndex(memberID, id)
			if _, isGroup := pool.groups[memberID]; isGroup {
				pool.dag.RemoveEdge(id, memberID)
			}
		}
		for parentID, parentGroup := range pool.groups {
			if parentID == id {
				continue
			}
			for index, memberID := range parentGroup.Members {
				if memberID == id {
					parentGroup.Members = append(parentGroup.Members[:index], parentGroup.Members[index+1:]...)
					parentGroup.UpdatedAt = time.Now()
					pool.removeMembershipIndex(id, parentID)
					pool.dag.RemoveEdge(parentID, id)
					break
				}
			}
		}
		pool.dag.RemoveNode(id)
		delete(pool.groups, id)
		return nil
	}

	return ErrObjectNotFound
}

func (pool *Pool) List() []uuid.UUID {
	pool.mu.RLock()
	defer pool.mu.RUnlock()

	ids := make([]uuid.UUID, 0, len(pool.objects))
	for id := range pool.objects {
		ids = append(ids, id)
	}
	return ids
}

func (pool *Pool) CreateObject(id uuid.UUID, objType string, properties map[string]interface{}) error {
	pool.mu.Lock()
	defer pool.mu.Unlock()

	if id == uuid.Nil {
		return fmt.Errorf("object ID cannot be nil")
	}
	if _, exists := pool.objects[id]; exists {
		return fmt.Errorf("object %s already exists", id)
	}

	now := time.Now()
	definition := &ObjectDefinition{
		ID:         id,
		Type:       objType,
		Properties: cloneProperties(properties),
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	if err := definition.Validate(); err != nil {
		return err
	}

	pool.objects[id] = definition
	return nil
}

func (pool *Pool) GetObject(id uuid.UUID) (*ObjectDefinition, error) {
	pool.mu.RLock()
	defer pool.mu.RUnlock()

	definition, exists := pool.objects[id]
	if !exists {
		return nil, ErrObjectNotFound
	}
	return definition.Clone(), nil
}

func (pool *Pool) UpdateObject(id uuid.UUID, properties map[string]interface{}) error {
	pool.mu.Lock()
	defer pool.mu.Unlock()

	definition, exists := pool.objects[id]
	if !exists {
		return ErrObjectNotFound
	}

	for key, value := range properties {
		if definition.Properties == nil {
			definition.Properties = make(map[string]interface{})
		}
		definition.Properties[key] = value
	}
	definition.UpdatedAt = time.Now()
	return nil
}

func (pool *Pool) DeleteObject(id uuid.UUID) error {
	pool.mu.Lock()
	defer pool.mu.Unlock()

	if _, exists := pool.objects[id]; !exists {
		return ErrObjectNotFound
	}
	pool.removeMemberFromAllGroups(id)
	delete(pool.objects, id)
	return nil
}

func (pool *Pool) ListObjects() []uuid.UUID {
	return pool.List()
}

func (pool *Pool) UpdateGroup(id uuid.UUID, properties map[string]interface{}) error {
	pool.mu.Lock()
	defer pool.mu.Unlock()

	definition, exists := pool.groups[id]
	if !exists {
		return ErrGroupNotFound
	}

	for key, value := range properties {
		if definition.Properties == nil {
			definition.Properties = make(map[string]interface{})
		}
		definition.Properties[key] = value
	}
	definition.UpdatedAt = time.Now()
	return nil
}

func (pool *Pool) ListGroups() []uuid.UUID {
	pool.mu.RLock()
	defer pool.mu.RUnlock()

	ids := make([]uuid.UUID, 0, len(pool.groups))
	for id := range pool.groups {
		ids = append(ids, id)
	}
	return ids
}

func (pool *Pool) CreateGroup(id uuid.UUID, name string, parentID *uuid.UUID, properties map[string]interface{}) error {
	pool.mu.Lock()
	defer pool.mu.Unlock()

	if id == uuid.Nil {
		return fmt.Errorf("group ID cannot be nil")
	}
	if _, exists := pool.groups[id]; exists {
		return fmt.Errorf("group %s already exists", id)
	}
	if parentID != nil {
		if *parentID == id {
			return ErrHierarchyCycle
		}
		if _, exists := pool.groups[*parentID]; !exists {
			return fmt.Errorf("parent group %s not found", *parentID)
		}
		if pool.dag.WouldCreateCycle(*parentID, id) {
			return ErrHierarchyCycle
		}
		if pool.wouldExceedHierarchyDepth(*parentID, id) {
			return ErrHierarchyDepthExceeded
		}
	}

	now := time.Now()
	definition := &GroupDefinition{
		ID:         id,
		Name:       name,
		Members:    make([]uuid.UUID, 0),
		ParentID:   parentID,
		Properties: cloneProperties(properties),
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	if err := definition.Validate(); err != nil {
		return err
	}

	pool.groups[id] = definition
	pool.dag.EnsureNode(id)
	if parentID != nil {
		parent := pool.groups[*parentID]
		if err := parent.AddMember(id); err != nil {
			delete(pool.groups, id)
			pool.dag.RemoveNode(id)
			return err
		}
		pool.addMembershipIndex(id, *parentID)
		pool.dag.AddEdge(*parentID, id)
	}
	return nil
}

func (pool *Pool) GetGroup(id uuid.UUID) (*GroupDefinition, error) {
	pool.mu.RLock()
	defer pool.mu.RUnlock()

	definition, exists := pool.groups[id]
	if !exists {
		return nil, ErrGroupNotFound
	}

	members := make([]uuid.UUID, len(definition.Members))
	copy(members, definition.Members)
	properties := cloneProperties(definition.Properties)

	copyDef := &GroupDefinition{
		ID:         definition.ID,
		Name:       definition.Name,
		Members:    members,
		ParentID:   definition.ParentID,
		Properties: properties,
		Locked:     definition.Locked,
		CreatedAt:  definition.CreatedAt,
		UpdatedAt:  definition.UpdatedAt,
	}
	return copyDef, nil
}

func (pool *Pool) AddGroupMember(groupID, memberID uuid.UUID) error {
	pool.mu.Lock()
	defer pool.mu.Unlock()

	group, exists := pool.groups[groupID]
	if !exists {
		return ErrGroupNotFound
	}

	if _, objectExists := pool.objects[memberID]; !objectExists {
		childGroup, groupExists := pool.groups[memberID]
		if !groupExists {
			return fmt.Errorf("member %s not found", memberID)
		}
		if childGroup.ParentID != nil {
			return fmt.Errorf("group %s already has parent %s", memberID, *childGroup.ParentID)
		}
		if groupID == memberID {
			return ErrHierarchyCycle
		}
		if pool.dag.WouldCreateCycle(groupID, memberID) {
			return ErrHierarchyCycle
		}
		if pool.wouldExceedHierarchyDepth(groupID, memberID) {
			return ErrHierarchyDepthExceeded
		}
		if err := group.AddMember(memberID); err != nil {
			return err
		}
		parentIDCopy := groupID
		childGroup.ParentID = &parentIDCopy
		childGroup.UpdatedAt = time.Now()
		pool.addMembershipIndex(memberID, groupID)
		pool.dag.AddEdge(groupID, memberID)
		return nil
	}
	if err := group.AddMember(memberID); err != nil {
		return err
	}
	pool.addMembershipIndex(memberID, groupID)
	return nil
}

func (pool *Pool) RemoveGroupMember(groupID, memberID uuid.UUID) error {
	pool.mu.Lock()
	defer pool.mu.Unlock()

	group, exists := pool.groups[groupID]
	if !exists {
		return ErrGroupNotFound
	}
	if err := group.RemoveMember(memberID); err != nil {
		return err
	}
	if childGroup, isGroup := pool.groups[memberID]; isGroup {
		if childGroup.ParentID != nil && *childGroup.ParentID == groupID {
			childGroup.ParentID = nil
			childGroup.UpdatedAt = time.Now()
		}
	}
	pool.removeMembershipIndex(memberID, groupID)
	if _, isGroup := pool.groups[memberID]; isGroup {
		pool.dag.RemoveEdge(groupID, memberID)
	}
	return nil
}

func (pool *Pool) LockGroup(groupID uuid.UUID) error {
	pool.mu.Lock()
	defer pool.mu.Unlock()

	group, exists := pool.groups[groupID]
	if !exists {
		return ErrGroupNotFound
	}
	group.Lock()
	return nil
}

func (pool *Pool) UnlockGroup(groupID uuid.UUID) error {
	pool.mu.Lock()
	defer pool.mu.Unlock()

	group, exists := pool.groups[groupID]
	if !exists {
		return ErrGroupNotFound
	}
	group.Unlock()
	return nil
}

func cloneProperties(properties map[string]interface{}) map[string]interface{} {
	if properties == nil {
		return map[string]interface{}{}
	}
	copyMap := make(map[string]interface{}, len(properties))
	for key, value := range properties {
		copyMap[key] = value
	}
	return copyMap
}

func (pool *Pool) GroupsForMember(memberID uuid.UUID) []uuid.UUID {
	pool.mu.RLock()
	defer pool.mu.RUnlock()

	groupSet := pool.memberToGroups[memberID]
	if len(groupSet) == 0 {
		return []uuid.UUID{}
	}

	groups := make([]uuid.UUID, 0, len(groupSet))
	for groupID := range groupSet {
		groups = append(groups, groupID)
	}
	return groups
}

func (pool *Pool) addMembershipIndex(memberID, groupID uuid.UUID) {
	if _, exists := pool.memberToGroups[memberID]; !exists {
		pool.memberToGroups[memberID] = make(map[uuid.UUID]struct{})
	}
	pool.memberToGroups[memberID][groupID] = struct{}{}
}

func (pool *Pool) removeMembershipIndex(memberID, groupID uuid.UUID) {
	groupSet, exists := pool.memberToGroups[memberID]
	if !exists {
		return
	}
	delete(groupSet, groupID)
	if len(groupSet) == 0 {
		delete(pool.memberToGroups, memberID)
	}
}

func (pool *Pool) removeMemberFromAllGroups(memberID uuid.UUID) {
	for groupID, group := range pool.groups {
		for index := 0; index < len(group.Members); index++ {
			if group.Members[index] != memberID {
				continue
			}
			group.Members = append(group.Members[:index], group.Members[index+1:]...)
			group.UpdatedAt = time.Now()
			pool.removeMembershipIndex(memberID, groupID)
			index--
		}
	}
}

func (pool *Pool) wouldExceedHierarchyDepth(parentID, childID uuid.UUID) bool {
	parentDepth := pool.groupDepth(parentID, map[uuid.UUID]int{}, map[uuid.UUID]struct{}{})
	childSubtreeHeight := pool.groupSubtreeHeight(childID, map[uuid.UUID]int{}, map[uuid.UUID]struct{}{})
	return parentDepth+childSubtreeHeight > maxGroupHierarchyDepth
}

func (pool *Pool) groupDepth(groupID uuid.UUID, memo map[uuid.UUID]int, visiting map[uuid.UUID]struct{}) int {
	if depth, exists := memo[groupID]; exists {
		return depth
	}
	if _, exists := visiting[groupID]; exists {
		return maxGroupHierarchyDepth + 1
	}
	visiting[groupID] = struct{}{}

	maxParentDepth := 0
	for parentID := range pool.memberToGroups[groupID] {
		depth := pool.groupDepth(parentID, memo, visiting)
		if depth > maxParentDepth {
			maxParentDepth = depth
		}
	}

	delete(visiting, groupID)
	depth := maxParentDepth + 1
	memo[groupID] = depth
	return depth
}

func (pool *Pool) groupSubtreeHeight(groupID uuid.UUID, memo map[uuid.UUID]int, visiting map[uuid.UUID]struct{}) int {
	if height, exists := memo[groupID]; exists {
		return height
	}
	if _, exists := visiting[groupID]; exists {
		return maxGroupHierarchyDepth + 1
	}
	visiting[groupID] = struct{}{}

	maxChildHeight := 0
	group := pool.groups[groupID]
	if group != nil {
		for _, memberID := range group.Members {
			if _, isGroup := pool.groups[memberID]; !isGroup {
				continue
			}
			height := pool.groupSubtreeHeight(memberID, memo, visiting)
			if height > maxChildHeight {
				maxChildHeight = height
			}
		}
	}

	delete(visiting, groupID)
	height := maxChildHeight + 1
	memo[groupID] = height
	return height
}
