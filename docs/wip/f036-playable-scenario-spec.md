# F-036 — Playable Scenario

## Purpose
Define the Playable Scenario layer: a named, server-scoped configuration that activates a Game Definition, selects which systems participate, sets initial conditions, and drives runtime universe state. Platform ships pre-built scenarios; operators and users can create custom ones.

## Status
📋 Not started

## Last Updated
2026-05-24

## Depends On
- F-034 — System Data Directory Structure
- F-035 — Game Definition (themes, factions)

## Unlocks
- F-037 — AI/NPC Console (scenarios configure NPC behavior parameters)
- Multi-server interlocking scenarios (see §6)

---

## 1. Motivation

A Playable Scenario answers: **"What is happening in this server's universe right now?"**

- Which systems are active?
- Which factions are contesting what?
- What resources exist and how do they evolve over time?
- What are the win/loss/progress conditions?

Scenarios are **per-server**. Each running server instance loads one scenario at startup. The simulation, faction state, and universe state are scoped to that server.

---

## 2. Directory Layout

Scenarios live inside a Game Definition's `scenarios/` directory:

```
data/
  game_definitions/
    vanilla/
      scenarios/
        exploration_run/          ← named scenario directory
          scenario.json           ← scenario manifest
          initial_state.json      ← starting universe state
          objectives.json         ← win/loss/progress conditions
          events.json             ← scripted time/condition triggers
        inner_system_war/
          scenario.json
          initial_state.json
          objectives.json
          events.json
```

A scenario directory is the **unit of deployment**: the whole directory can be copied, versioned, shared, or modified independently of the Game Definition.

### Runtime state (not shipped with the scenario, generated at server start):

```
game_state/
  <server-id>_<scenario-id>/
    universe_state.json           ← live, mutable universe state
    placement_log.json            ← faction placement decisions (from F-035)
    event_log.jsonl               ← append-only event log (extends existing Phase 5 format)
```

`game_state/` is outside `data/`; it is runtime-generated and server-local. It is not committed to the repository.

---

## 3. scenario.json Schema

```json
{
  "id": "exploration_run",
  "name": "Exploration Run",
  "version": "1.0",
  "schema_version": "1.0",
  "game_definition": "vanilla",
  "description": "A peaceful solo or co-op exploration of the solar system.",
  "active_systems": ["solar_system"],
  "active_themes": ["sol_frontier"],
  "mode": "cooperative",
  "time_limits": {
    "real_seconds": 0,
    "sim_years": 0
  },
  "manufacturing": {
    "enabled": false
  }
}
```

| Field | Notes |
|-------|-------|
| `game_definition` | Must match an `id` in `data/game_definitions/` |
| `active_systems` | Subset of `game_definition.systems`; only these systems run in this scenario |
| `active_themes` | Themes to activate; must be defined in the referenced Game Definition |
| `mode` | `"cooperative"`, `"competitive"`, `"sandbox"`, `"pve"` |
| `time_limits` | `0` means no limit |
| `manufacturing.enabled` | Whether time/resource-based manufacturing is active (see §5) |

---

## 4. Universe State

`universe_state.json` is the live, mutable record of game-play progress for a running server. It is written atomically (same pattern as Phase 5 persistence) at configurable intervals and on server shutdown.

```json
{
  "scenario_id": "exploration_run",
  "server_id": "sol-server-01",
  "started_at": "2026-05-24T14:00:00Z",
  "sim_time_elapsed_s": 1234567.0,
  "factions": {
    "earth_union": {
      "controlled_bodies": ["Earth", "Luna", "Mars"],
      "resource_credits": 4200,
      "active_ships": 12,
      "active_stations": 3
    },
    "outer_exiles": {
      "controlled_bodies": ["Ceres", "Titan"],
      "resource_credits": 1800,
      "active_ships": 7,
      "active_stations": 1
    }
  },
  "player_states": {
    "player-guid-abc": {
      "faction": "earth_union",
      "ship_guid": "ship-guid-xyz",
      "system": "solar_system",
      "credits": 500
    }
  }
}
```

Universe state is **server-scoped**: each server owns its own `universe_state.json`. Cross-server interlocking (§6) synchronizes a subset of this state between servers.

---

## 5. Manufacturing and Resource Economy

Manufacturing allows factions (and players) to produce new artifacts, ships, and stations over time, subject to resource availability. This makes the game universe dynamic rather than a static starting configuration.

### Phase 1 (scenario foundation — no manufacturing)
Manufacturing is disabled (`"manufacturing": {"enabled": false}`). All faction forces are placed at scenario start and do not change due to production. Forces can still be destroyed or captured.

