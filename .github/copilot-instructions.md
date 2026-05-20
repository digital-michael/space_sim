# Copilot Workspace Instructions

---

## Operational Modes

The mode governs how much up-front context loading and gate checking is required.
The user signals the mode in their first message of a chat. When no mode is
signalled, apply the **default** based on context (see table below).

| Signal | Mode | When to apply |
|--------|------|---------------|
| `[fast]` | **Fast-track** | Resuming work with an accurate session summary; task is already fully specified in `docs/wip/`; no cross-package surprises expected. |
| `[standard]` | **Standard** | Default for new chats with no summary or when the task touches boundaries or multiple packages. |
| `[full]` | **Full review** | Conflicts expected, refactor scope unclear, or user explicitly wants a plan approval gate before any code is written. |

**Auto-detect rule** (no explicit signal):
- Conversation summary present **and** request continues an active phase → **Fast-track**
- New chat, no summary → **Standard**
- Request contains words like "refactor", "redesign", "audit", or spans >3 packages → **Full review**

---

## Required Reading

Reading requirements vary by mode.

### Fast-track
- **Skip** re-reading docs already loaded in this chat session.
- Read only if the request explicitly touches something not covered by the
  session summary: specific wip phase doc, or a lessons-learned file by name.
- **Always** read `docs/wip/todo.md` if you have not read it this session.

### Standard (default)
Read all of these before producing any plan or code:

| # | File | Extract |
|---|------|---------|
| 1 | [`docs/standards/agent-readme.md`](../docs/standards/agent-readme.md) | Repository map, package ownership, architectural boundaries, preserved refactor intent, agent working defaults |
| 2 | [`docs/standards/guidance.md`](../docs/standards/guidance.md) | Delivery workflow (Design → Queue → Schedule → Execute), planning requirements, phase rules, documentation rules, approval gates |
| 3 | [`docs/standards/coding-standards.md`](../docs/standards/coding-standards.md) | Priority order for conflicting standards, core engineering rules, Go best practices, Definition of Done |
| 4 | [`docs/wip/todo.md`](../docs/wip/todo.md) | Active work items and blocking dependencies — align with in-progress work, do not introduce a parallel track |
| 5 | [`docs/wip/`](../docs/wip/) | Scan for additional phase docs, audits, or planning artifacts |
| 6 | [`docs/history/lessons-learned.md`](../docs/history/lessons-learned.md) | Verified root causes, confirmed anti-patterns, hard-won constraints |
| 7 | [`docs/history/lessons-learned-double-buffering.md`](../docs/history/lessons-learned-double-buffering.md) | Concurrency, cloning, synchronization, and double-buffer anti-patterns |

### Full review
Same as Standard **plus**: re-read any wip spec files relevant to the task even
if already read this session. Produce an explicit written plan and wait for user
approval before writing any code.

---

## Acknowledgment

After completing the required reading for the active mode, respond with **"I'm Locked-In now"** before anything else.

The acknowledgment must always communicate three things:

1. Why this context is loading.
2. Whether the context is new for this request or was already available.
3. Whether the context was loaded directly by user action or in support of another context.

**Message shape:** `At <date time>: I'm Locked-In now` + reason clause + lock-state clause

**Required variants** (all start with `At <date time>, `):

| Situation | Variant |
|-----------|---------|
| New user chat | `I'm Locked-In now by user action and this locked-in context was not already loaded.` |
| Delegated / supporting context | `I'm Locked-In now as a delegate and this locked-in context was not already loaded.` |
| Same chat, re-read for a reason | `I'm Locked-In now for <reason> and I already have access to the locked-in information and context.` |
| Same chat, first lock-in for a reason | `I'm Locked-In now for <reason> and this locked-in context was not already loaded.` |
| Fast-track continuation | `I'm Locked-In now [fast-track] continuing <phase/task name> — context already loaded.` |

Reason guidance: use `by user action` for a new chat; `as a delegate` when
supporting another context; `for <short phrase>` otherwise (e.g.
`for the selector planning review`).

Do not reload the locked-in context unnecessarily. If the required docs have
already been read in the current chat and no new reason requires re-reading them,
acknowledge that context is available and continue.

---

## Before Acting: Confirmation Gate

Gate requirements vary by mode.

### Fast-track gate (minimal)
Before writing code, confirm silently (no need to enumerate in the reply):
- Branch is not `main`/`master` (run `git branch`; prompt user if so).
- The change is within the scope described in the session summary or wip spec.
- No lessons-learned entry obviously prohibits the approach.

### Standard gate
Before writing code or a plan, confirm you can answer:

0. **Branch check** — run `git branch` and apply the Branch Check Rule in `guidance.md §11.1`. If the current branch is `main`/`master` or a newer related branch exists, prompt the user before proceeding.
1. Which package(s) does this change touch, and do those packages own the behavior being changed?
2. Does this change respect the architectural boundaries in `agent-readme.md`?
3. Does a lessons-learned entry warn against this approach?
4. Is there active work in `docs/wip/` this change should align with or defer to?
5. Does your plan satisfy the Definition of Done in `coding-standards.md`?

### Full review gate
Same as Standard **plus**: write out the answers to all five questions explicitly
in the reply and wait for user confirmation before proceeding.

Also answer **Q6 — Design principles compliance** explicitly:
- **DIP**: Do high-level modules (app, render) depend on interfaces, not concrete types? Are transport/network clients constructed at the `cmd/` composition root and injected, not built inside library packages?
- **DRY**: Are any code paths structurally duplicated (>80% identical)? If so, extract or accept with rationale.
- **SRP / GRASP Cohesion**: Does each new type/file have one clear responsibility? Are transport, rendering, and domain concerns separated?
- **Low Coupling / Protected Variations**: Is each transport or data-source seam isolated so swapping it doesn't touch unrelated layers?
- **IoC**: Are dependencies injected at construction time? Does any library package construct its own network clients or heavy resources?

For each violation found: state **accepted** (rationale + future TD note) or **corrected** in the plan. Accepted violations must not block a corrective refactor later.

---

## Project Tools

All project-specific MCP tools are served by `tools/write_file_server.py` and
registered in `.vscode/mcp.json`. See [`tools/mcp.json`](../tools/mcp.json) for
the full tool registry and [`docs/tools/write-file-mcp.md`](../docs/tools/write-file-mcp.md)
for enable/disable/extend instructions.

### `write_file`

Replaces the built-in `create_file` tool, which has a confirmed bug (writes
lines in reverse order).

**When creating a new file, always use `write_file` instead of `create_file`.**

If `write_file` is not available (server not loaded), use the fallback pattern:
1. `echo 'package x' > path/to/file.go` to create a minimal stub.
2. `replace_string_in_file` to fill in the full content.

### `get_current_datetime`

Returns the real wall-clock date and time in UTC and local time. Use this tool
whenever you need to know the current date or time — do not rely on a
training-data cutoff or a timestamp embedded in context.
