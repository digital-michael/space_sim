# Coding Standards

## Purpose
Provide the default implementation standards for LLM Agents and human contributors working in this repository. This document is ordered from highest-priority rules to lower-level design guidance so an agent can make correct tradeoffs quickly.

## Last Updated
2026-04-27

## Table of Contents
1. Priority Order
2. Core Engineering Rules
3. SOLID Guidance
	3.1 Single Responsibility Principle
	3.2 Open/Closed Principle
	3.3 Liskov Substitution Principle
	3.4 Interface Segregation Principle
	3.5 Dependency Inversion Principle
4. DRY Guidance
5. GRASP Guidance
6. IoC — Inversion of Control
7. Go Best Practices
	7.1 Design
	7.2 Errors
	7.3 Concurrency
	7.4 Data and APIs
	7.5 Control Flow and Style
	7.6 Testing
	7.7 Performance
	7.8 State Management
8. JSON Data Structure Best Practices
	8.1 Schema and Evolution
	8.2 Naming and Types
	8.3 Structure
	8.4 Integrity and Validation
	8.5 Maintainability
9. Definition of Done
10. Reference Material

## 1. Priority Order

Apply standards in this order when they conflict:

1. Preserve correctness and user-visible behavior.
2. Preserve architectural boundaries already established in the repository.
3. Prefer simple, explicit code over clever or dense code.
4. Improve maintainability and testability.
5. Improve performance, but not by making the code obscure.
6. Remove duplication only when the abstraction is clearer than the repetition.

## 2. Core Engineering Rules

- Fix root causes, not only symptoms.
- Keep changes local, minimal, and consistent with existing package boundaries.
- Prefer explicit names, explicit control flow, and explicit error handling.
- Avoid hidden coupling between simulation, UI state, rendering, and JSON loading.
- Keep public APIs small and stable.
- Add or update tests when behavior, invariants, or branching logic change.
- Document non-obvious behavior in docs when an agent would otherwise have to rediscover it.
- Use interfaces or generics only when they simplify usage, isolate a boundary, or reduce repeated boilerplate. Do not add abstraction for its own sake.

## 3. SOLID Guidance

### Single Responsibility Principle

- Each package, file, type, and function should have one clear reason to change.
- Keep simulation math in `internal/space/engine`, app orchestration in `internal/space/app`, generic UI state in `internal/space/ui`, and SOL-specific loading/generation in `internal/space`.
- Separate parsing, validation, transformation, and runtime mutation work instead of mixing them in one function.

### Open/Closed Principle

- Prefer extending behavior through new helpers, adapters, configuration, or new types instead of repeatedly modifying unrelated call sites.
- Add new object categories, UI modes, or JSON feature types through clear extension seams and exhaustive switch updates.
- When adding enums or mode flags, update the tests that lock in their intended values and ordering.

### Liskov Substitution Principle

- If an interface is introduced, implementations must preserve expected behavior and error semantics.
- Do not create “partial” implementations that silently no-op unless that is the documented contract.
- Keep data transformation helpers behaviorally consistent across body types and feature types.

### Interface Segregation Principle

- Prefer small interfaces at package boundaries.
- Consumers should depend only on the methods they actually need.
- Avoid “god interfaces” that combine loading, simulation, rendering, and persistence concerns.

### Dependency Inversion Principle

- Depend on stable abstractions at boundaries, but keep concrete implementations where abstraction adds no value.
- High-level orchestration should not depend on low-level rendering details when a narrower contract will do.
- Pure logic should remain decoupled from Raylib and OS-specific concerns when feasible.

## 4. DRY Guidance

- Eliminate duplicated business rules, constants, validation logic, and mapping tables.
- Keep a single source of truth for dataset sizes, category ordering, threshold sets, and schema assumptions.
- Do not force reuse when the resulting abstraction becomes harder to read than two explicit call sites.
- Prefer extraction when the duplicated logic is likely to evolve together.
- Prefer tables, typed constants, and focused helpers over repeated string literals and magic numbers.

## 5. GRASP Guidance

Apply GRASP mainly as a design sanity check:

