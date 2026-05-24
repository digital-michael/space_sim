# F-037 — AI/NPC Console

## Purpose
Define an abstracted, pluggable AI/LLM integration layer that powers three NPC profiles: Personal Copilot (per-player ship assistant), NPC Characters (individual automated actors), and NPC Theme Managers (faction-level orchestrators). The architecture decouples AI back-end (local or remote LLM, rule-based engine) from game-play mechanics via a stable internal interface.

## Status
📋 Not started

## Last Updated
2026-05-24

## Depends On
- F-035 — Game Definition (faction mindset profiles seed NPC Theme Manager behavior)
- F-036 — Playable Scenario (scenario context drives NPC objectives)
- F-020 — Multi-Client Session Layer (player sessions are targets for Personal Copilot)
- F-011 — IAAM (role-based permission for which AI actions are available to which profile)

## Unlocks
- F-036 Phase 2 — NPC manufacturing and spawning orders originate here
- F-025 — Ship-to-ship communications (NPC characters send comms via this spec)

---

## 1. Motivation

Three distinct AI use cases exist:

| Profile | Actor | Owner | Back-end |
|---------|-------|-------|----------|
| **Personal Copilot** | Per-player ship AI | Player | Server-side default LLM, or player's own AI (client-side) |
| **NPC Character** | Individual automated ship or station | Server | Server-side LLM |
| **NPC Theme Manager** | Faction-level orchestrator ("leader") | Server | Server-side LLM |

All three share a common MCP action vocabulary (game-play actions, not implementation actions), but differ in scope, decision frequency, and authorization level.

The existing `tools/write_file_server.py` MCP is for **developer tooling** and is entirely separate from this feature.

---

## 2. Architecture Overview

```
┌─────────────────────────────────────────────────────────┐
│                    LLM Abstraction Layer                 │
│  LLMProvider interface                                   │
│  ┌──────────────┐  ┌──────────────┐  ┌───────────────┐  │
│  │  LocalLLM    │  │  OpenAILLM   │  │  ClaudeLLM    │  │
│  │  (Ollama)    │  │  (GPT-4o)    │  │  (Anthropic)  │  │
│  └──────────────┘  └──────────────┘  └───────────────┘  │
└─────────────────────────────────────────────────────────┘
                           │
                   NPCAgent (per-profile)
                           │
              ┌────────────┴─────────────┐
              │    Game MCP Action Set   │
              │  (move, comms, scan,     │
              │   manufacture, claim...) │
              └────────────┬─────────────┘
                           │
              Server Event Queue (F-003/F-004)
```

---

## 3. LLM Provider Interface

```go
// internal/ai/llm/provider.go

type Message struct {
    Role    string // "system", "user", "assistant"
    Content string
}

type CompletionRequest struct {
    Messages    []Message
    MaxTokens   int
    Temperature float64
}

type CompletionResponse struct {
    Content string
    Usage   TokenUsage
}

type TokenUsage struct {
    PromptTokens     int
    CompletionTokens int
    TotalTokens      int
}

// LLMProvider is the single interface all back-ends implement.
type LLMProvider interface {
    Complete(ctx context.Context, req CompletionRequest) (CompletionResponse, error)
    Name() string       // e.g. "ollama/llama3", "openai/gpt-4o"
    IsLocal() bool      // true if no external network call
}
```

### Supported Back-ends (Phase 1 target: LocalLLM)

| Back-end | Implementation | Notes |
|----------|---------------|-------|
| `LocalLLM` | HTTP to local Ollama instance | Phase 1 target; no external API cost |
| `OpenAILLM` | OpenAI Chat Completions API | Phase 2 |
| `AnthropicLLM` | Anthropic Messages API | Phase 2 |
| `RuleBasedLLM` | Pure FSM; no LLM call | Useful for testing and low-resource scenarios |

### Back-end Registry
A server can configure **0–N providers**:

```json
{
  "ai_providers": [
    { "id": "local", "type": "ollama", "base_url": "http://localhost:11434", "model": "llama3" },
    { "id": "openai", "type": "openai", "api_key_env": "OPENAI_API_KEY", "model": "gpt-4o" }
  ],
  "default_server_provider": "local",
  "token_cap_per_request": {
    "local": 0,
    "openai": 2000
  }
}
```

`token_cap_per_request = 0` means no cap (intended for local providers). External providers must have a non-zero cap; the loader rejects a zero cap on a non-local provider.

---

## 4. Client-Side AI

Players may configure their own LLM provider for their Personal Copilot instead of using the server's default.

- **Default**: client uses the server-exposed AI endpoint (see §5).
- **Override**: client config declares `personal_copilot_provider` with connection details for a local or remote LLM the player controls.
- The API shape is identical to the server-side `LLMProvider` interface, so server-side and client-side copilots are interchangeable at the API level.
- UI/UX standards (chat panel, response formatting, action confirmation flow) must be **consistent** regardless of whether the backing LLM is local, server-exposed, or client-controlled.

---

## 5. Server-Exposed AI Endpoint

The server exposes an AI service over gRPC so clients can use the server's configured provider without managing their own:

```protobuf
service AIService {
  rpc Chat(ChatRequest) returns (stream ChatResponse);
}

message ChatRequest {
  string session_id = 1;
  string npc_profile = 2;  // "personal_copilot", "npc_character", "theme_manager"
  repeated Message messages = 3;
  int32 max_tokens = 4;
}

message ChatResponse {
  string content_delta = 1;
  bool done = 2;
  TokenUsage usage = 3;
}
```

Access is gated by IAAM roles (F-011): `user` role can access `personal_copilot`; `admin` can access all profiles.

---

## 6. NPC Profiles

### 6.1 Personal Copilot

