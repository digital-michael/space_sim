package eventloop

import (
	"testing"
	"time"

	"github.com/digital-michael/space_sim/internal/server/eventqueue"
	"github.com/digital-michael/space_sim/internal/server/pool"
	"github.com/digital-michael/space_sim/internal/server/pool/group"
	"github.com/digital-michael/space_sim/internal/server/routines"
	runtimepkg "github.com/digital-michael/space_sim/internal/server/runtime"
	"github.com/google/uuid"
)

type captureRoutine struct {
	name    string
	execute func(obj *pool.Object, dt float32) error
}

func (r *captureRoutine) Name() string {
	return r.name
}

func (r *captureRoutine) Execute(obj *pool.Object, dt float32) error {
	return r.execute(obj, dt)
}

// frameObjID is a stable UUID for frame tests.
var frameObjID = uuid.MustParse("22222222-2222-2222-2222-222222222222")

// newTestFrame builds a Frame wired to a fresh runtime with one registered
// object (frameObjID initialized at the origin with optional velocity v).
func newTestFrame(v runtimepkg.Vector3, dt float64) (*Frame, *runtimepkg.RuntimeEnvironment, *eventqueue.QueueManager) {
	defs := group.NewPool()
	_ = defs.CreateObject(frameObjID, "cube", nil)

	rt := runtimepkg.NewRuntimeEnvironment(defs)
	_ = rt.InitializeObject(frameObjID, runtimepkg.OriginPosition(), v)

	mgr := eventqueue.NewQueueManager(100)
	lib := routines.NewLibrary()

	frame := NewFrame(defs, rt, mgr, lib, dt)
	return frame, rt, mgr
}

// TestFrameCreation verifies that NewFrame returns a non-nil Frame.
func TestFrameCreation(t *testing.T) {
	defs := group.NewPool()
	rt := runtimepkg.NewRuntimeEnvironment(defs)
	mgr := eventqueue.NewQueueManager(10)
	lib := routines.NewLibrary()

	frame := NewFrame(defs, rt, mgr, lib, 1.0/60.0)
	if frame == nil {
		t.Fatal("expected non-nil Frame")
	}
}

// TestFrameProcessEmpty confirms Process() succeeds with no events or objects.
func TestFrameProcessEmpty(t *testing.T) {
	defs := group.NewPool()
	rt := runtimepkg.NewRuntimeEnvironment(defs)
	mgr := eventqueue.NewQueueManager(10)
	lib := routines.NewLibrary()

	frame := NewFrame(defs, rt, mgr, lib, 1.0/60.0)
	if err := frame.Process(); err != nil {
		t.Fatalf("unexpected error on empty frame: %v", err)
	}
}

// TestFrameApplyPhysicsIntegration verifies velocity is integrated into position.
// With velocity.X = 1.0 and deltaTime = 0.5 s, position.X should advance by 0.5.
func TestFrameApplyPhysicsIntegration(t *testing.T) {
	dt := 0.5
	v := runtimepkg.Vector3{X: 1.0, Y: 0.0, Z: 0.0}
	frame, rt, _ := newTestFrame(v, dt)

	if err := frame.applyPhysics(); err != nil {
		t.Fatalf("applyPhysics: %v", err)
	}

	state, err := rt.GetObjectState(frameObjID)
	if err != nil {
		t.Fatalf("GetObjectState: %v", err)
	}

	expected := v.X * dt
	if state.Position.X != expected {
		t.Errorf("expected Position.X = %f, got %f", expected, state.Position.X)
	}
}

