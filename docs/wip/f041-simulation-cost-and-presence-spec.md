# F-041 — Simulation Cost and Player Presence

## Purpose
Define what a running simulation costs, who creates demand for it, and what it does when nobody is there. Introduces a player presence model (Active / Away) and a server lifecycle policy driven by it. Replaces today's behaviour, in which a simulation advances at full rate and clones a snapshot of every object 30 times a second regardless of whether anything consumes it.

## Last Updated
2026-09-25

## Table of Contents
1. Framing
2. Locked Decisions
3. Vocabulary
4. Architecture
	4.1 Presence Model
	4.2 Demand and Lifecycle
	4.3 Snapshot Push Conditions
	4.4 Implementation Constraints
5. Rejected Alternatives
6. Wire Surface
7. Hard Prerequisites
8. Phases and Work Items
9. Acceptance Criteria
10. Known Interactions
11. Open Questions
12. Related Work

---

## 1. Framing

This began as a stray-process defect: a `space-sim-server` ran 47h43m at 100–142% CPU with zero clients attached. The practitioner reframed it, correctly, as a **cost of simulation** problem. The forgotten process was a detector, not the defect — if idle cost were near zero, a stray server would have been invisible.

Measured cost drivers, from code and observation:

| Driver | Rate | Gated on demand? |
|---|---|---|
| Physics tick (N-body named bodies, Keplerian belts) | `defaultSimHz` = 60 Hz | No |
| Snapshot deep `Clone()` of every object, pushed to subscribers | `snapshotHz` = 30 Hz | No — pushed to an empty subscriber list |

Roughly one core at 717 objects, the default dataset. `docs/wip/todo.md` documents datasets of 1,200–24,000 objects, about 33× that object count; the cost curve across that range has never been characterised.

**Why this is foundational rather than janitorial.** It is load-bearing for four planned features:

- **F-039 Concurrent simulations** — if one simulation costs a core, N cost N cores. Cost scales with simulations, not players. How many fit on a host is currently a number discovered by accident.
- **F-010 Multi-machine** — every headless box pays a core permanently from startup.
- **F-012 Federated compute** — the entire premise is distributing simulation cost. A partition strategy *is* a cost-allocation decision, and cannot be designed without a cost model.
- **F-011 / multi-client** — 100 clients on one simulation is cheap; 100 simulations with one client each is not. What a session costs depends on which shape is being offered.

---

## 2. Locked Decisions

Settled with the practitioner 2026-09-24/25. Not to be re-litigated during implementation. Where a decision was inferred rather than stated it is marked.

| # | Decision | Rationale |
|---|---|---|
| D1 | The frame is **cost of simulation**, not stray processes | A forgotten process mattered only because idle cost is ~1 core. Process hygiene is a separate, smaller concern |
| D2 | Two flags: `--idle=run\|suspend\|exit` and `--max-runtime=<duration>`. **`suspend` is the default** | A single-valued `--idle` makes invalid combinations unrepresentable, unlike a close-mode × idle-mode matrix with an incompatibility rule |
| D3 | The term is **active**, not *connected* | Lets a player declare themselves away without disconnecting, which no connection-state term can express |
| D4 | `active` = connected **and** not Away | |
| D5 | Presence states are **ACTIVE** and **AWAY** only | Extend when a feature needs it, not preemptively |
| D6 | Away is a **player declaration**; no heartbeat is involved. Network liveness is a **separate** transport concern | Conflating declared intent with connection liveness was an earlier agent proposal and was rejected. Intent and liveness fail differently and are detected differently |
| D7 | `demand = ACTIVE sessions where role ≠ ADMIN`. **Only ADMIN is exempt** | An admin manages the server; everyone else consumes the world |
| D8 | An admin **observes without interfering**: never creates demand, never wakes a suspended simulation, never prevents one suspending. Admin *mutations* apply but do not wake | State changes, lifecycle does not; the next active player finds the new world |
| D9 | Add `CLIENT_ROLE_SPECTATOR`. **Spectators create demand.** `UNSPECIFIED` and `OTHER` continue to clamp to `PLAYER` | Watching planets orbit is a legitimate use — this is a planetarium as much as a game — and a spectator watching a frozen world is pointless. Clamping an unknown role to SPECTATOR instead would give a forgetful renderer a frozen world, a far more confusing failure than a forgotten client costing a core. ADMIN is never a default or fallback for anything |
| D10 | A suspended world resumes at the **sim time it stopped at**, not the wall clock | Eliminates catch-up entirely: no elapsed-time estimation, no reconstructing events, and N-body path-dependence never arises |
| D11 | Suspend is implemented by **gating the tick**, never by cancelling the sim context | World state stays in memory, so resume is "start ticking again" rather than a teardown and rebuild. This is what makes near-instant resume achievable, and the tempting implementation is the wrong one |
| D12 | Simulation advance and snapshot push are **separate conditions** | An admin creates no demand but must still receive snapshots, or "observes without interfering" is unachievable |
| D13 | `suspend` triggers on **no demand**; `exit` requires **zero sessions**, full stop | *Agent-stated interpretation, offered for correction and uncorrected.* Suspend is free and reversible, exit is destructive and irreversible — asymmetric consequence justifies asymmetric thresholds. Without this, everyone stepping away kills the server under connected players |
| D14 | Away is a **demand signal, not a physics change** | An Away player still has a ship with position and velocity. While other players are active the simulation runs and that ship keeps drifting under gravity like any object. "Away" must not be read as "frozen" |
| D15 | `--max-runtime` is **unconditional** and can terminate a live session. Its help text must say so | It is a debug aid, not a policy mechanism |

