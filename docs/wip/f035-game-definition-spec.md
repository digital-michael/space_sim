# F-035 — Game Definition (Themes, Factions, Scenario Composition)

## Purpose
Define the "Game Definition" abstraction: a higher-level configuration layer that lives outside and above the system data directory. A Game Definition composes one or more named systems, assigns themes to them, defines cross-system factions, and declares the available playable scenarios that reference its configuration.

## Status
📋 Not started

## Last Updated
2026-05-24

## Depends On
- F-034 — System Data Directory Structure (system directories must be stable before a Game Definition can reference them by name)

## Unlocks
- F-036 — Playable Scenario (scenarios reference factions and themes defined here)
- F-037 — AI/NPC Console (NPC Theme Managers and faction AIs are seeded from Game Definition faction profiles)
- F-008 Phase 2 — Artifact faction assignment

---

## 1. Motivation

Systems are factual: Solar System, Alpha Centauri, etc. Themes and factions are **fictional overlays** that vary by gameplay context. The same system can be cast as a peaceful explorer's hub in one Game Definition and a contested warzone in another. Decoupling fictional overlay from factual system data means:

- System data files can be maintained for scientific accuracy without game-play concerns leaking in.
- Multiple Game Definitions can reference the same system without forking its data.
- Factions are defined once and applied across N systems; placement is dynamic at game-play load time.

---

## 2. Directory Layout

```
data/
  game_definitions/
    vanilla/                        ← Platform-provided default
      game_definition.json          ← Manifest
      themes/
        sol_frontier.json
        centauri_colony.json
      factions/
        earth_union.json
        nomad_collective.json
        outer_exiles.json
      scenarios/                    ← Playable scenario configs (see F-036)
        inner_system_war.json
        exploration_run.json
      assets/
        flags/
          earth_union.svg
          nomad_collective.png
        icons/
          earth_union_icon.svg
    custom_campaign/                ← User or server-provided
      game_definition.json
      ...
```

### Rules
- `game_definition.json` is required; absence is a fatal load error.
- All sub-directories are optional; absence means the Game Definition has no themes/factions/scenarios of that type.
- Platform ships one or more pre-built Game Definitions under `data/game_definitions/`.
- Users and server operators can create additional Game Definitions in the same directory.
- A running server loads exactly **one** active Game Definition (declared in startup config or CLI flag `--game-def <name>`).

---

## 3. game_definition.json Schema

```json
{
  "name": "Vanilla",
  "id": "vanilla",
  "version": "1.0",
  "schema_version": "1.0",
  "description": "The default platform game definition.",
  "systems": ["solar_system", "alpha_centauri_system", "barnards_star_system"],
  "default_scenario": "scenarios/exploration_run.json"
}
```

| Field | Notes |
|-------|-------|
| `id` | Machine-readable unique identifier; used by CLI flags and server config |
| `systems` | List of system directory names from `data/systems/` that this Game Definition participates in |
| `default_scenario` | Relative path to the scenario loaded if no scenario is specified at startup |

---

## 4. Theme Schema

A theme describes the fictional context layered onto a named system. Multiple themes may be active on the same system simultaneously; conflicts are resolved at game-play load time (see §6).

```json
{
  "id": "sol_frontier",
  "name": "Sol Frontier",
  "version": "1.0",
  "target_system": "solar_system",
  "lore": {
    "summary": "Earth's solar system in the early colonization era.",
    "backstory_file": "lore/sol_frontier_backstory.md",
    "era": "2217 CE"
  },
  "visual": {
    "color_theme": {
      "primary": [60, 120, 200, 255],
      "secondary": [200, 160, 60, 255],
      "accent": [255, 255, 255, 255]
    },
    "flag_asset": "assets/flags/sol_frontier.svg",
    "icon_asset": "assets/icons/sol_frontier_icon.svg"
  },
  "factions": ["earth_union", "outer_exiles"],
  "faction_placement": [
    {
      "faction_id": "earth_union",
      "home_body": "Earth",
      "controlled_bodies": ["Earth", "Luna", "Mars"],
      "artifact_density": 0.7,
      "force_scale": 1.0
    },
    {
      "faction_id": "outer_exiles",
      "home_body": null,
      "controlled_bodies": ["Ceres", "Titan"],
      "artifact_density": 0.3,
      "force_scale": 0.6
    }
  ]
}
```

### Color Theme Usage
`color_theme` values affect:
- LQM (Label/Marker/Artifact) branding colors for affiliated objects.
- Faction-affiliated artifact rendering tint.
- In-game communications channel color coding.

`color_theme` does **not** affect:
- HUD chrome or UI element colors (those are profile-driven per F-038).
- Navigation or text rendering.

### Visual Assets
- Formats: SVG (preferred for scalability), PNG, JPG.
- Managed in `assets/flags/` and `assets/icons/` within the Game Definition directory.
- No global texture pipeline changes required; assets are referenced by path and loaded on demand.

---

## 5. Faction Schema

Factions are defined **once** at the Game Definition level and are **system-agnostic**. A faction can be placed into any number of systems via theme `faction_placement` entries.

