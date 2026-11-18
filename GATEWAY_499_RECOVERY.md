# 499 自动恢复与会话关联：实现及验收记录

日期：2026-11-10。目标仓库：`D:\codes\2026\sub2api`。
需求来源：`D:\codes\2026\sub2api-helper\tmp\sub2api-499-recovery-prompt.md`。


## 2026-11-19 升级后的回归状态

v0.2.23 升级保留本次业务功能，并修复用量 drain 生命周期、恢复代理归属、WS 模型健康信号、SSE 读取批次和 compact 模型回退。与既定取消/统一预算/handler 终态所有权冲突的旧断言已逐项更新，新增终态 JSON/SSE 客户端交付测试。远程完整 unit 与容器 PostgreSQL integration 已通过（首次共同通过的提交为 `435c8a4b4`），不再依赖本文件旧记录中的测试 overlay；最终证据见 `UPGRADE_V0.2.23_2026-11-19.md`。

以下 2026-11-10 内容为当时的历史记录，其中“未推送”和旧测试失败不代表本次升级最终状态。原始 WS passthrough 未接入统一恢复、首输出超时协同和真实客户端端到端验证仍是既存限制；不得据此宣称所有 WS 模式已覆盖。生产数据库快照升级/回滚演练仍未进行，数据库上线未放行。

## 交付状态：实现中，尚未完成验收，禁止据此部署

本次本地提交记录当前实现。主要 HTTP/WS 恢复代码、日志字段、管理界面和专项测试已写入，但仍存在服务层回归失败，并且原始 WebSocket passthrough 模式尚未接入统一恢复策略。验收未完成；未推送、未部署、未重启服务、未执行数据库迁移。

任务开始时工作区已有其他改动，禁止 reset、清理未跟踪文件或整体覆盖。既有的未跟踪文件 `nul` 原样保留。数据库既有上线门禁不因单元测试通过而解除。

## 已写入的实现

### 恢复入口

- `backend/internal/service/openai_gateway_recovery.go`：HTTP 统一决策、原请求重放、退避、预算、响应前缀缓冲、资源关闭及尝试记录。已接入原生 Responses、HTTP 自动透传、Chat Completions 兼容及 WS→HTTP bridge。
- `backend/internal/service/openai_ws_recovery.go`、`openai_ws_forwarder_v2.go`、`openai_ws_forwarder_ingress.go`：原生 WS v2 / ctx_pool 每轮恢复，复用统一预算，不额外重复准入或计费钩子。
- 默认最多额外 2 次，退避约 250 ms / 750 ms 加少量抖动。池模式显式配置的重试次数优先，包括 0。预算不允许再由外层切换循环重复放大。恢复连接随后遇到未分类传输异常时终止，不重新启动另一套重试。
- 默认覆盖真实 HTTP 499、502/503/504/524、连接重置、意外 EOF、HTTP/2 流错误和有明确瞬态信号的流内失败。真实客户端取消立即停止；鉴权、策略、上下文和模型等确定性错误不默认重试；无显式 retry 规则的 429 保留原有容量处理。
- 同账号重试复用已准备好的请求体及头，不改模型、会话键、`prompt_cache_key` 或 `previous_response_id`。已经输出文本、可见推理、工具调用或参数后不重放；依赖原连接状态的续链保守拒绝重放。
- 失败尝试的协议前缀不混入成功尝试；目前前缀上限 64 KiB / 30 秒。普通 HTTP 未提交响应时恢复耗尽返回 503，适用时携带 `Retry-After: 1`；已提交流必须按对应协议终止。
- 非 OpenAI 平台复用该 Service 的调用保留原提供商处理，不把本次 OpenAI 默认策略强行套到 Grok 等平台。

### 规则动作

规则增加 `action`，按现有优先级第一条命中决定动作，仅匹配原始故障。支持空正文按状态码匹配。旧规则按原有状态码/正文透传组合映射，不能升级后突然开始重试。

| 场景 | action | 结果 |
| --- | --- | --- |
| 无规则，或只有无关规则 | 无 | 已接入的安全路径对瞬态故障执行默认恢复 |
| 状态码 499，明确保留上游错误 | `passthrough` | 不自动重试，按协议保留状态/安全错误内容 |
| 状态码 499，改写为 503 | `rewrite` | 直接按配置结束，不对改写出的 503 再匹配重试 |
| 状态码 499，自动恢复 | `retry` | 复用统一安全检查和预算，不能绕过取消或有效输出边界 |

这些规则说明适用于已接入路径；不代表尚未完成适配的原始 WS passthrough 模式已经具有相同语义。

### 会话关联和计费观察

`gateway_request_observation.go` 保存原始显式 conversation/session 标识与来源；只有缓存键时明确标记 fallback，未知标识保持空。下游 request ID、跨请求 conversation ID、内部 attempt 分开。日志不保存密钥和完整对话。

错误与用量持久化、管理 API、用量/错误页面接入关联字段；页面支持复制、精确筛选和尝试明细。同会话跳转同时携带用户与 API Key 范围。普通用户 DTO 不暴露内部账号/规则尝试明细。

失败尝试的上游成本未知时标记 unknown，不能当作零成本；成功路径仍走原 usage 计费。缓存读写量用于排障比较，不足以证明上游换号或物理缓存重建。

## 与“同账号重试状态码”的关系

旧设置位于 `account.go` 的池模式配置，HTTP 自动透传错误路径会检查它。对 API Key 池账号，把真实 HTTP 499 加入列表且重试次数非零，在该路径可以触发同账号重试。