- Information Expert: place behavior where the required data already lives.
- Creator: let types that aggregate or own data construct closely related values when practical.
- Controller: keep orchestration in application-level controllers, not in low-level entities.
- Low Coupling: reduce cross-package knowledge and avoid circular dependencies.
- High Cohesion: keep files and types narrowly focused.
- Polymorphism: use interfaces or type-driven dispatch when behavior truly varies by role.
- Pure Fabrication: create helper types when needed to keep domain types focused.
- Indirection: introduce an adapter only when it genuinely decouples unstable details.
- Protected Variations: isolate likely-to-change seams such as input handling, JSON formats, and rendering integrations.

## 6. IoC — Inversion of Control

Inversion of Control means that high-level modules define what they need (through parameters, interfaces, or constructor injection) and low-level modules supply it — rather than high-level modules reaching inward to construct or locate their own dependencies.

- **Inject dependencies at construction time.** Pass collaborators (e.g. a renderer, a physics engine, a clock, a logger) into a type's constructor rather than letting the type instantiate them internally. This makes wiring explicit and makes tests straightforward.
- **Accept interfaces at boundaries, return concrete types inside.** A type that depends on a storage layer should accept an interface; the concrete implementation is wired once at the composition root (`main` or the top-level app constructor).
- **Keep the composition root thin.** Wire dependencies in `main` or an `App` constructor. Constructors and workers should not reach into global state, call `os.Getenv`, or open files unless that is their explicit responsibility.
- **Avoid service locators and global registries.** Passing a locator object so code can look up its own dependencies is still hidden coupling; prefer explicit injection.
- **IoC is not a framework requirement.** In Go, constructor injection with plain structs and interfaces is sufficient. Avoid heavy DI containers unless the wiring is genuinely complex and the container earns its weight.
- **Apply at package boundaries, not inside packages.** Within a cohesive package, direct construction is fine. IoC is most valuable where packages cross architectural lines (e.g. simulation → rendering, server → persistence).

## 7. Go Best Practices

### Design

- Keep packages cohesive and import directions clean.
- Prefer constructors and zero-value-safe types where practical.
- Accept `context.Context` for operations that can block, run in loops, or be cancelled.
- Return concrete types unless callers need an interface.
- Keep exported APIs intentional; do not export by default.

### Errors

- Return errors with actionable context using `fmt.Errorf(... %w ...)`.
- Fail fast on invalid configuration and corrupted assumptions.
- Avoid panic except for unrecoverable startup/configuration paths that are intentionally fatal.
- Do not swallow errors silently.

### Concurrency

- Be explicit about ownership of mutable state.
- Keep lock scope short and easy to reason about.
- Avoid nested locking unless it is clearly safe and documented.
- Prefer message passing or phase-separated mutation over ad hoc shared-state writes.
- Preserve the existing double-buffer invariants when touching simulation state.

### Data and APIs

- Prefer typed constants over raw integers and strings.
- Prefer small structs with meaningful field names.
- Avoid boolean parameter lists when a config struct or option type is clearer.
- Keep serialization structs stable and backwards-compatible where possible.

### Control Flow and Style

- Write straightforward branches instead of compact but opaque expressions.
- Use early returns to reduce nesting.
- Keep functions short enough to understand in one pass, unless a longer function is the clearest expression of a state machine.
- Comments should explain intent, invariants, or unusual behavior, not restate obvious code.
- Run `gofmt` formatting expectations implicitly by keeping formatting idiomatic.

### Testing

- Add focused unit tests for invariants, enum stability, transformation rules, and edge-case logic.
- Prefer deterministic tests with fixed seeds and explicit expectations.
- Test behavior, not implementation trivia.
- When adding a switch or table that encodes important ordering, add a regression test for it.

### Performance

- Measure before optimizing.
- Prefer predictable allocations and simple data flow.
- Avoid micro-optimizations that make the codebase harder to evolve.
- When performance tradeoffs are non-obvious, document the invariant or benchmark result that justifies them.

### State Management

Goal: prevent stale-data access and partial-initialization bugs. These rules extend SRP §3.1, GRASP Information Expert and Creator §5, DRY §4, and the DDD Aggregate Root pattern.

