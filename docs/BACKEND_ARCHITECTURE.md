# Agent Skill Eval — 后端架构设计方案

> 参考 design.md，结合当前 MVP 现状，给出完整的后端设计方案

---

## 1. 设计定位

本框架的核心目标：

> 给定两组 Skill（或"有 Skill vs 无 Skill"），对同一批 Case 并发执行，收集执行过程和产物，进行自动评分和人工确认，输出结构化报告。

### 当前 MVP vs 目标架构 差异对比

| 维度 | 当前 MVP | 目标架构 |
|------|----------|---------|
| 执行方式 | MockAdapter 返回硬编码结果 | 真实 Agent Loop（LLM + 工具调用） |
| 评测模式 | 单版本顺序执行 | A/B 并发评测（v1 vs v2，或 with vs without Skill） |
| Skill 格式 | JSON 元数据（name + description） | SKILL.md 文件（内容直接注入 system prompt） |
| 校验方式 | HardChecker 硬规则 | HardChecker + LLM Scorer |
| 执行记录 | 无步骤追踪 | 事件回调机制，持久化为 JSONL |
| 工作空间 | 无 | 每个 Case 每个 Agent 独立隔离 workspace |
| CLI 参数 | --skills-dir / --cases-file | --skill-a / --skill-b / --model / --api-key |

---

## 2. 整体架构

```
┌─────────────────────────────────────────────────────────────┐
│                     CLI (main.go)                           │
│  skill-eval cases.json                                      │
│    --skill-a ./skills/v1/SKILL.md                           │
│    --skill-b ./skills/v2/SKILL.md   (可省略，省略为无Skill)  │
│    --model glm-5 --api-key xxx                              │
└──────────────────────┬──────────────────────────────────────┘
                       │ 构造 Agent A / B，加载 Cases
                       ▼
┌─────────────────────────────────────────────────────────────┐
│                   eval.Runner                               │
│                                                             │
│  for each Case:                                             │
│    ├── 创建隔离 workspace: {outputDir}/{caseID}/a/          │
│    │                      {outputDir}/{caseID}/b/          │
│    ├── go orchestrator.Run(agentA, case, handlerA) ─┐      │
│    ├── go orchestrator.Run(agentB, case, handlerB) ─┘ 并发  │
│    ├── wait both → PairResult                               │
│    └── 收集产物文件路径                                      │
│                                                             │
│  所有 Case 完成后:                                          │
│    └── scorer.Score(pairResults) → ScoredResult            │
└────────┬───────────────────────────────────────────────────┘
         │
         ▼
┌─────────────────────────────────────────────────────────────┐
│                 agent.Orchestrator                          │
│                                                             │
│  Orchestrator.Run(agent, input, eventHandler):              │
│    ┌────────────────────────────────────────────────────┐   │
│    │ 1. 初始化 RunContext（State + EventHandler）        │   │
│    │ 2. 构造 messages = [system(skill)] + [user(prompt)]│   │
│    │ 3. 调用 LLM（ChatFunc）                            │   │
│    │    → 触发 EventLLMCall                             │   │
│    │ 4. 解析响应 tool_calls                             │   │
│    │ 5. 若含 finish tool → 终止，取 result              │   │
│    │ 6. 若无 tool_calls（纯文本） → 终止，取文本        │   │
│    │ 7. 执行工具（FileSystem / Bash / UseSkill 等）     │   │
│    │    → 触发 EventToolExec                            │   │
│    │ 8. 追加工具结果到 messages，继续循环               │   │
│    │ 9. 超出 max_iters 或 max_tokens → 强制终止         │   │
│    └────────────────────────────────────────────────────┘   │
│                                                             │
│  返回 RunResult{Output, StopReason, ToolCalls, Steps}       │
└────────┬───────────────────────────────────────────────────┘
         │
         ▼
┌─────────────────────────────────────────────────────────────┐
│                  eval.Scorer                                │
│                                                             │
│  LLM 评分模式：                                             │
│    ├── 读取 workspace A / B 的产物文件（输出内容）           │
│    ├── 构造对比 prompt → 调用 LLM 打分                      │
│    └── 返回 ScoredPairResult（A/B 得分 + 理由 + 胜负）      │
│                                                             │
│  HardCheck 模式（保留现有 checker）：                       │
│    └── 对 RunResult.Output 执行规则验证                     │
└────────┬───────────────────────────────────────────────────┘
         │
         ▼
┌─────────────────────────────────────────────────────────────┐
│                   report.Writer                             │
│  输出 report.json：                                         │
│    ├── 汇总统计（总数 / 胜负 / Pass率）                      │
│    ├── 每个 Case 的 PairResult（A/B RunResult 对比）        │
│    └── 每个 Case 的 ScoredResult（得分 + 理由）             │
└─────────────────────────────────────────────────────────────┘
```

