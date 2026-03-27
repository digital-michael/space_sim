# Document Ownership

## Purpose
Record which migrated documents are owned in this repository, which remain shared, and which are transitional so future documentation cleanup can preserve the right source of truth.

## Last Updated
2026-03-26

## Table of Contents
1. Ownership Table
2. Lessons Learned Rule

This repository is the new primary home for the Space Sim application. The table below records which migrated documents are now owned here, which are shared, and which are intended to remain behind in the original repository once that cleanup happens.

## 1. Ownership Table

| Document | Decision | Notes |
| --- | --- | --- |
| [internal/space/package.md](../../internal/space/package.md) | Owned here | Canonical package and architecture reference for the application |
| [docs/history/app-refactor-plan.md](app-refactor-plan.md) | Owned here | Historical refactor rationale for the current application layout; no longer an active plan |
| [docs/wip/todo.md](../wip/todo.md) | Owned here | Canonical active and future work queue for the application |
| [docs/history/changelog.md](changelog.md) | Owned here | Completed work history moved out of the active queue |
| [docs/history/fullscreen-implementation.md](fullscreen-implementation.md) | Owned here | Application feature implementation history |
| [docs/history/json-only-migration.md](json-only-migration.md) | Owned here | JSON-driven application migration history |
| [docs/history/hardcoded-removal.md](hardcoded-removal.md) | Owned here | Application code/data migration history |
| [docs/history/implementation-summary.md](implementation-summary.md) | Owned here | JSON configuration system implementation summary |
| [docs/performance/performance-analysis.md](../performance/performance-analysis.md) | Owned here | Performance results and application analysis |
| [docs/performance/performance-results.md](../performance/performance-results.md) | Owned here | Performance result summary carried with the app |
| [docs/performance/debug-logging-guide.md](../performance/debug-logging-guide.md) | Owned here | Application debug workflow |
| [docs/schema/solar-system-json-schema.md](../schema/solar-system-json-schema.md) | Owned here | Canonical schema for solar-system datasets |
| [docs/schema/belt-features.md](../schema/belt-features.md) | Owned here | Application feature documentation |
| [data/README.md](../../data/README.md) | Owned here | Canonical dataset and asset layout guidance |
| [docs/history/lessons-learned.md](lessons-learned.md) | Transitional | Contains mixed content; app-specific sections belong here, Raylib-focused sections may remain in the original repo later |
| [docs/history/lessons-learned-double-buffering.md](lessons-learned-double-buffering.md) | Shared | Useful in both repositories |
| [docs/wip/smoke-test-origin.md](../wip/smoke-test-origin.md) | space_sim | Historical origin and planning document for the smoke test that became Space Sim |

## 2. Lessons Learned Rule

- Keep application architecture, design-decision, defect-history, and behavior-specific lessons here.
- Keep Raylib-specific graphics API constraints and general Raylib best-practice lessons in the original repo if they remain broadly useful there.
- Duplicate cross-cutting architectural lessons when both repositories benefit from them.