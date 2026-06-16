---
template-version: 1.0.0
---

# Agent Assignment — space_sim Framework Retrofit

**Title:** Retrofit space_sim into the LLM Agent Collaboration Framework
**Date:** 2026-06-12
**Assigned to model tier:** Implementation (docs + config; no production code changes)
**Related ADR(s):** None
**Change risk level:** Low

---

## Goals

- space_sim is enrolled in the framework: `.llm-framework.yml` exists at the project root with correct paths
- `docs/governance/` contains the four required files: load-order README, lessons-learned index, assignment template, session-context template
- Existing lessons (`docs/history/lessons-learned.md`, `docs/history/lessons-learned-double-buffering.md`) are indexed — not duplicated — into the framework format
- `llm-agent-domains/photon-datum/space_sim/README.md` exists with session-start load order, package map quick-reference, and external components table
- `docs/standards/agent-readme.md` receives a `## Current-State Package Map` addendum with layer classification, composition rules, and external reusable components
- `CLAUDE.md` is thinned to a bridge file + Claude Code-specific material; substantive content migrated to appropriate framework locations
- `docs/standards/guidance.md` and `docs/standards/coding-standards.md` are reduced to space_sim-unique content; framework-covered content removed

---

## Constraints

- `llm-agent-framework` is **read-only** — no writes, no modifications
- Do not restructure or rewrite existing `docs/` content beyond targeted reductions — additive changes or explicit removals only
- Do not write agent behavior rules into `docs/governance/` — those belong in `llm-agent-domains/photon-datum/space_sim/README.md`
- `CLAUDE.md` must not be deleted — thin to a bridge file (Claude Code auto-injects it; deleting it breaks Claude Code sessions)
- All declared paths in `.llm-framework.yml` must exist before writing the file

---

## Out of Scope / Deferred

| Item | Reason for Deferral |
|---|---|
| Personal profile layer | No personal profile repo exists; Team mode is sufficient |
| Promoting space_sim patterns to `llm-agent-framework` | No patterns validated as universal yet |
| Adding space_sim Go lessons to domain `library/go/governance-overlay.md` | Identified during audit step; promoted only after validation |
| Rewriting or restructuring `docs/history/` lessons files | Those are the authoritative source; `docs/governance/lessons-learned.md` indexes them |

---

## Q&A Log

| # | Question | Answer | Status |
|---|---|---|---|
| 1 | Which load order is authoritative for Claude Code — framework or CLAUDE.md? | Framework takes precedence; integrate CLAUDE.md into appropriate framework locations | Resolved |
| 2 | Where does the `## Current-State Package Map` addendum go? | Into `docs/standards/agent-readme.md` | Resolved |
| 3 | Which domain does space_sim belong to? | New domain: `photon-datum`, sibling to `cts` in `llm-agent-domains/` | Resolved |

---

## Implementation Plan

| # | Unit of Work | Status | Notes |
|---|---|---|---|
| 1 | **Audit verification** — confirm all source paths exist | Complete | All paths verified |
| 2 | **Audit existing space_sim docs for framework overlap** — `CLAUDE.md`, `guidance.md`, `coding-standards.md`, `.github/copilot-instructions.md` | Complete | Findings documented in unit 11 below |
| 3 | **Write `.llm-framework.yml`** | Complete | Points to llm-agent-framework + photon-datum |
| 4 | **Create `photon-datum` domain** — README, governance-overlay, library/go overlay, update llm-agent-domains README | Complete | All four files + Domains table updated |
| 5 | **Create `docs/governance/README.md`** — session-start load order | Complete | 12-step load order with profile boundaries |
| 6 | **Create `docs/governance/lessons-learned.md`** — framework-format index | Complete | Indexes both lessons files; no content duplication |
| 7 | **Copy `agent-assignment.md` template** → `docs/governance/agent-assignment-template.md` | Complete | Template copy; this file is the active assignment |
| 8 | **Copy `session-context.md` template** → `docs/governance/session-context.md` | Complete | |
| 9 | **Add `## Current-State Package Map` addendum** to `docs/standards/agent-readme.md` | Complete | Layer table, External Reusable Components, Composition Rules |
| 10 | **Create `llm-agent-domains/photon-datum/space_sim/README.md`** | Complete | Load order, package map, doc rules, git constraints |
| 11 | **Apply audit findings** — thin `CLAUDE.md`, `.github/copilot-instructions.md`, `guidance.md`, `coding-standards.md`; IoC section moved to `photon-datum/library/go/governance-overlay.md` | Complete | |
| 12 | **Self-verify** — walk session-start protocol; confirm all declared paths resolve | Complete | All 11 paths verified ✓ |

### Unit 11 — Audit Findings (awaiting approval)

**`CLAUDE.md` and `.github/copilot-instructions.md`**

Both files contain the same content: Operational Modes, Required Reading, Before Acting gates, Workflow, Documentation Rules. All of this is now covered by the framework (governance-overlay, repo context, framework workflow). User direction: "CLAUDE.md should defer into the framework."

Proposed outcome for **both** files — thin to:
- One-line project identity
- Pointer to `docs/governance/README.md` for the authoritative load order
- Claude Code / Copilot-specific tool notes only (MCP server reference; `write_file`, `get_current_datetime` tools)

All other content removed — it now lives in the framework layer.

---

**`docs/standards/guidance.md`**

