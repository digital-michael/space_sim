# Galaxy View Plan

## Last Updated
2026-04-10

## Table of Contexts
1. Description
2. Goals
3. Scope
	3.1 In Scope
	3.2 Out of Scope
4. Ownership and Constraints
5. Agent Execution Contract
6. Prioritized Implementation Plan
	6.1 Phase 1 - Schema and Data Contracts
	6.2 Phase 2 - System Metadata Rollout
	6.3 Phase 3 - Galaxy Catalog and Info Nodes
	6.4 Phase 4 - Discovery and Loadable Node Pipeline
	6.5 Phase 5 - Galaxy View State and Input Model
	6.6 Phase 6 - Galaxy Renderer and Backdrop
	6.7 Phase 7 - Selection, Loading, and Transitions
	6.8 Phase 8 - Validation, Docs, and Completion Pass
7. Blocking Questions Checklist
8. Validation Plan
9. Definition of Done

## 1. Description

Add a dedicated Galaxy View that acts as a visually rich, navigable system-selection tool separate from the current System View. Galaxy View is not a galaxy simulator. It is a Sol-centered galactic atlas that renders:

- all eligible systems discovered from [data/systems/](../../data/systems)
- a reactive, realistic-looking Milky Way backdrop
- curated info-only galactic annotations that are selectable but not loadable
- a clean handoff from Galaxy View selection into System View runtime loading

Galaxy View and System View remain separate solutions. System View continues to own local orbital simulation. Galaxy View owns galaxy-scale coordinates, rendering, navigation, and selection.

## 2. Goals

- Make Galaxy View the primary visual system-selection experience for multi-system exploration.
- Auto-include newly added system JSON files in Galaxy View when they provide the required galaxy metadata.
- Keep each system JSON self-contained by storing its true galactic-position metadata in the file itself.
- Support info-only galactic points of interest that can be traversed with `TAB` and `SHIFT+TAB` but do not load a system.
- Provide a realistic, reactive Milky Way backdrop that responds to camera movement without attempting full galaxy simulation.
- Preserve current System View behavior and avoid coupling galaxy-scale logic into the local orbital simulation engine.

## 3. Scope

### 3.1 In Scope

- System JSON schema extension for galactic metadata
- Automatic discovery of loadable galaxy nodes from [data/systems/](../../data/systems)
- Separate galaxy catalog file for non-system annotations and galaxy-view presentation data
- Galaxy View scene/state, input handling, and selection model
- `TAB` / `SHIFT+TAB` traversal across systems and info-only entries
- `ENTER` to load a selected system entry
- Zoom and camera movement in Galaxy View with reasonable relative spatial accuracy
- Reactive backdrop layers that move with the camera and reinforce the Milky Way shape
- Focused tests, docs, and manual validation for the complete flow

### 3.2 Out of Scope

- Full-galaxy physical simulation
- Evolving orbits for stars within the Milky Way
- Attempting scientific completeness for all stars in the galaxy
- Mixing galaxy-space coordinates into local system orbital integration
- Procedural generation of trillions of stars as simulation objects
- Replacing the current System View camera and simulation model

## 4. Ownership and Constraints

- [data/systems/](../../data/systems) owns per-system truth, including local orbital definitions and each system's galactic metadata.
- A new galaxy catalog file under [data/](../../data) or [configs/](../../configs) should own info-only galactic annotations and galaxy-view presentation data.
- [internal/client/go/raylib/app/](../../internal/client/go/raylib/app) should own galaxy-view mode switching, input dispatch, selection flow, and system-load handoff.
- [internal/client/go/raylib/ui/render/](../../internal/client/go/raylib/ui/render) should own galaxy-view rendering, node presentation, backdrop rendering, and view-specific overlays.
- [internal/sim/](../../internal/sim) should remain focused on local system loading and orbital simulation. It should not absorb galaxy-view scene behavior.
- New data loading must preserve backward compatibility where practical or fail clearly with actionable errors.
- Newly added systems should be automatically eligible for Galaxy View only when required metadata is present and valid.

## 5. Agent Execution Contract

This document is written for an LLM agent expected to execute the work to completion without stopping after initial scaffolding.

Required execution behavior:

- Before implementation begins, ask all blocking questions in one batch.
- Do not start coding until blocking questions are resolved or explicitly waived.
- Once implementation starts, continue through code changes, tests, docs, and validation instead of stopping after partial scaffolding.
- Do not leave the feature half-wired across schema, loader, app mode, renderer, and docs.
- If a phase reveals a hidden blocker, stop once, ask the smallest possible set of clarifying questions, then continue to completion after answers are provided.
- Prefer focused, minimal changes that fit current package ownership and preserve existing behavior.
- Validate each completed phase before moving to the next.

Execution rule for completion:

A phase is not complete until code, tests, and docs for that phase are updated and the resulting behavior is validated.

## 6. Prioritized Implementation Plan

### 6.1 Phase 1 - Schema and Data Contracts

**Priority**: Highest
**Status**: [ ] Not started
**Objective**: define the minimum data contracts needed for Galaxy View without destabilizing current system loading.

#### Tasks