### Phase 2 (manufacturing)
When `manufacturing.enabled = true`:
- Factions accumulate `resource_credits` over sim time at a rate governed by `controlled_bodies` and their economic output.
- Faction AI (F-037 NPC Theme Manager) can spend credits to queue ship or station construction.
- Construction completes after a sim-time duration; the new artifact appears in the simulation via the event queue.
- Player-driven manufacturing: players affiliated with a faction can queue construction if their role permits (IAAM F-011).

---

## 6. Multi-Server Interlocking Scenarios

Multiple servers can participate in the **same named scenario** while each running a different system. A player in the Solar System server can travel to Alpha Centauri, triggering a server handoff.

### Topology Example
```
vanilla/inner_system_war scenario
  → Server A: active_systems = ["solar_system"]
  → Server B: active_systems = ["alpha_centauri_system"]
  → Coordination layer: faction universe_state synchronized via a shared state service
```

### Player Handoff Protocol
1. Player issues warp/transit command to a body in a different system.
2. Current server validates the destination system is reachable (declared in scenario config).
3. Server issues a **transit token** to the player client (signed, short-lived JWT).
4. Client presents transit token to the destination server.
5. Destination server validates token and creates a `ClientSession` for the player at the system entry point.
6. Original server marks the player as `transited_out`; destination server marks as `arrived`.
7. Ship state (inventory, health, credits) is embedded in the transit token (server-to-server trust is explicit; tokens are not forged by the client).

### Shared Universe State
- Faction `resource_credits` and `controlled_bodies` are synchronized across servers at a configurable interval.
- Synchronization uses a **coordination service** (not yet designed; scope post F-011 IAAM).
- Phase 1 of multi-server support: **manual handoff** (admin teleports player; no automatic sync). Automatic handoff with sync is Phase 2.

### Phase 1 Multi-Server Scope
- `scenario.json` may declare `"peer_servers": [{"id": "sol-server-01", "system": "solar_system", "addr": "..."}]` — informational only in Phase 1.
- No automated player handoff in Phase 1.

---

## 7. Scripted Events

`events.json` defines triggers that fire game-play events at specific sim times or conditions.

```json
{
  "schema_version": "1.0",
  "events": [
    {
      "id": "outer_exiles_raid_01",
      "trigger": {
        "type": "sim_time",
        "sim_seconds": 86400
      },
      "action": {
        "type": "spawn_faction_fleet",
        "faction": "outer_exiles",
        "near_body": "Ceres",
        "ship_class": "scout_mk1",
        "count": 3
      }
    },
    {
      "id": "distress_call",
      "trigger": {
        "type": "condition",
        "condition": "earth_union.active_ships < 5"
      },
      "action": {
        "type": "broadcast_comms",
        "faction": "earth_union",
        "message": "Earth Union requesting assistance. Defend the inner system!"
      }
    }
  ]
}
```

Phase 1: `sim_time` triggers only (simplest to implement via the existing event queue). Condition triggers are Phase 2.

---

## 8. Acceptance Criteria

### Phase 1 (scenario data + loader + universe state)
- [ ] `data/game_definitions/vanilla/scenarios/exploration_run/` and `inner_system_war/` directories created with `scenario.json`, `initial_state.json`, `objectives.json`.
- [ ] Scenario loader reads and validates `scenario.json`; resolves referenced Game Definition, systems, and themes.
- [ ] `universe_state.json` written at startup and updated atomically at configurable interval and on shutdown.
- [ ] `--scenario <id>` CLI flag selects scenario at server startup.
- [ ] `sim_time` scripted event triggers fire correctly via event queue.
- [ ] All existing tests pass; new tests for scenario load and universe state write/read round-trip.

### Phase 2 (manufacturing + multi-server)
- [ ] Manufacturing economy loop active when `manufacturing.enabled = true`.
- [ ] Faction AI (F-037) can queue and fulfill construction orders.
- [ ] Manual player transit handoff between servers.
- [ ] `peer_servers` config honored in admin handoff workflow.

---

## 9. Related Documents
- [docs/wip/f034-system-data-structure-spec.md](f034-system-data-structure-spec.md) — System data format
- [docs/wip/f035-game-definition-spec.md](f035-game-definition-spec.md) — Game Definition (factions, themes)
- [docs/wip/f037-ai-npc-console-spec.md](f037-ai-npc-console-spec.md) — AI/NPC Console
- [docs/wip/f020-multi-client-spec.md](f020-multi-client-spec.md) — Multi-client session layer
- [docs/wip/f011-iaam.md](f011-iaam.md) — Identity and access (roles for manufacturing/transit)
