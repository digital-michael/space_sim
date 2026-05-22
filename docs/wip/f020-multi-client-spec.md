# F-020 — Multi-Client gRPC Session Layer

## Purpose

Define the session layer that allows up to 100 concurrent REPL clients to connect to a
single `space-sim-grpc` binary (Option A in-process model) and participate in the same
simulated world simultaneously. This feature is intentionally decoupled from IAAM (F-011):
identity is self-asserted and stub-based in early phases; the IAAM integration slot is
reserved but not implemented here.

Read this alongside:
- [`docs/standards/agent-readme.md`](../standards/agent-readme.md) — architectural boundaries
- [`docs/standards/coding-standards.md`](../standards/coding-standards.md) — implementation rules
- [`docs/wip/f021-physical-marker-spec.md`](f021-physical-marker-spec.md) — visual representation
- [`docs/wip/f022-client-movement-spec.md`](f022-client-movement-spec.md) — movement and physics
- [`docs/wip/f013-nbody-plan.md`](f013-nbody-plan.md) — gravity dependency

## Last Updated
2026-05-11

## Status
📋 Not started

---

## 1. Goals

| # | Goal |
|---|------|
| G1 | Allow up to 100 concurrent REPL clients to connect to one `space-sim-grpc` process |
| G2 | Each client has a stable identity for its session duration (pre-IAAM: self-asserted) |
| G3 | Each client owns: a world position, a POV vector, a physical marker, a display label, and a role |
| G4 | The server tracks all connected clients in a session registry |
| G5 | Concurrent commands from multiple clients resolve without silent corruption |
| G6 | IAAM-sourced UUIDs replace stub UUIDs when F-011 ships; no structural changes required |

---

## 2. Non-Goals (this feature)

- Full IAAM authentication (F-011 scope)
- Option B split into separate server/client binaries (F-010 scope)
- NPC behavior automation (deferred to F-022)
- Ship-to-ship collision detection and damage system (deferred to F-027)
- Ship-to-ship communications / messaging (deferred to F-025)
- Audio event system (deferred to F-026)
- Bandwidth optimization / frustum filtering (deferred; roadmap item — see F-020 §9)
- Client-side interpolation between snapshots (deferred)

---

## 3. Client Identity Model

### 3.1 ClientSession fields

| Field | Type | Source | Notes |
|-------|------|--------|-------|
| `SessionID` | `string` (UUID v4) | Server-generated on connect | Stable for session lifetime |
| `ClientUUID` | `string` (UUID v4) | Self-asserted by client at connect | Future: validated token from IAAM |
| `Label` | `string` | Self-asserted; max 32 UTF-8 chars | Display name shown in HUD |
| `Role` | `ClientRole` enum | Self-asserted with server default | Server may downgrade; see §5 |
| `Color` | `[3]uint8` (RGB) | Server-assigned from palette | Used by physical marker (F-021) |
| `Position` | `[3]float64` | Server-authoritative | World position in sim units |
| `POV` | `[3]float32` | Client-authoritative | Unit direction vector, camera forward |
| `MarkerRef` | `string` | Config/future catalog | Marker model path; phase 1 = empty |
| `ConnectedAt` | `time.Time` | Server-recorded | Session audit |
| `LastSeen` | `time.Time` | Server-updated on any RPC | Used for idle detection |
| `DamageRating` | `float32` (0.0–1.0) | Server-maintained | 0.0 = undamaged; 1.0 = destroyed; durable; see F-027 |
| `SpawnRef` | `string` | Server-computed at register | Name of the body or belt the client spawned near/in |

### 3.2 ClientRole enum

```proto
enum ClientRole {
  CLIENT_ROLE_UNSPECIFIED = 0;
  CLIENT_ROLE_PLAYER      = 1;
  CLIENT_ROLE_NPC         = 2;
  CLIENT_ROLE_ADMIN       = 3;
  CLIENT_ROLE_OTHER       = 4;
}
```

Role enforcement rules (pre-IAAM):
- Any client may request any role.
- Server clamps to `PLAYER` by default.
- `ADMIN` role requires the connecting client to supply a shared secret in the `RegisterClient`
  request metadata. Secret is configured in `configs/app.json` under `"admin_secret"`.
  This is a stop-gap; IAAM replaces it in F-011.