但原生 Responses 旧调用链的前置故障判断不包含 499，所以仅修改该列表不能统一所有入口。连接异常没有可匹配的实际 HTTP 状态；HTTP 200 中的 `response.failed` 也不等于 HTTP 499。现有列表本身不提供本需求中的流前缀隔离、跨层统一预算或会话日志关联。

同一个 Sub2API 账号可能只是同一个黑箱池端点，不能推断其后真实账号一定不变。503 或重试成功不保证客户端显示 reconnecting；本任务未在目标客户端环境验证自动重连。

## 已执行验证

- 前端：`pnpm run typecheck`、`pnpm run check:i18n`、涉及的页面/API/组件 ESLint 通过。
- 前端相关行为测试：3 个文件共 22 项通过，覆盖会话单元格、UsageView 和语言键完整性。
- 后端新增核心和实际入口测试覆盖 499 空/HTML/JSON 正文、三种动作、优先级、无关规则、耗尽、取消、工具输出边界、请求一致性和失败元数据丢弃。真实入口测试使用本地模拟上游，不是生产探测。
- 一轮相关包完整单元测试：21,076 个 pass、37 个 fail、23 个 skip 事件（包含子测试，父测试失败也计入）。model、handler、dto、admin、repository、migrations 包通过，service 包未通过。这不是整个后端 `./...` 通过。
- 该轮日志：`C:\Users\test\AppData\Local\Temp\sub2api-499-final-regression-5g_5okly.jsonl`。随后已修复旧处理链重复 Ops 事件和非 OpenAI 平台行为串扰，后续定向重跑结果另记，不能把该轮失败隐去。

### 最新定向重跑（修复重复事件和恢复所有权之后）

日志：`C:\Users\test\AppData\Local\Temp\sub2api-499-focused-final-1vg2ecpp.jsonl`。

结果：391 个 pass、28 个 fail 事件（含父测试与子测试）。其中本次新增的 `TestGatewayRecovery*` / `TestGatewayObservation*` 共 62 个 pass，0 个 fail。Grok 内容策略、compact 重试事件、代理归属、旧 transport 故障记录、Anthropic CC 兼容错误路径的若干回归已恢复通过。该定向重跑不能代替完整 service 包重新通过。

剩余失败集中在：旧测试期待请求结束后不取消上游 context、忽略客户端取消、瞬态错误交回外层 failover、响应失败规则直接写出、WS bridge 的旧错误返回形态、WS 连接上限/容量故障及健康信号。部分与需求要求的新语义冲突，部分仍需修复或澄清；没有把它们一概当作环境问题或简单删除测试。

此前完整运行中的模型目录分流测试在此次定向运行通过，但完整运行的失败原因未确认。该轮验证结束时 `git diff --check` 通过，暂存区为空，测试基于 `4b73a9b2c37496d7dcefb2ded899d56f17f8948f` 与当时的工作区改动。

### 测试环境限制

`backend/internal/service/response_model_billing_test.go` 的 `TestToUsageFields_ResponseModelSourcePassesThrough` 在当前文件和 HEAD 都重复定义两次，这是已核实的既有编译阻断。没有改动该原文件；测试使用 Go `-overlay` 在临时副本仅重命名第二个测试，两个测试体均保留。

Overlay：`C:\Users\test\AppData\Local\Temp\sub2api-499-test-overlay-q6vfmr21\overlay.json`。
Go：本机缓存工具链 `go1.27.3`，`GOMAXPROCS=2 GOGC=50 GOTOOLCHAIN=local`，`-p 1 -tags unit`。Windows 测试子进程 PATH 增加 Git 的 `sh.exe`，仅对子进程去除代理环境变量。

因此，不能声称未加 overlay 的标准全量测试已通过，也没有进行真实 PostgreSQL 集成演练、完整生产构建、竞态检查或目标客户端端到端验证。

## 剩余验收项

1. `openai_ws_v2_passthrough_adapter.go` 的原始双向透传尚未适配：必须保持客户端取消、重叠轮次保护和终止帧写出生命周期，不能简单重跑整个 relay 或重放已完成轮次。
2. 服务层遗留测试需要逐项区分真正回归与需求变更，包括取消不再忽略、耗尽不再交给外层嵌套重试、流内规则在 handler 前返回已决策错误、WS 容量/连接上限路径等。不得仅为通过测试弱化安全断言。模型协议目录分流失败的原因尚未确认。
3. 首输出缓冲等待上限与现有首 token / 高推理超时配置的协同需要进一步验证；连续失败后健康信号、冷却及分组切换策略也尚需完整验收。
4. 重新执行完整相关包回归、专项流式/取消测试、必要的竞态检查及最终差异审查，完成后才能确认代码验收通过。

## 迁移兼容与上线结论

按当前上游 v0.2.19 最高编号 248 与本批之前 Fork 最高编号 256 核对，新增：

- `257_error_passthrough_rule_action.sql`：新增 action，并兼容旧规则；
- `258_gateway_conversation_observation.sql`：usage/error 的可空关联字段和 JSONB 元数据，不伪造历史 ID；
- `259_gateway_conversation_indexes_notx.sql`：非事务并发构建包含 conversation/user/API key/time 的部分索引，支持中断后的无效索引清理重试。

未修改或重编号历史 SQL。未来批次必须重新核对上游和 Fork 的占用编号，不能把 260 当作永久约定的下次起点。`note.md`、`FORK_SEQUENCE.md` 和升级记录已同步。

**上线结论：BLOCKED。** 未检查生产 `schema_migrations` / checksum / 实际表结构，未备份生产数据库，未在隔离 PostgreSQL 上升级和回滚演练。既有程序和旧 hash 不代表包含这些工作区变更；禁止自动替换生产程序或重启服务。
