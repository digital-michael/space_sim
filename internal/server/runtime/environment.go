package runtime

import (
	"fmt"
	"sync"
	"time"

	"github.com/digital-michael/space_sim/internal/server/pool/group"
	"github.com/google/uuid"
)

type RuntimeEnvironment struct {
	mu           sync.RWMutex
	objects      map[uuid.UUID]*ObjectState
	groups       map[uuid.UUID]*GroupState
	definitions  *group.Pool
	nextPosIndex int
}

func NewRuntimeEnvironment(definitions *group.Pool) *RuntimeEnvironment {
	return &RuntimeEnvironment{
		objects:     make(map[uuid.UUID]*ObjectState),
		groups:      make(map[uuid.UUID]*GroupState),
		definitions: definitions,
	}
}

func (environment *RuntimeEnvironment) InitializeObject(id uuid.UUID, positionFunc PositionFunc, velocity Vector3) error {
	environment.mu.Lock()
	defer environment.mu.Unlock()

	if environment.definitions == nil {
		return fmt.Errorf("definitions pool is required")
	}
	if _, err := environment.definitions.GetObject(id); err != nil {
		return fmt.Errorf("object %s not found in definitions: %w", id, err)
	}
	if _, exists := environment.objects[id]; exists {
		return fmt.Errorf("object %s already initialized", id)
	}

	pos := OriginPosition()(0)
	if positionFunc != nil {
		pos = positionFunc(environment.nextPosIndex)
	}
	environment.nextPosIndex++

	now := time.Now()
	environment.objects[id] = &ObjectState{
		ID:            id,
		Position:      pos,
		Rotation:      Vector3{},
		Scale:         Vector3{X: 1, Y: 1, Z: 1},
		Velocity:      velocity,
		RoutineStates: map[string]interface{}{},
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	environment.invalidateGroupCachesLocked(id)

	return nil
}

func (environment *RuntimeEnvironment) GetObjectState(id uuid.UUID) (*ObjectState, error) {
	environment.mu.RLock()
	defer environment.mu.RUnlock()

	state, exists := environment.objects[id]
	if !exists {
		return nil, fmt.Errorf("object state %s not found", id)
	}

	return state.Clone(), nil
}

func (environment *RuntimeEnvironment) UpdateObjectState(id uuid.UUID, mutator func(*ObjectState)) error {
	environment.mu.Lock()
	defer environment.mu.Unlock()

	state, exists := environment.objects[id]
	if !exists {
		return fmt.Errorf("object state %s not found", id)
	}

	if mutator != nil {
		mutator(state)
	}
	state.UpdatedAt = time.Now()
	environment.invalidateGroupCachesLocked(id)
	return nil
}

func (environment *RuntimeEnvironment) RemoveObjectState(id uuid.UUID) error {
	environment.mu.Lock()
	defer environment.mu.Unlock()

	if _, exists := environment.objects[id]; !exists {
		return fmt.Errorf("object state %s not found", id)
	}

	delete(environment.objects, id)
	environment.invalidateGroupCachesLocked(id)
	return nil
}

func (environment *RuntimeEnvironment) ListObjectStates() []uuid.UUID {
	environment.mu.RLock()
	defer environment.mu.RUnlock()

	ids := make([]uuid.UUID, 0, len(environment.objects))
	for id := range environment.objects {
		ids = append(ids, id)
	}
	return ids
}

func (environment *RuntimeEnvironment) ListGroupStates() []uuid.UUID {
	environment.mu.RLock()
	defer environment.mu.RUnlock()

	ids := make([]uuid.UUID, 0, len(environment.groups))
	for id := range environment.groups {
		ids = append(ids, id)
	}
	return ids
}

func (environment *RuntimeEnvironment) GetObjectStatesByIDs(ids []uuid.UUID) []*ObjectState {
	environment.mu.RLock()
	defer environment.mu.RUnlock()

	result := make([]*ObjectState, 0, len(ids))
	for _, id := range ids {
		if state, exists := environment.objects[id]; exists {
			result = append(result, state.Clone())
		}
	}
	return result
}

func (environment *RuntimeEnvironment) GetAggregatesByIDs(ids []uuid.UUID) []*GroupState {
	environment.mu.Lock()
	defer environment.mu.Unlock()

	result := make([]*GroupState, 0, len(ids))
	for _, id := range ids {
		if state, exists := environment.groups[id]; exists {
			if !state.CachedValid {
				if err := environment.computeGroupStateLocked(state, map[uuid.UUID]struct{}{}); err == nil {
					state.CachedValid = true
				}
			}
			result = append(result, state.Clone())
		}
	}
	return result
}