// TestFrameApplyPhysicsNoVelocity verifies that zero velocity leaves position unchanged.
func TestFrameApplyPhysicsNoVelocity(t *testing.T) {
	frame, rt, _ := newTestFrame(runtimepkg.Vector3{}, 1.0/60.0)

	if err := frame.applyPhysics(); err != nil {
		t.Fatalf("applyPhysics: %v", err)
	}

	state, err := rt.GetObjectState(frameObjID)
	if err != nil {
		t.Fatalf("GetObjectState: %v", err)
	}

	if state.Position.X != 0 || state.Position.Y != 0 || state.Position.Z != 0 {
		t.Errorf("expected zero position after no-velocity physics, got %+v", state.Position)
	}
}

// TestFrameProcessEventsUpdate enqueues an update event and confirms it is applied.
func TestFrameProcessEventsUpdate(t *testing.T) {
	frame, rt, mgr := newTestFrame(runtimepkg.Vector3{}, 1.0/60.0)

	event := eventqueue.NewEvent(
		frameObjID,
		eventqueue.EventTypeUpdate,
		map[string]interface{}{
			"position": map[string]interface{}{"x": 3.0, "y": 2.0, "z": 1.0},
		},
		eventqueue.TransactionTypeNone,
	)
	if err := mgr.Enqueue(event); err != nil {
		t.Fatalf("Enqueue: %v", err)
	}

	if err := frame.processEvents(); err != nil {
		t.Fatalf("processEvents: %v", err)
	}

	state, err := rt.GetObjectState(frameObjID)
	if err != nil {
		t.Fatalf("GetObjectState: %v", err)
	}

	if state.Position.X != 3.0 || state.Position.Y != 2.0 || state.Position.Z != 1.0 {
		t.Errorf("unexpected position after update event: %+v", state.Position)
	}
}

// TestFrameProcessEventsEmptyQueue confirms no error when queue is empty.
func TestFrameProcessEventsEmptyQueue(t *testing.T) {
	frame, _, _ := newTestFrame(runtimepkg.Vector3{}, 1.0/60.0)

	if err := frame.processEvents(); err != nil {
		t.Fatalf("unexpected error on empty queue: %v", err)
	}
}

// TestFrameExecuteRoutinesNoRoutines confirms routine execution is a no-op when
// the object definition has no routines assigned.
func TestFrameExecuteRoutinesNoRoutines(t *testing.T) {
	frame, rt, _ := newTestFrame(runtimepkg.Vector3{}, 1.0/60.0)

	if err := frame.executeRoutines(); err != nil {
		t.Fatalf("executeRoutines: %v", err)
	}

	state, err := rt.GetObjectState(frameObjID)
	if err != nil {
		t.Fatalf("GetObjectState: %v", err)
	}

	// Rotation and position should remain untouched.
	if state.Rotation.Z != 0 {
		t.Errorf("expected no rotation change, got %f", state.Rotation.Z)
	}
}

// TestFrameExecuteRoutinesRotate assigns the built-in rotate routine and
// confirms the object's rotation changes after one frame.
func TestFrameExecuteRoutinesRotate(t *testing.T) {
	dt := 1.0 // 1 second delta so rotation change is easy to verify.

	defs := group.NewPool()
	_ = defs.CreateObject(frameObjID, "cube", map[string]interface{}{
		"routines": []string{"rotate"},
	})

	rt := runtimepkg.NewRuntimeEnvironment(defs)
	_ = rt.InitializeObject(frameObjID, runtimepkg.OriginPosition(), runtimepkg.Vector3{})

	mgr := eventqueue.NewQueueManager(100)
	lib := routines.NewLibrary()

	frame := NewFrame(defs, rt, mgr, lib, dt)

	if err := frame.executeRoutines(); err != nil {
		t.Fatalf("executeRoutines: %v", err)
	}

	state, err := rt.GetObjectState(frameObjID)
	if err != nil {
		t.Fatalf("GetObjectState: %v", err)
	}

	// Default rotation speed is 90 deg/sec, so after 1 s Rotation.Z should be 90.
	if state.Rotation.Z == 0 {
		t.Errorf("expected Rotation.Z to be non-zero after rotate routine, got 0")
	}
}

