# Sub2API Fork v0.2.23 升级记录（2026-11-19）

## 范围与基线

- 起点：本地 main `1c9e81eb2`，含 Fork v0.2.19 升级及 499 恢复/会话关联。
- 工作区：`D:/codes/2026/sub2api-upgrade-v0223`。
- 分支：`upgrade/sub2api-v0.2.23-20261119`。
- 目标：Wei-Shaw/sub2api `v0.2.23`（`5af9afb58`）及其紧随的纯 VERSION 同步提交 `2af3f1e15`。标签内 VERSION 仍为 0.2.22，后者仅将它同步到 0.2.23。
- 原始 main 的未跟踪文件 `nul` 保留。用户已授权完成后推送 Fork 远程 main，由 Actions 构建镜像；未授权生产数据库变更或部署。

## 逐版本合并

| 版本 | 合并提交 | 兼容处理 |
| --- | --- | --- |
| v0.2.20 | 55e005b46 | MCP OAuth 接入，保留首页链接、发票、工单、图片分组重定向与分组侧栏 |
| v0.2.21 | a5437ecc7 | 缓存 30m/12h 计费字段与会话观察字段的 SQL 列、参数、扫描顺序统一；保留完全信任模式 |
| v0.2.22 | fe591f672 | Ultrafast 分组字段贯穿 API/认证缓存/Ent，错误提取器兼容 Fork 规则动作 |
| v0.2.23 | b3bdfe710 | Ultrafast 渠道定价迁移、上游网关修复 |
| VERSION 同步 | 1c9b5fac1 | 合并上游 2af3f1e15，仅同步版本文件 |

合并中间提交不作为独立发布版本。迁移分配以整批目标的最终词典序为准。

## 迁移审查

逐文件比较 v0.2.19、v0.2.20、v0.2.21、v0.2.22、v0.2.23 的完整 SQL 集合：目标上游最高前缀为 252，当前 Fork 最高已占用前缀为 259，故本批在两者之后分配 260–265。这是本批计算结果，不是未来起点约定。

上游两组同号文件均独立分配 Fork 编号。v0.2.21 才引入的 cost 索引在最终上游词典序中位于 MCP 之前，因此 Fork cost 索引使用 260，MCP 使用 261；两者无相互依赖。

| 上游文件名 | Fork 文件名 | 依赖与执行方式 |
| --- | --- | --- |
| 249_add_usage_logs_api_key_cost_index_notx.sql | 260_add_usage_logs_api_key_cost_index_notx.sql | usage_logs；事务外并发建索引，保留中断后精确名称的无效索引清理 |
| 249_mcp_oauth.sql | 261_mcp_oauth.sql | users；先 grants，再 codes/credentials，含外键与索引 |
| 250_openai_cache_write_ttl.sql | 262_openai_cache_write_ttl.sql | usage_logs 与 081/082/101/106 创建的四张渠道价格表；增加默认值/可空字段 |
| 251_group_force_openai_ultrafast.sql | 263_group_force_openai_ultrafast.sql | groups；默认 FALSE |
| 251_group_usage_rollup_nonblocking_build.sql | 264_group_usage_rollup_nonblocking_build.sql | 222/223 的汇总状态、函数和触发器；事务内添加状态字段、替换函数，lock_timeout=5s |
| 252_channel_pricing_ultrafast_multiplier.sql | 265_channel_pricing_ultrafast_multiplier.sql | channel_model_pricing；可空倍率及正值 CHECK |

新增 SQL 与上游 Git blob **逐字节相同**。原有 309 个 SQL（001–259，含重号及字母后缀）文件名、SQL 字节与 `1c9e81eb2` 相同；没有修改旧 checksum、旧名兼容白名单或旧记录。迁移执行器、迁移单元测试、仓储集成测试和 MCP 文档中的新文件引用均已改为 Fork 名称。新增顺序回归同时锁定本批字节 SHA-256 和完整历史的聚合 SHA-256。