---

## 3. Vocabulary

Agreed before design, deliberately, because the earlier ambiguity in "connected" produced two candidate populations that would have given opposite answers.

| Term | Meaning |
|---|---|
| **Session** | A registered client in `session.Registry`. Absence from the registry is not a presence state — it is absence |
| **Presence** | A session's declared availability: `ACTIVE` or `AWAY`. Set by the client, never inferred from timing |
| **Active** | Connected and not Away |
| **Demand** | At least one ACTIVE session whose role is not ADMIN. What decides whether the simulation advances |
| **Liveness** | Whether a session's transport is still functioning. Independent of presence; detected by timeout, not declaration |
| **Suspended** | The simulation exists with its state intact but its tick is gated. Sim time does not advance |

---

## 4. Architecture

### 4.1 Presence Model

`ClientSession` gains a presence field defaulting to `ACTIVE` at registration. A client changes it by calling `SetPresence`; nothing else writes it. Presence is carried in `ClientSessionInfo` on the snapshot so peers can render it — F-021's marker should distinguish an Away player, or others read the stillness as being ignored.

Presence is orthogonal to liveness. A session that stops responding is neither Active nor Away: it is *stale*, and the liveness timeout unregisters it. Until that fires, a dead client still counts as demand — see §7.

### 4.2 Demand and Lifecycle

```
demand := any session where presence == ACTIVE and role != ADMIN
```

| `--idle` | Behaviour when demand is absent |
|---|---|
| `run` | Advance at full rate regardless. The pre-F-041 behaviour, kept for deliberate deployments |
| `suspend` | **Default.** Gate the tick. Sim time stops. Resume on the first ACTIVE non-admin session |
| `exit` | Terminate the process — but only when **zero sessions** exist at all (D13), not merely zero active ones |

`--max-runtime=<duration>` is orthogonal and unconditional: the process exits when it expires, whatever is connected.

Wake is triggered by any session transitioning to ACTIVE, whether a returning Away player or a new connection.

### 4.3 Snapshot Push Conditions

Independent of §4.2, and both clauses matter:

```
push := (at least one stream subscriber) and (the world advanced since the last snapshot)
```

The first clause removes an O(N) deep copy 30×/sec going to nobody. The second means an admin watching a suspended world receives one snapshot and then silence until something changes — push-on-change rather than push-on-timer. This is strictly better than today's behaviour and is correct whatever `--idle` is set to, so it can land independently of everything else here.

### 4.4 Implementation Constraints

- **Gate the tick; do not cancel the context** (D11). Context cancellation is one-way and would force a rebuild of the sim goroutine and worker pool on every resume.
- Worker pool threads may stay alive while suspended; idle threads cost nothing and keeping them avoids resume latency.
- The first snapshot after resume is a full clone, which is microseconds-to-milliseconds — not a latency concern.
- Perceived resume time is dominated by client connection setup (bundle cache hit ~3 ms, register, activate), not by the simulation. `suspend` is therefore viable in live multiplayer, not only in development.

