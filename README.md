# Agent Skill Eval Go

一个使用 Go 实现的、最小可运行但具备基本工程化能力的 Agent Skill 评测框架 MVP。

这个项目提供了一套小型评测框架，用来加载 agent skill、读取测试用例、执行确定性的 mock 行为、进行硬规则校验、校验配置合法性，并输出结构化评测报告。

---

## 项目目标

这个仓库是一个 **Go 版 Agent Skill Evaluation Framework 的 MVP**。

它试图解决一个非常直接的问题：

> 给定一组 agent skill 和一组测试用例，能否以可控方式运行它们、验证结果、发现配置问题，并生成报告？

当前版本仍然保持轻量，但已经具备了比较完整的命令行评测工具雏形。

---

## 当前 MVP 范围

当前项目已经包含以下组件：

- skill registry
- testcase loader
- mock adapter
- hard checker
- sequential runner
- report 模块
- CLI `run` 命令
- CLI `validate` 命令
- 文本 / JSON 双输出模式
- 原始 skill 文件校验
- 基础单元测试
- 基础 CLI 测试

---

## 当前版本已经能做什么

当前 MVP 已经可以：

- 从 JSON 文件加载 skill 定义
- 从 JSONL 文件加载 testcase
- 在执行前校验 skills 和 testcases 的一致性
- 检查原始 skill JSON 文件是否合法
- 通过确定性的 mock adapter 执行 skill 行为
- 使用硬规则对输出结果进行验证
- 单条 case 失败时不中断整个批次
- 生成机器可读的 JSON 报告
- 通过 CLI 运行完整评测流程
- 通过 `--json` 输出机器可读的 CLI 结果

---

## 当前版本还不能做什么

当前 MVP **刻意没有实现**这些能力：

- 多 agent 编排
- LLM judge
- web search
- memory 系统
- sandbox 隔离
- 分布式执行
- 云端部署
- GUI 或 Web 前端
- 真实模型 adapter
- 并发执行

---

## 项目结构

```text
.
├── AGENTS.md
├── README.md
├── cmd/
│   └── agent-eval/
│       ├── load.go
│       ├── main.go
│       ├── main_test.go
│       └── output.go
├── internal/
│   ├── adapters/
│   │   ├── adapter.go
│   │   └── mock.go
│   ├── checker/
│   │   ├── checker.go
│   │   └── checker_test.go
│   ├── registry/
│   │   ├── registry.go
│   │   └── registry_test.go
│   ├── report/
│   │   ├── report.go
│   │   └── report_test.go
│   ├── runner/
│   │   ├── runner.go
│   │   └── runner_test.go
│   ├── spec/
│   │   └── types.go
│   └── validate/
│       ├── validate.go
│       └── validate_test.go
├── reports/
├── testdata/
│   ├── cases/
│   │   └── mvp.jsonl
│   └── skills/
│       ├── echo.json
│       ├── hello_world.json
│       └── mock_tool_call.json
└── go.mod
```

---

## 核心概念

### Skill

Skill 是由 registry 从 JSON 文件中加载的定义。
当前 MVP 中，skill 元数据很简单，主要包含：

- `name`
- `description`

示例：

```json
{
  "name": "hello_world",
  "description": "用于 MVP registry 加载的最小示例 skill。"
}
```

### TestCase

TestCase 表示一条评测用例，用来描述：

- 要执行哪个 skill
- 输入 prompt 是什么
- 允许哪些工具
- 要做哪些 hard checks
- 超时时间是多少

示例：

```json
{
  "case_id": "case_hello_world",
  "prompt": "say hello",
  "allowed_tools": [],
  "skill": { "name": "hello_world" },
  "hard_checks": {
    "expected_output": "hello world"
  },
  "timeout_seconds": 3
}
```

### HardChecks

HardChecks 是确定性的规则校验。
当前 MVP 支持：

- `expected_output`
- `expected_tool_name`
- `expected_args`

示例：

```json
{
  "expected_tool_name": "mock_tool",
  "expected_args": {
    "value": "ok"
  }
}
```

### Adapter