**S1 — Cohesion: keep state structs narrowly focused.**
A state struct should contain only fields that are semantically coupled — values that are always updated together and only meaningful together. Fields that are only relevant in some modes or for some subjects belong in a sub-struct or a separate type. A struct that mixes unrelated concerns becomes a surface where partial resets are possible. (Extends SRP §3.1, GRASP High Cohesion §5.)

**S2 — Encapsulation: mutations go through named methods.**
External callers must not write struct fields directly. Provide constructor functions or named transition methods (e.g. `StartTracking`, `StartJumpTo`) that set all relevant fields as a unit. In Go: unexport fields where practical; expose behavior through methods. Direct field access from outside the owning package is a red flag. (Extends GRASP Information Expert §5, Parnas Information Hiding, Command-Query Separation.)

**S3 — Constructor completeness: a transition method owns all fields it touches.**
When a type has a named initializer or transition method, that method is responsible for resetting every field that is meaningful in that mode or context — not only the fields that existed when the method was first written. When a new field is added to a struct, immediately identify which constructor or transition functions should reset it and add the reset there. Never delegate that reset to individual call sites. Call-site resets drift apart as new paths are added without copying the full reset list. (Extends GRASP Creator §5, DDD Aggregate Root. See LL #39.)

**S4 — Owner-change invalidates all owned state.**
When the subject or owner of a state context changes (e.g. tracking a new object, switching datasets, changing a player target), treat all prior context as invalid and reset the entire context — not just the fields that are obviously stale. A partial reset is a latent bug waiting for the next field to be added. (DDD Aggregate Root: the root enforces invariants on every transition, including re-assignment.)

**S5 — Single locus of mutation.**
Each mutable field must have one authoritative write site. If a field is set inside a method on the owning type, external call sites must not also set it. If state genuinely must live in two places (e.g. a session copy and a persistent copy), define the sync contract explicitly at the definition site — which is authoritative, which is derived, and where writes must propagate. Do not discover the two-write requirement by debugging a persistence regression. (Extends DRY §4. See LL #33.)

**S6 — Pass snapshots across boundaries; never share live mutable state.**
When state is consumed by a second subsystem (renderer, network serializer, test), pass a snapshot taken at a defined phase boundary, not a live reference to the mutable struct. A reader that can observe a partially-written struct will produce incorrect output non-deterministically. This is the read-side counterpart to the double-buffer write discipline already required for simulation state.

**S7 — Make mode-dependent field validity explicit.**
When a struct contains fields that are only meaningful in a specific mode or state (e.g. `TrackDistance` and `TrackOffset` are undefined in `CameraModeFree`; a "selected target" index is undefined when nothing is selected), treat cross-mode field access as a design smell. At the design level, prefer making invalid states unrepresentable: extract mode-specific fields into a sub-struct or a dedicated type that only exists when the mode is active. Where that refactor is not yet done, guard every read of a mode-gated field with an explicit mode check and document the invariant at the field definition. Use this rule to scrutinize existing structs: if you find yourself checking `if mode == X` before reading a field, that field is a candidate for extraction. (Extends S1 Cohesion; relates to the "make illegal states unrepresentable" principle from type-driven design.)

## 8. JSON Data Structure Best Practices

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

## 9. Definition of Done

Work is not done until all of the following are true:

1. The implementation matches repository architecture and these standards.
2. The code or docs are formatted and readable.
3. Relevant tests or validation steps have been run.
4. New or changed behavior is documented when discovery cost would otherwise be high.
5. Temporary work products have been removed or promoted to a proper long-lived location.

## 10. Reference Material

- SOLID overview: https://en.wikipedia.org/wiki/SOLID
- DRY overview: https://en.wikipedia.org/wiki/Don%27t_repeat_yourself
- GRASP overview: https://en.wikipedia.org/wiki/GRASP_(object-oriented_design)
- DDD Aggregate pattern: https://martinfowler.com/bliki/DDD_Aggregate.html
- Command-Query Separation (CQS): https://martinfowler.com/bliki/CommandQuerySeparation.html
- Parnas Information Hiding: https://en.wikipedia.org/wiki/Information_hiding
- Effective Go: https://go.dev/doc/effective_go
- Go Code Review Comments: https://go.dev/wiki/CodeReviewComments
- Practical Go style guide: https://google.github.io/styleguide/go/
- JSON standard: https://www.rfc-editor.org/rfc/rfc8259
