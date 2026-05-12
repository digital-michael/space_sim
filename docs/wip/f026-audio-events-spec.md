# F-026 — Audio Events

## Purpose

Introduce a Raylib-backed audio manager that plays short sound cues tied to simulation
and game events. Audio is client-local: each Raylib client decides which events trigger
sound; the server sends no audio data.

Read this alongside:
- [`docs/standards/agent-readme.md`](../standards/agent-readme.md) — architectural layers
- [`docs/standards/coding-standards.md`](../standards/coding-standards.md)
- [`docs/wip/f027-collision-damage-spec.md`](f027-collision-damage-spec.md) — collision events trigger audio
- [`docs/wip/f025-ship-comms-spec.md`](f025-ship-comms-spec.md) — message-received event triggers audio
- [`docs/wip/f024-multiplayer-hud-spec.md`](f024-multiplayer-hud-spec.md) — proximity alert triggers audio

## Last Updated
2026-05-11

## Status
📋 Not started

---

## 1. Goals

| # | Goal |
|---|------|
| G1 | Short sound cues for the most important game events, starting with proximity and collision |
| G2 | Audio is purely client-side; zero server changes required |
| G3 | All audio can be disabled via `configs/app.json`; no audio dependency on game logic |
| G4 | Volume and per-event enable/disable are configurable without restarting |
| G5 | No external audio library beyond Raylib's built-in audio (`rl.InitAudioDevice`) |

---

## 2. Non-Goals (this feature)

- Ship-to-ship voice audio (F-025 Phase 3 scope)
- Procedural / synthesized audio (future)
- Positional / 3D spatialized audio (future; Raylib supports it but adds complexity)
- Background music system (future)

---

## 3. Event Vocabulary

| Event ID | Trigger | Default sound |
|----------|---------|---------------|
| `audio.proximity_alert` | Another client within proximity threshold (F-024) | Short ping |
| `audio.collision_impact` | Client-to-client or client-to-body impact (F-027) | Thud/clang |
| `audio.thruster_start` | Thrust input begins | Low engine hum (one-shot) |
| `audio.thruster_stop` | Thrust input ends | Hum fade (one-shot) |
| `audio.warp` | Superluminal jump executes | Whoosh |
| `audio.message_received` | Incoming `ShipMessage` text (F-025) | Soft chime |
| `audio.client_join` | New session registered | Brief positive tone |
| `audio.client_leave` | Session unregistered | Brief neutral tone |
| `audio.damage_critical` | `DamageRating` crosses 0.75 threshold | Warning klaxon |
| `audio.destroyed` | `DamageRating` reaches 1.0 | Explosion |

---

## 4. Architecture

### 4.1 Layer

Audio is a client concern only. The `AudioManager` lives in
`internal/client/go/raylib/audio/` — a new package at the Raylib client layer.
It must never be imported by `internal/sim`, `internal/server`, or `internal/api`.

### 4.2 AudioManager interface

```go
// AudioManager plays event-triggered sound cues.
// All methods are safe to call from the render loop.
type AudioManager interface {
    Play(event AudioEvent)
    SetVolume(event AudioEvent, volume float32) // 0.0–1.0
    SetEnabled(event AudioEvent, enabled bool)
    SetMasterVolume(volume float32)
    Shutdown()
}
```

### 4.3 AudioEvent enum

```go
type AudioEvent int

const (
    AudioEventProximityAlert AudioEvent = iota
    AudioEventCollisionImpact
    AudioEventThrusterStart
    AudioEventThrusterStop
    AudioEventWarp
    AudioEventMessageReceived
    AudioEventClientJoin
    AudioEventClientLeave
    AudioEventDamageCritical
    AudioEventDestroyed
    audioEventCount // sentinel; used to size arrays and validate enum stability
)
```

A regression test locks in the count and ordering.

### 4.4 Sound file manifest

Sound files live in `data/assets/sounds/`. The manifest is declared as a Go `[audioEventCount]string`
array constant (not a config file) since event-to-sound mapping is stable code behavior,
not user data:

```go
var defaultSoundPaths = [audioEventCount]string{
    "data/assets/sounds/proximity_alert.wav",
    "data/assets/sounds/collision_impact.wav",
    "data/assets/sounds/thruster_start.wav",
    "data/assets/sounds/thruster_stop.wav",
    "data/assets/sounds/warp.wav",
    "data/assets/sounds/message_received.wav",
    "data/assets/sounds/client_join.wav",
    "data/assets/sounds/client_leave.wav",
    "data/assets/sounds/damage_critical.wav",
    "data/assets/sounds/destroyed.wav",
}
```

Missing sound files log a warning and produce a no-op entry; they do not crash the app.
Placeholder `.wav` files (1-frame silence) are committed so the app loads cleanly from
the start.

### 4.5 Config

New fields in `configs/app.json`:

```json
"audio": {
  "enabled": true,
  "master_volume": 0.8,
  "events": {
    "proximity_alert":  { "enabled": true,  "volume": 1.0 },
    "collision_impact": { "enabled": true,  "volume": 1.0 },
    "thruster_start":   { "enabled": true,  "volume": 0.5 },
    "thruster_stop":    { "enabled": true,  "volume": 0.5 },
    "warp":             { "enabled": true,  "volume": 1.0 },
    "message_received": { "enabled": true,  "volume": 0.8 },
    "client_join":      { "enabled": true,  "volume": 0.6 },
    "client_leave":     { "enabled": true,  "volume": 0.6 },
    "damage_critical":  { "enabled": true,  "volume": 1.0 },
    "destroyed":        { "enabled": true,  "volume": 1.0 }
  }
}
```

