package eventqueue

import (
	"testing"

	"github.com/google/uuid"
)

func TestEventCreationAndValidation(t *testing.T) {
	guid := uuid.New()
	payload := map[string]interface{}{"key": "value"}

	event := NewEvent(guid, EventTypeCreate, payload, TransactionTypeNone)
	if event.ID == uuid.Nil {
		t.Fatal("expected event ID to be generated")
	}
	if event.GUID != guid {
		t.Fatalf("expected GUID %s, got %s", guid, event.GUID)
	}
	if event.Type != EventTypeCreate {
		t.Fatalf("expected type Create, got %s", event.Type)
	}
	if event.TransactionType != TransactionTypeNone {
		t.Fatalf("expected transaction type None, got %s", event.TransactionType)
	}

	if err := event.Validate(); err != nil {
		t.Fatalf("validation failed: %v", err)
	}
}

func TestEventValidationFailures(t *testing.T) {
	guid := uuid.New()

	testCases := []struct {
		name  string
		event *Event
	}{
		{
			name: "nil ID",
			event: &Event{
				ID:              uuid.Nil,
				GUID:            guid,
				Type:            EventTypeCreate,
				TransactionType: TransactionTypeNone,
			},
		},
		{
			name: "nil GUID",
			event: &Event{
				ID:              uuid.New(),
				GUID:            uuid.Nil,
				Type:            EventTypeCreate,
				TransactionType: TransactionTypeNone,
			},
		},
		{
			name: "empty type",
			event: &Event{
				ID:              uuid.New(),
				GUID:            guid,
				Type:            "",
				TransactionType: TransactionTypeNone,
			},
		},
		{
			name: "empty transaction type",
			event: &Event{
				ID:              uuid.New(),
				GUID:            guid,
				Type:            EventTypeCreate,
				TransactionType: "",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if err := tc.event.Validate(); err == nil {
				t.Fatalf("expected validation to fail for %s", tc.name)
			}
		})
	}
}

func TestEventClone(t *testing.T) {
	guid := uuid.New()
	payload := map[string]interface{}{"data": 123}
	event := NewEvent(guid, EventTypeUpdate, payload, TransactionTypeRollback)

	clone := event.Clone()
	if clone == nil {
		t.Fatal("expected clone to be non-nil")
	}
	if clone == event {
		t.Fatal("expected clone to be a different pointer")
	}
	if clone.ID != event.ID || clone.GUID != event.GUID {
		t.Fatal("expected cloned IDs to match")
	}

	clone.Payload["data"] = 999
	if event.Payload["data"] == 999 {
		t.Fatal("expected payload to be deep-copied")
	}
}

func TestEventTypesAndTransactionTypes(t *testing.T) {
	guid := uuid.New()

	eventTypes := []EventType{EventTypeCreate, EventTypeUpdate, EventTypeDelete, EventTypeMove, EventTypeCustom}
	for _, et := range eventTypes {
		event := NewEvent(guid, et, nil, TransactionTypeNone)
		if event.Type != et {
			t.Fatalf("expected type %s, got %s", et, event.Type)
		}
	}

	txTypes := []TransactionType{TransactionTypeNone, TransactionTypeBestEffort, TransactionTypeRollback}
	for _, tx := range txTypes {
		event := NewEvent(guid, EventTypeCreate, nil, tx)
		if event.TransactionType != tx {
			t.Fatalf("expected transaction type %s, got %s", tx, event.TransactionType)
		}
	}
}