- [ ] Define required top-level galactic metadata fields for each system JSON
- [ ] Decide the canonical coordinate frame for stored positions
- [ ] Decide whether to store both observational inputs and derived Cartesian values
- [ ] Add or update schema documentation for system galactic metadata
- [ ] Define the separate galaxy catalog file format for info-only annotations and presentation data
- [ ] Define annotation types such as `point`, `region`, and `label`
- [ ] Define the minimum metadata required for a system to auto-appear in Galaxy View
- [ ] Define loader behavior for systems that are missing required galaxy metadata

#### Exit Criteria

- [ ] System JSON metadata contract is documented and stable enough to implement
- [ ] Galaxy catalog schema for info-only annotations is documented and stable enough to implement
- [ ] Auto-inclusion rules are explicit

### 6.2 Phase 2 - System Metadata Rollout

**Priority**: Highest
**Status**: [ ] Not started
**Objective**: populate galactic metadata in all currently supported systems so Galaxy View has real, selectable content immediately.

#### Tasks

- [ ] Add galactic metadata to [data/systems/solar_system.json](../../data/systems/solar_system.json)
- [ ] Add galactic metadata to [data/systems/alpha_centauri_system.json](../../data/systems/alpha_centauri_system.json)
- [ ] Add galactic metadata to the nearby-system JSON files already present in [data/systems/](../../data/systems)
- [ ] Verify the metadata uses one consistent frame and unit convention across all systems
- [ ] Add source notes or provenance fields if included in the schema design
- [ ] Validate that existing system loading remains unaffected by the new metadata

#### Exit Criteria

- [ ] Every current system JSON has valid galactic metadata
- [ ] Existing system loading still works unchanged
- [ ] Galaxy View can rely on current systems as a seeded dataset

### 6.3 Phase 3 - Galaxy Catalog and Info Nodes

**Priority**: High
**Status**: [ ] Not started
**Objective**: create the separate galaxy catalog file and seed it with enough varied info-only entries to exercise the feature properly.

#### Tasks

- [ ] Create the galaxy catalog file in the agreed location
- [ ] Add at least 4 info-only annotations; target 6 to cover multiple types cleanly
- [ ] Include items such as Galactic Center, Orion Spur, Perseus Arm, Sagittarius Arm, Local Bubble, and Fermi Bubbles unless changed by approved answers
- [ ] Add descriptive text and selection-friendly labels for each annotation
- [ ] Distinguish loadable system entries from info-only annotations at the schema level
- [ ] Define ordering rules for mixed traversal of systems and info-only entries

#### Exit Criteria

- [ ] Galaxy catalog file exists and validates
- [ ] Mixed traversal across info-only and loadable nodes is well-defined
- [ ] There are enough seeded items to validate implementation behavior correctly

### 6.4 Phase 4 - Discovery and Loadable Node Pipeline

**Priority**: High
**Status**: [ ] Not started
**Objective**: build the data pipeline that discovers all systems, validates galaxy metadata, and exposes the combined node set to Galaxy View.

#### Tasks

- [ ] Add discovery logic that scans [data/systems/](../../data/systems) and loads galaxy metadata for each eligible system
- [ ] Exclude or warn on systems missing required metadata according to the Phase 1 rules
- [ ] Load info-only annotations from the galaxy catalog file
- [ ] Merge systems and annotations into one selection-ready collection while preserving type distinctions
- [ ] Add stable sort and traversal ordering rules
- [ ] Add test coverage for discovery, missing metadata behavior, and mixed-node ordering

#### Exit Criteria

- [ ] All eligible system files are auto-included with no manual registration step
- [ ] Info-only nodes are loaded from the separate catalog file
- [ ] Mixed discovery behavior is covered by focused tests

### 6.5 Phase 5 - Galaxy View State and Input Model

**Priority**: High
**Status**: [ ] Not started
**Objective**: add a dedicated Galaxy View mode with navigation, camera controls, and selection semantics.

#### Tasks

- [ ] Add a dedicated Galaxy View mode separate from System View
- [ ] Define input-state fields required for galaxy camera, zoom, selection index, and selected node details
- [ ] Implement `TAB` to move to the next selectable node
- [ ] Implement `SHIFT+TAB` to move to the previous selectable node
- [ ] Implement `ENTER` to load a selected system node
- [ ] Ensure `ENTER` on an info-only node does not attempt a load and instead shows details only
- [ ] Add camera zoom controls for galaxy-scale navigation
- [ ] Add camera pan/orbit controls with useful relative accuracy and predictable damping or response
- [ ] Add a return path back to System View without loading a new system
- [ ] Add focused input/state tests for selection traversal and action behavior

#### Exit Criteria

- [ ] Galaxy View can be entered, navigated, and exited cleanly
- [ ] Selection semantics differ correctly between systems and info-only nodes
- [ ] Input behavior is validated and does not conflict with existing System View behavior

### 6.6 Phase 6 - Galaxy Renderer and Backdrop

**Priority**: High
**Status**: [ ] Not started
**Objective**: render Galaxy View as a convincing, reactive Milky Way scene without promising scientific galaxy simulation.

#### Tasks

