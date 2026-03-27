# Coding Standards

## Purpose
Provide the default implementation standards for LLM Agents and human contributors working in this repository. This document is ordered from highest-priority rules to lower-level design guidance so an agent can make correct tradeoffs quickly.

## Last Updated
2026-03-26

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
6. Go Best Practices
	6.1 Design
	6.2 Errors
	6.3 Concurrency
	6.4 Data and APIs
	6.5 Control Flow and Style
	6.6 Testing
	6.7 Performance
7. JSON Data Structure Best Practices
	7.1 Schema and Evolution
	7.2 Naming and Types
	7.3 Structure
	7.4 Integrity and Validation
	7.5 Maintainability
8. Definition of Done
9. Reference Material

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

## 6. Go Best Practices

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

## 7. JSON Data Structure Best Practices

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

## 8. Definition of Done

Work is not done until all of the following are true:

1. The implementation matches repository architecture and these standards.
2. The code or docs are formatted and readable.
3. Relevant tests or validation steps have been run.
4. New or changed behavior is documented when discovery cost would otherwise be high.
5. Temporary work products have been removed or promoted to a proper long-lived location.

## 9. Reference Material

- SOLID overview: https://en.wikipedia.org/wiki/SOLID
- DRY overview: https://en.wikipedia.org/wiki/Don%27t_repeat_yourself
- GRASP overview: https://en.wikipedia.org/wiki/GRASP_(object-oriented_design)
- Effective Go: https://go.dev/doc/effective_go
- Go Code Review Comments: https://go.dev/wiki/CodeReviewComments
- Practical Go style guide: https://google.github.io/styleguide/go/
- JSON standard: https://www.rfc-editor.org/rfc/rfc8259