- `NPC` role is reserved for server-managed automated clients; see F-022.

### 3.3 Color palette

The server maintains a fixed palette of 100 visually distinct RGB colors
(deterministic, based on HSL hue rotation with constant saturation and lightness).
Colors are assigned in registration order; released on disconnect; reused FIFO.
The palette guarantees no two simultaneously connected clients share a color.

---

## 4. Session Registry

### 4.1 Location

New package: `internal/server/session/`

This keeps session concerns isolated from transport and from the simulation engine, preserving
the architectural boundary defined in `agent-readme.md` §3.2.

### 4.2 Registry interface

```go
// Registry tracks all active client sessions.
type Registry interface {
    Register(req RegisterRequest) (*ClientSession, error)
    Unregister(sessionID string)
    Get(sessionID string) (*ClientSession, bool)
    All() []*ClientSession           // snapshot copy; safe to iterate
    Count() int
    UpdatePosition(sessionID string, pos [3]float64) error
    UpdatePOV(sessionID string, pov [3]float32) error
}
```

- `Register` returns `ErrCapacityExceeded` when `Count() >= 100`.
- All methods are safe for concurrent use (internal RWMutex).
- `All()` returns a value snapshot; callers must not mutate returned sessions.

### 4.3 RegisterRequest

```go
type RegisterRequest struct {
    ClientUUID string
    Label      string
    Role       ClientRole
    AdminSecret string // empty unless requesting ADMIN role
}
```

### 4.4 Lifecycle

```
Client connects → RegisterClient RPC → Registry.Register() → SessionID returned
Client operates → UpdatePosition / UpdatePOV RPCs update registry
Client disconnects (RPC stream ends or timeout) → Registry.Unregister(sessionID)
Idle client (no RPC for > idle_timeout) → server sends Ping; if no response → Unregister
```

Default `idle_timeout`: 30 seconds (configurable in `configs/app.json`).

---

## 5. Proto Extensions

New RPCs in `api/proto/spacesim/v1/`:

```proto
service SessionService {
  // Register a new client session. Returns session metadata including server-assigned SessionID and Color.
  rpc RegisterClient(RegisterClientRequest) returns (RegisterClientResponse);

  // Graceful disconnect. Registry immediately releases the session and color slot.
  rpc UnregisterClient(UnregisterClientRequest) returns (UnregisterClientResponse);

  // Bidirectional stream: client pushes position+POV updates; server pushes registry snapshot deltas.
  rpc SessionStream(stream ClientUpdate) returns (stream SessionDelta);

  // One-shot query: returns all currently registered sessions (admin or own session).
  rpc ListSessions(ListSessionsRequest) returns (ListSessionsResponse);
}
```

`SessionDelta` contains only the changed sessions (add/update/remove) rather than the full
registry each tick. Reduces wire volume at scale.

---

## 6. Conflict Resolution Policy

Multiple clients sending simulation commands concurrently is intentional. The existing
`AppCmd` channel already serializes commands through the application loop (last-write-wins).
This is acceptable for simulation-level commands (speed, pause, load-world) because:

1. Two clients racing to set simulation speed is no worse than a single user double-pressing.
2. The simulation is server-authoritative; all clients observe the same resulting state.

For **client-local** state (position, POV) there is no conflict: each client owns its own
session and the registry is keyed by `SessionID`.

Explicit policy decisions:

| Command type | Resolution | Rationale |
|-------------|------------|-----------|
| `SetSpeed`, `Pause`, `Resume` | Last-write-wins via AppCmd | Acceptable; state is visible to all |
| `LoadSystem` | Admin-role only; others rejected with `ErrPermissionDenied` | Prevents accidental world reloads |
| `SetPosition` (own session) | Always accepted; no conflict possible | Per-session ownership |
| `SetPosition` (other session) | Admin-role only | Teleport / moderation use case |
| `KickClient` | Admin-role only | Session management |

---

## 7. Files to Touch

### Phase 1 files

