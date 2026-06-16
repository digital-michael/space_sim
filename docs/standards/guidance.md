# Guidance

## Purpose
Space_sim-specific delivery conventions that extend the LLM Agent Collaboration Framework. Workflow, planning, testing, and approval rules are defined in the framework; this document covers only what is unique to this project.

## Last Updated
2026-06-15

## Table of Contents
1. Commit Conventions
2. Documentation Rules
3. Performance Guidance

---

## 1. Commit Conventions

Commit at the end of each completed phase, not mid-phase. Messages must be terse and outcome-focused.

**Format:**
```
<type>: <brief outcome>
<work-id>: <brief outcome>   ← when a work identifier exists
```

**Examples:**
- `feat: add stellar classification system`
- `fix: strip path prefix in script dispatch`
- `docs: add agent governance`
- `F-020: session registry subscribe-before-bootstrap`
- `TODO-6: fix runtime memory defect`

Do not create commits unless the user has asked for commits or the task uses a commit-per-phase workflow. Avoid identifier-only commit messages with no human-readable title text.

---

## 2. Documentation Rules

### Required Metadata

All new and updated files in `docs/` must include and maintain:

- a brief `Purpose` section explaining what the document is for
- a `Last Updated` section
- a `Table of Contents` section (three-level outline maximum)

Keep these sections current on each meaningful update. Do not expand tables of contents beyond three outline levels.

### Documentation Placement

| Content | Location |
|---|---|
| Active and future work items | `docs/wip/todo.md` |
| In-progress specs, audits, phase plans | `docs/wip/` |
| Completed work (dated records) | `docs/history/changelog.md` |
| Lessons learned | `docs/history/lessons-learned.md` or `docs/history/lessons-learned-double-buffering.md` |
| Architecture and repo map | `docs/standards/agent-readme.md` |
| Coding and engineering standards | `docs/standards/coding-standards.md` |
| Active assignment | `docs/governance/agent-assignment.md` |
| Cross-session handoff | `docs/governance/session-context.md` |
| Performance data | `docs/performance/` |
| All new Markdown docs | Under `docs/` in an appropriate sublocation |
| Temporary Markdown files | `docs/` while in use; remove or promote when done |
| Root-level Markdown | Reserved for top-level repository files (`README.md`) |
| Non-README filenames | lowercase kebab-case |

### Documentation Quality

- Prefer concise headings, explicit scope, and actionable content.
- Avoid stale sections, placeholder text, or tables of contents that do not match the file.
- When code or workflow changes invalidate docs, update the docs in the same workstream.

### Work Tracking

- Any todo item or work section that moves to `in progress` must include a `Start Date`.
- When work moves from todo into changelog, record an `End Date`.
- Use `YYYY-MM-DD` for all work-tracking dates.
- Remove completed items from the active todo and move the outcome into the changelog.
- If a backlog item grows into a substantial plan, move it to a proper doc under `docs/` and leave only a short pointer in the todo.

---

## 3. Performance Guidance

Preserve correctness first. Measure when performance claims matter. Prefer explicit data flow and maintainable optimizations. Document non-obvious performance tradeoffs when they are important to future work.
