// Package eventloop provides frame processing for the simulation loop.
package eventloop

import (
	"context"
	"fmt"
	"time"

	"github.com/digital-michael/space_sim/internal/server/eventqueue"
	"github.com/digital-michael/space_sim/internal/server/pool"
	grouppool "github.com/digital-michael/space_sim/internal/server/pool/group"
	"github.com/digital-michael/space_sim/internal/server/routines"
	runtimepkg "github.com/digital-michael/space_sim/internal/server/runtime"
	"github.com/google/uuid"
)

const (
	bridgeKeyCreatedAt        = "__bridge_createdAt"
	bridgeKeyUpdatedAt        = "__bridge_updatedAt"
	bridgeKeyRuntimeVelocity  = "__bridge_runtimeVelocity"
	bridgeKeyDefinitionKeys   = "__bridge_definitionKeys"
	bridgeKeyRoutineStateKeys = "__bridge_routineStateKeys"
)

// Frame represents a single simulation frame.
// It processes events, executes routines, and applies physics in order.
type Frame struct {
	definitions *grouppool.Pool
	runtime     *runtimepkg.RuntimeEnvironment
	eventQueue  *eventqueue.QueueManager
	routineLib  *routines.Library
	deltaTime   float64
}

// NewFrame creates a new frame processor.
func NewFrame(
	definitions *grouppool.Pool,
	runtime *runtimepkg.RuntimeEnvironment,
	eventQueue *eventqueue.QueueManager,
	routineLib *routines.Library,
	deltaTime float64,
) *Frame {
	return &Frame{
		definitions: definitions,
		runtime:     runtime,
		eventQueue:  eventQueue,
		routineLib:  routineLib,
		deltaTime:   deltaTime,
	}
}

// Process executes one simulation frame in the correct order:
//  1. Process events (highest priority — mutate state)
//  2. Execute routines (behavior logic)
//  3. Apply physics (velocity integration)
func (f *Frame) Process() error {
	if err := f.processEvents(); err != nil {
		return fmt.Errorf("event processing failed: %w", err)
	}

	if err := f.executeRoutines(); err != nil {
		return fmt.Errorf("routine execution failed: %w", err)
	}

	if err := f.applyPhysics(); err != nil {
		return fmt.Errorf("physics update failed: %w", err)
	}

	return nil
}

// processEvents drains all pending events from the queue manager.
//
// Transaction scope in frame processing is intentionally per-event: each
// dequeued event gets its own TransactionContext and executes independently.
// Per-GUID ordering is maintained by the queue manager.
func (f *Frame) processEvents() error {
	for {
		event, _, err := f.eventQueue.DequeueNext()
		if err == eventqueue.ErrNoEvents {
			break
		}
		if err != nil {
			// Log and continue — one bad dequeue should not abort the frame.
			continue
		}

		// Per-event transaction context (not batch/frame-scoped).
		tc := eventqueue.NewTransactionContext(event.TransactionType)
		tc.AddEvent(event)

		if execErr := tc.Execute(f.runtime); execErr != nil {
			// Log error; transaction internally handles rollback if needed.
			// Continue processing remaining events.
			_ = execErr
		}
	}
	return nil
}

// executeRoutines runs all registered routines on every live object.
// Routines receive a temporary pool.Object populated from the runtime state,
// and their mutations are written back to the runtime after execution.
func (f *Frame) executeRoutines() error {
	if f.routineLib == nil || f.definitions == nil {
		return nil
	}

	ctx := context.Background()
	_ = ctx

	objectIDs := f.runtime.ListObjectStates()
	for _, objID := range objectIDs {
		if err := f.executeRoutinesForObject(objID); err != nil {
			// Log error but continue with remaining objects.
			_ = err
		}
	}
	return nil
}

