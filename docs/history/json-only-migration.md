# JSON-Only Configuration System

## Purpose
Explain the migration from fallback-based loading to required JSON-only configuration, including behavior changes, compatibility impact, testing outcomes, and file status after the transition.

## Last Updated
2026-03-26

## Table of Contents
1. Migration Complete
2. What Changed
  2.1 Before: Fallback System
  2.2 After: JSON-Only System
3. Key Changes
4. Testing Results
5. Usage
6. Benefits
7. Backward Compatibility
8. File Status

## Migration Complete ✅

The simulation has been successfully migrated from hard-coded + fallback to **JSON-only** configuration.

---

## What Changed

### Before (Fallback System)
```go
// If JSON loading failed, would fallback to createSolarSystem()
state, err = LoadSystemFromFile(configPath)
if err != nil {
    fmt.Println("Falling back to hard-coded solar system")
    state = createSolarSystem()  // FALLBACK
}
```

### After (JSON-Only System)
```go
// JSON loading is now REQUIRED - panic if file missing
state, err := LoadSystemFromFile(configPath)
if err != nil {
    panic(fmt.Sprintf("Failed to load system from %s: %v\n
        Please ensure the JSON configuration file exists or 
        run ./bin/export-system to generate it.", configPath, err))
}
```

---

## Key Changes

### `internal/space/simulation.go`

1. **NewSimulation()** - Now requires JSON file
   - Removed fallback to `createSolarSystem()`
   - Panics with helpful error message if JSON missing
   - Default path: `data/systems/solar_system.json`

2. **createSolarSystem()** - Marked internal-only
   - Updated documentation: "INTERNAL USE ONLY"
   - Used exclusively by export tool (`./bin/export-system`)
   - Not callable from normal application flow

---

## Testing Results

### Test 1: Valid JSON File ✅
```bash
$ go run ./cmd/test-json-required
✓ Loaded system from data/systems/solar_system.json
✓ Successfully loaded solar_system.json
```

### Test 2: Missing JSON File ✅
```bash
$ go run ./cmd/test-json-required
✓ Caught expected panic: Failed to load system from data/systems/nonexistent.json
Please ensure the JSON configuration file exists or run ./bin/export-system to generate it.
```

### Test 3: Export Tool Still Works ✅
```bash
$ ./bin/export-system
✓ Exported solar system to data/systems/solar_system.json
  Total bodies: 514
  Stars: 1, Planets: 8, Moons: 285, Dwarf Planets: 11, Rings: 9, Asteroids: 200
```

---

## Usage

### Running the Application
```bash
# Default: loads data/systems/solar_system.json
./bin/space-sim

# Custom config file
./bin/space-sim --system-config=path/to/custom.json
```

### If JSON File is Missing
```bash
# Regenerate from hard-coded system
./bin/export-system

# This creates data/systems/solar_system.json
```

---

## Benefits

1. **Single Source of Truth** - JSON is now the primary configuration method
2. **Clear Error Messages** - Helpful panic message guides users to fix the issue
3. **Consistent Behavior** - No surprising fallback behavior
4. **Simpler Code** - Removed conditional fallback logic
5. **Export Tool** - Still available for regenerating JSON from hard-coded data

---

## Backward Compatibility

⚠️ **Breaking Change**: Applications must have `data/systems/solar_system.json` to run.

**Migration Path:**
1. Run `./bin/export-system` to generate the JSON file
2. The JSON file is already included in the repository (171KB)
3. Custom configurations can be loaded via `--system-config` flag

---

## File Status

### Production Files (Required)
- ✅ `data/systems/solar_system.json` - Primary configuration (171KB)
- ✅ `internal/space/loader.go` - JSON loading infrastructure
- ✅ `internal/space/config.go` - JSON schema definitions

### Legacy Files (Export Only)
- 🔒 `createSolarSystem()` in `simulation.go` - Internal use only (export tool)
- 🔒 `ExportCreateSolarSystem()` - Exposed for export tool

### Tooling
- ✅ `cmd/export-system/main.go` - Regenerate JSON from hard-coded
- ✅ `cmd/test-json-required/main.go` - Test JSON requirement

---

## Date: February 17, 2026

**Status:** Production-ready, JSON-only configuration system active.
