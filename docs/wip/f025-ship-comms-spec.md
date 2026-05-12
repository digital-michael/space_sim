# F-025 — Ship-to-Ship Communications

## Purpose

Allow connected client sessions to send and receive in-game messages (text, and later
audio/video) to each other. Provides a durable in-simulation communications layer
independent of external chat tools.

Read this alongside:
- [`docs/wip/f020-multi-client-spec.md`](f020-multi-client-spec.md) — session registry;
  messages route by `SessionID`
- [`docs/wip/f024-multiplayer-hud-spec.md`](f024-multiplayer-hud-spec.md) — message display
  on the HUD

## Last Updated
2026-05-11

## Status
📋 Not started

---

## 1. Goals

| # | Goal |
|---|------|
| G1 | Any client can send a text message to any other client (or broadcast to all) |
| G2 | Messages are delivered in real-time over the existing gRPC session stream |
| G3 | Message history is visible in a HUD overlay (scrollable) |
| G4 | Ship-to-ship audio/video is reserved but not implemented in early phases |
| G5 | Admin clients can broadcast to all clients and can mute/silence specific sessions |

---

## 2. Non-Goals (early phases)

- Audio calls / audio streaming (Phase 3+)
- Video / screenshare (future)
- Persistent message history on disk (deferred; session-only in Phase 1)
- Message encryption beyond TLS (future; IAAM covers authentication)

---

## 3. Message Model

```proto
message ShipMessage {
  string  message_id   = 1;  // server-assigned UUID
  string  from_session = 2;  // sender SessionID
  string  to_session   = 3;  // recipient SessionID; empty = broadcast
  string  body         = 4;  // max 512 UTF-8 chars
  int64   sent_at_ns   = 5;  // server nanosecond timestamp
  MessageType type     = 6;
}

enum MessageType {
  MESSAGE_TYPE_UNSPECIFIED = 0;
  MESSAGE_TYPE_TEXT        = 1;  // plain text
  MESSAGE_TYPE_EMOTE       = 2;  // e.g., "/me waves"
  MESSAGE_TYPE_SYSTEM      = 3;  // server-generated (join, leave, damage events)
  MESSAGE_TYPE_AUDIO       = 4;  // reserved; payload TBD
}
```

---

## 4. Transport

Messages piggyback on `SessionStream` (F-020 §5). The server fans out incoming messages to
all targeted sessions' open streams. No separate gRPC service is required in Phase 1.

If `to_session` is empty, the message is broadcast to all connected sessions.
If `to_session` matches a single `SessionID`, it is a direct message (DM).

Admin-only: `to_session = "*"` broadcasts from the server as a system message.

---

## 5. HUD Integration

New HUD panel (added to F-024): a scrollable message log in the lower-left corner.
Maximum 50 messages displayed; oldest drop off the top.
Panel is visible by default; toggled by `hud.comms` binding (default: `M`).

---

## 6. REPL Command Reference

| Command | Description |
|---------|-------------|
| `msg all <text>` | Broadcast plain text to all connected sessions |
| `msg <label\|id> <text>` | Direct message to the named session or SessionID |
| `/me <action>` | Send emote (Phase 2; body prefix `/me `) |
| `session mute <id>` | Admin only: stop fan-out to target session (Phase 2) |

Message body rules:
- Max 512 UTF-8 characters; server rejects with `ErrMessageTooLong` if exceeded.
- Leading/trailing whitespace stripped by server.
- Empty body after stripping is rejected with `ErrMessageEmpty`.

---

## 7. Rate Limiting

To prevent spam, the server enforces a per-session rate limit:
- Max **10 messages per 5 seconds** per `SessionID`.
- Tokens refill continuously (token-bucket model).
- Excess sends return `ErrRateLimited` to the sender; not delivered to recipients.
- Rate limit parameters: `"message_rate_limit"` and `"message_rate_window_ms"` in `configs/app.json`.

---

## 8. Files to Touch

### Phase 1 files

