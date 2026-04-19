# Agent Skill Eval

`agent-skill-eval` 是一个正在演进中的 **Go-based A/B Skill Evaluation Framework**。  
仓库同时保留：

- **推荐的新架构入口**：根目录 `main.go`
- **legacy 兼容层**：`cmd/agent-eval` 与 `internal/*`

如果你是新用户，请优先使用根目录入口和本文档中的示例。

## Release Status

当前仓库更适合作为 **alpha** 版本交付：

- 新架构主链路已经可运行
- 文档、示例、配置文件、HTML report、API、SSE、run 管理命令已经打通
- 但仍保留若干“最小可用实现”与已知限制，不宜定义为 beta

## 当前能力概览

当前新架构已经具备以下最小可用能力：

- `SKILL.md` 最小格式解析与加载
- `stub` / `openai` provider
- 最小 tool schema 与 tool calling
- in-memory agent orchestrator
- single / pair evaluation
- checker、report 聚合、JSON 输出
- `report.json`、`report.html`、`events.jsonl`
- `index.json` 历史 run 索引
- 只读 HTTP API
- SSE 实时事件流
- 轻量 Web 页面
- 历史 run 管理命令：
  - rebuild index
  - list runs
  - archive runs
  - delete runs
  - prune runs

## Legacy 与新架构的边界

### 推荐入口

- `go run .`
- `go build ./...`

### Legacy 兼容层

- `cmd/agent-eval`
- `internal/*`

这些 legacy 代码仍然保留，但不再是新功能的推荐入口。

## 快速开始

## Recommended First Run

如果你只想先确认仓库是否可用，推荐先跑最稳妥的 single + stub 示例：

```bash
go run . --config configs/single.stub.json
```

### 1. 构建

```bash
go build ./...
```

### 2. 最快的 stub quick run

```bash
go run . --prompt "hello from the new framework"
```

### 3. 用配置文件跑 single stub

```bash
go run . --config configs/single.stub.json
```

### 4. 用配置文件跑 pair stub

```bash
go run . --config configs/pair.stub.json
```

## 配置文件运行

配置文件为 JSON，字段定义见 `eval/config.go`。

仓库已提供最小示例：

- `configs/single.stub.json`
- `configs/single.openai.json`
- `configs/pair.stub.json`
- `configs/pair.openai.json`

### 查看最终生效配置

```bash
go run . --config configs/single.stub.json --print-effective-config
```

CLI flags 优先级高于 config 文件。例如：

```bash
go run . --config configs/single.stub.json --max-iters 5
```

## Stub Provider 示例

```bash
go run . --config configs/single.stub.json
go run . --config configs/pair.stub.json
```

或直接用 flags：

```bash
go run . \
  --provider stub \
  --mode single \
  --cases examples/cases/sample_single.json \
  --skill-a examples/skills/simple-writer \
  --output-dir reports/manual-stub \
  --html-report
```

## OpenAI Provider 示例

推荐用环境变量，不要把 key 写死在命令里：

```bash
export OPENAI_API_KEY=your_key_here
go run . --config configs/single.openai.json
```

pair 示例：

```bash
export OPENAI_API_KEY=your_key_here
go run . --config configs/pair.openai.json
```

也可以用 flags：

```bash
go run . \
  --provider openai \
  --model gpt-4o-mini \
  --base-url https://api.openai.com/v1 \
  --cases examples/cases/sample_single.json \
  --skill-a examples/skills/simple-writer \
  --output-dir reports/openai-run \
  --html-report
```

## Single / Pair 模式

### Single

- 一个 agent
- 一个 skill
- 一组 cases
- 输出 `RunReport`

### Pair

- 同一个 case 分别跑 skill A 与 skill B
- 顺序执行，不做复杂并发
- 输出 `PairReport`

## HTML Report

开启：

```bash
go run . --config configs/single.stub.json --html-report
```

或配置文件里直接设置：

```json
{
  "output": {
    "html_report": true
  }
}
```

生成文件：

- `<output-root>/<run-id>/report.html`

这个 HTML 是离线静态页面，不依赖在线 API。

