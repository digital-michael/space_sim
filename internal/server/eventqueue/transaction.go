package eventqueue

import (
	"fmt"

	runtimepkg "github.com/digital-michael/space_sim/internal/server/runtime"
	"github.com/google/uuid"
)

type TransactionContext struct {
	transactionType TransactionType
	events          []*Event
}

type objectSnapshot struct {
	exists bool
	state  *runtimepkg.ObjectState
}

func NewTransactionContext(transactionType TransactionType) *TransactionContext {
	if transactionType == "" {
		transactionType = TransactionTypeNone
	}

	return &TransactionContext{
		transactionType: transactionType,
		events:          make([]*Event, 0),
	}
}

func (context *TransactionContext) AddEvent(event *Event) {
	if event == nil {
		return
	}
	context.events = append(context.events, event.Clone())
}

func (context *TransactionContext) Execute(environment *runtimepkg.RuntimeEnvironment) error {
	if environment == nil {
		return fmt.Errorf("runtime environment is required")
	}

	switch context.transactionType {
	case TransactionTypeRollback:
		return context.executeRollback(environment)
	case TransactionTypeBestEffort:
		return context.executeBestEffort(environment)
	case TransactionTypeNone:
		fallthrough
	default:
		return context.executeNone(environment)
	}
}

func (context *TransactionContext) executeNone(environment *runtimepkg.RuntimeEnvironment) error {
	for _, event := range context.events {
		if err := applyEvent(environment, event); err != nil {
			return err
		}
	}
	return nil
}

func (context *TransactionContext) executeBestEffort(environment *runtimepkg.RuntimeEnvironment) error {
	for _, event := range context.events {
		_ = applyEvent(environment, event)
	}
	return nil
}

func (context *TransactionContext) executeRollback(environment *runtimepkg.RuntimeEnvironment) error {
	snapshots := make(map[uuid.UUID]objectSnapshot)
	for _, event := range context.events {
		if _, exists := snapshots[event.GUID]; exists {
			continue
		}

		state, err := environment.GetObjectState(event.GUID)
		if err != nil {
			snapshots[event.GUID] = objectSnapshot{exists: false, state: nil}
			continue
		}
		snapshots[event.GUID] = objectSnapshot{exists: true, state: state}
	}

	for _, event := range context.events {
		if err := applyEvent(environment, event); err != nil {
			if restoreErr := restoreSnapshots(environment, snapshots); restoreErr != nil {
				return fmt.Errorf("transaction failed: %w (rollback failed: %v)", err, restoreErr)
			}
			return err
		}
	}

	return nil
}

func restoreSnapshots(environment *runtimepkg.RuntimeEnvironment, snapshots map[uuid.UUID]objectSnapshot) error {
	for objectID, snapshot := range snapshots {
		currentState, currentErr := environment.GetObjectState(objectID)

		if !snapshot.exists {
			if currentErr == nil && currentState != nil {
				if err := environment.RemoveObjectState(objectID); err != nil {
					return err
				}
			}
			continue
		}

		if currentErr != nil {
			if err := environment.InitializeObject(objectID, nil, snapshot.state.Velocity); err != nil {
				return err
			}
		}

		restoreState := snapshot.state.Clone()
		if err := environment.UpdateObjectState(objectID, func(state *runtimepkg.ObjectState) {
			state.Position = restoreState.Position
			state.Rotation = restoreState.Rotation
			state.Scale = restoreState.Scale
			state.Velocity = restoreState.Velocity
			state.RoutineStates = cloneRoutineStates(restoreState.RoutineStates)
			state.CreatedAt = restoreState.CreatedAt
			state.UpdatedAt = restoreState.UpdatedAt
		}); err != nil {
			return err
		}
	}

	return nil
}