When `"enabled": false` at the top level, `rl.InitAudioDevice` is never called.
Per-event `"enabled": false` means `Play(event)` is a no-op for that event.

---

## 5. Files to Touch

| File | Action | Notes |
|------|--------|-------|
| `internal/client/go/raylib/audio/audio.go` | **Create** | `AudioManager` interface + `raylibAudioManager` impl |
| `internal/client/go/raylib/audio/events.go` | **Create** | `AudioEvent` enum, sound path array, event name→ID map |
| `internal/client/go/raylib/audio/audio_test.go` | **Create** | Enum stability test; no-op manager test |
| `internal/client/go/raylib/app/app.go` | **Modify** | Add `audio AudioManager` field; init in `New()`; call `Shutdown()` in close |
| `internal/client/go/raylib/app/interactive.go` | **Modify** | Call `audio.Play(AudioEventThrusterStart/Stop)` on thrust input edges |
| `internal/client/go/raylib/ui/render/render.go` | **Modify** | Call `audio.Play(AudioEventProximityAlert)` from proximity check (Phase 2) |
| `configs/app.json` | **Modify** | Add `"audio"` config block |
| `data/assets/sounds/` | **Create** | Directory + 10 × 1-frame silence placeholder `.wav` files |

---

## 6. Phases

### Phase 1 — Audio Infrastructure + Thruster Sounds

**Architectural layer**: Raylib client layer only (`internal/client/go/raylib/audio/`).  
**No proto changes. No server changes.**

**Prerequisite**: None (fully independent).

**Files created/modified**: See §5.

**Work items**:
- [ ] Create `internal/client/go/raylib/audio/` package
- [ ] Implement `AudioEvent` enum with `audioEventCount` sentinel
- [ ] Implement `raylibAudioManager`: `InitAudioDevice`, `LoadSound` per event, `Play`, volume controls, `Shutdown`
- [ ] Write `noopAudioManager` (used when `audio.enabled: false`)
- [ ] Add `audio AudioManager` field to `App`; construct based on config; call `Shutdown` in teardown
- [ ] Wire `AudioEventThrusterStart` / `AudioEventThrusterStop` to thrust input edge detection
- [ ] Add `configs/app.json` `"audio"` block
- [ ] Commit 10 × 1-frame silence `.wav` files to `data/assets/sounds/`
- [ ] Write enum stability regression test in `audio_test.go`
- [ ] Write `noopAudioManager` implements `AudioManager` interface test

**Acceptance criteria**:
- `go vet ./internal/client/go/raylib/audio/...` passes ✓
- App starts cleanly with `"audio": { "enabled": false }` (no `InitAudioDevice` call) ✓
- App starts cleanly with `"audio": { "enabled": true }` and placeholder `.wav` files ✓
- Thruster input edge triggers `Play` (verified via `noopAudioManager` in test) ✓
- `audioEventCount` regression test passes ✓

### Phase 2 — Proximity, Join/Leave, Message Received

**Prerequisite**: Phase 1 complete; F-020 Phase 2 (position streaming) complete.

**Work items**:
- [ ] Call `AudioEventProximityAlert` from F-024 proximity check path
- [ ] Call `AudioEventClientJoin` / `AudioEventClientLeave` when session delta arrives
- [ ] Call `AudioEventMessageReceived` when `ShipMessage` arrives (F-025 Phase 1)
- [ ] Replace placeholder `.wav` files for these events with real sound assets (or keep placeholders)

**Acceptance criteria**:
- Proximity alert sound fires within one render frame of threshold crossing ✓
- Join/leave sounds fire on session delta events ✓
- Message received sound fires on incoming text message ✓

### Phase 3 — Collision, Damage, Warp

**Prerequisite**: Phase 2 complete; F-027 Phase 1 (collision events) complete.

**Work items**:
- [ ] Call `AudioEventCollisionImpact` on `ImpactEvent` received from server
- [ ] Call `AudioEventDamageCritical` when `DamageRating` crosses 0.75
- [ ] Call `AudioEventDestroyed` when `DamageRating` reaches 1.0
- [ ] Call `AudioEventWarp` when superluminal jump executes (F-022)
- [ ] Replace placeholder `.wav` files for these events with real sound assets

---

## 7. Dependencies

| Feature | Relationship |
|---------|-------------|
| F-022: Movement | Thruster events driven by movement input (Phase 1) |
| F-024: HUD | Proximity check path triggers `AudioEventProximityAlert` (Phase 2) |
| F-025: Ship Comms | Message-received event triggers audio (Phase 2) |
| F-027: Collision/Damage | Impact and damage events trigger audio (Phase 3) |
| F-020: Session | Join/leave events trigger audio (Phase 2) |

---

## 8. Open Questions

| # | Question | Status |
|---|----------|--------|
| Q1 | Should real sound assets be free-licensed WAV files committed to the repo, or generated procedurally? | Open — Phase 2 |
| Q2 | Should per-event volume be hot-reloaded alongside keybindings, or require restart? | Open — Phase 1 |
