# Scripts

This directory holds helper scripts for data maintenance, test runs, asset setup, and a small archived `legacy/` area for obsolete helpers kept only for historical reference.

## Standards

All scripts in this directory should follow these standards unless a script is explicitly marked as historical or one-off:

### General

- Scripts must have a single clear purpose and say what they modify or validate.
- Scripts with ongoing value belong in `scripts/`; temporary scripts should be deleted after the work is complete.
- Scripts that only target old repo layouts or obsolete workflows belong in `scripts/legacy/`.
- Prefer deterministic behavior, explicit paths, and clear failure messages.
- Default to safe behavior: validate inputs before making changes, and fail fast on bad state.
- If a script changes repo data, make the target files obvious in the usage text and logs.

### CLI and UX Contract

- Support `-h` and `--help` to print usage, key options, defaults, and examples.
- Support `-v` and `--verbose` to enable more detailed logging.
- Support `-q` and `--quiet` only when it does not hide actionable failures.
- If a script has destructive or write behavior, support a dry-run mode when practical.
- Exit with `0` on success and non-zero on failure.
- Write human-readable errors to stderr.
- Keep option names long-form and descriptive when more than one or two flags exist.

### Logging

- Default output should be concise and operator-focused.
- Verbose mode should explain what the script is checking, reading, or writing.
- Do not spam per-item output unless verbose mode is enabled or a failure occurs.
- Summarize changed files, validated files, or generated outputs at the end.

### Shell Script Standards

- Use `#!/usr/bin/env bash` unless a stricter shell requirement is intentional.
- Prefer `set -euo pipefail` for maintained scripts.
- Quote variables and paths.
- Prefer functions for non-trivial scripts.
- Use `command -v` to validate required external tools before use.
- Avoid fragile parsing pipelines when a simpler command or structured tool will do.

### Python Script Standards

- Use `#!/usr/bin/env python3`.
- Use `argparse` for maintained CLI scripts.
- Keep file I/O explicit and encoding-safe.
- Separate parsing, validation, and write logic into functions.
- Print concise summaries of what changed.

### Data and API Behavior

- Scripts that rewrite JSON should preserve valid structure and stable formatting expectations.
- Scripts should avoid inventing schema fields that are not understood by the runtime.
- If a script depends on repo-specific contracts, document those assumptions near the top of the file or in this README.
- Historical scripts that target old paths or obsolete APIs must be labeled clearly as legacy.

### Maintenance Expectations

- Keep scripts readable; performance matters only when it materially affects workflow.
- Reuse helper functions or shared conventions when it improves clarity.
- Remove or archive scripts that no longer match current repository layout.
- Update this README when a maintained script is added, removed, or substantially repurposed.

| Script | Purpose |
| --- | --- |
| `run_simple_test.sh` | Runs the app in performance mode with the `better` profile and 4 threads, streaming output to the terminal and saving console logs. |
| `run_debug_test.sh` | Runs the app in performance mode with the `worst` profile under a timeout, then summarizes debug and console log output. |
| `download_textures.sh` | Downloads texture assets for major solar system bodies into `data/assets/textures`. |
| `test_json_system.sh` | Validates the JSON system data, checks expected counts and key values, and verifies the app still builds. |
| `fix_orbital_periods.py` | Rewrites known body orbital periods in `data/systems/solar_system.json` using a curated set of astronomical values. |
| `fix_asteroid_periods.py` | Recomputes asteroid orbital periods in `data/systems/solar_system.json` from semi-major axis using Kepler's third law. |
| `fix_all_orbital_periods.py` | Performs a broader orbital-period rewrite for planets, dwarf planets, moons, and rings using a larger reference dataset. |

## Legacy Scripts

The following scripts are archived under `scripts/legacy/` because they target older file paths and obsolete workflows:

| Script | Legacy Reason |
| --- | --- |
| `legacy/add_importance.py` | Patches the removed `internal/smoke/simulation.go` path from an older layout. |
| `legacy/add_kuiper_belt_init.py` | Patches the removed `cmd/raylib-smoke/main.go` path from an older layout. |