// executeRoutinesForObject runs all routines assigned to a single object.
func (f *Frame) executeRoutinesForObject(objID uuid.UUID) error {
	// Look up the object definition to find its assigned routines.
	def, err := f.definitions.GetObject(objID)
	if err != nil {
		return nil // Object may not have a definition — skip gracefully.
	}

	routineNames, ok := def.Properties["routines"].([]string)
	if !ok || len(routineNames) == 0 {
		return nil
	}

	// Get a snapshot of the current state.
	state, err := f.runtime.GetObjectState(objID)
	if err != nil {
		return err
	}

	// Build a pool.Object bridge so routines can operate with their expected type.
	obj := stateToPoolObject(def, state)

	// Execute each routine on the bridged object.
	for _, routineName := range routineNames {
		routine, routineErr := f.routineLib.Get(routineName)
		if routineErr != nil {
			continue // Unknown routine — skip.
		}
		if execErr := routine.Execute(obj, float32(f.deltaTime)); execErr != nil {
			continue // Routine error — skip this routine, try others.
		}
	}

	// Write the mutated bridge object back into the runtime state.
	return f.runtime.UpdateObjectState(objID, func(s *runtimepkg.ObjectState) {
		poolObjectToState(obj, s)
	})
}

// applyPhysics integrates velocity into position for every live object.
// Uses the formula: position += velocity * deltaTime.
func (f *Frame) applyPhysics() error {
	dt := f.deltaTime
	objectIDs := f.runtime.ListObjectStates()

	for _, objID := range objectIDs {
		if err := f.runtime.UpdateObjectState(objID, func(s *runtimepkg.ObjectState) {
			s.Position.X += s.Velocity.X * dt
			s.Position.Y += s.Velocity.Y * dt
			s.Position.Z += s.Velocity.Z * dt
		}); err != nil {
			// Object may have been removed concurrently — skip gracefully.
			_ = err
		}
	}
	return nil
}

// --- Bridge helpers ---

// stateToPoolObject converts a runtime.ObjectState to a pool.Object so that
// the existing routines (which work on pool.Object) can execute unchanged.
func stateToPoolObject(def *grouppool.ObjectDefinition, state *runtimepkg.ObjectState) *pool.Object {
	objType := state.ID.String()
	if def != nil && def.Type != "" {
		objType = def.Type
	}

	obj := pool.NewObject(objType)
	obj.GUID = state.ID.String()
	obj.Type = objType
	obj.Position = pool.Position{
		X: float32(state.Position.X),
		Y: float32(state.Position.Y),
		Z: float32(state.Position.Z),
	}
	obj.Rotation = pool.Rotation{
		X: float32(state.Rotation.X),
		Y: float32(state.Rotation.Y),
		Z: float32(state.Rotation.Z),
	}
	obj.Scale = pool.Scale{
		X: float32(state.Scale.X),
		Y: float32(state.Scale.Y),
		Z: float32(state.Scale.Z),
	}

	if obj.Properties == nil {
		obj.Properties = make(map[string]interface{})
	}

	definitionKeys := make([]string, 0)
	if def != nil {
		for key, value := range def.Properties {
			obj.Properties[key] = cloneBridgeValue(value)
			definitionKeys = append(definitionKeys, key)
		}
		if routineNames, ok := def.Properties["routines"].([]string); ok {
			obj.Routines = append([]string(nil), routineNames...)
		}
	}

	for k, v := range state.RoutineStates {
		obj.Properties[k] = cloneBridgeValue(v)
	}

	routineStateKeys := make([]string, 0, len(state.RoutineStates))
	for key := range state.RoutineStates {
		routineStateKeys = append(routineStateKeys, key)
	}

	obj.Properties[bridgeKeyDefinitionKeys] = definitionKeys
	obj.Properties[bridgeKeyRoutineStateKeys] = routineStateKeys
	obj.Properties[bridgeKeyRuntimeVelocity] = map[string]interface{}{
		"x": state.Velocity.X,
		"y": state.Velocity.Y,
		"z": state.Velocity.Z,
	}
	if _, exists := obj.Properties["velocityX"]; !exists {
		obj.Properties["velocityX"] = state.Velocity.X
	}
	if _, exists := obj.Properties["velocityY"]; !exists {
		obj.Properties["velocityY"] = state.Velocity.Y
	}
	if _, exists := obj.Properties["velocityZ"]; !exists {
		obj.Properties["velocityZ"] = state.Velocity.Z
	}
	obj.Properties[bridgeKeyCreatedAt] = state.CreatedAt
	obj.Properties[bridgeKeyUpdatedAt] = state.UpdatedAt
	return obj
}

