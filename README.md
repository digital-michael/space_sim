# Space Sim

Space Sim is a standalone real-time solar system simulator built with Go and Raylib. It started life as a prototype in another repository, but this repository is now the primary home for the application, its simulation code, its JSON-driven system data, and its application-specific design history.

## Scope

- Interactive solar system simulation and visualization
- JSON-driven body, feature, and system configuration
- Performance testing and rendering experiments
- Application-focused architecture and implementation documentation

## Repository Layout

```text
space_sim/
├── cmd/space-sim/        # Application entrypoint
├── internal/space/       # Simulation, app, UI, and Raylib integration
├── configs/              # App configuration
├── data/                 # Solar-system datasets, templates, and assets
├── docs/                 # Application docs and retained lessons
└── scripts/              # Supporting validation scripts
```

## Prerequisites

- Go 1.24+
- Raylib available for the local build environment

## Common Commands

```bash
make build
make run
make test
make json-check
```

## Running

```bash
./bin/space-sim
./bin/space-sim --system-config=data/systems/solar_system.json
```

## Documentation

- [docs/README.md](docs/README.md): documentation index and folder guide
- [docs/history/doc-ownership.md](docs/history/doc-ownership.md): document ownership and migration decisions
- [docs/history/implementation-summary.md](docs/history/implementation-summary.md): JSON configuration system implementation summary
- [docs/history/fullscreen-implementation.md](docs/history/fullscreen-implementation.md): fullscreen and dynamic resize implementation notes
- [docs/history/json-only-migration.md](docs/history/json-only-migration.md): migration to required JSON-only system loading
- [docs/history/hardcoded-removal.md](docs/history/hardcoded-removal.md): removal of hard-coded solar-system data
- [docs/history/lessons-learned.md](docs/history/lessons-learned.md): implementation lessons and defect history
- [docs/performance/performance-results.md](docs/performance/performance-results.md): consolidated performance test results
- [docs/performance/performance-analysis.md](docs/performance/performance-analysis.md): performance analysis and investigation notes
- [docs/performance/debug-logging-guide.md](docs/performance/debug-logging-guide.md): debug logging workflow for performance hangs
- [internal/space/package.md](internal/space/package.md): package architecture and boundaries
- [docs/schema/solar-system-json-schema.md](docs/schema/solar-system-json-schema.md): solar-system JSON schema
- [data/README.md](data/README.md): data layout and configuration guidance

## Status

This repository is currently being established from the original prototype source. Expect some documentation references and scripts to continue being normalized as the migration is verified.