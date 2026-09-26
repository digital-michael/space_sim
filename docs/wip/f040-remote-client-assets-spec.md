# F-040 — Remote Client Asset Bundle

## Purpose
Specify how a truly remote `space-sim` client obtains the simulation content it needs to render — body metadata and texture assets — without any local `data/` directory. Replaces the current remote-renderer behaviour, in which the client draws flat category-coloured spheres because no rendering metadata crosses the wire.

## Last Updated
2026-09-22 (Phases 1–4 complete; D5 corrected, D7/D8 amended during implementation)

## Table of Contents
1. Goals and Non-Goals
2. Locked Decisions
3. Architecture
	3.1 Data Classification
	3.2 Bundle Contents
	3.3 Wire Surface
	3.4 Client Startup Flow
	3.5 Path Resolution
4. Security
5. Affected Files
6. Phases and Work Items
7. Acceptance Criteria
8. Open Questions
9. Related Work

---

## 1. Goals and Non-Goals

### Goals

| # | Goal |
|---|---|
| G1 | A `space-sim --server host:port` client renders with full fidelity — textures, night textures, atmosphere, luminosity, axial tilt, rotation — on a machine with no repo checkout and no `data/` directory |
| G2 | Content is server-authoritative; the server decides what the client renders |
| G3 | Assets are fetched once and cached across restarts; steady-state connect transfers nothing |
| G4 | Cache correctness requires no explicit invalidation step |
| G5 | One transport, one auth story — no second port or protocol beside ConnectRPC |

### Non-Goals

- Per-asset granular fetching. Deliberately deferred; see §8 and F-040-Later.
- Streaming texture assets for 3D models (F-021 Ph2 IQM, F-033 Ph2). The bundle carries whatever `data/assets/` holds, so these arrive for free once added there, but no model-specific work is in scope.
- Client-authoritative or user-supplied content. Out of scope until F-011 defines trust.
- Reducing per-frame snapshot bandwidth. Tracked separately under F-010 bandwidth mitigations.

---

## 2. Locked Decisions

These were confirmed by the practitioner on 2026-09-22 and are not to be re-litigated during implementation.

| # | Decision | Rationale |
|---|---|---|
| D1 | Visual fidelity ships before multi-client presence | A correct-looking single remote client is the more useful intermediate state; session wiring follows |
| D2 | **No `--data-root` dev override.** Remote mode *always* fetches the bundle, including on localhost | An untested remote path is how this class of bug recurs. Keeping it strictly uniform surfaces bundle defects in development rather than on first real deployment |
| D3 | Bundle delivered over the existing ConnectRPC transport, not a static file server | A file server adds a second protocol and port, carries path-traversal surface, offers no integrity guarantee, and sits outside whatever auth F-011 introduces |
| D4 | **No `GetBodyMetadata` RPC.** The bundle carries `data/systems/*.json`; the client runs the existing loader against it | Avoids hand-mirroring 27 `ObjectMetadata` fields into proto and re-mirroring on every future render field. The JSON is already the declared source of truth for body content |
| D5 | **CORRECTED 2026-09-22 — bundle INCLUDES `data/profiles/`.** The original decision excluded them | The original reasoning — "profiles are client-local user configuration" — was a misreading, caught by the isolated-client acceptance test. `input.LoadKeyMap` treats the profile file as **mandatory** and the user's own overrides file as optional, so `data/profiles/*.json` are *shipped hardware profiles*, i.e. app content. Excluding them protected nothing and made a genuinely remote client fail at startup with `reading profile "data/profiles/laptop.json": no such file`. Real user customization lives in the keybindings config file, which the bundle never touches |
| D6 | Bundle versioned by content hash | Content addressing means the cache is never stale and needs no invalidation protocol |
| D7 | The active system path is reported by `GetBundleInfo`, **not** obtained from `SystemService.GetActiveSystem` | Amended 2026-09-22 during Phase 2. `GetActiveSystem` returns 404 on a headless server: its handler routes through the Raylib app's command channel and no such app exists. Rather than make F-040 depend on fixing that boundary violation, the server reports what it loaded. It is the same connect-time question — "what content should I load?" — and answering both in one call also removes a round trip |
| D8 | RPCs named `GetBundleInfo` / `FetchBundle` inside `AssetService` | Shortened from the draft's `GetAssetBundleInfo` / `FetchAssetBundle`: the service name already carries "Asset", so the longer forms stutter |

