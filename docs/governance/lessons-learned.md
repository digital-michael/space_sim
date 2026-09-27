# Lessons Learned Index — space_sim

## Purpose
Framework-format index of all lessons-learned sources for this repository. This file is an index — do not duplicate lesson content here. Follow the pointers to the authoritative source files.

## Last Updated
2026-09-26

## Table of Contents
1. Source Files
2. LLM Agent Lessons Summary
3. Technologist Lessons Summary
4. Tech Stack Lessons Summary
5. Promotion Log

---

## 1. Source Files

| File | Topic | Sessions Covered |
|---|---|---|
| [`docs/history/lessons-learned.md`](../history/lessons-learned.md) | Performance testing, profiling methodology, visibility system bugs, UI rendering, Raylib constraints, recording/video, multi-client session layer (F-020), concurrency patterns, gitignore scope defects, F-040 asset bundle, transport boundary violations, Raylib conditional-draw diagnosis, floating-origin thresholds, h2c and duplex requirements, adaptive presence publishing | Feb–May 2026, Sep 2026 |
| [`docs/history/lessons-learned-double-buffering.md`](../history/lessons-learned-double-buffering.md) | Double-buffer clone discipline, Go pointer semantics, cross-thread deadlocks, visibility synchronization, type safety in graphics | Feb–Mar 2026 |

---

## 2. LLM Agent Lessons Summary

Key recurring patterns where agent behavior has been corrected or validated. See source files for full detail.

| # | Lesson | Source |
|---|---|---|
| A-1 | Diagnose all root causes before fixing any — sequential debugging without full system trace causes fix chains | lessons-learned.md §Development Process Issues |
| A-2 | `go build ./...` does not compile test files — always run `go test ./...` after signature changes | lessons-learned.md #14 |
| A-3 | Subscribe before bootstrapping in pub-sub patterns — never read state before attaching the event channel | lessons-learned.md #15 |
| A-4 | For field renames across multiple files: update all callsites atomically or use additive migration; never end a session with a partial rename | lessons-learned.md §LabelMode rename |
| A-5 | Verify existing behavior before implementing — read the relevant function before designing a new one | lessons-learned.md §Verify Existing Behavior |
| A-6 | A claim about *why* code exists must be verified (`git show --stat`, grep for callers) before it is recorded or acted on — a commit-subject keyword match is not evidence | lessons-learned.md #43 |
| A-7 | Probe the actual code path; do not infer capability from a different client's failure, nor treat a comment describing a limitation as proof it currently binds | lessons-learned.md #44 |
| A-8 | Anchor `.gitignore` patterns with a leading `/`; diagnose apparently-missing files with `git check-ignore -v`, never `git status`, which reports an ignored dir as clean | lessons-learned.md #41 |
| A-9 | Determine feature status by reading code, not the roadmap or the spec — when the two disagree, both are stale | lessons-learned.md #42 |
| A-10 | Dogfood against real repository content; synthetic fixtures verify code against the author's beliefs, not the project | lessons-learned.md #45 |
| A-11 | A doc comment describing behaviour is a claim to verify — code whose comment promises logging it does not perform is a defect hiding behind its own documentation | lessons-learned.md #43, #45 |
| A-12 | When a governance/bootstrap doc disagrees with live config, say so and fix it — do not silently prefer the correct source, which leaves the stale instruction to misdirect the next session | lessons-learned.md #49 |
| A-13 | Classify a directory as user config vs shipped content by reading the loader (optional or mandatory?), not by its name — a mandatory input is content whatever it is called | lessons-learned.md #50 |
| A-14 | An end-to-end run of the real binary in the real deployment shape catches a class of defect no unit test will, because unit tests never execute startup | lessons-learned.md #50 |
| A-15 | Restate a terse answer to a structural question before it shapes design; re-read accumulated answers as a set, not incrementally | lessons-learned.md #52 |
| A-16 | Never write a spec on a premise already in doubt — a consistent document aimed at the wrong architecture reviews as finished and buries the load-bearing assumption | lessons-learned.md #52 |
| A-17 | When an option matrix feels bloated, find the undefined term generating the options; semantic clarification collapses design surface faster than feature-cutting | lessons-learned.md #52 |
| A-18 | A test asserting a magic number will outlive the reason and defend the wrong behaviour — assert the invariant the number was chosen to satisfy | lessons-learned.md #54 |
| A-19 | When recording a known limitation, name the capability it blocks — an unlinked note will not be found by whoever hits the symptom | lessons-learned.md #55 |
| A-20 | When a formula sums two mechanisms, hold one at zero and verify the other against its closed form before trusting the combination | lessons-learned.md #57 |

---

## 3. Technologist Lessons Summary

| # | Lesson | Source |
|---|---|---|
| T-1 | Always benchmark on AC power — battery mode CPU throttling (15–60% slowdown) invalidates results | lessons-learned.md #9, #12 |
| T-2 | Provide a warmup period before measuring — GPU compilation, GC cycles, and scheduler balance take 1–8 seconds | lessons-learned.md #10 |
| T-3 | Test with realistic camera positions — god-view hides optimization effectiveness | lessons-learned.md #7 |
| T-4 | A `make run-server` left running pegs ~140% CPU indefinitely with zero clients attached; check for an existing listener before starting another instance | lessons-learned.md #47 |
| T-5 | Never trust `pkill -f` — a relative-path launch (`./space-sim`) will not match an absolute-path pattern, and killing zero processes is not an error. Kill by PID and verify with `pgrep` | lessons-learned.md #51 |

---

## 4. Tech Stack Lessons Summary