| File | Action | Notes |
|------|--------|-------|
| `api/proto/spacesim/v1/session.proto` | **Modify** | Add `ShipMessage`, `MessageType` enum; extend `SessionStream` payload union |
| `api/gen/spacesim/v1/` | **Regenerate** | `make proto` |
| `internal/transport/grpc/session_handler.go` | **Modify** | Fan-out `ShipMessage` to target sessions via `SessionStream`; broadcast when `to_session` empty; rate-limit check |
| `internal/transport/grpc/session_handler_test.go` | **Modify** | Add DM delivery, broadcast delivery, rate-limit rejection tests |
| `internal/client/commands/commands.go` | **Modify** | Add `MsgCmd` type (fields: `ToSession`, `Body`) |
| `internal/client/repl/repl.go` | **Modify** | Add `msg` dispatch case; parse `msg all` vs `msg <target>` |
| `internal/client/go/raylib/ui/render/hud_comms.go` | **Create** | `drawCommsLog()` function; ring buffer of max 50 `ShipMessage`; `M` toggle |
| `internal/client/go/raylib/ui/render/hud_comms_test.go` | **Create** | Ring-buffer overflow test; timestamp format test |
| `internal/client/go/raylib/ui/render/render.go` | **Modify** | Call `drawCommsLog()` in HUD render pass |
| `internal/client/go/raylib/app/interactive.go` | **Modify** | Wire `M` key (via `hud.comms` InputAction) to comms panel toggle |
| `configs/app.json` | **Modify** | Add `"message_rate_limit": 10`, `"message_rate_window_ms": 5000`, `"comms_log_max_messages": 50` |

### Phase 2 files

| File | Action | Notes |
|------|--------|-------|
| `internal/transport/grpc/session_handler.go` | **Modify** | Add mute map; block fan-out to muted sessions; emit `MESSAGE_TYPE_SYSTEM` on mute/unmute |
| `internal/client/repl/repl.go` | **Modify** | Add `session mute <id>` and `session unmute <id>` dispatch |
| `internal/client/commands/commands.go` | **Modify** | Add `SessionMuteCmd`, `SessionUnmuteCmd` types |

---

## 9. Phases

### Phase 1 — Text Messaging

**Architectural layer**: Wire protocol (`api/proto/`), transport layer (`internal/transport/grpc/`), REPL client, Raylib UI render layer.
**Prerequisites**: F-020 Phase 2 complete (SessionStream must exist; sessions must have IDs).

**Value**: Clients can send text messages to each other in real time; messages appear in HUD.

Work items:
- [ ] Add `ShipMessage` proto message and `MessageType` enum to `session.proto`
- [ ] Extend `SessionStream` payload union to carry `ShipMessage`
- [ ] `make proto` to regenerate stubs
- [ ] Implement fan-out in `session_handler.go`: DM to target; broadcast when `to_session` empty
- [ ] Implement token-bucket rate limiter per session
- [ ] Add `msg` REPL command (`msg all <text>`, `msg <label|id> <text>`)
- [ ] Auto-post system messages on client join and leave (`MESSAGE_TYPE_SYSTEM`)
- [ ] Create `hud_comms.go` with `drawCommsLog()` and 50-message ring buffer
- [ ] Wire `M` key toggle to comms panel via `InputAction`
- [ ] Unit tests: DM delivery, broadcast, rate-limit rejection, ring-buffer overflow

Acceptance criteria:
- `msg all hello` posts visible message to all connected sessions' HUDs ✓
- `msg <label> hello` delivers only to the named session ✓
- Sending 11 messages in 5 seconds returns `ErrRateLimited` on the 11th ✓
- Join/leave events appear as system messages in all sessions' comms log ✓
- Ring buffer retains the 50 most recent messages; 51st drops the oldest ✓

### Phase 2 — Emotes and Moderation

**Architectural layer**: Transport layer, REPL client.
**Prerequisites**: Phase 1 complete.

**Value**: Richer communication and admin moderation capability.

Work items:
- [ ] Emote syntax: `/me <action>` prefix sets `MESSAGE_TYPE_EMOTE` on server
- [ ] Damage event system messages from F-027 fan-out to all comms logs
- [ ] `session mute <id>` blocks fan-out to target session (admin only)
- [ ] `session unmute <id>` restores fan-out
- [ ] Unit tests: mute suppresses delivery; unmute restores; damage event appears in log

### Phase 3 — Audio (scope TBD)

Reserved. Ship-to-ship audio streaming requires a separate audio service and is
outside the scope of this spec. Design separately when Phase 1 is complete.

---

## 10. Open Questions

| # | Question | Status |
|---|----------|--------|
| Q1 | Maximum message body: 512 chars sufficient? | Open |
| Q2 | Should DMs be visible to admin? (moderation vs. privacy) | Open |
| Q3 | Message rate-limit per session to prevent spam? | Open — Phase 1 |
