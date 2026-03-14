# CLAUDE.md

Unified working standard for LLM agents in this repository.
Goal: predictable quality, minimal regression risk, transparent changes, fast delivery.
Workflow: **Spec-Driven Development (SDD)** via OpenSpec .agent/ .

---

## 0) Purpose & Priorities

These rules optimize maintainability, safety, and developer velocity.

### Rule levels
- **MUST** — mandatory (blocks task completion).
- **SHOULD** — strong recommendation.
- **CAN** — optional, allowed when appropriate.

### Language
All agent responses, reasoning, planning, and questions to the user MUST be in Ukrainian.
Code, identifiers, commit messages, and spec examples remain in English.

### Conflict resolution
1. **MUST > SHOULD > CAN**
2. If rules conflict, choose the lowest-risk and simplest correct option.
3. Any trade-off must be explicitly captured in the active OpenSpec change's `tasks.md` (Review section).

---

## 1) Operating Mode — OpenSpec SDD

All non-trivial work (3+ steps, architectural choices, multi-module impact) MUST follow the OpenSpec lifecycle.

## 1.1 OpenSpec Lifecycle (mandatory for non-trivial tasks)

```
/opsx:new <feature-name>   → creates openspec/changes/<feature-name>/
/opsx:ff                   → generates proposal.md, specs/, design.md, tasks.md
                             (human reviews and refines before any code)
/opsx:apply                → implements tasks from tasks.md step-by-step
/opsx:archive              → archives change, updates living specs
```

**Before writing any code**: spec must exist and be reviewed.
**Before marking done**: all tasks in `tasks.md` checked off and verified.

## 1.2 OpenSpec Artifact Roles

| Artifact | Purpose |
|---|---|
| `proposal.md` | Why we're doing this, what changes |
| `specs/<area>/spec.md` | Functional requirements (SHALL/GIVEN/WHEN/THEN) |
| `design.md` | Technical decisions, trade-offs |
| `tasks.md` | Implementation checklist (source of truth for progress) |

Living specs in `openspec/specs/` are the canonical documentation of system behavior.
**Always read relevant specs before starting implementation.**

## 1.3 Spec Quality Standards

- **SDD-1 (MUST)** Use SHALL for requirements, GIVEN/WHEN/THEN for scenarios.
- **SDD-2 (MUST)** Spec delta must reflect every behavioral change — no silent modifications.
- **SDD-3 (SHOULD)** Specs are small, scoped, and named after capabilities (e.g., `auth-session`, `meter-reading`).
- **SDD-4 (SHOULD)** When fixing a bug, update the relevant spec scenario to prevent regression.
- **SDD-5 (CAN)** Reference related specs via relative paths in `design.md`.

## 1.4 Subagent strategy

Use subagents/tasks to:
- research existing specs and codebase context before proposing,
- parallelize independent task groups from `tasks.md`,
- isolate context for large changes.

One subagent = one focused task or spec area.

## 1.5 Autonomous execution (bugs / hotfixes)

For reported bugs:
1. Check if a relevant spec exists in `openspec/specs/` — if yes, the scenario describes the correct behavior.
2. Identify root cause via logs/tests/traces.
3. Implement minimal correct fix.
4. Update spec scenario if behavior was undocumented.
5. Prove correctness via verification.

Hotfixes skipping `/opsx:new` are allowed but MUST still update `openspec/specs/` if behavior changes.

---

## 2) Task Management Protocol

1. **Spec First**: before planning, read `openspec/specs/` for relevant areas.
2. **Plan via OpenSpec**: use `/opsx:new` + `/opsx:ff` to generate `tasks.md`.
3. **Track Progress**: mark items in `tasks.md` as you go.
4. **Explain Changes**: briefly note what changed and why in `design.md` or `tasks.md`.
5. **Verify Before Done**: tests/logs/behavior diff as relevant.
6. **Document Results**: add review summary to `tasks.md` (Review section).
7. **Archive**: run `/opsx:archive` to close the change and update living specs.

---

## 3) Mistakes & Corrections

If a mistake relates to **system behavior** → fix it in `openspec/specs/`.

