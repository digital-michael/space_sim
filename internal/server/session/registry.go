package session

import (
	"errors"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Errors returned by the registry.
var (
	ErrCapacityExceeded = errors.New("session: capacity exceeded")
	ErrNotFound         = errors.New("session: session not found")
)

// SessionEventType classifies a change published to Subscribe() callers.
type SessionEventType int

const (
	SessionEventAdd    SessionEventType = iota // new session registered
	SessionEventUpdate                         // existing session position/POV changed
	SessionEventRemove                         // session unregistered
)

// SessionEvent is emitted by the registry whenever a session changes.
// Session is a copy of the post-change state; it is nil for Remove events.
// ID carries the session_id for Remove events.
type SessionEvent struct {
	Type    SessionEventType
	Session *ClientSession // non-nil for Add/Update
	ID      string         // set for Remove; also equals Session.SessionID for Add/Update
}

// Registry tracks all active client sessions.
// All methods are safe for concurrent use.
type Registry interface {
	// Register creates a new session. Returns ErrCapacityExceeded when full.
	Register(req RegisterRequest) (*ClientSession, error)
	// Unregister removes a session and releases its color slot. No-op if not found.
	Unregister(sessionID string)
	// Get returns a copy of the named session, or (nil, false) if not found.
	Get(sessionID string) (*ClientSession, bool)
	// All returns a snapshot copy of all sessions; safe to iterate.
	All() []*ClientSession
	// Count returns the number of currently registered sessions.
	Count() int
	// UpdatePosition stores the last-known world position for a session.
	// Returns ErrNotFound if the session does not exist.
	UpdatePosition(sessionID string, pos [3]float64) error
	// UpdatePOV stores the last-known point-of-view direction for a session.
	// Returns ErrNotFound if the session does not exist.
	UpdatePOV(sessionID string, pov [3]float32) error
	// Subscribe returns a buffered channel that receives session change events
	// and a cancel func that removes the subscription. The channel is closed
	// when cancel is called. Non-blocking: slow consumers may miss events.
	Subscribe() (<-chan SessionEvent, func())
}

// Config configures the session registry.
type Config struct {
	MaxSessions int    // default 100
	AdminSecret string // required when requesting ClientRoleAdmin; "" disables admin role
}

// DefaultConfig returns a Config with safe defaults.
func DefaultConfig() Config {
	return Config{MaxSessions: 100}
}

// NewRegistry creates a thread-safe in-memory session registry.
func NewRegistry(cfg Config) Registry {
	if cfg.MaxSessions <= 0 {
		cfg.MaxSessions = 100
	}
	return &inMemoryRegistry{
		cfg:      cfg,
		sessions: make(map[string]*ClientSession),
		colors:   make([]bool, 100),
	}
}

type inMemoryRegistry struct {
	cfg      Config
	mu       sync.RWMutex
	sessions map[string]*ClientSession
	colors   []bool // colors[i] == true means colorPalette[i] is in use

	subsMu sync.Mutex
	subs   []chan SessionEvent
}

func (r *inMemoryRegistry) Register(req RegisterRequest) (*ClientSession, error) {
	r.mu.Lock()

	if len(r.sessions) >= r.cfg.MaxSessions {
		r.mu.Unlock()
		return nil, ErrCapacityExceeded
	}

	// Resolve role: clamp to Player unless admin secret matches.
	role := req.Role
	switch role {
	case ClientRoleAdmin:
		if r.cfg.AdminSecret == "" || req.AdminSecret != r.cfg.AdminSecret {
			role = ClientRolePlayer
		}
	case ClientRoleNPC, ClientRoleUnspecified:
		role = ClientRolePlayer
	}

	label := req.Label
	if len(label) > 32 {
		label = label[:32]
	}
	if label == "" {
		label = "Player"
	}

	idx := r.allocColor()
	now := time.Now()
	sess := &ClientSession{
		SessionID:   uuid.NewString(),
		ClientUUID:  req.ClientUUID,
		Label:       label,
		Role:        role,
		Color:       colorPalette[idx],
		ConnectedAt: now,
		LastSeen:    now,
	}
	r.sessions[sess.SessionID] = sess
	r.colors[idx] = true
	snap := copySession(sess)
	r.mu.Unlock()
	r.notify(SessionEvent{Type: SessionEventAdd, Session: snap, ID: snap.SessionID})
	return snap, nil
}

func (r *inMemoryRegistry) Unregister(sessionID string) {
	r.mu.Lock()

	sess, ok := r.sessions[sessionID]
	if !ok {
		r.mu.Unlock()
		return
	}
	for i, c := range colorPalette {
		if c == sess.Color {
			r.colors[i] = false
			break
		}
	}
	delete(r.sessions, sessionID)
	r.mu.Unlock()
	r.notify(SessionEvent{Type: SessionEventRemove, ID: sessionID})
}

func (r *inMemoryRegistry) Get(sessionID string) (*ClientSession, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	sess, ok := r.sessions[sessionID]
	if !ok {
		return nil, false
	}
	return copySession(sess), true
}

func (r *inMemoryRegistry) All() []*ClientSession {
	r.mu.RLock()
	defer r.mu.RUnlock()

	out := make([]*ClientSession, 0, len(r.sessions))
	for _, s := range r.sessions {
		out = append(out, copySession(s))
	}
	return out
}

func (r *inMemoryRegistry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.sessions)
}