func TestFrameExecuteRoutinesSkipsUnknownRoutineAndContinues(t *testing.T) {
	defs := group.NewPool()
	if err := defs.CreateObject(frameObjID, "ship", map[string]interface{}{
		"routines": []string{"unknown-routine", "rotate"},
	}); err != nil {
		t.Fatalf("CreateObject: %v", err)
	}

	rt := runtimepkg.NewRuntimeEnvironment(defs)
	if err := rt.InitializeObject(frameObjID, runtimepkg.OriginPosition(), runtimepkg.Vector3{}); err != nil {
		t.Fatalf("InitializeObject: %v", err)
	}

	frame := NewFrame(defs, rt, eventqueue.NewQueueManager(10), routines.NewLibrary(), 1.0)
	if err := frame.executeRoutines(); err != nil {
		t.Fatalf("executeRoutines: %v", err)
	}

	state, err := rt.GetObjectState(frameObjID)
	if err != nil {
		t.Fatalf("GetObjectState: %v", err)
	}
	if state.Rotation.Z == 0 {
		t.Fatal("expected known routine to execute even when unknown routine is present")
	}
}

func TestFrameExecuteRoutinesRotateUsesDefinitionSpeed(t *testing.T) {
	defs := group.NewPool()
	if err := defs.CreateObject(frameObjID, "ship", map[string]interface{}{
		"routines":      []string{"rotate"},
		"rotationSpeed": 45.0,
	}); err != nil {
		t.Fatalf("CreateObject: %v", err)
	}

	rt := runtimepkg.NewRuntimeEnvironment(defs)
	if err := rt.InitializeObject(frameObjID, runtimepkg.OriginPosition(), runtimepkg.Vector3{}); err != nil {
		t.Fatalf("InitializeObject: %v", err)
	}

	frame := NewFrame(defs, rt, eventqueue.NewQueueManager(10), routines.NewLibrary(), 1.0)
	if err := frame.executeRoutines(); err != nil {
		t.Fatalf("executeRoutines: %v", err)
	}

	state, err := rt.GetObjectState(frameObjID)
	if err != nil {
		t.Fatalf("GetObjectState: %v", err)
	}
	if state.Rotation.Z != 45.0 {
		t.Fatalf("expected rotation from definition speed (45 deg), got %f", state.Rotation.Z)
	}
}

func TestFrameExecuteRoutinesScaleUsesDefinitionConfig(t *testing.T) {
	defs := group.NewPool()
	if err := defs.CreateObject(frameObjID, "ship", map[string]interface{}{
		"routines":   []string{"scale"},
		"scaleMin":   2.0,
		"scaleMax":   2.0,
		"scaleSpeed": 0.0,
	}); err != nil {
		t.Fatalf("CreateObject: %v", err)
	}

	rt := runtimepkg.NewRuntimeEnvironment(defs)
	if err := rt.InitializeObject(frameObjID, runtimepkg.OriginPosition(), runtimepkg.Vector3{}); err != nil {
		t.Fatalf("InitializeObject: %v", err)
	}

	frame := NewFrame(defs, rt, eventqueue.NewQueueManager(10), routines.NewLibrary(), 1.0)
	if err := frame.executeRoutines(); err != nil {
		t.Fatalf("executeRoutines: %v", err)
	}

	state, err := rt.GetObjectState(frameObjID)
	if err != nil {
		t.Fatalf("GetObjectState: %v", err)
	}
	if state.Scale.X != 2.0 || state.Scale.Y != 2.0 || state.Scale.Z != 2.0 {
		t.Fatalf("expected scale from definition config to be 2.0, got %+v", state.Scale)
	}
}

