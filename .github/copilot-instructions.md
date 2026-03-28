# Copilot Workspace Instructions

## Required Reading

Read all of these before producing any plan or code.

| # | File | Extract |
|---|------|---------|
| 1 | [`docs/standards/agent-readme.md`](../docs/standards/agent-readme.md) | Repository map, package ownership, architectural boundaries, preserved refactor intent, agent working defaults |
| 2 | [`docs/standards/guidance.md`](../docs/standards/guidance.md) | Delivery workflow (Design → Queue → Schedule → Execute), planning requirements, phase rules, documentation rules, approval gates |
| 3 | [`docs/standards/coding-standards.md`](../docs/standards/coding-standards.md) | Priority order for conflicting standards, core engineering rules, Go best practices, Definition of Done |
| 4 | [`docs/wip/todo.md`](../docs/wip/todo.md) | Active work items and blocking dependencies — align with in-progress work, do not introduce a parallel track |
| 5 | [`docs/wip/`](../docs/wip/) | Scan for additional phase docs, audits, or planning artifacts |
| 6 | [`docs/history/lessons-learned.md`](../docs/history/lessons-learned.md) | Verified root causes, confirmed anti-patterns, hard-won constraints |
| 7 | [`docs/history/lessons-learned-double-buffering.md`](../docs/history/lessons-learned-double-buffering.md) | Concurrency, cloning, synchronization, and double-buffer anti-patterns |

---

## Acknowledgment

After reading and understanding all required documents above, respond with **"I'm Locked-In now."** before anything else. This signals to the user that the workspace rules are in effect.

---

## Before Acting: Confirmation Gate

Before writing code or a plan, confirm you can answer:

1. Which package(s) does this change touch, and do those packages own the behavior being changed?
2. Does this change respect the architectural boundaries in `agent-readme.md`?
3. Does a lessons-learned entry warn against this approach?
4. Is there active work in `docs/wip/` this change should align with or defer to?
5. Does your plan satisfy the Definition of Done in `coding-standards.md`?
