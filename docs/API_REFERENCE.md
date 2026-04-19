# API Reference

The built-in server is read-only and serves persisted outputs plus live SSE streams.

## `GET /healthz`

Returns:

```json
{ "ok": true }
```

## `GET /api/runs`

Returns active runs from `index.json` or a rebuilt index.

Query params:

- `mode=single|pair`
- `status=failed|errored|timed_out|passed`
- `limit=N`

## `GET /api/runs/{runID}`

Returns the raw `report.json` content for one run.

## `GET /api/runs/{runID}/summary`

Returns a lightweight summary view with:

- `run_id`
- `report_id`
- `created_at`
- `total_cases`
- `mode`
- `summary`
- `metadata`

## `GET /api/runs/{runID}/cases/{caseID}/events`

Returns parsed event objects from `events.jsonl`.

Pair mode supports:

- `?side=a`
- `?side=b`

## `GET /api/runs/{runID}/stream`

SSE endpoint for live events while a run is active.

Response headers:

- `Content-Type: text/event-stream`
- `Cache-Control: no-cache`
- `Connection: keep-alive`