func TestFrameExecuteRoutinesRoutineStateOverridesDefinitionConfig(t *testing.T) {
	defs := group.NewPool()
	if err := defs.CreateObject(frameObjID, "ship", map[string]interface{}{
		"routines":      []string{"rotate"},
		"rotationSpeed": 45.0,
	}); err != nil {
		t.Fatalf("CreateObject: %v", err)
	}

	rt := runtimepkg.NewRuntimeEnvironment(defs)
	if err := rt.InitializeObject(frameObjID, runtimepkg.OriginPosition(), runtimepkg.Vector3{}); err != nil {
		t.Fatalf("InitializeObject: %v", err)
	}
	if err := rt.UpdateObjectState(frameObjID, func(state *runtimepkg.ObjectState) {
		state.RoutineStates["rotationSpeed"] = 30.0
	}); err != nil {
		t.Fatalf("UpdateObjectState: %v", err)
	}

	frame := NewFrame(defs, rt, eventqueue.NewQueueManager(10), routines.NewLibrary(), 1.0)
	if err := frame.executeRoutines(); err != nil {
		t.Fatalf("executeRoutines: %v", err)
	}

	state, err := rt.GetObjectState(frameObjID)
	if err != nil {
		t.Fatalf("GetObjectState: %v", err)
	}
	if state.Rotation.Z != 30.0 {
		t.Fatalf("expected routine-state precedence rotation=30, got %f", state.Rotation.Z)
	}
}

func TestFrameBridgeExposesDefinitionTypeAndRuntimeMetadata(t *testing.T) {
	defs := group.NewPool()
	if err := defs.CreateObject(frameObjID, "ship", map[string]interface{}{
		"routines":      []string{"capture"},
		"rotationSpeed": 45.0,
		"label":         "alpha",
	}); err != nil {
		t.Fatalf("CreateObject: %v", err)
	}

	rt := runtimepkg.NewRuntimeEnvironment(defs)
	velocity := runtimepkg.Vector3{X: 1.5, Y: 2.5, Z: 3.5}
	if err := rt.InitializeObject(frameObjID, runtimepkg.OriginPosition(), velocity); err != nil {
		t.Fatalf("InitializeObject: %v", err)
	}
	createdAt := time.Date(2026, time.March, 20, 10, 0, 0, 0, time.UTC)
	updatedAt := createdAt.Add(2 * time.Minute)
	if err := rt.UpdateObjectState(frameObjID, func(state *runtimepkg.ObjectState) {
		state.RoutineStates["scalePhase"] = 90.0
		state.CreatedAt = createdAt
		state.UpdatedAt = updatedAt
	}); err != nil {
		t.Fatalf("UpdateObjectState: %v", err)
	}
	storedState, err := rt.GetObjectState(frameObjID)
	if err != nil {
		t.Fatalf("GetObjectState: %v", err)
	}

	mgr := eventqueue.NewQueueManager(100)
	lib := routines.NewLibrary()

	var capturedType string
	var capturedRoutines []string
	var capturedLabel string
	var capturedRotationSpeed float64
	var capturedScalePhase float64
	var capturedCreatedAt time.Time
	var capturedUpdatedAt time.Time
	var capturedVelocity map[string]interface{}

	if err := lib.Register(&captureRoutine{
		name: "capture",
		execute: func(obj *pool.Object, dt float32) error {
			capturedType = obj.Type
			capturedRoutines = append([]string(nil), obj.Routines...)
			capturedLabel, _ = obj.Properties["label"].(string)
			capturedRotationSpeed, _ = obj.Properties["rotationSpeed"].(float64)
			capturedScalePhase, _ = obj.Properties["scalePhase"].(float64)
			capturedCreatedAt, _ = obj.Properties[bridgeKeyCreatedAt].(time.Time)
			capturedUpdatedAt, _ = obj.Properties[bridgeKeyUpdatedAt].(time.Time)
			capturedVelocity, _ = obj.Properties[bridgeKeyRuntimeVelocity].(map[string]interface{})
			return nil
		},
	}); err != nil {
		t.Fatalf("Register: %v", err)
	}

	frame := NewFrame(defs, rt, mgr, lib, 1.0)
	if err := frame.executeRoutinesForObject(frameObjID); err != nil {
		t.Fatalf("executeRoutinesForObject: %v", err)
	}

	if capturedType != "ship" {
		t.Fatalf("expected bridged object type 'ship', got %q", capturedType)
	}
	if len(capturedRoutines) != 1 || capturedRoutines[0] != "capture" {
		t.Fatalf("expected bridged routines [capture], got %v", capturedRoutines)
	}
	if capturedLabel != "alpha" {
		t.Fatalf("expected definition property label=alpha, got %q", capturedLabel)
	}
	if capturedRotationSpeed != 45.0 {
		t.Fatalf("expected definition property rotationSpeed=45, got %f", capturedRotationSpeed)
	}
	if capturedScalePhase != 90.0 {
		t.Fatalf("expected routine state scalePhase=90, got %f", capturedScalePhase)
	}
	if !capturedCreatedAt.Equal(storedState.CreatedAt) {
		t.Fatalf("expected bridged createdAt %v, got %v", storedState.CreatedAt, capturedCreatedAt)
	}
	if !capturedUpdatedAt.Equal(storedState.UpdatedAt) {
		t.Fatalf("expected bridged updatedAt %v, got %v", storedState.UpdatedAt, capturedUpdatedAt)
	}
	if capturedVelocity["x"] != velocity.X || capturedVelocity["y"] != velocity.Y || capturedVelocity["z"] != velocity.Z {
		t.Fatalf("expected bridged velocity %+v, got %+v", velocity, capturedVelocity)
	}
}