- [ ] Render system nodes at their galactic positions using a galaxy-view-specific representation
- [ ] Render info-only annotations distinctly from loadable systems
- [ ] Add a Milky Way disc or band backdrop with a visible central bulge
- [ ] Add layered reactive background elements that respond to camera movement
- [ ] Add parallax or depth cues so the backdrop feels spatial rather than flat
- [ ] Add visual emphasis for the currently selected node
- [ ] Add labels that scale or fade cleanly with zoom level
- [ ] Ensure the scene remains readable at both zoomed-out and local-neighborhood scales
- [ ] Validate performance on the current desktop target without introducing a new bottleneck

#### Exit Criteria

- [ ] Galaxy View is visually legible and obviously distinct from System View
- [ ] The backdrop feels reactive to camera movement
- [ ] Selected nodes are easy to identify and traverse

### 6.7 Phase 7 - Selection, Loading, and Transitions

**Priority**: Medium
**Status**: [ ] Not started
**Objective**: complete the handoff from Galaxy View to System View cleanly and predictably.

#### Tasks

- [ ] Reuse or extend current runtime system loading from Galaxy View node selection
- [ ] Ensure selected system nodes load the intended file with no manual mapping step
- [ ] Preserve Galaxy View as a separate mode rather than merging camera state into System View
- [ ] Decide whether the first version uses immediate load or a short transition animation
- [ ] If a transition is approved, implement a short non-blocking camera or UI sequence before load
- [ ] Reuse welcome-banner or status messaging only if it improves clarity and does not clutter the handoff
- [ ] Add focused validation for selection-to-load flow and no-op behavior on info-only nodes

#### Exit Criteria

- [ ] Entering on a system node loads the correct system reliably
- [ ] Entering on an info-only node does not trigger a system load
- [ ] Transition behavior, if implemented, is deliberate and minimal

### 6.8 Phase 8 - Validation, Docs, and Completion Pass

**Priority**: Medium
**Status**: [ ] Not started
**Objective**: finish the feature end-to-end and leave no partial implementation seams.

#### Tasks

- [ ] Add or update package-level tests for new discovery, state, and rendering-adjacent logic where practical
- [ ] Run focused package tests for the touched app, UI, render, and loader packages
- [ ] Perform manual runtime validation for Galaxy View controls, selection, system load handoff, and info-only annotations
- [ ] Update relevant docs in [docs/standards/](../standards) and [docs/history/](../history) once implementation is complete
- [ ] Update [docs/wip/todo.md](todo.md) with Galaxy View status if the work becomes active
- [ ] Remove stale comments, dead scaffolding, or placeholder-only implementation notes
- [ ] Confirm that newly added systems auto-appear when metadata is valid

#### Exit Criteria

- [ ] Code, docs, and validation are all complete
- [ ] The feature works end-to-end without manual registration of new systems
- [ ] No phase remains half-finished

## 7. Blocking Questions Checklist

The implementing agent must ask these before coding unless the answers are already explicitly settled in the current conversation.

- [ ] What is the exact top-level field name for system galactic metadata?
- [ ] Which coordinate frame should be canonical for stored Cartesian positions?
- [ ] Should system JSON files store both raw astronomy inputs and derived Cartesian coordinates, or only derived coordinates?
- [ ] Where should the separate galaxy catalog file live?
- [ ] Should `TAB` / `SHIFT+TAB` traverse one merged list of systems and info-only entries, or two modes/categories?
- [ ] Should `ENTER` on an info-only node open a detail card, do nothing, or center the camera on that item?
- [ ] What key or command enters Galaxy View from System View?
- [ ] Should the first implementation include a transition animation when loading a system from Galaxy View?
- [ ] What minimum metadata failure behavior is preferred for a new system file: hide with warning, visible but disabled, or fail load hard?
- [ ] What exact initial set of galaxy annotations should ship in the first catalog file if the proposed list needs adjustment?

## 8. Validation Plan

- Focused loader/data tests for system metadata parsing and galaxy catalog parsing
- Focused app/state tests for view switching, traversal, and system-load decisions
- Focused discovery tests proving new systems in [data/systems/](../../data/systems) auto-appear when metadata is valid
- Manual runtime validation for:
	- entering Galaxy View
	- traversing systems with `TAB` and `SHIFT+TAB`
	- traversing info-only entries
	- loading a system with `ENTER`
	- refusing to load info-only entries
	- zooming and camera movement in Galaxy View
	- backdrop responsiveness to camera movement
	- returning to System View cleanly

## 9. Definition of Done

Galaxy View is done when all of the following are true:

- A dedicated Galaxy View exists and is separate from System View.
- All current systems with valid galactic metadata are auto-discovered and selectable.
- The separate galaxy catalog file loads info-only annotations successfully.
- `TAB` / `SHIFT+TAB` traversal works across the intended selectable entries.
- `ENTER` loads a selected system and does not load info-only entries.
- The Milky Way backdrop is reactive to camera movement and visually coherent.
- Newly added systems in [data/systems/](../../data/systems) require no manual map registration when their metadata is valid.
- The work is tested, documented, and complete enough that a follow-on agent does not need to rediscover the intended architecture.
