# 实施路线图

## Phase 1: 核心数据结构和基础模块（Week 1）

### Step 1.1: 扩展 `spec/types.go` - 执行追踪支持
- 新增 `ExecutionPhase` 结构
- 新增 `ToolInvocation` 结构
- 新增 `ExecutionTrace` 结构
- 扩展 `AgentOutput` 添加 `ExecutionTrace` 字段
- 新增 `ComparisonMetrics` 结构

**时间**：2-3 小时
**输出**：更新的 `spec/types.go`，所有新类型

---

### Step 1.2: 实现 Tracer 模块
- 创建 `internal/tracer/tracer.go`
- 实现 `Tracer` 接口
- 实现 `DefaultTracer` 
- 添加 `generateTraceID()` 辅助函数
- 编写单元测试

**时间**：2-3 小时
**输出**：可工作的 Tracer 模块，带单测

**依赖**：Step 1.1 ✅

---

### Step 1.3: 实现 BaselineAdapter
- 创建 `internal/adapters/baseline.go`
- 实现 `BaselineAdapter` 结构
- 预定义低准确率的返回值
- 测试与 MockAdapter 的兼容性

**时间**：1-2 小时
**输出**：`baseline.go`，验证编译通过

**依赖**：无

---

### Step 1.4: 扩展 MockAdapter 支持追踪
- 修改 `internal/adapters/mock.go`
- 添加 `Tracer` 字段到 `MockAdapter`
- 在各个 skill case 中集成 tracer 调用
- 确保向后兼容（Tracer 为 nil 时）
- 添加单元测试

**时间**：2 小时
**输出**：升级的 `mock.go`，兼容旧代码

**依赖**：Step 1.2 ✅

---

### Step 1.5: 本地验证编译
```bash
go build ./...
go test ./...
```

**时间**：30 分钟
**输出**：项目成功编译，所有单测通过

---

### ✅ Phase 1 完成标志
- 所有新数据结构定义完成
- Tracer 模块可工作
- BaselineAdapter 可用
- MockAdapter 支持追踪
- 项目编译通过，单测通过

---

## Phase 2: 多版本对比执行引擎（Week 2）

### Step 2.1: 新增 `MultiScenarioRunner`
- 创建 `internal/runner/multi_scenario.go`
- 实现 `MultiScenarioRunner` 结构
- 实现 `RegisterScenario()` 方法
- 实现 `RunAllScenarios()` 方法（并行执行）
- 添加单元测试

**时间**：3-4 小时
**输出**：`multi_scenario.go`，支持并行执行

**依赖**：Step 1.4 ✅

---

### Step 2.2: 实现对比指标计算
- 在 `internal/runner/runner.go` 中新增 `ComputeMetrics()` 函数
- 实现场景间差异计算
- 实现改进指标计算逻辑
- 添加单元测试

**时间**：2-3 小时
**输出**：对比指标计算逻辑验证

**依赖**：Step 2.1 ✅

---

### Step 2.3: 扩展 `internal/report/` 支持多场景汇总
- 修改 `report.go` 以支持多场景结果
- 新增 `SummarizeMultiScenario()` 函数
- 新增 `MergeReports()` 函数用于合并多场景报告
- 为每个场景生成独立的 `SummaryItem`
- 添加单元测试

**时间**：2-3 小时
**输出**：升级的 `report.go`，支持多场景统计

**依赖**：Step 2.1 ✅

---

### Step 2.4: 扩展 Registry 支持版本管理
- 修改 `internal/registry/registry.go`
- 添加版本索引能力
- 实现 `GetVersions(skillName)` 方法
- 实现版本查询逻辑
- 添加单元测试

**时间**：2 小时
**输出**：升级的 `registry.go`

**依赖**：无

---

### Step 2.5: 集成测试 - 完整的多版本执行流
- 创建集成测试用例
- 验证 MultiScenarioRunner 端到端
- 验证对比指标计算正确性
- 验证报告汇总正确性

**时间**：2 小时
**输出**：集成测试通过