```json
{
  "id": "earth_union",
  "name": "Earth Union",
  "version": "1.0",
  "description": "The federated government of humanity's home world and inner colonies.",
  "color_theme": {
    "primary": [30, 80, 180, 255],
    "secondary": [220, 220, 240, 255],
    "accent": [255, 200, 0, 255]
  },
  "flag_asset": "assets/flags/earth_union.svg",
  "icon_asset": "assets/icons/earth_union_icon.svg",
  "mindset": {
    "archetype": "defender",
    "disposition": "territorial",
    "aggression": 0.4,
    "expansion": 0.3,
    "description": "Earth Union forces defend established territories. They are not expansionist but respond aggressively to incursions."
  },
  "ai_profile": {
    "npc_archetype": "guardian",
    "preferred_formations": ["patrol", "convoy_escort"],
    "technology_level": 0.8,
    "scavenge_alien_tech": false
  },
  "artifact_resources": {
    "ship_classes": ["scout_mk1", "freighter_t1", "explorer_x1"],
    "station_types": ["orbital_outpost"],
    "initial_force_scale": 1.0
  }
}
```

### Mindset Archetypes
Archetypes guide AI/NPC behavior (F-037). Platform-defined archetypes:

| Archetype | Description |
|-----------|-------------|
| `defender` | Holds territory; responds to intrusion |
| `conqueror` | Expands aggressively; manifest destiny |
| `nomad` | No fixed territory; trades and roams |
| `isolationist` | Avoids contact; hostile when approached |
| `scavenger` | Opportunistic; acquires technology from others |
| `infiltrator` | Operates covertly within other factions |
| `homeworld_guardian` | Extreme territorial focus on a single body |

### Artifact Resources
The `artifact_resources` block seeds faction presence in a system:
- `ship_classes` — ships drawn from `data/ships/` catalog; spawned as NPCs.
- `station_types` — station archetypes spawned near `controlled_bodies`.
- `initial_force_scale` — scalar applied to spawn counts at scenario load time (overridable per theme `faction_placement`).

---

## 6. Multi-Theme Conflict Resolution

When multiple themes are active on the same system, the following rules apply at game-play load time:

1. **Non-overlapping bodies**: Each faction controls disjoint bodies → no conflict; both themes coexist.
2. **Overlapping controlled bodies**: The theme with higher `faction_placement.force_scale` claims the body; the lower-scale faction retains presence but not control.
3. **Complete mismatch** (themes require bodies that do not exist in the target system): The loader inserts a synthetic object at a calculated orbital position to represent the faction's presence. The synthetic object may be:
   - A **rogue space station** artifact (default): inserted into the system's `artifacts.json` runtime list as a faction installation.
   - A **synthetic rogue body** (rogue planet or large asteroid): inserted into the system's `rogues.json` runtime list, named after the faction, to serve as a territorial anchor. Used when the faction's `mindset.archetype` is `homeworld_guardian` or when no inhabited body is within range.
   Both options are tagged `synthetic_placement: true` and are visible in admin/debug modes. The placement choice is logged in `placement_log.json`.
4. **Unresolvable conflict**: Logged as a warning; the later-loaded theme's conflicting entries are skipped.

Conflict resolution outcomes are written to a `game_state/<scenario_id>/placement_log.json` at scenario startup.

---

## 7. Loading Sequence

1. Server starts with `--game-def vanilla`.
2. Load `data/game_definitions/vanilla/game_definition.json`.
3. Load all referenced system directories (via F-034 `LoadSystemFromDir`).
4. Load all faction JSON files from `factions/`.
5. Load all theme JSON files from `themes/`.
6. Validate faction references in each theme.
7. For the active scenario (F-036), resolve `faction_placement` into the live simulation: spawn NPC ships and stations via the event queue.
8. Write `placement_log.json`.

---

## 8. Acceptance Criteria

### Phase 1 (data schema + loader)
- [ ] `data/game_definitions/vanilla/` created with `game_definition.json`, at least two faction definitions, and at least one theme definition for `solar_system`.
- [ ] Game Definition loader reads manifest, factions, and themes; validates all faction references in themes resolve.
- [ ] Multi-theme conflict resolution logic implemented and tested for the two-faction-one-body case.
- [ ] Faction and theme JSON schema versioned; mismatch produces actionable error.
- [ ] `--game-def <id>` CLI flag added to server startup.
- [ ] Placement log written to `game_state/<scenario_id>/placement_log.json`.

### Phase 2 (integration with F-036, F-037)
- [ ] Faction NPC ships and stations spawned at scenario start via event queue.
- [ ] AI/NPC Theme Manager (F-037) seeded with faction mindset profile.
- [ ] Synthetic placement artifact rendered in-world for extreme-mismatch scenarios.

---

## 9. Related Documents
- [docs/wip/f034-system-data-structure-spec.md](f034-system-data-structure-spec.md) — System data format (prerequisite)
- [docs/wip/f036-playable-scenario-spec.md](f036-playable-scenario-spec.md) — Playable scenario (uses Game Definition)
- [docs/wip/f037-ai-npc-console-spec.md](f037-ai-npc-console-spec.md) — AI/NPC Console (consumes faction mindset profiles)
- [docs/wip/f033-ship-definition-spec.md](f033-ship-definition-spec.md) — Ship catalog (ship classes referenced by factions)