---

## 3. 分层设计（核心）

### 3.1 三层结构：Agent / Orchestrator / RunContext

```
Agent（静态配置，可复用）
  ├── Model: 使用的 LLM 模型名
  ├── Skill: Skill 内容（SKILL.md 原文）
  ├── Tools: 可用工具列表
  └── MaxIters: 最大迭代次数

Orchestrator（无状态引擎）
  └── Run(agent, input, eventHandler) → RunResult

RunContext（单次运行状态）
  ├── Messages: 消息历史
  ├── ToolCalls: 本次运行记录的所有工具调用
  ├── Iteration: 当前迭代次数
  └── EventHandler: 事件回调（记录/推送）
```

设计理由：评测时对同一个 Agent 跑多个 Case，复用 Agent 配置，每次 Run 创建独立 RunContext，天然隔离。

### 3.2 事件机制：回调优于 Channel

```go
type EventHandler func(Event)

type Event struct {
    Type    EventType              // EventLLMCall / EventToolExec / EventFinish
    CaseID  string
    Data    map[string]interface{} // 事件具体数据
    Time    time.Time
}
```

优势：
- **简单**：不需要管理 chan 的创建、消费、关闭
- **灵活**：调用方自由决定处理方式（写日志、写文件、推 WebSocket）
- **可组合**：多个 handler 可以链式组合

```go
// 持久化到 JSONL
orchestrator.Run(agent, input, func(e Event) {
    switch e.Type {
    case EventLLMCall:
        store.SaveLLMCall(e.Data)
    case EventToolExec:
        store.SaveToolCall(e.Data)
    }
})

// 组合多个 handler
orchestrator.Run(agent, input, composeHandlers(
    jsonFileLogger("run_001.jsonl"),
    consoleLogger(),
    wsForwarder(wsConn), // 转发给前端 WebSocket
))
```

### 3.3 Agent Loop 终止策略

```
if 有 tool_calls:
    if 包含 finish tool → 完成，取 finish.result 作为最终输出
    else                → 执行工具，继续循环
else（纯文本回复）       → 也视为完成，取文本内容作为最终输出
```

终止原因类型：

```go
type StopReason string

const (
    StopFinish    StopReason = "finish"     // Agent 主动调用 finish
    StopMaxIters  StopReason = "max_iters"  // 超出最大迭代次数
    StopMaxTokens StopReason = "max_tokens" // 超出 token 限制
    StopError     StopReason = "error"      // 不可恢复错误
    StopTextReply StopReason = "text_reply" // 纯文本兜底
)
```

评测报告中会记录 StopReason，可以分析"是模型主动完成还是被截断"。

### 3.4 Workspace 隔离

A/B 评测必须使用独立工作目录，避免两个 Agent 互相干扰：

```
{outputDir}/
  {caseID}/
    a/        ← Agent A 的工作目录（绑定 FileSystem / Bash 工具）
    b/        ← Agent B 的工作目录
```

评分时分别读取 a/ 和 b/ 下的产物文件内容进行对比。

---

## 4. 数据结构设计

### 4.1 核心类型（对应 agent/types.go）

```go
// Agent 静态配置（不可变，评测时复用）
type Agent struct {
    Name     string
    Model    string
    Skill    *skill.Skill // nil 表示无 Skill（基准对照组）
    Tools    []tool.Tool
    MaxIters int
    ChatFunc ChatFunc     // LLM 调用函数
}

// ChatFunc 是 LLM 调用的函数签名（便于测试 mock）
type ChatFunc func(ctx context.Context, messages []openai.ChatCompletionMessageParamUnion) (*openai.ChatCompletion, error)

// RunResult 单次执行结果
type RunResult struct {
    Output     string          // 最终输出（finish.result 或纯文本）
    StopReason StopReason      // 终止原因
    ToolCalls  []ToolCallRecord // 所有工具调用记录（含输入 + 输出）
    Iters      int             // 实际迭代次数
    DurationMS int64
    Error      string
}

// ToolCallRecord 工具调用的完整记录（输入 + 输出聚合）
type ToolCallRecord struct {
    ID         string                 // OpenAI tool_call_id
    ToolName   string
    Args       map[string]interface{}
    Result     string
    Error      string
    Iter       int   // 第几轮迭代触发
    DurationMS int64
}
```

