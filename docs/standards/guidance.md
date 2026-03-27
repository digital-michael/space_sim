# Guidance

## Purpose
Define the default way of working for LLM Agents and human contributors in this repository, especially for planning, execution flow, documentation hygiene, temporary artifacts, approvals, and phase-based delivery.

## Last Updated
2026-03-26

## Table of Contents
1. Role of This Document
2. Working Priorities
3. Required Delivery Workflow
4. Planning Requirements
5. Phase Execution Rules
6. Testing Requirements
7. User Approval and Commit Rules
8. Documentation Rules
	8.1 Required Metadata for `docs/`
	8.2 Documentation Placement
	8.3 Documentation Quality
	8.4 Work Tracking Documents
9. Code and Design Guidance
10. Temporary Artifact Rules
	10.1 Temporary Scripts
	10.2 Temporary Documents and Notes
11. Agent Default Behavior

## 1. Role of This Document

This document defines how work should be performed in this repository.

Use it together with:

- [docs/standards/coding-standards.md](coding-standards.md) for implementation standards.
- [docs/standards/agent-readme.md](agent-readme.md) for repository architecture and operating context.
- [docs/history/lessons-learned.md](../history/lessons-learned.md) and related lessons-learned documents when intent or risk is unclear.

If information is missing, agents should prefer repository docs and lessons learned over improvisation.

## 2. Working Priorities

Apply these priorities in order:

1. Understand the task and repository context before editing.
2. Produce a defined, refined, and finalized implementation plan before substantial work.
3. Keep work aligned with architecture and coding standards.
4. Test between meaningful steps, not only at the end.
5. Keep docs accurate as the implementation evolves.
6. Keep code performant, but not obscurely so.
7. Remove temporary artifacts when the task is complete.

## 3. Required Delivery Workflow

All non-trivial work should follow this sequence:

1. Design
	- Understand the problem, current architecture, constraints, and likely impact area.
2. Queue
	- Break the work into concrete tasks or phases.
3. Schedule
	- Order the phases so dependencies and validation points are clear.
4. Implement
	- Make focused changes in the planned order.
5. Test between steps
	- Validate each phase or meaningful checkpoint before moving on.
6. Test after all phases
	- Run final validation across the completed work.
7. Check for user approval
	- Confirm with the user before phase-finalizing actions when appropriate.
8. Commit at end of each phase
	- Use a terse commit message if committing is requested or part of the agreed workflow.

For very small tasks, the workflow can be compressed, but not skipped. The planning still needs to exist, even if it is brief.

## 4. Planning Requirements

Before substantial implementation work, define a plan that is:

- Defined
  - The scope, target files, expected behavior, and validation approach are identified.
- Refined
  - Risks, dependencies, unclear assumptions, and sequencing are considered.
- Finalized
  - The plan is stable enough to execute in phases without constant reinvention.

Good plans in this repository should:

- identify the architectural layer being changed;
- identify whether the work affects code, data, tests, scripts, or docs;
- state what will be validated after each phase;
- identify where user approval is expected;
- avoid mixing unrelated refactors into feature or bug-fix work.

## 5. Phase Execution Rules

- Work in bounded phases.
- Each phase should have a clear objective and a clear validation step.
- Do not begin a later phase if the current phase is still failing basic checks.
- Keep phase boundaries visible in progress updates and summaries.
- Prefer completing one coherent area correctly before widening the change surface.

Examples of sensible phases:

- repo survey and design updates;
- schema or data changes;
- runtime implementation changes;
- test additions or fixes;
- documentation cleanup and final validation.

## 6. Testing Requirements

- Test between phases or meaningful steps.
- Test again after all planned phases are complete.
- Choose the narrowest useful validation first, then broaden when needed.
- If a relevant test cannot be run, say so explicitly and explain why.
- If the work changes behavior without existing tests, add tests or document the gap.
- Docs-only changes should still be checked for consistency, structure, and obvious errors.

## 7. User Approval and Commit Rules

- Check for user approval before finalizing major direction changes, multi-phase completion points, or requested approval gates.
- Do not assume approval for architectural redesigns or broad scope expansion.
- If commits are part of the workflow, commit at the end of each completed phase, not mid-phase.
- Commit messages should be terse and outcome-focused.
- When a work identifier already exists, use it to shorten the message further, but always keep a brief title after the identifier.
- Good format: `<work-id>: <brief outcome>`.
- Avoid identifier-only commit messages with no human-readable title text.
- Do not create commits unless the user has asked for commits or has clearly adopted a commit-per-phase workflow for the task.

