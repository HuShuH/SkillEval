# Alpha Release Checklist

这是新架构 alpha 发布前的最小检查清单。

## 1. Build / Test

- `go test ./...`
- `go build ./...`

## 2. Recommended First Run Smoke Test

- `go run . --config configs/single.stub.json`
- 确认生成：
  - `report.json`
  - `report.html`
  - `events.jsonl`
  - `index.json`

## 3. Config Example Check

- `configs/single.stub.json`
- `configs/single.openai.json`
- `configs/pair.stub.json`
- `configs/pair.openai.json`

确认：

- 字段与 `eval/config.go` 对齐
- 路径引用真实存在
- OpenAI 示例优先使用 `api_key_env`

## 4. Skill Example Check

- `examples/skills/simple-writer/SKILL.md`
- `examples/skills/cautious-writer/SKILL.md`

确认：

- 可被当前 skill loader 加载
- 包含 `Instructions`
- 包含 `Tools`

## 5. Case Example Check

- `examples/cases/sample_single.json`
- `examples/cases/sample_pair.json`

确认：

- 可被 case loader 读取
- 结构与当前 `eval.Case` 对齐

## 6. API / Web / HTML Check

- `go run . --serve --output-dir <reports-dir>`
- 检查：
  - `GET /healthz`
  - `GET /api/runs`
  - `GET /api/runs/{runID}`
  - `GET /api/runs/{runID}/summary`
  - `GET /api/runs/{runID}/cases/{caseID}/events`
  - `GET /api/runs/{runID}/stream`
- 打开 `/`
- 打开生成的 `report.html`

## 7. Output Layout Check

确认 active run 输出中存在：

- `<output-dir>/<run-id>/report.json`
- `<output-dir>/<run-id>/report.html`
- `<output-dir>/<run-id>/cases/.../events.jsonl`
- `<output-dir>/index.json`

确认 archive 输出中存在：

- `<output-dir>/_archive/<run-id>/`

## 8. Run Management Check

- `--list-runs`
- `--rebuild-index`
- `--archive-runs ... --dry-run`
- `--delete-runs ... --dry-run`
- `--prune-keep N --prune-status all --dry-run`

确认：

- dry-run 不修改文件
- real run 管理后 `index.json` 会更新

## 9. Known Limits

- JSON-only config
- minimal `SKILL.md` parser
- no Web write operations
- archive is directory move only
- prune currently deletes, not archives
- no database or advanced search

## 10. Alpha Release Conclusion Template

```text
Alpha release check result:
- build/test: PASS
- config examples: PASS
- skill examples: PASS
- case examples: PASS
- API/Web/HTML smoke test: PASS
- run management smoke test: PASS
- known limits reviewed: YES

Conclusion:
Ready for alpha release.
```