### 4.2 评测层类型（对应 eval/）

```go
// Case 一条评测用例
type Case struct {
    CaseID         string            `json:"case_id"`
    Prompt         string            `json:"prompt"`
    AllowedTools   []string          `json:"allowed_tools"`
    HardChecks     HardChecks        `json:"hard_checks,omitempty"` // 保留现有 HardChecker
    TimeoutSeconds int               `json:"timeout_seconds"`
    // 注意：不再需要 skill 字段，skill 由 Agent 配置携带
}

// EvalPair 一次 A/B 评测的配置
type EvalPair struct {
    AgentA *agent.Agent
    AgentB *agent.Agent // nil 时自动创建无 Skill 对照组
}

// PairResult 一个 Case 的 A/B 对比结果
type PairResult struct {
    CaseID     string
    Prompt     string
    ResultA    agent.RunResult
    ResultB    agent.RunResult
    WorkspaceA string // A 的产物目录路径
    WorkspaceB string // B 的产物目录路径
    HardChecks *HardCheckResult // 硬规则校验结果（可选）
}

// HardCheckResult 硬规则校验结果
type HardCheckResult struct {
    PassedA bool
    PassedB bool
    ReasonsA []string
    ReasonsB []string
}

// ScoredPairResult LLM 评分结果
type ScoredPairResult struct {
    PairResult
    ScoreA   int    // 0-10 分
    ScoreB   int    // 0-10 分
    Winner   string // "a" / "b" / "tie"
    Reason   string // LLM 给出的评分理由
}

// EvalReport 最终报告
type EvalReport struct {
    GeneratedAt string
    SkillA      string // Skill A 的名称/路径
    SkillB      string // Skill B 的名称/路径（空=无Skill）
    Model       string
    Total       int
    AWins       int
    BWins       int
    Ties        int
    AvgScoreA   float64
    AvgScoreB   float64
    Results     []ScoredPairResult
}
```

### 4.3 Skill 类型（对应 skill/skill.go）

```go
// Skill 从 SKILL.md 加载
type Skill struct {
    Name        string // 从目录名或文件名推断
    Content     string // SKILL.md 的完整内容（直接注入 system prompt）
    SourcePath  string // SKILL.md 文件路径
    Version     string // 可选版本标记（从文件名/目录名解析）
}

// Load 从 SKILL.md 文件加载
func Load(path string) (*Skill, error)

// LoadDir 从目录加载（找 SKILL.md）
func LoadDir(dir string) (*Skill, error)
```

---

## 5. 工具清单

| Tool | 文件 | 职责 |
|------|------|------|
| **FileSystem** | `tool/filesystem.go` | 文件读写、编辑、目录列表（绑定 workspace） |
| **Bash** | `tool/bash.go` | Shell 命令执行（以 workspace 为 cwd） |
| **Finish** | `tool/finish.go` | Agent 主动交付结果 + 产物文件列表 |
| **UseSkill** | `tool/use_skill.go` | 从 Skill Registry 加载技能内容 |

关于 CodeExec：不单独实现，模型通过 Bash 自行执行 `python3 xxx.py` / `node xxx.js`，这与 Claude Code 的做法一致。

---

## 6. 模块结构

