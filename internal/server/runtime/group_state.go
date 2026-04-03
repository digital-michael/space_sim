package runtime

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

type GroupState struct {
	ID          uuid.UUID
	MemberCount int
	Center      Vector3
	AvgVelocity Vector3
	BoundingMin Vector3
	BoundingMax Vector3
	CachedValid bool
	UpdatedAt   time.Time
}

func (state *GroupState) Clone() *GroupState {
	if state == nil {
		return nil
	}
	return &GroupState{
		ID:          state.ID,
		MemberCount: state.MemberCount,
		Center:      state.Center,
		AvgVelocity: state.AvgVelocity,
		BoundingMin: state.BoundingMin,
		BoundingMax: state.BoundingMax,
		CachedValid: state.CachedValid,
		UpdatedAt:   state.UpdatedAt,
	}
}

func (environment *RuntimeEnvironment) InitializeGroup(id uuid.UUID) error {
	environment.mu.Lock()
	defer environment.mu.Unlock()

	if environment.definitions == nil {
		return fmt.Errorf("definitions pool is required")
	}
	if _, err := environment.definitions.GetGroup(id); err != nil {
		return fmt.Errorf("group %s not found in definitions: %w", id, err)
	}
	if _, exists := environment.groups[id]; exists {
		return fmt.Errorf("group %s already initialized", id)
	}

	environment.groups[id] = &GroupState{ID: id}
	return nil
}

func (environment *RuntimeEnvironment) GetGroupState(id uuid.UUID) (*GroupState, error) {
	environment.mu.Lock()
	defer environment.mu.Unlock()

	state, exists := environment.groups[id]
	if !exists {
		return nil, fmt.Errorf("group state %s not found", id)
	}

	if !state.CachedValid {
		if err := environment.computeGroupStateLocked(state, map[uuid.UUID]struct{}{}); err != nil {
			return nil, err
		}
		state.CachedValid = true
	}

	return state.Clone(), nil
}

func (environment *RuntimeEnvironment) computeGroupStateLocked(state *GroupState, visiting map[uuid.UUID]struct{}) error {
	if _, seen := visiting[state.ID]; seen {
		return fmt.Errorf("cycle detected while computing group state %s", state.ID)
	}
	visiting[state.ID] = struct{}{}
	defer delete(visiting, state.ID)

	definition, err := environment.definitions.GetGroup(state.ID)
	if err != nil {
		return fmt.Errorf("group %s not found in definitions: %w", state.ID, err)
	}

	var (
		totalPos       Vector3
		totalVel       Vector3
		count          int
		boundingMin    Vector3
		boundingMax    Vector3
		haveBoundValue bool
	)

	for _, memberID := range definition.Members {
		if objectState, exists := environment.objects[memberID]; exists {
			totalPos = addVector3(totalPos, objectState.Position)
			totalVel = addVector3(totalVel, objectState.Velocity)
			count++

			if !haveBoundValue {
				boundingMin = objectState.Position
				boundingMax = objectState.Position
				haveBoundValue = true
			} else {
				boundingMin = minVector3(boundingMin, objectState.Position)
				boundingMax = maxVector3(boundingMax, objectState.Position)
			}
			continue
		}

		groupState, exists := environment.groups[memberID]
		if !exists {
			continue
		}
		if !groupState.CachedValid {
			if err := environment.computeGroupStateLocked(groupState, visiting); err != nil {
				return err
			}
			groupState.CachedValid = true
		}
		if groupState.MemberCount == 0 {
			continue
		}

		totalPos = addVector3(totalPos, scaleVector3(groupState.Center, float64(groupState.MemberCount)))
		totalVel = addVector3(totalVel, scaleVector3(groupState.AvgVelocity, float64(groupState.MemberCount)))
		count += groupState.MemberCount

		if !haveBoundValue {
			boundingMin = groupState.BoundingMin
			boundingMax = groupState.BoundingMax
			haveBoundValue = true
		} else {
			boundingMin = minVector3(boundingMin, groupState.BoundingMin)
			boundingMax = maxVector3(boundingMax, groupState.BoundingMax)
		}
	}

	state.MemberCount = count
	state.UpdatedAt = time.Now()
	if count == 0 {
		state.Center = Vector3{}
		state.AvgVelocity = Vector3{}
		state.BoundingMin = Vector3{}
		state.BoundingMax = Vector3{}
		return nil
	}

	reciprocal := 1.0 / float64(count)
	state.Center = scaleVector3(totalPos, reciprocal)
	state.AvgVelocity = scaleVector3(totalVel, reciprocal)
	state.BoundingMin = boundingMin
	state.BoundingMax = boundingMax
	return nil
}

func addVector3(left, right Vector3) Vector3 {
	return Vector3{X: left.X + right.X, Y: left.Y + right.Y, Z: left.Z + right.Z}
}

func scaleVector3(value Vector3, factor float64) Vector3 {
	return Vector3{X: value.X * factor, Y: value.Y * factor, Z: value.Z * factor}
}

func minVector3(left, right Vector3) Vector3 {
	return Vector3{
		X: minFloat(left.X, right.X),
		Y: minFloat(left.Y, right.Y),
		Z: minFloat(left.Z, right.Z),
	}
}

func maxVector3(left, right Vector3) Vector3 {
	return Vector3{
		X: maxFloat(left.X, right.X),
		Y: maxFloat(left.Y, right.Y),
		Z: maxFloat(left.Z, right.Z),
	}
}

func minFloat(left, right float64) float64 {
	if left < right {
		return left
	}
	return right
}

func maxFloat(left, right float64) float64 {
	if left > right {
		return left
	}
	return right
}