---

## 3. Architecture

### 3.1 Data Classification

The current problem is tractable once two different concerns are separated:

| Concern | Nature | Delivery |
|---|---|---|
| **Simulation content** — body definitions, orbital elements, render metadata, texture binaries, ship catalogs | Server-authoritative, changes only when the operator changes data files | Bundle (§3.2) |
| **Client configuration** — the user's keybinding overrides, window geometry, render toggles | User-local, per-machine, per-person | Stays local, in `configs/app.json` and the keybindings config path. Note these live *outside* `data/`; shipped hardware profiles under `data/profiles/` are content and ARE bundled (D5) |

Conflating these is what makes "just ship the data directory" feel wrong: it would have the server dictate the user's keybindings.

### 3.2 Bundle Contents

A zip archive containing simulation content only, rooted so that entries match their current repo-relative paths:

```
data/systems/**        # system + body definitions (596K)
data/bodies/**         # body templates (16K)
data/ships/**          # ship catalog (12K)
data/profiles/**       # shipped keybinding hardware profiles (16K) — mandatory at startup
data/assets/**         # textures (4.5M)
```

Nothing under `data/` is excluded. User configuration is excluded by virtue of living
outside `data/` entirely — in `configs/app.json` and the keybindings config path (D5).

Total ≈ 5.1 MB uncompressed today. At this size a whole-bundle transfer is a sub-second LAN operation and per-asset granularity earns nothing; see §8.

Entries keep their `data/`-prefixed paths so that a bundle root plus an existing relative path composes correctly, and so that a bundle is inspectable against the repo it came from.

### 3.3 Wire Surface

A new `AssetService` in its own `api/proto/spacesim/v1/asset.proto`, following the precedent of `session.proto` and the codebase's pattern of many small focused services rather than a few broad ones.

| RPC | Status | Purpose |
|---|---|---|
| `AssetService.GetBundleInfo` | **Built — Phase 2** | Returns content hash, byte size, entry count, and `active_system_path`. Cheap unary call made on every connect |
| `AssetService.FetchBundle` | **Built — Phase 2** | Server-streaming, 64 KiB chunks. `expected_hash` mismatch returns `FailedPrecondition` rather than streaming a different bundle than the client asked for |
| `SystemService.GetActiveSystem` | **Not used** — see D7 | Returns 404 on a headless server; superseded by `GetBundleInfo.active_system_path` |
| `SystemService.ListSystems` | **Unavailable headless** | Would populate the in-app system selector, but returns 404 on `space-sim-server`. Phase 4 must not depend on it; tracked as its own defect |

Server builds the bundle and computes its hash at startup by walking the included trees. Measured on the real tree: 94 entries, 4.03 MB compressed, ~0.7 s. The hash is reproducible across separate processes, verified.

The active system path comes from `world.World.SystemPath()`, added in Phase 2. `World` is the authoritative owner of which system it loaded, and it applies the `DefaultSystemPath` fallback internally — so the server reports the resolved path, never an empty string.

### 3.4 Client Startup Flow

Remote mode (`--server` set):

1. Call `GetBundleInfo` → hash `H`.
2. If `<cache>/bundles/H/` exists and is marked complete, use it. Go to 6.
3. Call `FetchBundle`, streaming into a temporary file.
4. Verify the received bytes hash to `H`. Mismatch is a hard failure — do not unpack.
5. Unpack into `<cache>/bundles/H.partial/` with traversal validation (§4), then atomically rename to `<cache>/bundles/H/`. The rename is the completeness marker; a crashed unpack leaves a `.partial` directory that is discarded on the next run.
6. Set the session's asset root to `<cache>/bundles/H/`.
7. Take `active_system_path` from the `GetBundleInfo` response (step 1 — no extra call) and load it from under the asset root. Use `sim.LoadSystemFromDir`: since F-034 a system is a directory of per-category files, so `LoadSystemFromFile` is the legacy v1 path and does not apply to current content.
8. Build a `name → engine.ObjectMetadata` map from the loaded state. This is used for metadata lookup only — positions, velocity, and visibility remain server-authoritative from the snapshot stream.
9. In `protoToSnapshot`, populate each body's `Meta` from the map by name, resolving texture paths against the asset root. Bodies absent from the map (procedurally generated belt asteroids) retain the existing category-default fallback.