| Fork 编号 | 文件字节 SHA-256 | schema_migrations checksum |
| --- | --- | --- |
| 260 | 789d4d5a5c186531cb6edd65ea2ecc07f5e488272f7876c6f46240e2fed0c30b | 94f124c3e50405f7b75746885111a4f662c331985a697de46fe946a17ff1f7d9 |
| 261 | 881ff2e8ecf430202c593c699b1441db1d2196ac2c312d8a3849acb6c97609e5 | 751a1dcee6108afcecbd580b6e3140cca3480853fecd2174861b02f3a71e54a1 |
| 262 | c470116c49042425f5663c421f0aea2a6aa6eae75b7e489dc95c12fc303562b2 | 532551ce7aee86b2ee0a78d397d422424ceed4051e31d5f542277c80c21b8b98 |
| 263 | f91352f2d3b1bffbbea5f6b7fc17e37c5ad64f04c3426cb7168778d10135a7bf | d78e605d4fc2038c57f620dff398022823a850ea8994b129b50774ac49e20af1 |
| 264 | da504646104cec14aa7bcf5a0af6f06f5cd40883459379ae915db27c1d4cec8b | 5b6e9b57f9b354a96c6870185c034db8adc7def944efcddf654e39cd97770718 |
| 265 | 01eb249a933e5c3bef3a8e2106098abf56d20a5f2890fcc6490b1f83ec05687f | 458752b3523254ee60ab4a821261ab64df9216a6f208c1960e7b816f8c7cfd1d |

运行器对 SQL 使用 `strings.TrimSpace` 后再计算 SHA-256，因此数据库 checksum 与原始文件字节 SHA-256 分列保存。不得以本表代替生产实查。

## 验证记录

日志位于仓库外部 `D:/codes/2026/sub2api-upgrade-v0223-*`。使用 Go 1.27.4、Node 24.4.0、pnpm 10.14.0；Go 测试子进程提供 Git 的 sh.exe 并去除代理环境变量。

- frozen lockfile 安装、前端生产构建（含 vue-tsc/i18n）、最终 ESLint 通过。
- 最终全量 Vitest：393 文件、3189 用例全部通过。首轮发现的 OpsErrorLogTable 测试缺少子组件隔离、上游 Ultrafast 提示文案断言过期已修复；会话组件独立行为测试保留并通过。
- `go test ./migrations -count=1` 通过，覆盖原有顺序/别名相关 SQL 约束和新增顺序/历史字节回归。
- Ent 完整重新生成完成（为避免依赖解析网络等待使用本机缓存与 GOPROXY=off）：仅修正 mutation 容量和 15 处 Group 默认值索引，消除初始化 panic；Wire 重新生成无差异。生成时补充的无关 go.sum 元数据已撤销。
- `go test -tags=integration ./... -run '^$' -count=1 -p 1` 所有包编译通过；本机缺 Docker，仓储 TestMain 跳过真实容器数据库测试，不能算 PostgreSQL 集成验收。
- 全量 unit 首轮完成：service 包尚有三处新 prompt 参数测试编译问题（已修复）；sysutil 的一个符号链接用例因 Windows 缺权限失败，随后在 Linux 子系统运行整个包全部通过；其余包通过。
- 用户随后要求停止本地运行、改在远程验证。已终止本地 service 重跑；终止前仍发现 compact 回退、请求生命周期、流式刷新次数、错误规则及恢复/failover 等回归，不能声称后端单元测试通过。保留原断言，在远程 Actions 复测并核对。远程结果将在完成后补齐。

Fork 基线已有两个同名、同体的 response billing 测试，合并后仍然存在，删除重复副本后不再依赖 overlay（上游目标只有一个副本）。另适配上游新增 UsageBillingRepository 构造参数、WS 搜索重连 payload 参数，并清除自动合并的 integrationDSN 重复声明。

额外跨平台检查限制：macOS apple-container 测试在 Git Bash 下的 `stat -f %Lp` 不兼容；release helper 在 Windows 下的 Bash 路径不兼容，在 WSL 中又缺 Python 3.11 的 `hashlib.file_digest`（CI 指定 Python 3.12）。没有为这些本机限制改动上游部署/发布脚本，后续由 Actions 原生环境验证。

## 数据库上线门禁与回退

**数据库上线未放行（BLOCKED）**。尚未读取生产 PostgreSQL 的 schema_migrations / 实际结构，未执行生产备份，未取得真实快照并在隔离 PostgreSQL 中按历史升级与回滚演练。没有执行任何生产 SQL、服务重启或部署。Actions 镜像构建/发布成功也不解除此门禁。

部署前须完整备份 PostgreSQL、Redis、配置和密钥，核对旧 242–248 别名及严格 checksum；在生产快照隔离副本逐步升级，确认 257–259 既有业务迁移与 260–265 本批迁移的实际应用记录、结构和数据。特别验证并发索引中断恢复、MCP 授权/撤销、缓存 TTL 计费、Ultrafast 与分组汇总失效/重建并发行为。遇到未知 checksum、部分执行或对象不一致，应停止并人工核对，不修改历史记录。

回退须以经过演练的数据库/Redis/配置/密钥完整备份恢复为依据，明确停写和数据恢复点；禁止把旧二进制替换视作数据库回滚，也不在生产盲目删除新增表、列或函数。演练证据齐全并取得明确部署确认后才能上线。