| File | Action | Notes |
|------|--------|-------|
| `internal/server/session/session.go` | **Create** | `ClientSession`, `ClientRole`, `RegisterRequest`, `ShipProfile`, color palette types |
| `internal/server/session/registry.go` | **Create** | `Registry` interface + `inMemoryRegistry` impl; RWMutex; capacity guard |
| `internal/server/session/registry_test.go` | **Create** | Capacity, concurrent register/unregister, color reuse, idle timeout |
| `api/proto/spacesim/v1/session.proto` | **Create** | `SessionService`, `RegisterClientRequest/Response`, `ListSessionsRequest/Response`, `ClientRole` enum |
| `api/gen/spacesim/v1/` | **Regenerate** | `make proto` after proto change |
| `internal/transport/grpc/session_handler.go` | **Create** | `RegisterClient`, `UnregisterClient`, `ListSessions` handlers; role enforcement |
| `internal/transport/grpc/session_handler_test.go` | **Create** | Handler unit tests via bufconn |
| `internal/client/commands/commands.go` | **Modify** | Add `SessionRegisterCmd`, `SessionUnregisterCmd`, `SessionListCmd` types |
| `internal/client/repl/repl.go` | **Modify** | Add `register`, `unregister`, `list sessions` dispatch cases |
| `configs/app.json` | **Modify** | Add `"admin_secret"`, `"idle_timeout_seconds"`, `"max_sessions"` (default 100) |

### Phase 2 files

| File | Action | Notes |
|------|--------|-------|
| `internal/server/session/registry.go` | **Modify** | Add `UpdatePosition`, `UpdatePOV` to interface + impl |
| `internal/server/session/registry_test.go` | **Modify** | Add position/POV update tests |
| `api/proto/spacesim/v1/session.proto` | **Modify** | Add `SessionStream` RPC, `ClientUpdate`, `SessionDelta` messages |
| `api/gen/spacesim/v1/` | **Regenerate** | `make proto` |
| `internal/transport/grpc/session_handler.go` | **Modify** | Add `SessionStream` bidirectional stream handler |
| `internal/protocol/snapshot.go` | **Modify** | Add `ClientSessions []ClientSessionSummary` field to `WorldSnapshot` |
| `internal/client/go/raylib/app/app.go` | **Modify** | Consume `ClientSessions` from snapshot; expose to marker renderer |

### Phase 3 files

| File | Action | Notes |
|------|--------|-------|
| `api/proto/spacesim/v1/session.proto` | **Modify** | Add `KickClient`, `TeleportClient` RPCs |
| `api/gen/spacesim/v1/` | **Regenerate** | `make proto` |
| `internal/transport/grpc/session_handler.go` | **Modify** | Add kick/teleport handlers with admin role guard |
| `internal/client/commands/commands.go` | **Modify** | Add `SessionKickCmd`, `SessionTeleportCmd` |
| `internal/client/repl/repl.go` | **Modify** | Add `session kick`, `session teleport` dispatch |
| `internal/persist/eventlog.go` | **Modify** | Add admin action event type; write on kick/teleport |

---

## 8. Phases

### Phase 1 — Session Registry + Identity + Proto

**Architectural layer**: Server session layer (`internal/server/session/`), wire protocol (`api/proto/`), transport layer (`internal/transport/grpc/`), REPL client (`internal/client/`).
**Prerequisites**: None — fully independent of all other features.

**Value**: The server can distinguish clients; each client has a name, color, and role.
Physical markers become possible.

Work items:
- [ ] Add `internal/server/session/` package with `Registry`, `ClientSession`, `ClientRole`
- [ ] Add `SessionService` proto + generated Go stubs
- [ ] Add `session_handler.go` in `internal/transport/grpc/`
- [ ] Wire `RegisterClient` / `UnregisterClient` RPCs to registry
- [ ] Assign color from palette on register; release on unregister
- [ ] Add `RegisterClient` / `UnregisterClient` commands to REPL
- [ ] Add `list sessions` REPL command (maps to `ListSessions` RPC)
- [ ] Unit tests: registry capacity, concurrent register/unregister, color reuse
- [ ] Integration test: two REPL clients register and appear in `list sessions`

Acceptance criteria:
- 100 concurrent `RegisterClient` calls succeed; the 101st returns capacity error ✓
- `list sessions` output shows all connected clients with name, role, color, position ✓
- Disconnecting a client releases its color for reuse ✓
- Race detector passes ✓

### Phase 2 — Position and POV Streaming

**Architectural layer**: Server session layer, wire protocol, transport layer, Raylib app layer (`internal/client/go/raylib/app/`), protocol snapshot (`internal/protocol/`).
**Prerequisites**: Phase 1 complete.

