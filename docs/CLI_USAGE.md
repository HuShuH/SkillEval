# CLI Usage

This repository now has two CLI surfaces:

- New architecture entrypoint: root `main.go`
- Legacy compatibility entrypoint: `cmd/agent-eval`

Use the new architecture entrypoint for all examples below.

## Build

```bash
go build ./...
```

## Quick Run

Stub provider:

```bash
go run . --prompt "hello from the new framework"
```

Single run with config:

```bash
go run . --config configs/single.stub.json
```

Pair run with config:

```bash
go run . --config configs/pair.stub.json
```

## Useful Flags

- `--config`
- `--print-effective-config`
- `--mode`
- `--cases`
- `--prompt`
- `--skill-a`
- `--skill-b`
- `--provider`
- `--model`
- `--base-url`
- `--api-key`
- `--timeout`
- `--provider-timeout`
- `--max-retries`
- `--retry-backoff-ms`
- `--output-dir`
- `--run-id`
- `--html-report`
- `--serve`
- `--stream`

## Management Commands

These commands do not start a new evaluation run:

```bash
go run . --output-dir reports --list-runs --output-format json
go run . --output-dir reports --rebuild-index
go run . --output-dir reports --archive-runs run-1,run-2 --dry-run
go run . --output-dir reports --delete-runs run-1 --dry-run
go run . --output-dir reports --prune-keep 20 --prune-status all --dry-run
```