---

## 5. Rejected Alternatives

Recorded with reasons so they do not return unexamined.

| Rejected | Why |
|---|---|
| `reduce-by NN`, `slow-until-clients`, `--idle=hz=N` | For a fixed-step integrator, lowering the tick rate is not a cost knob — it is an accuracy or a time-rate knob. Keep `dt` and world time advances slower than wall clock; stretch `dt` and integration error grows (leapfrog energy error scales with `dt²`, and F-013 made named bodies genuinely path-dependent). Suspend is honest; slow is silently lossy |
| `pause-until-clients-and-estimated-simulation-catchup` | Analytic catch-up is exact and free for Keplerian bodies but **impossible** for N-body, which cannot be evaluated at time *t* without integrating through it. Anything stateful in the gap — F-027 collisions, F-035/F-036 events, NPC decisions — simply did not happen and cannot be reconstructed. D10 eliminates the need entirely |
| `suspend-until-npc` | External NPCs are clients: they register sessions and are already covered by demand. Internal NPCs run inside the simulation, so their actions cannot be a wake condition — the simulation must run to produce them. A scheduled-actor model that sleeps until the next event is legitimate but is **discrete-event simulation**, a different execution model rather than a flag on a fixed-tick loop |
| `--server-close` as a flag family separate from `--as-idle` | Collapsed into a single-valued `--idle` plus an orthogonal `--max-runtime`, eliminating a 3 × 5 matrix and its incompatibility rule |
| `./space-sim-server --shutdown` | `ShutdownService.Shutdown` already exists in the proto and is simply unregistered on the headless server. A flag meaning "act as a client and tell another process to die" is a command wearing a flag's clothing; it belongs in `space-sim-admin` |
| Defaulting an unknown role to ADMIN-like (no-demand) semantics | An earlier agent suggestion, rejected: inheriting admin-adjacent semantics for an unspecified role is backwards from least privilege. See D9 |

---

## 6. Wire Surface

| Change | Detail |
|---|---|
| `ClientRole` | Add `CLIENT_ROLE_SPECTATOR` |
| `PresenceState` | New enum: `PRESENCE_UNSPECIFIED = 0`, `PRESENCE_ACTIVE`, `PRESENCE_AWAY` |
| `ClientSession` | Add presence field, defaulting to ACTIVE at registration |
| `ClientSessionInfo` | Add presence so peers can render it |
| `SessionService.SetPresence` | New RPC. A declaration; carries no timeout semantics |
| Lifecycle visibility | The run state (running / suspended) must be readable, or an admin cannot distinguish a suspended world from a hung server — which would make D8's "sees it as it is" true in letter and false in practice. Proposed on `GetSimulationTimeResponse`, since anything asking for the time wants to know whether it is advancing |
| `ShutdownService` | Register it on `space-sim-server`; it exists but returns 404 there |

---

## 7. Hard Prerequisites

Both gate **correctness**, not convenience.

1. **F-020 graphical-client session wiring** (Ledger `cc28512b`). The renderer never registers today, so "zero active sessions" would be true while a player is watching — with `suspend` as the default, the server would freeze the world under them. No registration means no presence, which means no demand signal.

2. **Liveness timeout** (Ledger `260e55b3`). `ClientSession.LastSeen` is written once at registration and never updated, and no reaper exists. A client that dies without unregistering still counts as demand and pins the simulation at full rate — which is the original 47-hour failure arriving by a second route. **`suspend` being the default makes this load-bearing rather than optional.** A half-open connection (laptop sleep, network partition) leaves a stream that looks alive, so stream closure alone is not sufficient detection.

---

## 8. Phases and Work Items

### Phase 0 — Snapshot push conditions

Independent of everything else; can land immediately.

- [ ] Skip the `Clone()` and push when the subscriber list is empty
- [ ] Skip the push when the world has not advanced since the last snapshot
- [ ] Test: a subscriber attached to a non-advancing world receives one snapshot, then none

### Phase 1 — Presence model

**Depends on:** F-020 session wiring.