**依赖**：Step 2.1-2.4 ✅

---

### Step 2.6: 本地验证编译
```bash
go build ./...
go test ./...
```

**时间**：30 分钟
**输出**：项目编译通过，所有测试通过

---

### ✅ Phase 2 完成标志
- MultiScenarioRunner 支持多版本并行执行
- 对比指标计算正确
- 报告多场景汇总工作
- Registry 支持版本管理
- 完整集成测试通过

---

## Phase 3: 存储层实现（Week 2-3）

### Step 3.1: 实现 Storage 接口
- 创建 `internal/storage/storage.go`
- 定义 `Storage` 接口
- 实现 `FileStorage` 基础版本
- 支持 save/get/list/delete 操作
- 添加单元测试

**时间**：3 小时
**输出**：`storage.go`，FileStorage 实现

**依赖**：Step 1.1 ✅

---

### Step 3.2: 集成 Storage 到 CLI
- 修改 `cmd/agent-eval/main.go`
- 添加 `--storage-path` 参数
- 执行完成后自动保存到存储
- 支持从存储加载历史结果
- 测试存储的读写

**时间**：2 小时
**输出**：CLI 支持持久化存储

**依赖**：Step 3.1 ✅

---

### Step 3.3: 本地验证
```bash
go build ./...
go test ./...
go run ./cmd/agent-eval run --storage-path ./test_storage
```

**时间**：30 分钟
**输出**：CLI 可成功保存和读取结果

---

### ✅ Phase 3 完成标志
- Storage 层工作正常
- 结果可持久化
- CLI 集成存储成功

---

## Phase 4: REST API 服务（Week 3-4）

### Step 4.1: 创建 API 服务框架
- 创建 `cmd/eval-server/main.go`
- 创建 `internal/api/server.go`
- 初始化 HTTP 服务
- 添加基础中间件（日志、CORS）
- 能够启动并监听端口

**时间**：2-3 小时
**输出**：可启动的 API 服务器框架

**依赖**：Step 3.1 ✅

---

### Step 4.2: 实现核心 REST 端点
- `POST /api/v1/evaluations` - 创建评测
- `GET /api/v1/evaluations` - 列表
- `GET /api/v1/evaluations/:id` - 获取进度/结果
- `GET /api/v1/evaluations/:id/report` - 获取报告
- `DELETE /api/v1/evaluations/:id` - 删除评测

**时间**：4-5 小时
**输出**：所有 REST 端点实现

**依赖**：Step 4.1 ✅, Step 2.1 ✅

---

### Step 4.3: 实现后台评测任务队列
- 创建 `internal/api/evaluationservice.go`
- 实现异步评测任务执行
- 支持多个并发评测任务
- 跟踪任务进度和状态
- 支持任务的暂停/继续（可选）

**时间**：3-4 小时
**输出**：任务队列系统工作

**依赖**：Step 4.2 ✅

---

### Step 4.4: 实现 WebSocket 推送
- 创建 `internal/api/ws_handler.go`
- 实现 `WS /api/v1/evaluations/:id/stream`
- 推送进度事件（progress）
- 推送完成事件（case_completed, completed）
- 处理客户端连接/断开

**时间**：3 小时
**输出**：WebSocket 推送工作

**依赖**：Step 4.3 ✅

---

### Step 4.5: 集成 API 服务测试
- 编写 API 端点集成测试
- 测试 HTTP 请求/响应
- 测试 WebSocket 推送
- 测试错误处理

**时间**：3 小时
**输出**：API 集成测试通过

**依赖**：Step 4.1-4.4 ✅

---

### Step 4.6: 本地启动测试
```bash
go build -o bin/eval-server ./cmd/eval-server

# Terminal 1
./bin/eval-server --port 8080

# Terminal 2
curl -X POST http://localhost:8080/api/v1/evaluations -d '...'
curl http://localhost:8080/api/v1/evaluations
```

**时间**：1 小时
**输出**：API 服务可本地运行，端点可访问

---