```
skill-eval/
├── cmd/
│   └── agent-eval/             # 现有 CLI（保留，逐步迁移）
│       ├── main.go
│       ├── load.go
│       └── output.go
│
├── agent/                      # 【新增】Agent 执行引擎
│   ├── types.go                # Agent, RunResult, StopReason, ToolCallRecord
│   ├── run_context.go          # RunContext, State, Event, EventHandler
│   └── orchestrator.go         # Orchestrator, ChatFunc, Agent Loop
│
├── eval/                       # 【新增】评测层
│   ├── case.go                 # Case 定义与 JSON/JSONL 加载
│   ├── runner.go               # EvalPair, PairResult, Runner（A/B 并发）
│   └── scorer.go               # LLM 对比评分
│
├── providers/                  # 【新增】LLM 提供商
│   └── openai.go               # OpenAIProvider（返回 SDK 原生类型）
│
├── skill/                      # 【新增】Skill 加载
│   └── skill.go                # Skill 定义、SKILL.md 解析
│
├── tool/                       # 【新增】工具实现
│   ├── types.go                # Tool 接口
│   ├── filesystem.go
│   ├── bash.go
│   ├── finish.go
│   └── use_skill.go
│
├── internal/                   # 【保留】现有 MVP 模块（过渡期保留）
│   ├── spec/types.go           # 现有数据结构（新结构稳定后可废弃）
│   ├── registry/               # 现有 Skill 注册表（被 skill/ 替代）
│   ├── adapters/               # MockAdapter（保留用于测试）
│   ├── checker/                # HardChecker（保留，集成进 eval 层）
│   ├── runner/                 # 现有 Runner（被 eval.Runner 替代）
│   ├── report/                 # 现有 Report（被 eval 层替代）
│   └── validate/               # 配置校验（保留）
│
├── testdata/
│   ├── cases/
│   │   └── mvp.jsonl           # 现有测试用例（格式兼容）
│   └── skills/
│       ├── echo.json           # 现有 JSON 格式 Skill（过渡期保留）
│       └── v1/SKILL.md         # 【新增】SKILL.md 格式
│       └── v2/SKILL.md         # 【新增】对比用的 v2
│
├── docs/
│   ├── design.md               # 原始设计文档
│   ├── BACKEND_ARCHITECTURE.md # 本文档
│   └── IMPLEMENTATION_ROADMAP.md
│
├── go.mod
└── main.go                     # 新 CLI 入口（未来替代 cmd/agent-eval）
```

---

## 7. 运行流程

```
1. CLI 解析参数
   --skill-a ./skills/v1/SKILL.md
   --skill-b ./skills/v2/SKILL.md   (省略则 Agent B = 无 Skill 对照组)
   --model glm-5  --api-key xxx
   --output ./eval-output
   cases.json

2. 加载 Skill A 和 Skill B（读取 SKILL.md 内容）

3. 构造 Agent A（绑定 Skill A + Tools + ChatFunc）
   构造 Agent B（绑定 Skill B 或 nil + Tools + ChatFunc）

4. 加载评测 Cases（从 JSON/JSONL 文件）

5. 创建 eval.Runner
   - 配置事件回调（JSONL 持久化到 {outputDir}/events.jsonl）
   - 配置 HardChecker（可选）

6. for each Case：
   a. 创建 workspace A：{outputDir}/{caseID}/a/
   b. 创建 workspace B：{outputDir}/{caseID}/b/
   c. goroutine: orchestrator.Run(agentA, case.Prompt, handlerA)
   d. goroutine: orchestrator.Run(agentB, case.Prompt, handlerB)
   e. 等待两者完成 → PairResult
   f. 运行 HardChecker（可选）

7. 所有 Case 完成后，LLM 评分：
   - 读取 workspaceA / workspaceB 下的产物文件
   - 调用 Scorer，得到每条 Case 的 ScoredPairResult

8. 生成报告：
   - 汇总统计：总数 / A胜 / B胜 / 平局 / 平均分
   - 写入 {outputDir}/report.json
```

---

## 8. CLI 用法（目标）

```bash
# A/B 版本对比
skill-eval cases.json \
  --skill-a ./skills/pdf-v1/SKILL.md \
  --skill-b ./skills/pdf-v2/SKILL.md \
  --model glm-5 \
  --base-url https://open.bigmodel.cn/api/paas/v4 \
  --api-key <your-key> \
  --max-iters 10 \
  --output ./eval-output

# 有 Skill vs 无 Skill 对比（省略 --skill-b）
skill-eval cases.json \
  --skill-a ./skills/my-skill/SKILL.md \
  --model glm-5 \
  --api-key <your-key> \
  --output ./eval-output

# 只做 HardCheck，不用 LLM 评分
skill-eval cases.json \
  --skill-a ./skills/v1/SKILL.md \
  --model glm-5 \
  --api-key <your-key> \
  --no-llm-score \
  --output ./eval-output
```

---

## 9. 报告格式（目标）

