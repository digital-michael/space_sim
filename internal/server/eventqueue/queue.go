package eventqueue

import (
"errors"
"fmt"
"sync"

"github.com/google/uuid"
)

var ErrQueueFull = errors.New("queue is full")

type GUIDQueue struct {
mu       sync.Mutex
events   []*Event
capacity int
guid     uuid.UUID
}

func NewGUIDQueue(guid uuid.UUID, capacity int) *GUIDQueue {
if capacity <= 0 {
capacity = 1000
}
return &GUIDQueue{
events:   make([]*Event, 0, capacity),
capacity: capacity,
guid:     guid,
}
}

func (q *GUIDQueue) Enqueue(event *Event) error {
q.mu.Lock()
defer q.mu.Unlock()

if len(q.events) >= q.capacity {
return ErrQueueFull
}
if event == nil {
return fmt.Errorf("cannot enqueue nil event")
}
if event.GUID != q.guid {
return fmt.Errorf("event GUID %s does not match queue GUID %s", event.GUID, q.guid)
}

q.events = append(q.events, event.Clone())
return nil
}

func (q *GUIDQueue) Dequeue() (*Event, error) {
q.mu.Lock()
defer q.mu.Unlock()

if len(q.events) == 0 {
return nil, fmt.Errorf("queue is empty")
}

event := q.events[0]
q.events = q.events[1:]
return event, nil
}

func (q *GUIDQueue) Peek() (*Event, error) {
q.mu.Lock()
defer q.mu.Unlock()

if len(q.events) == 0 {
return nil, fmt.Errorf("queue is empty")
}

return q.events[0].Clone(), nil
}

func (q *GUIDQueue) Len() int {
q.mu.Lock()
defer q.mu.Unlock()

return len(q.events)
}

func (q *GUIDQueue) Full() bool {
q.mu.Lock()
defer q.mu.Unlock()

return len(q.events) >= q.capacity
}

func (q *GUIDQueue) DequeueAll() []*Event {
q.mu.Lock()
defer q.mu.Unlock()

result := make([]*Event, len(q.events))
for i, e := range q.events {
result[i] = e.Clone()
}
q.events = q.events[:0]
return result
}
