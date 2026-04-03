package eventqueue

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

type EventType string

const (
	EventTypeCreate EventType = "create"
	EventTypeUpdate EventType = "update"
	EventTypeDelete EventType = "delete"
	EventTypeMove   EventType = "move"
	EventTypeCustom EventType = "custom"
)

type TransactionType string

const (
	TransactionTypeNone       TransactionType = "none"
	TransactionTypeBestEffort TransactionType = "best_effort"
	TransactionTypeRollback   TransactionType = "rollback"
)

type Event struct {
	ID              uuid.UUID
	GUID            uuid.UUID
	Type            EventType
	Payload         map[string]interface{}
	TransactionType TransactionType
	Priority        int
	EnqueuedAt      time.Time
}

func NewEvent(guid uuid.UUID, eventType EventType, payload map[string]interface{}, txType TransactionType) *Event {
	return &Event{
		ID:              uuid.New(),
		GUID:            guid,
		Type:            eventType,
		Payload:         clonePayload(payload),
		TransactionType: txType,
		Priority:        0,
		EnqueuedAt:      time.Now(),
	}
}

func (e *Event) Validate() error {
	if e.ID == uuid.Nil {
		return fmt.Errorf("event ID cannot be nil")
	}
	if e.GUID == uuid.Nil {
		return fmt.Errorf("event GUID cannot be nil")
	}
	if e.Type == "" {
		return fmt.Errorf("event type cannot be empty")
	}
	if e.TransactionType == "" {
		return fmt.Errorf("transaction type cannot be empty")
	}
	return nil
}

func (e *Event) Clone() *Event {
	if e == nil {
		return nil
	}
	return &Event{
		ID:              e.ID,
		GUID:            e.GUID,
		Type:            e.Type,
		Payload:         clonePayload(e.Payload),
		TransactionType: e.TransactionType,
		Priority:        e.Priority,
		EnqueuedAt:      e.EnqueuedAt,
	}
}

func clonePayload(payload map[string]interface{}) map[string]interface{} {
	if payload == nil {
		return map[string]interface{}{}
	}
	clone := make(map[string]interface{}, len(payload))
	for k, v := range payload {
		clone[k] = v
	}
	return clone
}
