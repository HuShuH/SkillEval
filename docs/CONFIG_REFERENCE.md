# Config Reference

Run configuration is JSON-only and is defined by `eval/config.go`.

## Top-Level

```json
{
  "mode": "single",
  "cases": "examples/cases/sample_single.json",
  "prompt": "",
  "provider": {},
  "execution": {},
  "output": {},
  "skills": {}
}
```

## Fields

### `mode`

- `single`
- `pair`

### `cases`

Path to a JSON or JSONL case file.

### `prompt`

Optional quick-run prompt when you do not want to use a case file.

### `provider`

- `name`: `stub` or `openai`
- `model`
- `base_url`
- `api_key_env`
- `api_key`
- `provider_timeout`

Recommended: prefer `api_key_env` over `api_key`.

### `execution`

- `timeout`
- `max_retries`
- `retry_backoff_ms`
- `max_iters`
- `workspace_root`

### `output`

- `output_dir`
- `run_id`
- `html_report`

### `skills`

- `skill_a`
- `skill_b`

`skill_b` is required in `pair` mode.

## Override Rules

- Config file values load first
- Explicit CLI flags override config values
- Unset flags do not clear config values