Examples of acceptable terse commit messages:

- `docs: add agent guidance`
- `space: fix belt dataset visibility`
- `ui: cover selection defaults`
- `TODO-4: rename CLI options`
- `TODO-6: fix runtime memory defect`

## 8. Documentation Rules

### Required Metadata for `docs/`

All new and updated files in `docs/` must include and maintain:

- a brief `Purpose` section explaining what the document is for;
- a `Last Updated` section;
- a `Table of Contents` section.

These sections must be kept up to date on each meaningful update.

Table of contents entries should use a three-level outline maximum. In practice, document tables of contents should cover:

- level 1 major sections;
- level 2 subsections when they improve navigation;
- level 3 subsections only when needed for clarity.

Do not expand document tables of contents beyond three outline levels.

### Documentation Placement

- All new Markdown documents should live under `docs/` in an appropriate sublocation.
- If a temporary Markdown file is needed during work, place it in `docs/`, not in the repository root or an arbitrary folder.
- New documentation should be organized so a future agent can locate it by topic.
- Reserve [README.md](../../README.md) for directory entry documents; otherwise use lowercase kebab-case filenames for Markdown documents.

### Documentation Quality

- Make documentation work nice: readable, structured, current, and useful.
- Prefer concise headings, explicit scope, and actionable content.
- Avoid stale sections, placeholder text, or tables of contents that do not match the file.
- When code or workflow changes invalidate docs, update the docs in the same workstream.

### Work Tracking Documents

- Keep active and future work in [docs/wip/todo.md](../wip/todo.md), not in package directories.
- Keep finished work in [docs/history/changelog.md](../history/changelog.md) as concise dated completion records.
- Any todo item or work section that moves to `in progress` must include a `Start Date`.
- When work is moved from the todo into the changelog, record an `End Date` for that completed work.
- Use `YYYY-MM-DD` for all work-tracking dates.
- When work is completed, remove it from the active todo and move the outcome into the changelog instead of leaving completed backlog items mixed into the live queue.
- Keep decision records and architecture rationale out of the live todo unless the decision is still open and blocking execution.
- If a backlog item grows into a substantial design or implementation plan, move that plan into a proper document under `docs/` and leave only a short pointer in the todo.

## 9. Code and Design Guidance

- Code should conform to [docs/standards/coding-standards.md](coding-standards.md).
- Code should be adapted to the existing architecture, not forced around it.
- Prefer simple solutions when possible.
- Make code performant, but not obscurely so.
- Use interfaces and/or generics when they materially improve boundaries, reuse, or clarity.
- Do not introduce abstraction that makes the system harder to understand.
- Referenced or copied patterns must be adapted correctly to this repository’s architecture and naming conventions.

Performance guidance in this repository means:

- preserve correctness first;
- measure when performance claims matter;
- prefer explicit data flow and maintainable optimizations;
- document non-obvious performance tradeoffs when they are important to future work.

## 10. Temporary Artifact Rules

### Temporary Scripts

- Temporary scripts should be deleted when the work is completed.
- If a script has lasting operational value, convert it into a maintained script and place it under `scripts/`.
- Maintained scripts must conform to the standards in [scripts/README.md](../../scripts/README.md).

### Temporary Documents and Notes

- Temporary Markdown files belong in `docs/` while in use.
- Remove them when they are no longer needed, or promote them into proper long-lived documentation.
- Avoid leaving behind scratch files that future contributors may mistake for authoritative docs.

## 11. Agent Default Behavior

Unless the user explicitly requests otherwise, agents should:

1. inspect the relevant code, docs, data, and tests first;
2. create a defined, refined, and finalized plan;
3. implement in phases;
4. test between phases;
5. perform final validation after all phases;
6. update docs when needed;
7. ask for approval at meaningful checkpoints when the workflow or task calls for it;
8. keep temporary artifacts under control;
9. avoid broad, unrelated cleanup during focused work.

This document is intended to improve LLM Agent and human interaction by making expectations explicit, repeatable, and easy to follow.