func TestPoolObjectToStateSkipsDefinitionAndBridgeMetadata(t *testing.T) {
	state := &runtimepkg.ObjectState{
		ID:            frameObjID,
		Scale:         runtimepkg.Vector3{X: 1, Y: 1, Z: 1},
		Velocity:      runtimepkg.Vector3{X: 7, Y: 8, Z: 9},
		RoutineStates: map[string]interface{}{"scalePhase": 45.0},
	}

	createdAt := time.Date(2026, time.March, 20, 11, 0, 0, 0, time.UTC)
	updatedAt := createdAt.Add(time.Minute)
	obj := pool.NewObject("ship")
	obj.Properties["scalePhase"] = 180.0
	obj.Properties["rotationSpeed"] = 45.0
	obj.Properties["label"] = "alpha"
	obj.Properties[bridgeKeyDefinitionKeys] = []string{"rotationSpeed", "label"}
	obj.Properties[bridgeKeyRoutineStateKeys] = []string{"scalePhase"}
	obj.Properties[bridgeKeyRuntimeVelocity] = map[string]interface{}{"x": 1.0, "y": 2.0, "z": 3.0}
	obj.Properties[bridgeKeyCreatedAt] = createdAt
	obj.Properties[bridgeKeyUpdatedAt] = updatedAt

	poolObjectToState(obj, state)

	if state.RoutineStates["scalePhase"] != 180.0 {
		t.Fatalf("expected routine state scalePhase to persist, got %v", state.RoutineStates["scalePhase"])
	}
	if _, exists := state.RoutineStates["rotationSpeed"]; exists {
		t.Fatal("expected definition property rotationSpeed to be excluded from routine state writeback")
	}
	if _, exists := state.RoutineStates["label"]; exists {
		t.Fatal("expected definition property label to be excluded from routine state writeback")
	}
	if _, exists := state.RoutineStates[bridgeKeyRuntimeVelocity]; exists {
		t.Fatal("expected bridge runtime velocity metadata to be excluded from routine state writeback")
	}
	if _, exists := state.RoutineStates["velocityX"]; exists {
		t.Fatal("expected velocityX to map to runtime velocity, not routine state")
	}
	if state.Velocity.X != 7 || state.Velocity.Y != 8 || state.Velocity.Z != 9 {
		t.Fatalf("expected runtime velocity to remain unchanged, got %+v", state.Velocity)
	}
}