## API / SSE / Web

### 启动只读 API

```bash
go run . --serve --output-dir reports --listen :8080
```

### 主要 API

- `GET /healthz`
- `GET /api/runs`
- `GET /api/runs/{runID}`
- `GET /api/runs/{runID}/summary`
- `GET /api/runs/{runID}/cases/{caseID}/events`
- `GET /api/runs/{runID}/stream`

### Web 页面

启动 `--serve` 后，直接打开：

- `http://localhost:8080/`

页面支持：

- runs 列表
- report 查看
- case events 查看
- live SSE 查看

## Run 管理命令

这些命令不会启动新的 eval run。

### 重建索引

```bash
go run . --output-dir reports --rebuild-index
```

### 列出 runs

```bash
go run . --output-dir reports --list-runs --output-format json
```

### 归档 runs

```bash
go run . --output-dir reports --archive-runs run-1,run-2 --dry-run
go run . --output-dir reports --archive-runs run-1,run-2
```

### 删除 runs

```bash
go run . --output-dir reports --delete-runs run-1 --dry-run
go run . --output-dir reports --delete-runs run-1
```

### 保留最近 N 次

```bash
go run . --output-dir reports --prune-keep 20 --prune-status all --dry-run
go run . --output-dir reports --prune-keep 20 --prune-status all
```

可用状态：

- `all`
- `failed`
- `errored`
- `timed_out`
- `passed`

## 输出目录结构

本文统一使用：

- `output dir`：CLI 参数 `--output-dir`
- `output root`：概念上指同一个根目录
- `workspace root`：执行工具时的工作目录根

### Single

```text
<output-root>/<run-id>/
  report.json
  report.html
  cases/
    <case-id>/
      events.jsonl
```

### Pair

```text
<output-root>/<run-id>/
  report.json
  report.html
  cases/
    <case-id>/
      a/events.jsonl
      b/events.jsonl
```

### 根目录索引与归档

```text
<output-root>/
  index.json
  _archive/
```

## `report.json` / `report.html` / `events.jsonl` / `index.json`

- `report.json`
  - 机器可读的完整 run 结果
- `report.html`
  - 离线静态 HTML 报告
- `events.jsonl`
  - 每个 case 的结构化事件流
- `index.json`
  - 活跃 runs 的缓存索引

## SKILL.md 最小格式

最小支持格式：

```md
# simple_writer

Description: Minimal example skill
Version: 0.1.0

## Instructions
Do the task.
Return a concise answer.

## Tools
- filesystem
- finish
```

示例见：

- `examples/skills/simple-writer/SKILL.md`
- `examples/skills/cautious-writer/SKILL.md`

## 示例文件

### Configs

- `configs/single.stub.json`
- `configs/single.openai.json`
- `configs/pair.stub.json`
- `configs/pair.openai.json`

### Skills

- `examples/skills/simple-writer/SKILL.md`
- `examples/skills/cautious-writer/SKILL.md`

### Cases

- `examples/cases/sample_single.json`
- `examples/cases/sample_pair.json`

## 相关文档

- `docs/CLI_USAGE.md`
- `docs/CONFIG_REFERENCE.md`
- `docs/SKILL_FORMAT.md`
- `docs/OUTPUT_LAYOUT.md`
- `docs/API_REFERENCE.md`
- `docs/RELEASE_NOTES_ALPHA.md`
- `docs/BACKEND_ARCHITECTURE.md`
- `docs/design.md`
- `docs/IMPLEMENTATION_ROADMAP.md`
- `docs/frontend-design.md`

## 当前已知限制

- 不支持 YAML 配置
- `SKILL.md` 解析器仍是最小规则集，不是完整 Markdown AST
- provider/tool calling 仍以最小可用能力为主
- run 管理命令仅提供 CLI，不提供 Web 写操作
- archive 目前是目录移动，不做压缩
- prune 当前仅删除，不做“prune to archive”
- 没有引入数据库、复杂搜索或复杂调度系统

## 测试与校验

```bash
go test ./...
go build ./...
```