### Git policy
- Commit: `openspec/specs/`, `openspec/changes/*/`
- Ignore: `tasks/todo.md`

---

## 4) Core Engineering Principles

- **Simplicity First**: minimal code change, maximal clarity.
- **No Laziness**: find root causes, avoid temporary patches.
- **Minimal Impact**: change only what is necessary; minimize blast radius.
- **Spec Fidelity**: implementation must match the spec — diverge only by updating the spec first.
- **Balanced Elegance**: for non-trivial work, ask:
  _"Is there a more elegant solution without over-engineering?"_

---

## 5) Before Coding

- **BP-1 (MUST)** Ask clarifying questions when critical ambiguity exists.
- **BP-2 (MUST)** Generate and review `proposal.md` + `specs/` before implementation.
- **BP-3 (MUST)** If >2 approaches exist, capture pros/cons in `design.md` with rationale.
- **BP-4 (SHOULD)** Define testing strategy and observability signals in `design.md`.
- **BP-5 (SHOULD)** Read existing `openspec/specs/` for affected areas before proposing.

---

## 6) Go Standards

## 6.1 Modules & dependencies
- **MD-1 (SHOULD)** Prefer stdlib first; add dependencies only with clear payoff.
- **MD-2 (CAN)** Use `govulncheck` for vulnerability checks.
- **MD-3 (CAN)** Use Go 1.26+ features when project-compatible.

## 6.2 Code style
- **CS-1 (MUST)** `gofmt` and `go vet` must pass.
- **CS-2 (MUST)** Avoid stutter naming (`package kv; type Store`).
- **CS-3 (SHOULD)** Small interfaces near consumers; composition over hierarchy.
- **CS-4 (SHOULD)** Avoid reflection on hot paths; use generics when they simplify and improve clarity/perf.
- **CS-5 (MUST)** Use input structs when function takes >2 non-context args. Do not place `context.Context` inside input structs.
- **CS-6 (SHOULD)** Declare input structs right before the consuming function.
- **CS-7 (SHOULD)** All user-facing text/comments/log messages are in English.
- **CS-8 (CAN)** Use `embed` for static assets.

## 6.3 Errors
- **ERR-1 (MUST)** Wrap errors with `%w` and context.
- **ERR-2 (MUST)** Use `errors.Is` / `errors.As` for control flow; no string matching.
- **ERR-3 (SHOULD)** Define package-level sentinel errors and document behavior.
- **ERR-4 (CAN)** Use `context.WithCancelCause` / `context.Cause`.

## 6.4 Concurrency
- **CC-1 (MUST)** Sender closes channels; receivers never close.
- **CC-2 (MUST)** Bind goroutine lifetime to `context.Context`.
- **CC-3 (MUST)** Protect shared state with `sync.Mutex`/`atomic`; no "probably safe" races.
- **CC-4 (SHOULD)** Use `errgroup` for fan-out and cancel on first error.
- **CC-5 (CAN)** Use buffered channels only with explicit rationale.

## 6.5 Contexts
- **CTX-1 (MUST)** If a function takes `ctx`, it must be first argument; never store context in structs.
- **CTX-2 (MUST)** Propagate non-nil context; honor Done/deadlines/timeouts.
- **CTX-3 (CAN)** Expose `WithX(ctx)` helpers derived from config.

## 6.6 Testing
- **T-1 (MUST)** Table-driven tests; deterministic and hermetic by default.
- **T-2 (MUST)** Run `-race` in CI; teardown via `t.Cleanup`.
- **T-3 (SHOULD)** Use `t.Parallel()` where safe.
- **T-4 (SHOULD)** Test scenarios map 1:1 to spec scenarios (GIVEN/WHEN/THEN → test case name).

## 6.7 Logging & observability
- **OBS-1 (MUST)** Structured logging (`slog`) with levels and consistent fields.
- **OBS-2 (SHOULD)** Correlate logs/metrics/traces via request IDs from context.
- **OBS-3 (CAN)** Provide debug/pprof endpoints behind auth or local-only access.

