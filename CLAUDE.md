# Claude Code — Project Instructions

Adapted from `.github/copilot-instructions.md` for Claude Code.

---

## Operational Modes

Signal the mode in your first message. When no mode is given, auto-detect.

| Signal | Mode | When |
|--------|------|------|
| `[fast]` | **Fast-track** | Resuming work with an accurate session summary; task fully specified in `docs/wip/`; no cross-package surprises. |
| `[standard]` | **Standard** | Default for new chats or tasks touching multiple packages or boundaries. |
| `[full]` | **Full review** | Conflicts expected, refactor scope unclear, or explicit plan approval gate wanted before code. |

**Auto-detect rule (no explicit signal):**
- Conversation summary present and continuing an active phase → Fast-track
- New chat, no summary → Standard
- Request contains "refactor", "redesign", "audit", or spans > 3 packages → Full review

---

## Required Reading

### Fast-track
- Skip re-reading docs already loaded this session.
- Read only what the request explicitly touches beyond the session summary.
- **Always** read `docs/wip/todo.md` if you haven't this session.

### Standard (default)

Read all of these before producing any plan or code:

| # | File | What to extract |
|---|------|-----------------|
| 1 | `docs/standards/agent-readme.md` | Repository map, package ownership, architectural boundaries, agent working defaults |
| 2 | `docs/standards/guidance.md` | Delivery workflow, planning requirements, phase rules, documentation rules, approval gates |
| 3 | `docs/standards/coding-standards.md` | Priority order, core engineering rules, Go best practices, Definition of Done |
| 4 | `docs/wip/todo.md` | Active work items and blocking dependencies — align with in-progress work |
| 5 | `docs/wip/` | Scan for additional phase docs, audits, or planning artifacts |
| 6 | `docs/history/lessons-learned.md` | Verified root causes, confirmed anti-patterns, hard-won constraints |
| 7 | `docs/history/lessons-learned-double-buffering.md` | Concurrency, cloning, synchronization, and double-buffer anti-patterns |

### Full review
Same as Standard **plus**: re-read any wip spec files relevant to the task. Produce an explicit written plan and wait for approval before writing any code.

---

## Before Acting

### Fast-track gate (silent check)
- Branch is not `main`/`master`.
- Change is within the scope of the session summary or wip spec.
- No lessons-learned entry obviously prohibits the approach.

### Standard gate
Before writing code or a plan, be able to answer:

1. Which packages does this change touch, and do those packages own the behavior being changed?
2. Does this change respect the architectural boundaries in `agent-readme.md`?
3. Does a lessons-learned entry warn against this approach?
4. Is there active work in `docs/wip/` this change should align with or defer to?
5. Does the plan satisfy the Definition of Done in `coding-standards.md`?

**Branch check (§11.1 of `guidance.md`):** Run `git branch` at the start of every session. Prompt before proceeding if on `main`/`master` or if a related feature branch exists.

### Full review gate
Same as Standard **plus**: write out the answers to all five questions explicitly and wait for user confirmation. Also answer **Q6 — Design principles compliance** (DIP, DRY, SRP/GRASP, Low Coupling, IoC). For each violation: state **accepted** (rationale + future TD note) or **corrected**.

---

## Workflow

Follow the delivery sequence from `docs/standards/guidance.md §3`:

1. Design → Queue → Schedule → Implement → Test between steps → Test after all phases → Commit

Commit at the end of each completed phase, not mid-phase. Keep commit messages terse and outcome-focused (e.g. `feat: add stellar classification system`).

---

## Documentation Rules

When code or behavior changes, update the relevant docs in the same workstream:
- Active work → `docs/wip/todo.md`
- Completed work → `docs/history/changelog.md`
- Architecture/enum changes → `docs/standards/agent-readme.md`

All files under `docs/` must maintain `Purpose`, `Last Updated`, and `Table of Contents` sections.

---

## Project Tools

MCP tools are configured in `.vscode/mcp.json` and served by `tools/write_file_server.py`. Claude Code has native file-editing tools — use the built-in Read/Edit/Write tools for file operations. The MCP server is available for other integrations when needed.
