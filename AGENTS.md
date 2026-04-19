# Agent Instructions for Agent Skill Eval

## Repository Status

This repository is in migration from the original legacy MVP toward a new A/B Skill Evaluation Framework.

During this phase:

- keep the repository buildable at every step
- preserve current behavior unless the task explicitly changes it
- prefer compatibility-first migration before replacement
- align implementation with the docs before adding new modules

## Architecture References

Use these documents as the migration references:

- `docs/BACKEND_ARCHITECTURE.md` for the target backend architecture
- `docs/design.md` for the overall design intent
- `docs/IMPLEMENTATION_ROADMAP.md` for implementation order

`docs/frontend-design.md` is a forward-looking reference for later `server/` and `web/` work and must not be treated as already implemented.

## Legacy Compatibility Layer

The following paths are currently retained as a legacy / compatibility layer and should stay in place unless a task explicitly replaces them:

- `cmd/agent-eval`
- `internal/*`

Do not remove or rewrite these areas wholesale during repository governance or migration prep.
Prefer compatibility bridges and phased replacement.

## Allowed New Modules

The migration may introduce these new top-level modules when required:

- `agent/`
- `eval/`
- `providers/`
- `skill/`
- `tool/`
- `server/`
- `web/`

A root `main.go` is also allowed for the new entrypoint.

## Allowed Capability Expansion

The migration may introduce the following capabilities when explicitly in scope:

- OpenAI-compatible providers
- A/B evaluation
- event streaming
- SSE
- HTML reports

Do not present planned capabilities as completed features.

## Engineering Discipline

- Use Go as the primary language for backend logic.
- Keep changes incremental, explicit, and easy to review.
- Preserve local compilability after every step.
- Prefer adding or updating tests around touched behavior when practical.
- Keep old behavior working before replacing it.
- Avoid broad refactors unless the task clearly requires them.

## Migration Workflow

For each step:

1. choose one concrete target
2. implement it with minimal surface area
3. verify build or tests for the touched scope
4. keep the repository stable
5. then move to the next step

Prefer small checkpoint-worthy changes over large rewrites.

## Scope Guardrails

When the task is repository governance, documentation alignment, cleanup, or migration prep:

- update docs and repository metadata as needed
- clean up generated artifacts and ignore rules when confirmed
- avoid unnecessary runtime behavior changes
- avoid large edits to `cmd/` and `internal/`

When the task is implementation:

- use `docs/BACKEND_ARCHITECTURE.md` and `docs/design.md` as the target architecture references
- use `docs/IMPLEMENTATION_ROADMAP.md` as the sequencing reference
- keep legacy CLI and internals working until intentional replacement is complete

## Definition Of Done

A step is complete only if:

- the targeted change is finished
- the repository still builds locally
- relevant tests pass when they exist
- no unrelated files were changed unnecessarily
- the result is ready for a small checkpoint commit
