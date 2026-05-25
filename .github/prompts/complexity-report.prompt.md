---
description: Complexity and technical debt audit. Produces a unified, timestamped top-10 report saved to ./output/. Run in Agent mode (Claude Sonnet 4.6). Reads docs/wip/todo.md to populate cross-reference columns.
---

# Tech Debt Complexity Report

Generate a **top 10 technical debt report** for this repository and write it as a timestamped Markdown file to `./output/tech-debt-report-<YYYY-MM-DD>.md`. Scope: all local Go files under `internal/` and `cmd/`. Exclude `api/gen/`, `vendor/`, and `bin/`.

## Step 1 — Data collection (run these before writing anything)

1. `wc -l` on all in-scope `.go` files, sorted descending — identifies the largest files.
2. `awk '/^func /{...}'` function-boundary line counts across the top 15 files — identifies the largest individual functions.
3. `go test -cover ./...` — records package-level coverage for every candidate.
4. Read `docs/wip/todo.md` — extracts pending feature IDs and names needed for cross-reference columns.
5. Read the top 6–8 candidate files — confirms function call depth and direct field-mutation patterns.

## Step 2 — Scoring and ranking

Score each candidate on three axes, then sum:

| Axis | Signal | Weight |
|---|---|---|
| Size | Lines (file + largest function) | High |
| Coverage | Package `go test -cover` % | High (0% = Critical multiplier) |
| Standards | Count of S1–S7, SRP, OCP, DRY violations found by reading | Medium |

Rank the top 10 by combined score, highest risk first. Assign a Risk Tier: **Critical** (0–10% coverage or 600+ line function), **High** (10–40% or 300+ line function), **Medium** (40–60% or 200+ line function).

## Step 3 — What to look for in each candidate

- Functions over 200 lines (read for nesting depth and direct field mutation)
- Files over 500 lines mixing more than one concern
- Dead code: verify by running `grep -rn "<funcName>" ./internal ./cmd --include="*.go"` for every top-level function in a suspected dead file. A file is dead only if **no function in it has a caller outside that file**. Do not infer dead code from filename conventions such as `legacy_*` — verify the call graph.
- **S1**: struct mixes unrelated fields from different contexts or modes
- **S2**: external callers write struct fields directly instead of using transition methods
- **S3**: state resets scattered at call sites instead of centralized in the owning constructor/transition function
- **S7**: struct fields valid only in a specific mode, read/written without a mode guard
- OCP violations: adding a feature requires editing a large monolithic switch/function
- Duplicated logic across files or packages
- Undocumented invariants on safety-critical code (physics, concurrency)

## Step 4 — Output: unified report file

Write `./output/tech-debt-report-<YYYY-MM-DD>.md` with the following structure:

### File structure

```
# Tech Debt Complexity Report
Title / Last Updated / Branch / Scope / Methodology (header block)

## Summary
3–5 sentence narrative of the dominant patterns found.

## Table of Contents
Linked list of all sections.

## Summary Table
One row per item. Columns (in order):
  # | Area | Coverage | Risk | Issues | Blocks/Impacts | Depends On

  - # : item number, hyperlinked to its full section below
  - Area : file path · function or struct name
  - Coverage : package %
  - Risk : Critical / High / Medium
  - Issues : comma list (SRP, S2, S3, S7, dead-code, low-coverage, duplication, call-depth, OCP)
  - Blocks/Impacts : pending feature IDs + short names (from todo.md) that this debt makes harder
  - Depends On : feature or debt items that must be done first (usually empty)

  Note: Size, Pairs With, and Rules Violated are omitted from the table — they appear in each item's full section below.

Sequencing note below the table: two explicit orderings:
  - **Foundational batch** (no feature dependency, do first): items with no feature unlock and that structurally unblock others
  - **Feature-driven sequence** (after foundational batch): table with columns Session | Debt Items | Feature Unlock, ordered by feature value

## #N — Short Name  (one section per item, 10 total)

Each section contains exactly these subsections:

### Description
2–4 paragraphs. What the code is, how it got this way, what the concrete problem is.

### Issues
Bullet list with one line per problem, cross-referencing the standard violated.

### Rules Violated
Markdown table: Rule | Reference | Violation (one row per violated rule).

### Acceptance Criteria
Bullet list derived directly from the violated standards. For each rule violated, state one concrete, verifiable pass/fail criterion. Include the verification method inline:
- **S2, S3, S7**: grep command that must return zero matches (state the exact pattern)
- **SRP / OCP**: line count of the refactored function — must be ≤ stated target (state the target)
- **Coverage**: `go test -cover` % — must be ≥ stated floor (state the floor and current baseline)
- **Dead-code**: grep for all top-level function names must return zero external callers
- **Build**: `go build ./...` must exit 0
All tests in the AC must run without Raylib unless the code explicitly requires a display.

### Fix
2–5 sentences describing the extraction or encapsulation approach. Include concrete
renamed files or types where the approach is clear.

### Pairs With
Which other item numbers and why.

### Blocks / Impacts
Bullet list of pending feature IDs + short names.

## Recommendations
Numbered list R1–RN of items not already covered in the individual entries:
- Coverage floor CI policy
- Lint gate for function length / cyclomatic complexity
- Quick-win sequencing rationale
- Any structural suggestions (interface extraction, test scaffold approaches)
- F-013 or other high-value feature dependency observations

## Footer
"Report generated by GitHub Copilot (Claude Sonnet 4.6) on <timestamp>. Re-run using .github/prompts/complexity-report.prompt.md in Agent mode."
```

## Step 5 — After writing the file

1. Print the file path and line count to confirm it was written.
2. Ask the user which items they want added to `docs/wip/todo.md` as active work. Do not update `todo.md` without explicit approval.
3. Do not create any other markdown files.

## Reference: coding standards locations

The rules referenced in "Rules Violated" columns are defined in:
- `docs/standards/coding-standards.md` §3 (SOLID), §4 (DRY), §5 (GRASP), §7.8 (State Management S1–S7), §9 (Definition of Done)
- `docs/history/lessons-learned.md` LL #39 (constructor completeness, TrackOffset root cause)
- `docs/history/lessons-learned-double-buffering.md` (concurrency and snapshot invariants)