- **Scope**: Per-player session.
- **Role**: Ship assistant — answers navigation questions, explains nearby objects, advises on ship status, relays comms.
- **Context window**: Player's current position, nearest bodies, ship status, recent comms history.
- **Action permissions**: Read-only by default (query sim state). Can request navigation assists (track, warp) with player confirmation.
- **UI**: Chat panel in Player HUD (F-038 Communications/Chat HUD).
- **Owner**: Player; can be redirected to player's own provider.

### 6.2 NPC Character

- **Scope**: Per-NPC-entity (individual ship or station).
- **Role**: Simulate individual behavior — patrol route decisions, threat response, communications.
- **Context window**: NPC's own state, nearby entities, faction orders from Theme Manager.
- **Action permissions**: Movement, comms broadcast, fire (if armed). Cannot modify universe state directly.
- **Decision frequency**: Configurable per-NPC; default 1 decision per N sim seconds (low frequency to manage API cost).
- **Owner**: Server; not player-configurable.

### 6.3 NPC Theme Manager ("Leader")

- **Scope**: Per-faction, per-system.
- **Role**: Faction-level strategy — assign NPC Characters to objectives, queue manufacturing orders, respond to player actions at strategic level.
- **Context window**: Full faction state (controlled bodies, resource credits, active ships/stations), opposing faction state, scenario objectives.
- **Action permissions**: Can dispatch orders to NPC Characters, queue manufacturing (F-036 Phase 2), broadcast faction-wide comms.
- **Decision frequency**: Low — one strategic review per N sim minutes. High-compute budget; use local LLM or external with generous token cap.
- **Owner**: Server. A player with an appropriate role could optionally "take over" a Theme Manager slot (faction leader play), at which point the AI defers to the human.

---

## 7. Game MCP Action Set

NPCs interact with the simulation via a game-play MCP action set. This is **separate** from the developer MCP (`tools/write_file_server.py`).

Each action maps to one or more event queue dispatches. The AI generates structured action calls; the game engine validates and executes them.

### Initial Action Vocabulary

| Action | Profile access | Description |
|--------|---------------|-------------|
| `move_to` | Character, Copilot (confirm) | Navigate ship toward a named body or coordinate |
| `warp_to` | Character, Copilot (confirm) | Initiate warp toward a distant target |
| `scan` | Character, Copilot | Query sim state for nearby objects |
| `broadcast_comms` | All | Send a text message on a comms channel |
| `send_comms` | All | Send a targeted private message to a player or NPC |
| `dock` | Character, Copilot (confirm) | Dock at a station or large body |
| `undock` | Character, Copilot (confirm) | Undock from current location |
| `patrol` | Character | Set a patrol route between named waypoints |
| `claim_body` | Theme Manager | Assert faction control over an unclaimed body |
| `queue_manufacture` | Theme Manager | Queue construction of a ship or station |
| `assign_order` | Theme Manager | Dispatch a specific order to a named NPC Character |
| `retreat` | Character | Move NPC away from a threat at maximum speed |

Actions that mutate simulation state (move, warp, claim, manufacture) require the invoking profile to have the corresponding permission in the IAAM role table (F-011).

---

## 8. Token Cap and Rate Limiting

| Provider type | Default cap | Override |
|---------------|------------|---------|
| Local (`IsLocal() == true`) | 0 (no cap) | Always overridable |
| External | Required non-zero | Per-provider config |

Per-profile rate limits are enforced by the AI service layer, not the LLM provider:
- `personal_copilot`: max N requests per player per minute (configurable).
- `npc_character`: max N decisions per NPC per sim-second (configurable).
- `theme_manager`: max N strategic reviews per sim-minute (configurable).

Rate limit violations: request is queued (backpressure); if queue depth exceeds threshold, the `RuleBasedLLM` fallback is used for that decision cycle.

---

## 9. Acceptance Criteria

### Phase 1 (LLM abstraction + Personal Copilot via local LLM)
- [ ] `LLMProvider` interface and `LocalLLM` (Ollama) implementation in `internal/ai/llm/`.
- [ ] Provider registry loaded from server config; `default_server_provider` honored.
- [ ] `AIService` gRPC handler with `Chat` streaming RPC.
- [ ] Personal Copilot chat panel in Player HUD (F-038 Phase 1 prerequisite: Communications HUD slot exists).
- [ ] Context window includes: player position, 10 nearest named bodies, ship status summary.
- [ ] Token cap enforced on non-local providers; zero cap on local accepted.
- [ ] `RuleBasedLLM` fallback available for testing without a running LLM service.
- [ ] All tests pass; unit tests for provider registration and cap enforcement.

### Phase 2 (NPC Characters + Theme Managers + external LLMs)
- [ ] `OpenAILLM` and `AnthropicLLM` back-ends implemented.
- [ ] NPC Character decision loop active; dispatches actions via event queue.
- [ ] NPC Theme Manager strategic review loop; can issue orders to NPC Characters.
- [ ] Client-side copilot provider override configurable via client config.
- [ ] Per-profile rate limiting enforced; `RuleBasedLLM` fallback on queue overflow.
- [ ] Game MCP action set validated against IAAM permissions (F-011 required for full enforcement).

---

## 10. Related Documents
- [docs/wip/f035-game-definition-spec.md](f035-game-definition-spec.md) — Faction mindset profiles
- [docs/wip/f036-playable-scenario-spec.md](f036-playable-scenario-spec.md) — Scenario context
- [docs/wip/f025-ship-comms-spec.md](f025-ship-comms-spec.md) — NPC comms channel
- [docs/wip/f038-hud-profiles-spec.md](f038-hud-profiles-spec.md) — Personal Copilot chat panel lives in Player HUD
