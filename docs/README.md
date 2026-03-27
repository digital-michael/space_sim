# Documentation Guide

## Purpose
Provide a directory-level index for the `docs/` tree so contributors and agents can quickly find standards, historical implementation notes, performance investigations, schema references, and work-in-progress material.

## Last Updated
2026-03-26

## Table of Contents
1. Folder Layout
2. Key Entry Points
3. Naming and Maintenance Rules

## 1. Folder Layout

- [standards/](standards/): agent behavior, coding standards, and workflow guidance.
- [history/](history/): implementation history, migrations, lessons learned, and ownership notes.
- [performance/](performance/): performance test guides, analysis, and consolidated results.
- [schema/](schema/): JSON/data-structure reference documents.
- [wip/](wip/): active backlog, retained planning, and work-in-progress material.

## 2. Key Entry Points

- [standards/agent-readme.md](standards/agent-readme.md): repository orientation for LLM agents.
- [standards/coding-standards.md](standards/coding-standards.md): implementation standards.
- [standards/guidance.md](standards/guidance.md): planning, approval, and execution workflow.
- [wip/todo.md](wip/todo.md): active and future work queue.
- [history/changelog.md](history/changelog.md): completed work history moved out of the live queue.
- [history/lessons-learned.md](history/lessons-learned.md): defect history and operational lessons.
- [performance/performance-results.md](performance/performance-results.md): consolidated performance findings.
- [schema/solar-system-json-schema.md](schema/solar-system-json-schema.md): JSON schema reference.

## 3. Naming and Maintenance Rules

- Use [README.md](../README.md) only for directory entry documents.
- Use lowercase kebab-case for other Markdown filenames.
- Keep substantive project documentation under `docs/`.
- Update links when moving documents so references stay navigable.