| Section | Disposition |
|---|---|
| §1 Role of This Document | Remove — covered by framework |
| §2 Working Priorities | Remove — covered by framework assignment-workflow + collaboration-directives |
| §3 Required Delivery Workflow | Remove — covered by framework assignment-workflow |
| §4 Planning Requirements | Remove — covered by framework assignment-workflow §3 |
| §5 Phase Execution Rules | Remove — covered by framework |
| §6 Testing Requirements | Remove — covered by framework testing-standards |
| §7 User Approval and Commit Rules | **Keep — commit message format and work-id examples are space_sim-specific** |
| §8 Documentation Rules | **Keep in full — doc metadata format, placement, and work tracking are space_sim-specific** |
| §9 Code and Design Guidance | Remove most; keep only the performance emphasis paragraph |
| §10 Temporary Artifact Rules | Remove — covered by framework |
| §11 Agent Default Behavior | Remove — covered by framework collaboration-directives + assignment-workflow |
| §11.1 Branch Check Rule | Already migrated to `photon-datum/space_sim/README.md` — remove from guidance.md |

Proposed outcome: guidance.md retains §7 (commit format), §8 (doc rules), and one paragraph from §9. Updated Purpose and ToC.

---

**`docs/standards/coding-standards.md`**

| Section | Disposition |
|---|---|
| §1 Priority Order | Remove — covered by framework engineering-principles |
| §2 Core Engineering Rules | Thin — keep only "Avoid hidden coupling between simulation, UI state, rendering, and JSON loading" (space_sim architectural boundary) |
| §3 SOLID Guidance (§3.1–§3.5) | **Keep — package-specific package names make this space_sim-specific** |
| §4 DRY Guidance | Remove — covered by framework engineering-principles |
| §5 GRASP Guidance | Remove — covered by framework engineering-principles |
| §6 IoC | **Keep — detailed Go IoC guidance more specific than framework's DIP summary** |
| §7 Go Best Practices (§7.1–§7.7) | Remove — covered by framework library/go/instructions.md; keep §7.8 |
| §7.8 State Management (S1–S7) | **Keep in full — validated space_sim-specific patterns** |
| §8 JSON Best Practices | **Keep in full — space_sim JSON schema-specific** |
| §9 Definition of Done | Remove — covered by framework base system prompt |
| §10 Reference Material | Remove — covered by framework |

Proposed outcome: coding-standards.md retains §3 SOLID (with package refs), §6 IoC, §7.8 State Management, §8 JSON. Updated Purpose and ToC.

---

## Testing Requirements

No automated tests apply (docs + config only). Validation is structural:

- Every path in `.llm-framework.yml` and `docs/governance/README.md` resolves to an existing file
- `docs/governance/lessons-learned.md` has a pointer row for each existing lessons file; no content is duplicated
- `CLAUDE.md` retains its bridge role and remains non-empty
- `llm-agent-domains/photon-datum/space_sim/README.md` is loadable as a session-start document (all referenced paths resolve)
- `docs/standards/guidance.md` and `docs/standards/coding-standards.md` contain no content already covered verbatim by the framework

---

## Quality Gate — Final Checklist

Complete at assignment close.

- [ ] `.llm-framework.yml` exists at space_sim root with correct, verified paths
- [ ] `docs/governance/` contains all four required files
- [ ] `docs/governance/lessons-learned.md` indexes all existing lessons files (no duplication)
- [ ] `photon-datum` domain fully created (4 files + Domains table updated)
- [ ] `llm-agent-domains/photon-datum/space_sim/README.md` complete with load order and package map
- [ ] `agent-readme.md` addendum written and coherent with existing content
- [ ] `CLAUDE.md` thinned to bridge + Claude Code-specific material
- [ ] `docs/standards/guidance.md` reduced to space_sim-unique content
- [ ] `docs/standards/coding-standards.md` reduced to space_sim-unique content
- [ ] All declared paths verified to exist (unit 12 passed)
- [ ] No writes made to `llm-agent-framework/` (read-only constraint)
- [ ] Lessons learned captured (all three sources)

---

## Quality Gate — Final Checklist

- [x] `.llm-framework.yml` exists at space_sim root with correct, verified paths
- [x] `docs/governance/` contains all four required files
- [x] `docs/governance/lessons-learned.md` indexes all existing lessons files (no duplication)
- [x] `photon-datum` domain fully created (4 files + Domains table updated)
- [x] `llm-agent-domains/photon-datum/space_sim/README.md` complete with load order and package map
- [x] `agent-readme.md` addendum written and coherent with existing content
- [x] `CLAUDE.md` thinned to bridge + Claude Code-specific material
- [x] `.github/copilot-instructions.md` thinned to bridge + Copilot-specific material
- [x] `docs/standards/guidance.md` reduced to space_sim-unique content
- [x] `docs/standards/coding-standards.md` reduced to space_sim-unique content; IoC moved to domain layer
- [x] All declared paths verified to exist (unit 12 passed)
- [x] No writes made to `llm-agent-framework/` (read-only constraint honoured)

---

## Lessons Learned

### LLM Agent

- The `.github/copilot-instructions.md` file was discovered mid-retrofit via user IDE observation — it was not in the initial audit scope. A complete file audit (find all instruction/governance files across hidden directories) should be step 1 of any future retrofit.
- The IoC section classification required a mid-execution correction: it was initially planned to stay in the project repo but correctly identified as domain-level HOW during the unit 11 discussion. The user's mental model question ("which layer owns HOW vs WHAT vs WHY") was the right gate to catch this before execution.

### Technologist

- The three-layer mental model (framework = universal HOW; domain = domain HOW + project WHAT; project = WHAT + WHY) is accurate and should be the framing question at the start of any future retrofit or governance review.

### Tech Stack

- No surprises. Framework templates transferred cleanly. All paths use `~/Projects/active/` prefix consistently.

---

## Retrospective Notes

Retrofit complete 2026-06-15. The photon-datum domain is established and space_sim is fully enrolled. Future projects joining photon-datum only need units 3, 5–8, 9–10, and 11 — the domain infrastructure (unit 4) is already in place.