func (r *inMemoryRegistry) UpdatePosition(sessionID string, pos [3]float64) error {
	r.mu.Lock()
	sess, ok := r.sessions[sessionID]
	if !ok {
		r.mu.Unlock()
		return ErrNotFound
	}
	sess.Position = pos
	snap := copySession(sess)
	r.mu.Unlock()
	r.notify(SessionEvent{Type: SessionEventUpdate, Session: snap, ID: snap.SessionID})
	return nil
}

func (r *inMemoryRegistry) UpdatePOV(sessionID string, pov [3]float32) error {
	r.mu.Lock()
	sess, ok := r.sessions[sessionID]
	if !ok {
		r.mu.Unlock()
		return ErrNotFound
	}
	sess.POV = pov
	snap := copySession(sess)
	r.mu.Unlock()
	r.notify(SessionEvent{Type: SessionEventUpdate, Session: snap, ID: snap.SessionID})
	return nil
}

// Subscribe registers a listener for session change events. The returned
// channel receives events until the returned cancel func is called, which
// closes the channel. Slow consumers may drop events (non-blocking send).
func (r *inMemoryRegistry) Subscribe() (<-chan SessionEvent, func()) {
	ch := make(chan SessionEvent, 64)
	r.subsMu.Lock()
	r.subs = append(r.subs, ch)
	r.subsMu.Unlock()

	cancel := func() {
		r.subsMu.Lock()
		for i, s := range r.subs {
			if s == ch {
				r.subs = append(r.subs[:i], r.subs[i+1:]...)
				break
			}
		}
		r.subsMu.Unlock()
		close(ch)
	}
	return ch, cancel
}

// notify sends e to all currently registered subscriber channels.
// Uses non-blocking sends; slow consumers miss events.
func (r *inMemoryRegistry) notify(e SessionEvent) {
	r.subsMu.Lock()
	subs := make([]chan SessionEvent, len(r.subs))
	copy(subs, r.subs) //nolint:govet // shadow ok: builtin copy
	r.subsMu.Unlock()
	for _, ch := range subs {
		select {
		case ch <- e:
		default:
		}
	}
}

// allocColor returns the first available color index. Caller must hold mu.
func (r *inMemoryRegistry) allocColor() int {
	for i, used := range r.colors {
		if !used {
			return i
		}
	}
	return 0 // should not reach here; capacity guard prevents overflow
}

// copySession returns a shallow copy to prevent callers from mutating state.
func copySession(s *ClientSession) *ClientSession {
	c := *s
	return &c
}
