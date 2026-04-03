package eventqueue

import (
	"testing"

	"github.com/digital-michael/space_sim/internal/server/pool/group"
	runtimepkg "github.com/digital-michael/space_sim/internal/server/runtime"
	"github.com/google/uuid"
)

func TestTransactionContextExecuteNone(t *testing.T) {
	definitions := group.NewPool()
	objectID := uuid.New()
	if err := definitions.CreateObject(objectID, "test", map[string]interface{}{}); err != nil {
		t.Fatalf("create object definition failed: %v", err)
	}

	environment := runtimepkg.NewRuntimeEnvironment(definitions)
	if err := environment.InitializeObject(objectID, nil, runtimepkg.Vector3{}); err != nil {
		t.Fatalf("initialize object failed: %v", err)
	}

	transactionContext := NewTransactionContext(TransactionTypeNone)
	transactionContext.AddEvent(NewEvent(objectID, EventTypeUpdate, map[string]interface{}{
		"position": map[string]interface{}{"x": 1.0, "y": 2.0, "z": 3.0},
	}, TransactionTypeNone))

	if err := transactionContext.Execute(environment); err != nil {
		t.Fatalf("execute none failed: %v", err)
	}

	state, err := environment.GetObjectState(objectID)
	if err != nil {
		t.Fatalf("get state failed: %v", err)
	}
	if state.Position.X != 1.0 || state.Position.Y != 2.0 || state.Position.Z != 3.0 {
		t.Fatalf("unexpected position: %+v", state.Position)
	}
}

func TestTransactionContextBestEffortContinues(t *testing.T) {
	definitions := group.NewPool()
	goodID := uuid.New()
	if err := definitions.CreateObject(goodID, "test", map[string]interface{}{}); err != nil {
		t.Fatalf("create object definition failed: %v", err)
	}

	environment := runtimepkg.NewRuntimeEnvironment(definitions)
	if err := environment.InitializeObject(goodID, nil, runtimepkg.Vector3{}); err != nil {
		t.Fatalf("initialize object failed: %v", err)
	}

	missingID := uuid.New()

	transactionContext := NewTransactionContext(TransactionTypeBestEffort)
	transactionContext.AddEvent(NewEvent(missingID, EventTypeUpdate, map[string]interface{}{
		"position": map[string]interface{}{"x": 99.0, "y": 99.0, "z": 99.0},
	}, TransactionTypeBestEffort))
	transactionContext.AddEvent(NewEvent(goodID, EventTypeMove, map[string]interface{}{
		"dx": 5.0,
		"dy": 0.0,
		"dz": 0.0,
	}, TransactionTypeBestEffort))

	if err := transactionContext.Execute(environment); err != nil {
		t.Fatalf("best-effort should continue without fatal error, got: %v", err)
	}

	state, err := environment.GetObjectState(goodID)
	if err != nil {
		t.Fatalf("get state failed: %v", err)
	}
	if state.Position.X != 5.0 {
		t.Fatalf("expected X position 5.0, got %f", state.Position.X)
	}
}

func TestTransactionContextRollbackRestoresState(t *testing.T) {
	definitions := group.NewPool()
	goodID := uuid.New()
	if err := definitions.CreateObject(goodID, "test", map[string]interface{}{}); err != nil {
		t.Fatalf("create object definition failed: %v", err)
	}

	environment := runtimepkg.NewRuntimeEnvironment(definitions)
	if err := environment.InitializeObject(goodID, nil, runtimepkg.Vector3{}); err != nil {
		t.Fatalf("initialize object failed: %v", err)
	}
	if err := environment.UpdateObjectState(goodID, func(state *runtimepkg.ObjectState) {
		state.Position = runtimepkg.Vector3{X: 2, Y: 0, Z: 0}
	}); err != nil {
		t.Fatalf("seed state failed: %v", err)
	}

	missingID := uuid.New()

	transactionContext := NewTransactionContext(TransactionTypeRollback)
	transactionContext.AddEvent(NewEvent(goodID, EventTypeMove, map[string]interface{}{
		"dx": 3.0,
		"dy": 0.0,
		"dz": 0.0,
	}, TransactionTypeRollback))
	transactionContext.AddEvent(NewEvent(missingID, EventTypeUpdate, map[string]interface{}{
		"position": map[string]interface{}{"x": 10.0, "y": 0.0, "z": 0.0},
	}, TransactionTypeRollback))

	if err := transactionContext.Execute(environment); err == nil {
		t.Fatal("expected rollback transaction to fail")
	}

	state, err := environment.GetObjectState(goodID)
	if err != nil {
		t.Fatalf("get state failed: %v", err)
	}
	if state.Position.X != 2.0 {
		t.Fatalf("expected rollback to restore X=2.0, got %f", state.Position.X)
	}
}

func TestApplyEventCreateRequiresDefinition(t *testing.T) {
	definitions := group.NewPool()
	environment := runtimepkg.NewRuntimeEnvironment(definitions)

	objectID := uuid.New()
	transactionContext := NewTransactionContext(TransactionTypeNone)
	transactionContext.AddEvent(NewEvent(objectID, EventTypeCreate, nil, TransactionTypeNone))

	if err := transactionContext.Execute(environment); err == nil {
		t.Fatal("expected create without definition to fail")
	}
}