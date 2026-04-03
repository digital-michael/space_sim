package runtime

import (
	"time"

	"github.com/google/uuid"
)

type Vector3 struct {
	X float64
	Y float64
	Z float64
}

type ObjectState struct {
	ID            uuid.UUID
	Position      Vector3
	Rotation      Vector3
	Scale         Vector3
	Velocity      Vector3
	RoutineStates map[string]interface{}
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

func (state *ObjectState) Clone() *ObjectState {
	routineStates := make(map[string]interface{}, len(state.RoutineStates))
	for key, value := range state.RoutineStates {
		routineStates[key] = value
	}

	return &ObjectState{
		ID:            state.ID,
		Position:      state.Position,
		Rotation:      state.Rotation,
		Scale:         state.Scale,
		Velocity:      state.Velocity,
		RoutineStates: routineStates,
		CreatedAt:     state.CreatedAt,
		UpdatedAt:     state.UpdatedAt,
	}
}
