# Space Sim

[![Inner Solar Tour](https://img.youtube.com/vi/JB9y0OEQ5go/0.jpg)](https://www.youtube.com/watch?v=JB9y0OEQ5go)

Space Sim is a standalone real-time solar system simulator built with Go and Raylib. It started life as a prototype in another repository, but this repository is now the primary home for the application, its simulation code, its JSON-driven system data, and its application-specific design history.

## Scope

- Interactive solar system simulation and visualization
- JSON-driven body, feature, and system configuration
- Performance testing and rendering experiments
- Application-focused architecture and implementation documentation

## Repository Layout

```text
space_sim/
├── cmd/space-sim-direct/ # In-process binary (Raylib + server, no network)
├── cmd/space-sim-grpc/   # Embedded ConnectRPC binary (Raylib + gRPC server)
├── cmd/space-sim-repl/   # Interactive CLI client for a running grpc binary
├── internal/             # Simulation, server, client, transport, and protocol
├── api/                  # Protobuf definitions and generated Go stubs
├── configs/              # App configuration (app.json)
├── data/                 # Solar-system datasets, templates, and assets
├── docs/                 # Application docs and retained lessons
└── scripts/              # Supporting scripts and tour files
```

## Prerequisites

- Go 1.24+
- Raylib available for the local build environment
- ffmpeg (optional — required only for video recording; see [README-extra.md](README-extra.md))

## Common Commands

```bash
make build          # Build all three binaries
make run            # Build and run space-sim-direct (in-process)
make run-grpc       # Build and run space-sim-grpc
make test           # Run all tests
make json-check     # Validate solar-system JSON files
```

## Running

```bash
# In-process (no gRPC, simplest):
./bin/space-sim-direct
./bin/space-sim-direct --system-config=data/systems/solar_system.json

# gRPC-coupled (Raylib client + embedded ConnectRPC server):
./bin/space-sim-grpc

# REPL client (requires a running space-sim-grpc):
./bin/space-sim-repl
./bin/space-sim-repl --script scripts/solar-tour.txt

# Reset app.json to factory defaults:
./bin/space-sim-grpc --reset
```

## Key Flags (space-sim-grpc)

| Flag | Description |
|------|-------------|
| `--system-config <path>` | Load an alternate solar system JSON |
| `--render-scale <n>` | Fixed render scale (e.g. `2` for 2×) — not saved to app.json |
| `--render-size <WxH>` | Fixed render resolution — not saved to app.json |
| `--no-msaa` | Disable MSAA 4× (enabled by default) |
| `--reset` | Write factory defaults to app.json and exit |

## Documentation

- [docs/README.md](docs/README.md): documentation index and folder guide
- [docs/history/changelog.md](docs/history/changelog.md): completed work archive
- [docs/history/lessons-learned.md](docs/history/lessons-learned.md): implementation lessons and defect history
- [docs/history/lessons-learned-double-buffering.md](docs/history/lessons-learned-double-buffering.md): concurrency and double-buffer anti-patterns
- [docs/performance/performance-results.md](docs/performance/performance-results.md): consolidated performance test results
- [docs/performance/performance-analysis.md](docs/performance/performance-analysis.md): performance analysis and investigation notes
- [docs/performance/debug-logging-guide.md](docs/performance/debug-logging-guide.md): debug logging workflow for performance hangs
- [docs/schema/solar-system-json-schema.md](docs/schema/solar-system-json-schema.md): solar-system JSON schema
- [docs/standards/agent-readme.md](docs/standards/agent-readme.md): repository map and architectural boundaries
- [data/README.md](data/README.md): data layout and configuration guidance
- [README-extra.md](README-extra.md): optional dependencies and platform setup (video recording, etc.)
- [internal/sim/package.md](internal/sim/package.md): sim package architecture and boundaries

## Status

Active development. Core simulation, gRPC transport, persistence, and rendering pipeline are complete. See [docs/wip/todo.md](docs/wip/todo.md) for the current work queue.