Adapter 是执行层，负责根据 testcase 和 skill 产出执行结果。

当前 MVP 使用的是 `MockAdapter`，它提供固定、可预测的行为：

- `hello_world`
- `echo`
- `mock_tool_call`

这样可以在不接真实模型 API 的前提下，把整个评测链路跑通。

### AgentOutput

AgentOutput 表示一次执行后的输出结果。
当前主要包含：

- `final_output`
- `tool_calls`
- `error`

示例：

```json
{
  "final_output": "hello world"
}
```

或者：

```json
{
  "tool_calls": [
    {
      "tool_name": "mock_tool",
      "args": {
        "value": "ok"
      }
    }
  ]
}
```

### RunResult

RunResult 表示单条 testcase 跑完并校验后的结果。
当前主要包含：

- `case_id`
- `skill`
- `agent_output`
- `passed`
- `reasons`
- `error`
- `duration_ms`

### ReportSummary

ReportSummary 是整个批次运行结束后的汇总结果。
当前主要包含：

- `total`
- `passed`
- `failed`
- `pass_rate`
- `generated_at`
- `results`

### Validate

`validate` 用于在正式执行之前检查配置和数据质量。
当前会检查：

- 原始 skill JSON 文件是否合法
- skill name 是否为空
- skill name 是否重复
- `case_id` 是否为空
- `case_id` 是否重复
- testcase 引用的 skill 是否存在
- `expected_args` 是否在缺少 `expected_tool_name` 的情况下被错误使用

### Runner

Runner 负责把这些模块串起来：

- 读取 testcase
- 从 registry 中解析 skill
- 调用 adapter 执行
- 调用 checker 校验
- 产出 `RunResult`

当前 runner 是串行执行的。

### Report

Report 模块负责：

- 汇总所有运行结果
- 统计通过/失败数量
- 计算 `pass_rate`
- 写入 `generated_at`
- 将 JSON 报告写入磁盘

---

## 当前已实现的示例 skill

### `hello_world`

行为：

- 返回 `hello world`

### `echo`

行为：

- 直接返回 testcase 中的 `prompt` 作为最终输出

### `mock_tool_call`

行为：

- 产生一条工具调用记录：
  - tool name: `mock_tool`
  - args: `{"value":"ok"}`

---

## 当前已实现的示例 testcase

当前 MVP JSONL 测试集包含 3 条 case：

- `case_hello_world`
- `case_echo`
- `case_mock_tool_call`

文件位置：

- `testdata/cases/mvp.jsonl`

内容示例：

```jsonl
{"case_id":"case_hello_world","prompt":"say hello","allowed_tools":[],"skill":{"name":"hello_world"},"hard_checks":{"expected_output":"hello world"},"timeout_seconds":3}
{"case_id":"case_echo","prompt":"echo this text","allowed_tools":[],"skill":{"name":"echo"},"hard_checks":{"expected_output":"echo this text"},"timeout_seconds":3}
{"case_id":"case_mock_tool_call","prompt":"call the mock tool","allowed_tools":["mock_tool"],"skill":{"name":"mock_tool_call"},"hard_checks":{"expected_tool_name":"mock_tool","expected_args":{"value":"ok"}},"timeout_seconds":3}
```

---

## 执行流程

当前 MVP 的执行流程如下：

1. 从 skills 目录加载 skill 定义
2. 从 JSONL 文件加载 testcase
3. 可选地先执行 `validate` 检查配置合法性
4. 使用 `MockAdapter` 执行 skill
5. 使用 `HardChecker` 校验输出
6. 为每条 case 生成 `RunResult`
7. 汇总所有结果为 `ReportSummary`
8. 将结果写入 report JSON 文件

---

## CLI 用法

### 运行完整评测流程

```bash
go run ./cmd/agent-eval run
```

默认参数：

- `--skills-dir ./testdata/skills`
- `--cases-file ./testdata/cases/mvp.jsonl`
- `--out ./reports/run.json`

### 运行评测并输出 JSON 摘要

```bash
go run ./cmd/agent-eval run --json
```

### 执行配置校验

```bash
go run ./cmd/agent-eval validate
```

