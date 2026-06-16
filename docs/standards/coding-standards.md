# Coding Standards

## Purpose
Space_sim-specific engineering rules that extend the LLM Agent Collaboration Framework. Universal principles (SOLID, DRY, GRASP, IoC, Go best practices, Definition of Done) are defined in the framework; this document covers only what is unique to this project's packages, data model, and validated patterns.

## Last Updated
2026-06-15

## Table of Contents
1. Architectural Boundary Rule
2. SOLID — Package-Specific Guidance
3. State Management
4. JSON Data Structure Best Practices

---

## 1. Architectural Boundary Rule

Avoid hidden coupling between simulation, UI state, rendering, and JSON loading. These four concerns are intentionally separated across packages and must not leak into each other.

---

## 2. SOLID — Package-Specific Guidance

The framework defines SOLID principles. These rules apply them to space_sim's specific package layout.

### Single Responsibility

- Keep simulation math in `internal/sim/engine`, app orchestration in `internal/client/go/raylib/app/`, generic UI state in `internal/client/go/raylib/ui/`, and SOL-specific loading/generation in `internal/sim/`.
- Separate parsing, validation, transformation, and runtime mutation — do not mix them in one function.

### Open/Closed

- Add new object categories, UI modes, or JSON feature types through clear extension seams and exhaustive switch updates.
- When adding enums or mode flags, update the tests that lock in their intended values and ordering.

### Liskov Substitution

- If an interface is introduced, implementations must preserve expected behavior and error semantics.
- Do not create "partial" implementations that silently no-op unless that is the documented contract.
- Keep data transformation helpers behaviorally consistent across body types and feature types.

### Interface Segregation

- Prefer small interfaces at package boundaries. Consumers should depend only on the methods they actually need.
- Avoid "god interfaces" that combine loading, simulation, rendering, and persistence concerns.

### Dependency Inversion

- Depend on stable abstractions at boundaries, but keep concrete implementations where abstraction adds no value.
- High-level orchestration must not depend on low-level rendering details when a narrower contract will do.
- Pure logic must remain decoupled from Raylib and OS-specific concerns when feasible.

---

## 3. State Management

These rules were validated through repeated bugs in space_sim's simulation and UI layers. They extend SRP, GRASP Information Expert and Creator, DRY, and the DDD Aggregate Root pattern.

**S1 — Cohesion: keep state structs narrowly focused.**
A state struct should contain only fields that are semantically coupled — values that are always updated together and only meaningful together. Fields relevant only in some modes or for some subjects belong in a sub-struct or a separate type. A struct that mixes unrelated concerns becomes a surface where partial resets are possible.

**S2 — Encapsulation: mutations go through named methods.**
External callers must not write struct fields directly. Provide constructor functions or named transition methods (e.g. `StartTracking`, `StartJumpTo`) that set all relevant fields as a unit. In Go: unexport fields where practical; expose behavior through methods. Direct field access from outside the owning package is a red flag.

**S3 — Constructor completeness: a transition method owns all fields it touches.**
When a type has a named initializer or transition method, that method is responsible for resetting every field that is meaningful in that mode or context — not only the fields that existed when the method was first written. When a new field is added to a struct, immediately identify which constructor or transition functions should reset it and add the reset there. Never delegate that reset to individual call sites. Call-site resets drift apart as new paths are added without copying the full reset list.

**S4 — Owner-change invalidates all owned state.**
When the subject or owner of a state context changes (e.g. tracking a new object, switching datasets, changing a player target), treat all prior context as invalid and reset the entire context — not just the fields that are obviously stale. A partial reset is a latent bug waiting for the next field to be added.

**S5 — Single locus of mutation.**
Each mutable field must have one authoritative write site. If a field is set inside a method on the owning type, external call sites must not also set it. If state genuinely must live in two places, define the sync contract explicitly at the definition site — which is authoritative, which is derived, and where writes must propagate.

**S6 — Pass snapshots across boundaries; never share live mutable state.**
When state is consumed by a second subsystem (renderer, network serializer, test), pass a snapshot taken at a defined phase boundary, not a live reference to the mutable struct. A reader that can observe a partially-written struct will produce incorrect output non-deterministically. This is the read-side counterpart to the double-buffer write discipline.

**S7 — Make mode-dependent field validity explicit.**
When a struct contains fields only meaningful in a specific mode (e.g. `TrackDistance` is undefined in `CameraModeFree`), treat cross-mode field access as a design smell. Prefer making invalid states unrepresentable: extract mode-specific fields into a sub-struct or dedicated type that only exists when the mode is active. Where that refactor is not yet done, guard every read of a mode-gated field with an explicit mode check and document the invariant at the field definition.

---

## 4. JSON Data Structure Best Practices

These rules apply to space_sim's JSON system and body definition files.

### Schema and Evolution

- Treat JSON as a stable contract.
- Use explicit object shapes; avoid overloaded fields whose meaning changes by context.
- Prefer additive evolution over breaking field renames or semantic changes.
- Keep required fields truly required and document defaults clearly.

### Naming and Types

- Use clear, consistent snake_case field names.
- Keep units explicit and consistent across related fields.
- Prefer numbers for numeric concepts and strings for identifiers or enumerated labels.
- Avoid ambiguous mixed-type fields.

### Structure

- Group related values into nested objects such as `orbit`, `physical`, and `rendering`.
- Use arrays only for ordered collections; use objects/maps only when keyed lookup is intended.
- Keep repeated structures consistent across stars, planets, moons, rings, and procedural features.
- Use templates for shared defaults instead of duplicating full body definitions.

### Integrity and Validation

- Validate references such as parent names, template names, and feature types.
- Avoid duplicate names for runtime-addressable bodies.
- Preserve deterministic behavior for generated content by keeping seeds explicit.
- Keep enum-like strings normalized and documented.

### Maintainability

- Prefer readability over dense or compressed JSON.
- Keep comments and rationale in Markdown docs, not embedded in data files.
- When a JSON rule is subtle, encode it in both documentation and validation or tests.
- Update schema docs when fields are added, removed, or repurposed.