| # | Lesson | Source |
|---|---|---|
| S-1 | Double-buffer: clone on swap, not pointer exchange, when buffers hold complex synchronized state | lessons-learned.md #18; lessons-learned-double-buffering.md §1 |
| S-2 | Clone() must be deep — `&objCopy` in a loop creates pointers to the same stack location | lessons-learned-double-buffering.md §1 |
| S-3 | Unlock before notify — `defer mu.Unlock()` holds the lock during subscriber callbacks and risks deadlock | lessons-learned.md #13 |
| S-4 | Same-package proto files still require explicit imports | lessons-learned.md #16 |
| S-5 | Raylib: 2D drawing primitives must appear after `EndMode3D()`, not inside the 3D context | lessons-learned.md #19 |
| S-6 | Raylib: window resolution is locked at `InitWindow()` — fullscreen transitions require window recreation | lessons-learned.md §Window Resolution |
| S-7 | More goroutine workers ≠ better performance at small object counts — profile thread scaling before assuming benefit | lessons-learned.md #6, #8 |
| S-8 | Native render mode has no render texture — always verify `HasRenderTarget()` before pixel readback | lessons-learned.md §Video Recording |
| S-9 | Apple Silicon / OpenGL-via-Metal: use a fresh FBO with explicit color attachment for `glReadPixels`; `rl.LoadImageFromTexture` does not work | lessons-learned.md §Apple Silicon |
| S-10 | Deterministic zip requires sorted entries + fixed modtime (1980-01-01; MS-DOS cannot encode earlier) + fixed mode; hash the archive **bytes** so verification precedes extraction | lessons-learned.md #46 |
| S-11 | `space-sim-server` registers only 2 of 11 services; `SystemService` cannot be registered headless because `SystemHandler` is coupled to the Raylib app's command channel | lessons-learned.md #48 |
| S-12 | The server speaks Connect over HTTP/1.1; h2c is not wired (`server.go:69`). `buf curl` needs `--schema api/proto`, and `--http2-prior-knowledge` fails — but `connect.WithGRPC()` clients stream fine | lessons-learned.md #44 |
| S-13 | Deflate crushes periodic data — a fixture needing incompressible bytes must use a seeded PRNG, not an arithmetic pattern | lessons-learned.md #45 |
| S-14 | A conditional Raylib draw fails silently — instrument the per-element decision before theorising about why nothing appears | lessons-learned.md #53 |
| S-15 | In a floating-origin renderer, visibility thresholds belong in screen/angular terms, not world distance; two coordinate spaces coexist and both look plausible | lessons-learned.md #54 |
| S-16 | Cleartext gRPC in Go requires h2c; its absence is masked because unary and server-streaming are half-duplex and survive HTTP/1.1, while bidirectional streaming cannot | lessons-learned.md #55 |
| S-17 | Locate a cost by following the call chain, not by assuming the plausible-looking loop is the hot one | lessons-learned.md #56 |
| S-18 | Gate a lower layer on a higher-layer condition by injecting a predicate — the import you are about to add is often the mirror of a violation already filed | lessons-learned.md #56 |
| S-19 | Express "does this matter to the viewer" in angular terms once and reuse it; pixels depend on resolution, sim units on view scale, angular size on neither | lessons-learned.md #57 |

---

## 5. Promotion Log

*Lessons promoted from source files to `llm-agent-domains/HobbyPro/library/go/governance-overlay.md` or to `llm-agent-framework`.*

| Date | Lesson | Promoted To |
|---|---|---|
| 2026-09-22 | #46 Content-addressed artifacts: normalize before hashing; hash the bytes the consumer verifies; size guards as vars so they are testable | `llm-agent-domains/HobbyPro/library/go/governance-overlay.md` §Content-Addressed Artifacts (Go) |
| 2026-09-22 | #48 One handler serving two topologies belongs behind an injected port; a boundary violation that breaks a supported deployment is a functional defect | `llm-agent-domains/HobbyPro/library/go/governance-overlay.md` §Applying IoC When One Handler Serves Two Topologies |
| 2026-09-22 | #43, #44, #49 Agent failure modes — fabricated causation from a keyword match, capability inferred from another client's failure, stale governance worked around silently | `llm-agent-personal/collaboration-preferences.md` §Where I Tend to Fail |
| 2026-09-22 | #45 A comment asserting a property the test depends on must be verified, not assumed | `llm-agent-personal/collaboration-preferences.md` §Where I Tend to Fail |
| 2026-09-25 | Simulation cost and state closure — tick-rate reduction is an accuracy knob not a cost knob; classify state as time-closed vs path-dependent before designing any skip/pause/catch-up; separate command surface from invoking policy | `llm-agent-domains/HobbyPro/library/go/governance-overlay.md` §Simulation Cost and State Closure (Go) |
| 2026-09-25 | Semantic clarification collapses option space — the highest-leverage move in a design discussion | `llm-agent-personal/collaboration-patterns.md` P-5 |
| 2026-09-25 | Practitioner reframes reduce scope; re-derive rather than defend the prior framing | `llm-agent-personal/collaboration-patterns.md` P-6 |
| 2026-09-25 | Capability is not obligation — verify a proposed demonstration runs on the machine at hand | `llm-agent-personal/collaboration-preferences.md` §What I Want |
| 2026-09-25 | #52 Mark restatement vs addition; a definition is not a decision about use; over-reading a terse structural answer | `llm-agent-personal/collaboration-preferences.md` §Where I Tend to Fail |
| 2026-09-26 | #55 Cleartext gRPC requires h2c; half-duplex call shapes mask its absence | `llm-agent-domains/HobbyPro/library/go/governance-overlay.md` §Cleartext gRPC Requires h2c (Go) |
| 2026-09-26 | #56 Inject a predicate rather than importing the consumer | `llm-agent-domains/HobbyPro/library/go/governance-overlay.md` §Inversion of Control (Go) |
