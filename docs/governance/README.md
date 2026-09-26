# Session-Start Load Order — space_sim

## Purpose
Define the authoritative context loading sequence for any agent working in this repository. Follow this order at every session start. Profile selection (minimal/standard/full) determines how much of this list is loaded; the sequence order is always the same.

## Last Updated
2026-09-22

## Table of Contents
1. Load Order
2. Profile Selection
3. Index of Governance Files

---

## 1. Load Order

Load in this sequence. Stop at the profile boundary indicated in §2.

| # | File | Profile | What to extract |
|---|---|---|---|
| 1 | `~/Documents/Entities/frameworks/llm-agent-framework/governance/system-prompts/base-system-prompt.md` | all | Guardrails, communication rules, code quality gates, process rules |
| 2 | `~/Documents/Entities/frameworks/llm-agent-framework/library/go/instructions.md` | all | Go-specific governance: error wrapping, context, interfaces, testing |
| 3 | `~/Documents/Entities/frameworks/llm-agent-domains/HobbyPro/governance-overlay.md` | all | Domain operating modes, HITL rules, commit conventions |
| 4 | `~/Documents/Entities/frameworks/llm-agent-domains/HobbyPro/library/go/governance-overlay.md` | all | Go lessons validated across HobbyPro projects |
| 5 | `~/Documents/Entities/frameworks/llm-agent-domains/HobbyPro/space_sim/README.md` | all | Repo identity, package map quick-reference, doc rules, git constraints |
| 6 | `docs/governance/lessons-learned.md` | all | Index of verified root causes and hard-won constraints |
| 7 | `docs/wip/todo.md` | all | Active work items — align before planning or coding |
| 8 | `docs/standards/agent-readme.md` (§2 package map + §3 architecture) | standard, full | Package ownership, architectural boundaries, process flows |
| 9 | `docs/standards/guidance.md` | standard, full | Doc placement rules, commit format, branch conventions |
| 10 | `docs/standards/coding-standards.md` | standard, full | Space_sim-specific engineering rules (SOLID package guidance, state management, JSON) |
| 11 | `docs/wip/` (scan for phase docs and planning artifacts) | standard, full | In-progress specs and audits that constrain the current task |
| 12 | `docs/governance/session-context.md` | resuming only | Cross-session handoff state; load when resuming a prior session |

**Also required, every session, regardless of profile:** the personal profile at
`~/Documents/Entities/frameworks/llm-agent-personal/` — `context-protocol.md`,
`collaboration-preferences.md`, `collaboration-patterns.md`, `roles-guide.md`. It is a
mode-level setting (who you are working with), not task-scoped, so a `minimal` profile
includes it too. Load it **before** the domain and project files.

`.llm-framework.yml` sets `personal_identity: digital-michael`; that is a resolution key,
**not a filename** — there is no `digital-michael.md`. See
`llm-agent-framework/governance/agent-context-protocol.md` §Personal Profile Resolution.

---

## 2. Profile Selection

| Profile | Load steps | When |
|---|---|---|
| `minimal` | 1–7 | `[fast]` signal; narrow single-file task; no cross-package surprises |
| `standard` | 1–11 | `[standard]` signal or auto-detected new session |
| `full` | 1–12 + relevant wip specs | `[full]` signal; refactor; redesign; spans > 3 packages |

---

## 3. Index of Governance Files

| File | Role |
|---|---|
| `docs/governance/README.md` | This file — load order and profile map |
| `docs/governance/lessons-learned.md` | Framework-format lessons index (pointers to detail files) |
| `docs/governance/agent-assignment-template.md` | Assignment template — copy for each new assignment |
| `docs/governance/agent-assignment.md` | Active assignment for the framework retrofit |
| `docs/governance/session-context.md` | Cross-session handoff template |