**Value**: The server tracks each client's world position and view direction. Physical
markers can be placed accurately in the scene.

Work items:
- [ ] Add `UpdatePosition` / `UpdatePOV` RPCs to `SessionService`
- [ ] Add `SessionStream` bidirectional stream handler
- [ ] REPL client sends position+POV on each movement command
- [ ] Registry persists last-known position and POV
- [ ] `WorldSnapshot` includes connected client sessions (added to snapshot proto)
- [ ] Raylib renderer reads client sessions from snapshot; passes to marker renderer (F-021 hook)

Acceptance criteria:
- Client's position visible in `list sessions` after a `nav jump` command ✓
- Two clients can move independently and their positions do not interfere ✓

### Phase 3 — Admin Controls + Conflict Policy Enforcement ✅ COMPLETE (2026-05-22)

**Architectural layer**: Wire protocol, transport layer, REPL client, persistence layer (`internal/persist/`).
**Prerequisites**: Phase 1 complete. Phase 2 not required.

**Value**: Admin clients can view all sessions, kick misbehaving clients, and teleport
any session.

Work items:
- [x] Add `KickClient` and `TeleportClient` RPCs (admin-role only)
- [x] Enforce role-gated commands in `session_handler.go`
- [ ] `LoadSystem` gated to admin role — deferred; touches simulation command pipeline
- [x] Admin REPL commands: `session kick <id>`, `session teleport <id> <body>`
- [x] Audit log entries for admin actions (writes to event log via `persist/eventlog`)

Acceptance criteria:
- Non-admin client receives `ErrPermissionDenied` on `LoadSystem` — deferred (see note above)
- Admin `kick` removes the target from registry and closes their stream ✓
- Audit log shows kick/teleport events ✓

### Phase 4 — IAAM Slot (stub, reserved)

This phase does nothing today. When F-011 ships, `ClientUUID` validation moves here:
`RegisterClientRequest` carries a bearer token; the session handler validates it against
the IAAM service and extracts the canonical UUID and role. No structural changes to the
registry or proto shape are expected.

---

## 8. Dependencies

| Feature | Relationship |
|---------|-------------|
| F-021: Physical Marker | Consumes `ClientSession.Color`, `ClientSession.Position`, `MarkerRef` |
| F-022: Client Movement | Consumes `SessionService` to update position and POV |
| F-023: Keyboard Config | Drives the movement commands that call `UpdatePosition` |
| F-013: N-Body | Client ships become simulation objects; gravity applied to their positions |
| F-011: IAAM | Phase 4 token validation slot; no earlier dependency |
| F-010: Option B Split | Registry moves to `space-sim-server`; client side unchanged |
| F-008: Artifact Type | Client physical markers are artifact-type objects (Phase 2+) |
| F-025: Ship-to-Ship Comms | Messaging uses session registry to route messages by SessionID |
| F-027: Ship Collision/Damage | Collision events update `DamageRating` in the session registry |

---

## 9. Open Questions

| # | Question | Decision | Status |
|---|----------|----------|---------|
| Q1 | Should `SessionID` be stable across reconnects (tied to `ClientUUID`)? | TBD | Open — Phase 2 |
| Q2 | Should client position persist to disk on disconnect? | Possible via IAAM external persistence profile; to be evaluated when F-011 design starts | Deferred |
| Q3 | What is the default spawn position for new clients? | **Resolved**: random orbit around any named body (star, planet, dwarf planet, moon) or inside any asteroid belt. Server picks randomly at register time using the loaded system's body list. | Closed |
| Q4 | Should NPC clients (role=NPC) be server-instantiated or remotely registered? | Server-instantiated. External AI agents may drive them via a future MCP/REPL bridge (F-022 Phase 3 extended). | Deferred |
| Q5 | Maximum label length: 32 chars sufficient? | TBD | Open — Phase 1 |
| Q6 | Bandwidth / frustum filtering (100 clients × full WorldSnapshot) | **Deferred**. Asteroid volume scaling and movement speed changes help. POV frustum filtering is the key mitigation. Tackle before double-digit concurrent users. | Deferred |
| Q7 | Client-side interpolation between received snapshots | **Deferred**. Needed for smooth marker animation at reduced stream rates; not blocking Phase 1–2. | Deferred |
