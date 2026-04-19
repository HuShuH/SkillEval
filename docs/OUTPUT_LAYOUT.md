# Output Layout

The new architecture writes results under one output root.

## Active Runs

Single mode:

```text
<output-root>/<run-id>/
  report.json
  report.html
  cases/
    <case-id>/
      events.jsonl
```

Pair mode:

```text
<output-root>/<run-id>/
  report.json
  report.html
  cases/
    <case-id>/
      a/
        events.jsonl
      b/
        events.jsonl
```

## Root-Level Files

```text
<output-root>/
  index.json
```

## Archive Layout

Archived runs are moved to:

```text
<output-root>/_archive/<run-id>/
```

Archived runs keep their internal files unchanged.

## File Meanings

- `report.json`: machine-readable run result
- `report.html`: offline static HTML summary
- `events.jsonl`: per-case structured events
- `index.json`: cached active run index
