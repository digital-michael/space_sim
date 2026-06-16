---
template-version: 1.0.0
---

# Session Context

> Use this file to capture state when switching models, pausing work, or resuming in a new session.
> This is the source of truth for cross-session handoff.

---

## Project Summary

**Project:**
**Date captured:**
**Current model tier in use:** <!-- Planning/Reasoning | Implementation | Minor task -->

*One paragraph: what is this project, what is it trying to achieve?*

---

## Active Assignment

**Assignment file:** <!-- path to agent-assignment.md -->
**Assignment title:**
**Current phase:** <!-- Intake | Q&A | Planning | Implementation | Quality Gate | Retrospective -->

---

## Last Known Status

*What was the last completed unit of work? What was the immediate next step?*

**Last completed:**

**Next step:**

**Any open questions or blockers:**

---

## Implementation Plan — Current Snapshot

*Copy the current implementation plan table here so a new session can orient without reading the full assignment.*

| # | Unit of Work | Status |
|---|---|---|
| | | |

---

## Governance Context

**Load order file:** `docs/governance/README.md`
**Active lessons-learned file:** `docs/governance/lessons-learned.md`
**Language overlay(s) in use:** Go (`llm-agent-framework/library/go/instructions.md`, `llm-agent-domains/photon-datum/library/go/governance-overlay.md`)

---

## Model Context Notes

*Anything the next model instance needs to know that is not in the assignment or governance files.*

-

---

## Resumption Instructions

*Clear instructions for the next session to pick up work without loss of context.*

1. Load context per `docs/governance/README.md` — use `full` profile if resuming mid-assignment
2. Read this file in full before reading the assignment file
3. Read `docs/governance/agent-assignment.md` and locate the last completed unit
4.