### ✅ Phase 4 完成标志
- API 服务框架完成
- 所有 REST 端点可用
- 后台任务队列工作
- WebSocket 推送工作
- API 集成测试通过

---

## Phase 5: CLI 集成优化（Week 4）

### Step 5.1: 为 CLI 添加新命令
- `agent-eval run-multi` - 多版本评测
- `agent-eval list` - 列出历史评测
- `agent-eval compare <id1> <id2>` - 对比两次评测

**时间**：2 小时
**输出**：新 CLI 命令

**依赖**：Phase 2, 3 ✅

---

### Step 5.2: 本地测试完整流程
```bash
# 单版本评测
go run ./cmd/agent-eval run

# 多版本评测（需要多个 skill 版本）
go run ./cmd/agent-eval run-multi

# 启动 API 服务
go run ./cmd/eval-server

# 通过 API 提交评测
curl -X POST http://localhost:8080/api/v1/evaluations ...
```

**时间**：2 小时
**输出**：完整流程可运行

**依赖**：Phase 1-4 ✅

---

### ✅ Phase 5 完成标志
- CLI 命令齐全
- 支持多版本评测
- 支持历史查询和对比
- 整个后端系统可用

---

## 总体时间估算

| Phase | 任务数 | 工作量 | 预计天数 |
|-------|--------|--------|---------|
| Phase 1 | 5 | 10-15h | 2-3 |
| Phase 2 | 6 | 15-20h | 3-4 |
| Phase 3 | 3 | 5-8h | 1-2 |
| Phase 4 | 6 | 18-24h | 3-4 |
| Phase 5 | 2 | 4h | 1 |
| **总计** | **22** | **52-71h** | **10-14 天** |

---

## Git 检查点计划

在完成每个 Phase 后创建检查点：

```bash
# Phase 1 完成
git add -A
git commit -m "feat: extend data structures and add tracer module (Phase 1)"

# Phase 2 完成
git commit -m "feat: implement multi-scenario runner and comparison metrics (Phase 2)"

# Phase 3 完成
git commit -m "feat: implement storage layer (Phase 3)"

# Phase 4 完成
git commit -m "feat: implement REST API and WebSocket service (Phase 4)"

# Phase 5 完成
git commit -m "feat: enhance CLI with new commands (Phase 5)"
```

---

## 建议优先级和依赖关系

```
Phase 1 (必须)
    ├─-> Phase 2 (必须)
    │      ├─-> Phase 4 (关键)
    │      └─-> Phase 5 (增强)
    ├─-> Phase 3 (必须)
    │      └─-> Phase 4 (关键)
    └─-> Phase 4 (关键)
            └─-> 前端开发准备完毕
```

**建议执行顺序**：Phase 1 → Phase 2 → Phase 3 → Phase 4 → Phase 5

---

## 里程碑检查

### ✅ 里程碑 1：数据结构完成
- [ ] spec/types.go 所有新类型定义完成
- [ ] 项目编译通过
- [ ] 相关单测通过

### ✅ 里程碑 2：多版本执行完成
- [ ] MultiScenarioRunner 实现完成
- [ ] 对比指标计算完成
- [ ] 集成测试通过
- [ ] CLI 可执行多版本评测

### ✅ 里程碑 3：存储和 API 完成
- [ ] Storage 层实现
- [ ] API 服务可启动
- [ ] 所有 REST 端点可用
- [ ] WebSocket 推送工作

### ✅ 里程碑 4：系统可用
- [ ] 完整后端系统运行
- [ ] CLI 和 API 并存
- [ ] 所有功能测试通过
- [ ] 文档更新完成

---

## 前提条件检查

- [ ] Go 1.22+ 已安装
- [ ] 项目在 main 分支且工作树干净
- [ ] 已备份原始代码
- [ ] 清楚项目的 AGENTS.md 规范

---

## 建议从这里开始

现在可以开始 **Phase 1, Step 1.1**：

1. 打开 `internal/spec/types.go`
2. 逐步添加新的数据结构
3. 验证编译通过
4. 逐步推进到后续 Step

需要帮助吗？告诉我哪一步开始！