## 6.8 Performance
- **PERF-1 (MUST)** Measure before optimizing (`pprof`, benchmarks, `benchstat`).
- **PERF-2 (SHOULD)** Reduce allocations on hot paths; reuse buffers carefully.
- **PERF-3 (CAN)** Add microbenchmarks for critical paths and track regressions.

## 6.9 Configuration
- **CFG-1 (MUST)** Config via env/flags; validate on startup; fail fast.
- **CFG-2 (MUST)** Treat config as immutable after init; pass explicitly (not globals).
- **CFG-3 (SHOULD)** Provide sane defaults and clear docs.
- **CFG-4 (CAN)** Support hot reload only when correctness is preserved and tested.

## 6.10 APIs & boundaries
- **API-1 (MUST)** Document exported items; keep exported surface minimal.
- **API-2 (MUST)** Accept interfaces where variation is needed; return concrete types by default.
- **API-3 (SHOULD)** Keep functions small, orthogonal, composable.
- **API-5 (CAN)** Use options pattern for extensibility.

## 6.11 Security
- **SEC-1 (MUST)** Validate inputs, set explicit I/O timeouts, prefer TLS.
- **SEC-2 (MUST)** Never log secrets; manage secrets outside code.
- **SEC-3 (SHOULD)** Apply least-privilege defaults for filesystem/network access.
- **SEC-4 (CAN)** Add fuzz tests for untrusted inputs.

---

## 7) CI/CD & Tooling Gates

- **CI-1 (MUST)** Every PR: lint, vet, test (`-race`), build.
- **CI-2 (MUST)** Reproducible builds (`-trimpath`), version via ldflags.
- **CI-3 (SHOULD)** Require review sign-off for MUST-rule changes.
- **CI-4 (SHOULD)** PR description links to the OpenSpec change folder.
- **CI-5 (CAN)** Publish SBOM and run vuln/license checks.
- **CI-6 (CAN)** Use distroless base image for production.

### Gates
- **G-1 (MUST)** `go vet ./...`
- **G-2 (MUST)** `golangci-lint run`
- **G-3 (MUST)** `go test -race ./...`
- **G-4 (CAN)** `gh pr view <num>`, `gh pr diff <num>` for review context.

---

## 8) Definition of Done

A task is not complete until proven:
1. Functional behavior works (tests/scenario checks).
2. No relevant regressions in adjacent behavior.
3. Logs/errors/metrics match expected outcomes.
4. MUST rules are satisfied.
5. `openspec/changes/<n>/tasks.md` contains a final review summary.
6. `openspec/specs/` updated to reflect any behavioral changes.
7. `/opsx:archive` executed — change closed, living specs current.

Control question: **"Would a staff engineer approve this — code AND spec?"**

---

## 9) Function Design Best Practices

Before refactoring a function, check:
1. Can it be followed easily and honestly? If yes, avoid unnecessary changes.
2. Is cyclomatic complexity/high nesting too high? Simplify.
3. Would a better data structure/algorithm help (parser/tree/stack/queue)?
4. Are hidden dependencies better passed explicitly as arguments?
5. Brainstorm 3 better names; ensure current naming is truly best-fit.

---

## 10) Non-Negotiables for Agents

- Never mark done without verification.
- Never ship temporary hacks without explicit risk note.
- Never expand scope without necessity — and without updating `proposal.md`.
- Never skip `/opsx:ff` planning for non-trivial tasks.
- Specs are the contract — implementation divergence requires spec update first.

---

## Appendix — OpenSpec Quick Reference

```bash
# Ініціалізація (одноразово)
npm install -g @fission-ai/openspec@latest
openspec init

# Стандартний цикл
/opsx:new <feature-name>   # створити зміну
/opsx:ff                   # згенерувати всі артефакти планування
# → review proposal.md, specs/, design.md, tasks.md
/opsx:apply                # виконати tasks.md
/opsx:archive              # закрити зміну, оновити живі специфікації

# Допоміжні команди
/opsx:onboard              # первинне налаштування
/opsx:update               # оновити інструкції агента
```

### Why stable IDs matter

Stable rule IDs (BP-1, ERR-2, CC-2, SDD-3, etc.) enable precise code review comments, decision auditing, and automated policy checks. Do not renumber IDs; deprecate with notes instead.