Cache location: `os.UserCacheDir()/space-sim/bundles/<hash>/`.

Standalone mode (no `--server`) is unchanged and continues to read `./data`. It has the files by definition and there is no server to fetch from. D2's uniformity requirement applies to the remote path, which must never shortcut to a local `data/`.

Mid-session system change: an operator `LoadSystem` alters the body set. The client re-queries `GetBundleInfo` and rebuilds the metadata map when it observes a snapshot body name absent from the current map, rate-limited so a belt of unmatched asteroids cannot trigger a refetch storm. No polling in steady state.

### 3.5 Path Resolution

Five call sites currently hardcode a `data/`-relative path. Each needs an explicit resolution decision.

| Site | Current | Resolution |
|---|---|---|
| `renders_atmosphere.go:265` | `const skyTexPath = "data/assets/textures/starfield_8k.jpg"` | Resolve against the renderer's asset root. **This is a real renderer change** — the skysphere path does not arrive via `ObjectMetadata`, so it is not covered by rooting metadata paths |
| `system_selector.go:14` | `const defaultSystemConfigPath = "data/systems/solar_system"` | Asset-root relative in remote mode |
| `system_selector.go:129` | `discoverSystemOptionsFromDir("data/systems")` | Remote mode must use the `ListSystems` RPC instead of scanning a local directory |
| `app.go:22` | `defaultProfilesDir = "data/profiles"` | **Rooted** via `Config.profilesDir()`. Originally marked unchanged per the mistaken D5; a remote client cannot start without a profile |
| `input/loader.go:37` | Doc comment only | Update wording if it becomes misleading |

Texture loading itself needs no change: `loadTexture` (`renders_assets.go:13`) takes a path string and caches on it, so paths resolved to the bundle root before they reach `Meta.TexturePath` work as-is.

---

## 4. Security

Unpacking an archive is the one genuinely hazardous step in this design and must not be treated as incidental.

- **Zip-slip / path traversal.** Every entry path must be validated to resolve strictly inside the target root before any write. Reject absolute paths, reject any path whose cleaned form escapes the root via `..`, reject symlink entries outright.
- **Size limits.** Enforce a maximum total uncompressed size and a maximum per-entry size to bound a decompression-bomb archive.
- **Integrity before unpack.** Hash verification (§3.4 step 4) happens before unpacking, not after.
- **Trust boundary.** The bundle is trusted exactly as much as the server is. This is acceptable pre-F-011 because the server is already fully trusted for simulation state. When F-011 lands, bundle fetch inherits its transport auth with no design change, since it rides the same ConnectRPC connection (D3).

---

## 5. Affected Files

| File | Change |
|---|---|
| `api/proto/spacesim/v1/asset.proto` | **Done.** New file: `AssetService` with `GetBundleInfo`, `FetchBundle` |
| `internal/sim/world/world.go` | **Done.** Added `SystemPath()` accessor and exported `DefaultSystemPath` |
| `api/gen/spacesim/v1/**` | Regenerate — `make proto`, never edited by hand |
| `internal/server/assets/` | **Done.** Bundle construction, hashing, entry enumeration |
| `internal/transport/grpc/asset_handler.go` | **Done.** `GetBundleInfo`, `FetchBundle` handlers |
| `cmd/space-sim-server/main.go` | **Done.** Builds the bundle at startup, adds `--data-dir`, registers `AssetService` on its mux |
| `internal/transport/grpc/server.go` | Added `Asset` to `Handlers` — but note `grpcserver.New`/`Handlers` is currently DEAD CODE, nothing calls it; the live registration is in `cmd/space-sim-server/main.go` |
| `internal/client/assets/` | **New package.** Bundle fetch, hash verification, safe unpack, cache management, asset-root resolution |
| `cmd/space-sim/main.go` | Fetch bundle before app start in remote mode; build and consult the metadata map in `protoToSnapshot`; retire `categoryDefaultColor` / `categoryDefaultMaterial` to fallback-only duty |
| `internal/client/go/raylib/ui/render/renders_atmosphere.go` | Root the skysphere texture path |
| `internal/client/go/raylib/app/system_selector.go` | Asset-root default path; `ListSystems` RPC in remote mode |
| `internal/client/go/raylib/app/session.go` | Carry the asset root on the session |

