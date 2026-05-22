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
}

func (r *inMemoryRegistry) Register(req RegisterRequest) (*ClientSession, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if len(r.sessions) >= r.cfg.MaxSessions {
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
	return copySession(sess), nil
}

func (r *inMemoryRegistry) Unregister(sessionID string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	sess, ok := r.sessions[sessionID]
	if !ok {
		return
	}
	for i, c := range colorPalette {
		if c == sess.Color {
			r.colors[i] = false
			break
		}
	}
	delete(r.sessions, sessionID)
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
