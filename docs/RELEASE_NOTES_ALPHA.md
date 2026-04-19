# Alpha Release Notes

## Title / Version

**Agent Skill Eval — New Architecture Alpha**

## Release Status

This release should be treated as an **alpha**.

It is suitable for internal evaluation, early testing, and architectural review, but it is **not** beta and **not** production-ready.

## Scope Of This Alpha

This alpha packages the current end-to-end new-architecture path into a usable, documented, and testable repository state.

The goal of this release is to provide:

- a runnable CLI entrypoint
- reproducible config-driven runs
- persisted outputs and offline HTML reports
- read-only API / SSE / Web visibility
- lightweight run history management

The goal is **not** to claim feature completeness or production hardening.

## Included Capabilities

This alpha includes:

- JSON config-driven runs
- single mode and pair mode execution
- minimal `SKILL.md` loading
- `stub` and OpenAI-compatible provider paths
- minimal tool calling with basic validation
- persisted output artifacts:
  - `report.json`
  - `report.html`
  - `events.jsonl`
  - `index.json`
- read-only HTTP API
- SSE live event streaming
- lightweight Web viewer
- run management commands for:
  - archive
  - delete
  - prune
  - rebuild index

## Recommended First Run

The most stable first trial is:

```bash
go run . --config configs/single.stub.json
```

This path avoids external provider dependencies and exercises the main new-architecture flow with the bundled examples.

## Output Artifacts

A successful run can produce:

- `report.json`
- `report.html`
- per-case `events.jsonl`
- root-level `index.json`

Archived runs are moved under:

- `<output-dir>/_archive/<run-id>/`

## API / Web / SSE Availability

This alpha includes a read-only server mode that can expose:

- `GET /healthz`
- `GET /api/runs`
- `GET /api/runs/{runID}`
- `GET /api/runs/{runID}/summary`
- `GET /api/runs/{runID}/cases/{caseID}/events`
- `GET /api/runs/{runID}/stream`

It also includes:

- a minimal Web viewer
- SSE live event streaming during active runs

## Run Management Support

This alpha includes CLI-only run management commands:

- rebuild index
- list runs
- archive runs
- delete runs
- prune runs

These management actions are explicit and support `--dry-run` where applicable.

## Known Limitations

Current known limitations include:

- JSON-only config support
- minimal `SKILL.md` parser
- minimal provider / tool schema support
- CLI-only management commands
- archive is implemented as directory move only
- prune is currently delete-only
- no database
- no advanced search
- no scheduling system

## Suggested Next Milestones

Natural follow-up milestones after this alpha include:

- archived runs read-only browsing
- experiment matrix runs
- richer tool schema support
- stronger provider robustness and hardening
- beta-level UX and operational polish