// poolObjectToState writes mutated pool.Object fields back into a runtime.ObjectState.
func poolObjectToState(obj *pool.Object, state *runtimepkg.ObjectState) {
	state.Position = runtimepkg.Vector3{
		X: float64(obj.Position.X),
		Y: float64(obj.Position.Y),
		Z: float64(obj.Position.Z),
	}
	state.Rotation = runtimepkg.Vector3{
		X: float64(obj.Rotation.X),
		Y: float64(obj.Rotation.Y),
		Z: float64(obj.Rotation.Z),
	}
	state.Scale = runtimepkg.Vector3{
		X: float64(obj.Scale.X),
		Y: float64(obj.Scale.Y),
		Z: float64(obj.Scale.Z),
	}

	velocityX, hasVelocityX := bridgeToFloat64(obj.Properties["velocityX"])
	velocityY, hasVelocityY := bridgeToFloat64(obj.Properties["velocityY"])
	velocityZ, hasVelocityZ := bridgeToFloat64(obj.Properties["velocityZ"])
	if hasVelocityX || hasVelocityY || hasVelocityZ {
		if hasVelocityX {
			state.Velocity.X = velocityX
		}
		if hasVelocityY {
			state.Velocity.Y = velocityY
		}
		if hasVelocityZ {
			state.Velocity.Z = velocityZ
		}
	}

	definitionKeys := make(map[string]struct{})
	if rawKeys, ok := obj.Properties[bridgeKeyDefinitionKeys].([]string); ok {
		for _, key := range rawKeys {
			definitionKeys[key] = struct{}{}
		}
	}

	routineOwnedKeys := make(map[string]struct{})
	if rawKeys, ok := obj.Properties[bridgeKeyRoutineStateKeys].([]string); ok {
		for _, key := range rawKeys {
			routineOwnedKeys[key] = struct{}{}
		}
	}
	for key := range builtInRoutineStateKeys {
		routineOwnedKeys[key] = struct{}{}
	}

	nextRoutineStates := make(map[string]interface{})
	for k, v := range obj.Properties {
		if isBridgeMetadataKey(k) {
			continue
		}
		if k == "velocityX" || k == "velocityY" || k == "velocityZ" {
			continue
		}
		if _, isDefinitionProperty := definitionKeys[k]; isDefinitionProperty {
			continue
		}
		if _, isRoutineOwned := routineOwnedKeys[k]; !isRoutineOwned {
			continue
		}
		nextRoutineStates[k] = cloneBridgeValue(v)
	}
	state.RoutineStates = nextRoutineStates
}

var builtInRoutineStateKeys = map[string]struct{}{
	"scalePhase": {},
}

func isBridgeMetadataKey(key string) bool {
	switch key {
	case bridgeKeyCreatedAt, bridgeKeyUpdatedAt, bridgeKeyRuntimeVelocity, bridgeKeyDefinitionKeys:
		return true
	case bridgeKeyRoutineStateKeys:
		return true
	default:
		return false
	}
}

func cloneBridgeValue(value interface{}) interface{} {
	switch typed := value.(type) {
	case map[string]interface{}:
		clone := make(map[string]interface{}, len(typed))
		for key, nestedValue := range typed {
			clone[key] = cloneBridgeValue(nestedValue)
		}
		return clone
	case []string:
		return append([]string(nil), typed...)
	case []interface{}:
		clone := make([]interface{}, len(typed))
		for index, nestedValue := range typed {
			clone[index] = cloneBridgeValue(nestedValue)
		}
		return clone
	case time.Time:
		return typed
	default:
		return value
	}
}

func bridgeToFloat64(value interface{}) (float64, bool) {
	switch typed := value.(type) {
	case float64:
		return typed, true
	case float32:
		return float64(typed), true
	case int:
		return float64(typed), true
	case int64:
		return float64(typed), true
	case int32:
		return float64(typed), true
	default:
		return 0, false
	}
}