### 执行配置校验并输出 JSON

```bash
go run ./cmd/agent-eval validate --json
```

### 指定路径运行

```bash
go run ./cmd/agent-eval run \
  --skills-dir ./testdata/skills \
  --cases-file ./testdata/cases/mvp.jsonl \
  --out ./reports/run.json
```

### 指定路径校验

```bash
go run ./cmd/agent-eval validate \
  --skills-dir ./testdata/skills \
  --cases-file ./testdata/cases/mvp.jsonl
```

---

## CLI 输出示例

### `run` 文本模式

```text
total: 3
passed: 3
failed: 0
report: ./reports/run.json
```

### `run --json`

```json
{
  "ok": true,
  "total": 3,
  "passed": 3,
  "failed": 0,
  "report_path": "./reports/run.json"
}
```

### `validate` 文本模式

```text
skills loaded: 3
testcases loaded: 3
validation: ok
```

### `validate --json`

```json
{
  "ok": true,
  "skills_loaded": 3,
  "testcases_loaded": 3,
  "errors": []
}
```

---

## 报告示例

执行：

```bash
go run ./cmd/agent-eval run
```

会生成：

- `reports/run.json`

示例结构：

```json
{
  "total": 3,
  "passed": 3,
  "failed": 0,
  "pass_rate": 1,
  "generated_at": "2026-04-15T13:57:14Z",
  "results": [
    {
      "case_id": "case_hello_world",
      "skill": {
        "name": "hello_world"
      },
      "agent_output": {
        "final_output": "hello world"
      },
      "passed": true,
      "reasons": [
        "expected_output matched: \"hello world\""
      ],
      "duration_ms": 0
    }
  ]
}
```

---

## 当前开发过程

这个项目是按增量方式一步一步做出来的。

目前已经完成的阶段包括：

- 初始化项目骨架
- 定义核心数据结构
- 实现 skill registry
- 实现 mock adapter
- 实现 testcase loader
- 实现 hard checker
- 实现 sequential runner
- 实现 report 模块
- 接通 CLI `run`
- 添加最小单元测试
- 补齐缺失的 MVP skill fixture
- 增加 `validate` 子命令
- 增加 CLI 测试
- 补齐 `validate` 失败路径测试
- 抽离 `validate` 逻辑到 `internal/validate`
- 增强 skill 文件原始校验
- 为 `validate` 增加 `--json`
- 为 `run` 增加 `--json`
- 抽离 CLI 输出辅助逻辑
- 抽离 CLI 共享加载逻辑
- 增强 report summary，增加 `pass_rate` 和 `generated_at`

---

## 构建与测试

### 构建

```bash
go build ./...
```

### 测试

```bash
go test ./...
```

### 运行评测

```bash
go run ./cmd/agent-eval run
```

### 执行校验

```bash
go run ./cmd/agent-eval validate
```

---

## 当前测试覆盖情况

当前已经有基础测试的模块：

- `cmd/agent-eval`
- `internal/checker`
- `internal/registry`
- `internal/report`
- `internal/runner`
- `internal/validate`

当前还没有单测的包：

- `internal/adapters`
- `internal/spec`

---

## 设计原则

当前项目遵循这些原则：

- 保持 MVP 小而可运行
- 优先使用标准库
- 模块结构清晰、容易理解
- 避免过早抽象
- 采用增量开发
- 每一步都保持本地可编译
- 评测逻辑尽量确定性、可复现
- CLI 输出同时兼顾人读和机器读

---

## 后续可能的演进方向

在当前 MVP 之上，比较合理的后续方向包括：

- 给 CLI JSON 输出增加 `generated_at`
- 扩展 report 的聚合统计
- 增强 hard checks 能力
- 增加并发执行
- 接入真实模型 adapter
- 增加可选 judge 评测能力

---

## 当前状态

当前状态：**MVP 已完成并可运行，且具备基础工程化能力**。

已确认可用：

```bash
go build ./...
go test ./...
go run ./cmd/agent-eval run
go run ./cmd/agent-eval run --json
go run ./cmd/agent-eval validate
go run ./cmd/agent-eval validate --json
```