- [ ] `PresenceState` enum and `SetPresence` RPC in `session.proto`
- [ ] Presence on `ClientSession`, defaulting to ACTIVE; registry accessor and mutator
- [ ] Presence on `ClientSessionInfo`; populate in `world_handler`
- [ ] `CLIENT_ROLE_SPECTATOR` added; confirm UNSPECIFIED/OTHER still clamp to PLAYER
- [ ] Client: a way to declare Away, and render a peer's Away state distinctly
- [ ] Tests: presence defaults to ACTIVE; SetPresence round-trips; presence reaches the snapshot

### Phase 2 — Demand and lifecycle

**Depends on:** Phase 1, and the liveness timeout.

- [ ] Demand predicate per D7, with ADMIN excluded
- [ ] `--idle=run|suspend|exit`, defaulting to `suspend`
- [ ] Suspend by gating the tick (D11); assert the sim goroutine survives a suspend/resume cycle
- [ ] `exit` requires zero sessions (D13)
- [ ] Wake on any session becoming ACTIVE
- [ ] Admin mutations apply without waking (D8)
- [ ] Tests: suspend with all-Away; no suspend with one ACTIVE player; no suspend prevented by an ADMIN-only connection; exit does not fire with an Away session present

### Phase 3 — Runtime bound and visibility

- [ ] `--max-runtime=<duration>`, unconditional, with help text stating it can terminate a live session (D15)
- [ ] Run state readable by an admin
- [ ] Register `ShutdownService` on `space-sim-server`

### Phase 4 — Cost characterisation

Feeds F-039 and F-012; not blocking.

- [ ] Benchmark CPU against object count from ~700 to ~24,000
- [ ] Record the curve so per-host simulation budgets are chosen from data

---

## 9. Acceptance Criteria

Verified behaviourally, by running the binaries.

- [ ] A server with no clients, default flags, settles to near-zero CPU rather than ~1 core
- [ ] A single connected renderer keeps the world advancing
- [ ] All players declaring Away suspends the world; any one returning resumes it, from the sim time it stopped at
- [ ] An Away player's ship continues drifting while other players are active (D14)
- [ ] An ADMIN-only connection does **not** keep the world advancing, and the admin can tell it is suspended rather than hung
- [ ] An admin `LoadSystem` against a suspended world applies without waking it
- [ ] `--idle=exit` does not fire while an Away session is still connected
- [ ] Resume is perceptually immediate
- [ ] `go vet ./...` and `go test -race ./...` pass

---

## 10. Known Interactions

| Item | Interaction |
|---|---|
| **F-015 Epoch-accurate start ("start from today")** | Suspend makes sim time diverge from wall clock, so a world that paused no longer corresponds to the real sky. These goals are **incompatible**, not merely in tension. Named here rather than discovered later |
| F-021 Client physical marker | An Away player's marker should be visually distinct |
| F-038 HUD profiles | Presence belongs in a multiplayer HUD |
| F-027 / F-035 / F-036 | Reinforces D10: nothing stateful happens while suspended, and with sim-time resume nothing needs to |
| Autosave | If driven by sim time it pauses with the world, which is correct. If wall-clock driven it will fire against an unchanging world — harmless but wasteful; worth checking |

---

## 11. Open Questions

| # | Question | Status |
|---|---|---|
| Q1 | Liveness timeout duration | Open — pick with the timeout implementation, not here |
| Q2 | Should `CLIENT_ROLE_OTHER` be deprecated? It now behaves identically to UNSPECIFIED and has no distinct purpose | Open — flagged, not decided. Young enum, nothing depends on its meaning |
| Q3 | Cost curve from 700 to 24,000 objects | Open — Phase 4 measures it; do not design a budget before then |
| Q4 | Does a per-host simulation budget exist for F-039, and who enforces it? | Deferred to F-039, but it depends on Q3 |

---

## 12. Related Work

| Item | Relationship |
|---|---|
| F-020 Multi-Client Session Layer | Hard prerequisite — presence lives on sessions |
| F-039 Concurrent simulations | Primary beneficiary; cost per simulation decides how many fit a host |
| F-012 Federated compute | Cannot be designed without the cost model this establishes |
| F-010 Multi-machine | Every headless box currently pays a core permanently |
| F-013 N-body | The reason slow-down and analytic catch-up were both rejected |
| F-022 Client locomotion | An Away player's ship keeps drifting under its physics (D14) |