func applyEvent(environment *runtimepkg.RuntimeEnvironment, event *Event) error {
	if event == nil {
		return fmt.Errorf("event cannot be nil")
	}

	switch event.Type {
	case EventTypeCreate:
		velocity := runtimepkg.Vector3{}
		if value, exists := event.Payload["velocity"]; exists {
			if parsed, ok := parseVector3(value); ok {
				velocity = parsed
			}
		}

		if err := environment.InitializeObject(event.GUID, nil, velocity); err != nil {
			return err
		}

		if value, exists := event.Payload["position"]; exists {
			if position, ok := parseVector3(value); ok {
				return environment.UpdateObjectState(event.GUID, func(state *runtimepkg.ObjectState) {
					state.Position = position
				})
			}
		}
		return nil

	case EventTypeUpdate:
		return environment.UpdateObjectState(event.GUID, func(state *runtimepkg.ObjectState) {
			if value, exists := event.Payload["position"]; exists {
				if parsed, ok := parseVector3(value); ok {
					state.Position = parsed
				}
			}
			if value, exists := event.Payload["rotation"]; exists {
				if parsed, ok := parseVector3(value); ok {
					state.Rotation = parsed
				}
			}
			if value, exists := event.Payload["scale"]; exists {
				if parsed, ok := parseVector3(value); ok {
					state.Scale = parsed
				}
			}
			if value, exists := event.Payload["velocity"]; exists {
				if parsed, ok := parseVector3(value); ok {
					state.Velocity = parsed
				}
			}
		})

	case EventTypeMove:
		return environment.UpdateObjectState(event.GUID, func(state *runtimepkg.ObjectState) {
			if value, exists := event.Payload["position"]; exists {
				if parsed, ok := parseVector3(value); ok {
					state.Position = parsed
					return
				}
			}

			dx, _ := toFloat64(event.Payload["dx"])
			dy, _ := toFloat64(event.Payload["dy"])
			dz, _ := toFloat64(event.Payload["dz"])
			state.Position = runtimepkg.Vector3{
				X: state.Position.X + dx,
				Y: state.Position.Y + dy,
				Z: state.Position.Z + dz,
			}
		})

	case EventTypeDelete:
		return environment.RemoveObjectState(event.GUID)

	case EventTypeCustom:
		return nil

	default:
		return fmt.Errorf("unsupported event type: %s", event.Type)
	}
}

func parseVector3(value interface{}) (runtimepkg.Vector3, bool) {
	mapValue, ok := value.(map[string]interface{})
	if !ok {
		return runtimepkg.Vector3{}, false
	}

	x, okX := toFloat64(firstPresent(mapValue, "x", "X"))
	y, okY := toFloat64(firstPresent(mapValue, "y", "Y"))
	z, okZ := toFloat64(firstPresent(mapValue, "z", "Z"))

	if !okX || !okY || !okZ {
		return runtimepkg.Vector3{}, false
	}

	return runtimepkg.Vector3{X: x, Y: y, Z: z}, true
}

func firstPresent(values map[string]interface{}, keys ...string) interface{} {
	for _, key := range keys {
		if value, exists := values[key]; exists {
			return value
		}
	}
	return nil
}

func toFloat64(value interface{}) (float64, bool) {
	switch typed := value.(type) {
	case float64:
		return typed, true
	case float32:
		return float64(typed), true
	case int:
		return float64(typed), true
	case int8:
		return float64(typed), true
	case int16:
		return float64(typed), true
	case int32:
		return float64(typed), true
	case int64:
		return float64(typed), true
	case uint:
		return float64(typed), true
	case uint8:
		return float64(typed), true
	case uint16:
		return float64(typed), true
	case uint32:
		return float64(typed), true
	case uint64:
		return float64(typed), true
	default:
		return 0, false
	}
}

func cloneRoutineStates(source map[string]interface{}) map[string]interface{} {
	if source == nil {
		return map[string]interface{}{}
	}

	cloned := make(map[string]interface{}, len(source))
	for key, value := range source {
		cloned[key] = value
	}
	return cloned
}