func TestPoolObjectToStateDropsNonRoutineTransientProperties(t *testing.T) {
	state := &runtimepkg.ObjectState{
		ID:            frameObjID,
		RoutineStates: map[string]interface{}{"scalePhase": 90.0},
	}

	obj := pool.NewObject("ship")
	obj.Properties["scalePhase"] = 180.0
	obj.Properties["tempColor"] = "red"
	obj.Properties[bridgeKeyRoutineStateKeys] = []string{"scalePhase"}

	poolObjectToState(obj, state)

	if _, exists := state.RoutineStates["tempColor"]; exists {
		t.Fatal("expected transient non-routine property to be dropped")
	}
	if state.RoutineStates["scalePhase"] != 180.0 {
		t.Fatalf("expected routine-owned scalePhase to persist, got %v", state.RoutineStates["scalePhase"])
	}
}

func TestFrameMoveRoutineUpdatesVelocityThenPhysicsMovesOnce(t *testing.T) {
	dt := 1.0

	defs := group.NewPool()
	if err := defs.CreateObject(frameObjID, "ship", map[string]interface{}{
		"routines":  []string{"move"},
		"velocityX": 4.0,
	}); err != nil {
		t.Fatalf("CreateObject: %v", err)
	}

	rt := runtimepkg.NewRuntimeEnvironment(defs)
	if err := rt.InitializeObject(frameObjID, runtimepkg.OriginPosition(), runtimepkg.Vector3{}); err != nil {
		t.Fatalf("InitializeObject: %v", err)
	}

	frame := NewFrame(defs, rt, eventqueue.NewQueueManager(10), routines.NewLibrary(), dt)
	if err := frame.Process(); err != nil {
		t.Fatalf("Process: %v", err)
	}

	state, err := rt.GetObjectState(frameObjID)
	if err != nil {
		t.Fatalf("GetObjectState: %v", err)
	}

	if state.Velocity.X != 4.0 {
		t.Fatalf("expected velocity.X=4.0 after move routine, got %f", state.Velocity.X)
	}
	if state.Position.X != 4.0 {
		t.Fatalf("expected one integration step (position.X=4.0), got %f", state.Position.X)
	}
}

func TestFrameScaleRoutinePersistsScalePhaseWithoutPriorRoutineState(t *testing.T) {
	defs := group.NewPool()
	if err := defs.CreateObject(frameObjID, "ship", map[string]interface{}{
		"routines": []string{"scale"},
	}); err != nil {
		t.Fatalf("CreateObject: %v", err)
	}

	rt := runtimepkg.NewRuntimeEnvironment(defs)
	if err := rt.InitializeObject(frameObjID, runtimepkg.OriginPosition(), runtimepkg.Vector3{}); err != nil {
		t.Fatalf("InitializeObject: %v", err)
	}

	frame := NewFrame(defs, rt, eventqueue.NewQueueManager(10), routines.NewLibrary(), 1.0)
	if err := frame.executeRoutines(); err != nil {
		t.Fatalf("executeRoutines: %v", err)
	}

	state, err := rt.GetObjectState(frameObjID)
	if err != nil {
		t.Fatalf("GetObjectState: %v", err)
	}
	if _, exists := state.RoutineStates["scalePhase"]; !exists {
		t.Fatal("expected scalePhase to persist as built-in routine-owned state")
	}
}