---

## 6. Phases and Work Items

### Phase 1 — Server-side bundle ✅ Complete 2026-09-22

- [x] `internal/server/assets`: walk the included trees, build a zip in memory, compute a content hash
- [x] Enumerate entry count and total size for `GetBundleInfo`
- [x] Exclude `data/profiles/` per D5
- [x] Unit tests: deterministic hash for identical trees, hash changes when any included file changes, `profiles/` absent from output
- [x] Determinism secured by sorted entries, fixed modtime (1980-01-01, earliest the zip format represents without clamping) and fixed 0644 mode. Without these, identical content hashes differently per build and invalidates every client cache on restart
- [x] Symlinks skipped at build time, so no entry can resolve outside the client cache root after unpack

### Phase 2 — Wire surface ✅ Complete 2026-09-22

- [x] New `asset.proto` with `AssetService`; every message carries a `version` field per existing convention
- [x] `make proto`
- [x] Implement `asset_handler.go`; chunk size 64 KiB, reasoning recorded inline
- [x] `FetchBundle` rejects a mismatched `expected_hash` with `FailedPrecondition`
- [x] `world.World.SystemPath()` + exported `DefaultSystemPath`, so the server reports a resolved active-system path rather than an empty string
- [x] Registered on the live `cmd/space-sim-server` mux, with a `--data-dir` flag (default `data`). Bundle-build failure warns loudly and serves `Unavailable` rather than refusing to start
- [x] Integration tests over a real `httptest` server rather than bufconn — streaming RPCs need an actual transport. Covers round trip, reassembly hashing to the advertised value, multi-chunk spanning, hash-mismatch rejection, empty-hash permitted, nil-bundle `Unavailable`
- [x] Verified end-to-end against a live server: 94 entries, 4.03 MB, 65 chunks over the wire, identical hash across separate processes

### Phase 3 — Client fetch and cache ✅ Complete 2026-09-22

- [x] `internal/client/assets`: `GetBundleInfo` → cache hit/miss → stream → verify → safe unpack → atomic rename
- [x] Traversal, symlink, and size-limit validation per §4, with tests using deliberately malicious archives
- [x] Discard stale `.partial` directories on startup
- [x] Unit tests: cache hit skips fetch entirely (proved with a fetch counter); corrupt payload fails before unpack; traversal entry rejected
- [x] `Unpack` validates **every** entry before writing **any**, so a hostile archive leaves no partial output even when used standalone
- [x] Size limits are package vars rather than consts, so the bomb guards are testable without generating hundreds of megabytes
- [x] `Root.Resolve(rel)` joins a bundle-relative path, so existing `ObjectMetadata.TexturePath` values compose against the cache root with no rewriting
- [x] Test defines its own fake `AssetService` rather than reusing the server handler, honouring the rule that `internal/client/*` never imports `internal/server/*`

**Known coverage limit:** archives written by `archive/zip` carry accurate declared sizes, so the declared-size check fires before the `io.LimitReader` bound. That LimitReader is defence-in-depth against a lying size field and is not directly exercised; doing so would require hand-crafting a zip with a false header.

### Phase 4 — Metadata and path rooting ✅ Complete 2026-09-22

