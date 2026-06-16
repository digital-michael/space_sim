# Lessons Learned Index — space_sim

## Purpose
Framework-format index of all lessons-learned sources for this repository. This file is an index — do not duplicate lesson content here. Follow the pointers to the authoritative source files.

## Last Updated
2026-06-15

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
| [`docs/history/lessons-learned.md`](../history/lessons-learned.md) | Performance testing, profiling methodology, visibility system bugs, UI rendering, Raylib constraints, recording/video, multi-client session layer (F-020), concurrency patterns | Feb–May 2026 |
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

---

## 3. Technologist Lessons Summary

| # | Lesson | Source |
|---|---|---|
| T-1 | Always benchmark on AC power — battery mode CPU throttling (15–60% slowdown) invalidates results | lessons-learned.md #9, #12 |
| T-2 | Provide a warmup period before measuring — GPU compilation, GC cycles, and scheduler balance take 1–8 seconds | lessons-learned.md #10 |
| T-3 | Test with realistic camera positions — god-view hides optimization effectiveness | lessons-learned.md #7 |

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

---

## 5. Promotion Log

*Lessons promoted from source files to `llm-agent-domains/photon-datum/library/go/governance-overlay.md` or to `llm-agent-framework`.*

| Date | Lesson | Promoted To |
|---|---|---|
| — | — | — |