func TestFrameProcessEventsUsesPerEventTransactionScope(t *testing.T) {
	defs := group.NewPool()
	if err := defs.CreateObject(frameObjID, "ship", nil); err != nil {
		t.Fatalf("CreateObject: %v", err)
	}

	rt := runtimepkg.NewRuntimeEnvironment(defs)
	if err := rt.InitializeObject(frameObjID, runtimepkg.OriginPosition(), runtimepkg.Vector3{}); err != nil {
		t.Fatalf("InitializeObject: %v", err)
	}

	frame := NewFrame(defs, rt, eventqueue.NewQueueManager(10), routines.NewLibrary(), 1.0/60.0)

	goodEvent := eventqueue.NewEvent(
		frameObjID,
		eventqueue.EventTypeUpdate,
		map[string]interface{}{
			"position": map[string]interface{}{"x": 10.0, "y": 0.0, "z": 0.0},
		},
		eventqueue.TransactionTypeNone,
	)
	if err := frame.eventQueue.Enqueue(goodEvent); err != nil {
		t.Fatalf("Enqueue good event: %v", err)
	}

	missingID := uuid.MustParse("33333333-3333-3333-3333-333333333333")
	failingRollbackEvent := eventqueue.NewEvent(
		missingID,
		eventqueue.EventTypeUpdate,
		map[string]interface{}{
			"position": map[string]interface{}{"x": 99.0, "y": 0.0, "z": 0.0},
		},
		eventqueue.TransactionTypeRollback,
	)
	if err := frame.eventQueue.Enqueue(failingRollbackEvent); err != nil {
		t.Fatalf("Enqueue failing rollback event: %v", err)
	}

	if err := frame.processEvents(); err != nil {
		t.Fatalf("processEvents: %v", err)
	}

	state, err := rt.GetObjectState(frameObjID)
	if err != nil {
		t.Fatalf("GetObjectState: %v", err)
	}
	if state.Position.X != 10.0 {
		t.Fatalf("expected successful event to remain applied despite later rollback failure, got %+v", state.Position)
	}
}

// TestFrameFullPipeline runs Process() end-to-end:
// an update event sets position, a rotate routine increments rotation,
// and physics integrates velocity.
func TestFrameFullPipeline(t *testing.T) {
	dt := 1.0

	defs := group.NewPool()
	_ = defs.CreateObject(frameObjID, "cube", map[string]interface{}{
		"routines": []string{"rotate"},
	})

	rt := runtimepkg.NewRuntimeEnvironment(defs)
	// Initialize with velocity.Y = 2.0; physics should advance Y by 2*dt.
	_ = rt.InitializeObject(frameObjID, runtimepkg.OriginPosition(), runtimepkg.Vector3{Y: 2.0})

	mgr := eventqueue.NewQueueManager(100)
	lib := routines.NewLibrary()

	// Enqueue update event: set X = 5.
	event := eventqueue.NewEvent(
		frameObjID,
		eventqueue.EventTypeUpdate,
		map[string]interface{}{
			"position": map[string]interface{}{"x": 5.0, "y": 0.0, "z": 0.0},
		},
		eventqueue.TransactionTypeNone,
	)
	_ = mgr.Enqueue(event)

	frame := NewFrame(defs, rt, mgr, lib, dt)

	if err := frame.Process(); err != nil {
		t.Fatalf("Process: %v", err)
	}

	state, err := rt.GetObjectState(frameObjID)
	if err != nil {
		t.Fatalf("GetObjectState: %v", err)
	}

	// After events: Position.X = 5.0
	if state.Position.X != 5.0 {
		t.Errorf("expected Position.X = 5.0, got %f", state.Position.X)
	}

	// After routines: Rotation.Z should be non-zero (rotate ran).
	if state.Rotation.Z == 0 {
		t.Errorf("expected Rotation.Z non-zero after rotate routine")
	}

	// After physics: Position.Y += Velocity.Y * dt = 0 + 2*1 = 2.0.
	// (Position.Y was set to 0 by the update event, so physics takes it to 2.0.)
	// Note: update event set Y=0, velocity=2.0; physics: Y += 2.0*1.0 = 2.0.
	if state.Position.Y != 2.0 {
		t.Errorf("expected Position.Y = 2.0 after physics, got %f", state.Position.Y)
	}
}