- [x] Load the active system from under the asset root via `sim.LoadSystemFromDir`; build the `name → ObjectMetadata` map
- [x] Populate `Meta` in `protoToSnapshot` from the map; category defaults remain as the unmatched-body fallback. The wire stays authoritative for name, category, parent and radius, so stale local content can never contradict a server-side override
- [x] Root the skysphere path via `Renderer.resolveAsset`
- [x] `system_selector` scans the **bundle** rather than calling `ListSystems` — that RPC returns 404 headless (D7's root cause). `Config.AssetRoot` drives `App.systemsDir()`
- [x] Rate-limited metadata rebuild on unknown body name, event-driven with no steady-state polling: only a miss on a category that *should* appear in a system definition counts, so procedurally generated belt members never trigger it
- [x] Keybinding profiles rooted via `Config.profilesDir()` — see the D5 correction
- [x] `render.New` gained an `assetRoot` parameter; its one test caller updated (`go build` does not compile tests — project lesson A-2)

**Verified behaviourally, not by inspection.** A binary copied alone into `/tmp/isolated` (no repo, no `data/`, cold cache), run against a live server:

| Check | Result |
|---|---|
| Bundle fetched and unpacked, cold | 96 entries, 4.03 MB, **40 ms** |
| Presentation metadata loaded | **717 bodies** from `data/systems/solar_system` |
| Skysphere from cache | `starfield_8k.jpg` 8192×4096 loaded from the cache path |
| Body textures from cache | `venusmap.jpg` and others — 8 textures, 0 errors |
| Warm start | **2 ms**, no transfer |

The first isolated run failed on the mandatory keybinding profile, which is what exposed the D5 error. That acceptance criterion earned its place.

---

## 7. Acceptance Criteria

Verified behaviourally, by running the binaries — not by reading the diff.

- [ ] `space-sim --server` renders Earth, Sol, the Moon, and Saturn's rings with their textures, matching standalone output
- [ ] Earth shows its night-side texture on the unlit hemisphere; Sol shows its corona
- [ ] The Milky Way skysphere renders
- [ ] Copy **only the built binary** to a directory with no `data/` and no repo checkout; it connects, fetches, and renders correctly
- [ ] Second launch in that directory transfers no bundle bytes
- [ ] A deliberately malicious archive with a `../` entry is rejected without writing outside the cache
- [ ] Operator `LoadSystem` to a different system updates client visuals without a client restart
- [ ] `go vet ./...` and `go test -race ./...` pass

---

## 8. Open Questions

| # | Question | Status |
|---|---|---|
| Q1 | Chunk size for `FetchBundle` | **Resolved: 64 KiB.** Far below gRPC's 4 MB default message limit, behaves well under stream flow control. A 4.03 MB bundle is 65 messages, verified over the wire, so per-message overhead is negligible and larger chunks buy nothing |
| Q2 | Should the server cap concurrent bundle streams? 100 clients × 5 MB on a simultaneous cold start is 500 MB of egress | Open — likely reuse the existing connection-limit interceptor rather than adding a second mechanism |
| Q3 | When do per-asset fetches become worth it? | Deferred. Trigger conditions: assets materially exceeding ~50 MB, or clients needing a subset. Bundle-by-hash is already coarse content addressing, so migration does not change the metadata contract |
| Q4 | Does the bundle need to carry `data/README.md`? | Trivial; include for fidelity unless it complicates exclusion rules |

---

## 9. Related Work

| Item | Relationship |
|---|---|
| F-010 Multi-Machine Architecture | This feature is a hard prerequisite for a genuinely remote renderer. Bandwidth mitigations remain separate |
| F-011 IAAM | Bundle fetch inherits transport auth with no design change once it lands |
| F-016 Wire Rendering Data Pipeline | Largely already implemented despite its "not started" status — `ObjectMetadata` carries texture, atmosphere, and luminosity fields today. D4 depends on that being true |
| F-020 Multi-Client Session Layer | Sequenced after this per D1 |
| F-021 Client Physical Marker | Ph2/Ph3 model assets arrive via the bundle at no extra cost |
| F-033 Ship Definition | Ship catalog is bundle content |