```json
{
  "generated_at": "2026-04-17T10:00:00Z",
  "skill_a": "skills/pdf-v1/SKILL.md",
  "skill_b": "skills/pdf-v2/SKILL.md",
  "model": "glm-5",
  "total": 10,
  "a_wins": 3,
  "b_wins": 6,
  "ties": 1,
  "avg_score_a": 6.2,
  "avg_score_b": 8.1,
  "results": [
    {
      "case_id": "case_001",
      "prompt": "帮我把这份 PDF 转成结构化 Markdown",
      "result_a": {
        "output": "...",
        "stop_reason": "finish",
        "iters": 4,
        "duration_ms": 3200,
        "tool_calls": [
          {
            "tool_name": "FileSystem",
            "args": {"op": "read", "path": "input.pdf"},
            "result": "...",
            "iter": 1,
            "duration_ms": 120
          }
        ]
      },
      "result_b": { "..." },
      "score_a": 6,
      "score_b": 9,
      "winner": "b",
      "reason": "B 的输出格式更清晰，标题层级正确，A 丢失了部分表格内容"
    }
  ]
}
```

---

## 10. 实施优先级

### Phase 1 — 保留并稳定现有 MVP（不改动，先用它跑通 HardCheck 流程）

当前 MVP 的以下模块**保持不变**，作为过渡期的稳定基础：
- `internal/spec/types.go`
- `internal/registry/`
- `internal/adapters/` (MockAdapter)
- `internal/checker/`（HardChecker 保留，后续复用）
- `internal/validate/`
- `cmd/agent-eval/`（现有 CLI）

### Phase 2 — 新增 skill/ 和 tool/ 模块

| 步骤 | 文件 | 工作量 |
|------|------|--------|
| 2.1 | `skill/skill.go` — Skill 加载（SKILL.md） | 2h |
| 2.2 | `tool/types.go` — Tool 接口定义 | 1h |
| 2.3 | `tool/finish.go` — Finish 工具 | 1h |
| 2.4 | `tool/filesystem.go` — 文件操作工具 | 3h |
| 2.5 | `tool/bash.go` — Bash 执行工具 | 2h |

### Phase 3 — 实现 agent/ 模块（核心引擎）

| 步骤 | 文件 | 工作量 |
|------|------|--------|
| 3.1 | `agent/types.go` — Agent, RunResult, StopReason | 2h |
| 3.2 | `agent/run_context.go` — RunContext, Event | 2h |
| 3.3 | `providers/openai.go` — ChatFunc 实现 | 2h |
| 3.4 | `agent/orchestrator.go` — Agent Loop | 5h |

### Phase 4 — 实现 eval/ 模块（评测层）

| 步骤 | 文件 | 工作量 |
|------|------|--------|
| 4.1 | `eval/case.go` — Case 加载（兼容现有 JSONL 格式）| 1h |
| 4.2 | `eval/runner.go` — A/B 并发执行 | 4h |
| 4.3 | `eval/scorer.go` — LLM 评分 | 3h |

### Phase 5 — 新 CLI 入口

| 步骤 | 文件 | 工作量 |
|------|------|--------|
| 5.1 | `main.go` — 新 CLI 入口（替代 cmd/agent-eval） | 2h |
| 5.2 | report 输出（report.json） | 1h |

### Phase 6 — API Server（前端对接）

| 步骤 | 内容 | 工作量 |
|------|------|--------|
| 6.1 | REST API（提交评测、查询进度、获取报告） | 5h |
| 6.2 | WebSocket 推送（事件实时转发） | 3h |
| 6.3 | 存储层（评测历史持久化） | 3h |

---

## 11. 与现有代码的关系

```
现有代码                  新代码                     关系
─────────────────────────────────────────────────────────
internal/spec/types.go   agent/types.go             新结构替代，旧的过渡期保留
internal/registry/       skill/skill.go             SKILL.md 格式替代 JSON
internal/adapters/mock   providers/openai.go        真实 LLM 替代 Mock
internal/checker/        eval/runner.go 中调用       复用，集成进评测层
internal/runner/         eval/runner.go             新的 A/B 并发 Runner 替代
internal/report/         eval/scorer.go + report    扩展为含评分的报告
cmd/agent-eval/          main.go                    新 CLI 入口（逐步替代）
```

过渡策略：**新旧代码共存**，不破坏现有 MVP 功能，逐步将新模块接入，稳定后再清理旧代码。
