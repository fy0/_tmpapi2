# Sub2API Fork 逐版本升级记录（2026-11-02）

## 升级边界

- 原始仓库：D:\codes\2026\sub2api，main 基线 96d0e5241（v0.2.13）。
- 独立 worktree：D:\codes\2026\sub2api-upgrade-v0219。
- 分支：upgrade/sub2api-v0.2.19-20261102。
- 官方上游：Wei-Shaw/sub2api，目标标签 v0.2.19。
- 原始 main 和未跟踪文件 nul 未改动；未推送远程或部署生产服务。

## 逐版合并

| 版本 | 合并提交 | 处理内容 |
| --- | --- | --- |
| v0.2.14 | d129ec489 | 无冲突 |
| v0.2.15 | 8c8616ecf | 协调测试、Fork 自定义提示词 |
| v0.2.16 | a9b9c5ccd | 保留图片生成重定向及分组国际化 |
| v0.2.17 | e87602704 | 无冲突 |
| v0.2.18 | 685ee8f0d | 保留完全信任模式并兼容上游配额判断 |
| v0.2.19 | bfc2aaf98 | 整合管理员密钥、图片计费、Fork 首页与缓存字段 |

补充提交：
- 026a4e5bf：Fork VERSION 改为 0.2.19。
- 035e7a220：修复 Ent 字段索引、管理员 Grok 诊断接口及路由测试。
- 23a404d3c：修复图片密钥响应分组本地化。

## 本地验证

- Go 工具链 go1.27.3。
- 后端所有包编译：go test ./... -run ^$ -count=1，通过。
- 后端全部测试：go test ./... -count=1 -p 4 -timeout 12m，通过。
- 前端 frozen lockfile 安装、pnpm run typecheck、pnpm run build（包含 i18n）均通过。
- 前端 pnpm exec vitest run --maxWorkers=3 --minWorkers=1：380 个测试文件和 3048 个测试用例全部通过。
- Windows 上运行仓库测试时把 C:\Program Files\Git\bin 放入 PATH 以提供 sh.exe，且取消 HTTP_PROXY / HTTPS_PROXY，以免 SDK 本地连接测试误判。
- 前端生产构建仍有非阻断性的 Browserslist、Vite 模块分包和大型 chunk 警告。

日志（保存在 worktree 外部）：
- D:\codes\2026\sub2api-upgrade-v0219-go-compile-final.log
- D:\codes\2026\sub2api-upgrade-v0219-go-tests-final.log
- D:\codes\2026\sub2api-upgrade-v0219-vitest-final.log

## Windows 可执行程序

- 路径：D:\codes\2026\sub2api-upgrade-v0219-build\sub2api.exe
- 构建：go build -o 目标路径 ./cmd/server
- 体积：164340224 字节。
- SHA-256：1C8CF72AC78611F4DA1AC6AFEAE6738FB7B5CEF602E68E16B5431738FC878D54。
- 服务端已嵌入新构建的前端资源。

## 数据库上线门禁

本次新增 8 个 SQL 迁移文件：
- 249_drop_platform_check_constraints.sql
- 250_group_i18n.sql
- 251_gemini_cached_contents.sql
- 252_gemini_cached_contents_channel_usage.sql
- 253_gemini_cached_contents_mutation_lease.sql
- 254_enable_gemini_image_generation_groups.sql
- 255_scoped_admin_api_keys.sql
- 256_group_image_count_billing.sql

Fork 已存在 241 及此前的迁移，未覆盖旧的 SQL 内容和校验和。迁移器通过 filename 与 SHA-256 跟踪历史迁移，禁止将 Fork 历史迁移重新编号。

**尚未做真实 PostgreSQL 升级演练。** 当前运行环境没有 Docker / PostgreSQL / psql 命令或数据库连接。本次验收覆盖代码编译、单元测试和前端构建，不能替代真实数据库迁移及生产网关、支付与业务数据的验证。

部署前必须先备份 PostgreSQL、Redis、配置与密钥，并用生产备份的隔离副本完整执行历史迁移，核对 schema_migrations、工单、发票、图片密钥、主页链接、管理员权限及新增图片计费。在模拟回滚后，再评估正式部署。数据库结构可能前进，不能假定旧二进制直接回退就能恢复全部功能。
## 2026-11-02 迁移序列补正

本次初版升级曾直接沿用上游 242–248 的文件名，这不符合 Fork 后续迁移独立编号的维护要求。

已核对 Fork 0.2.13 共有 298 个 SQL 文件，上游同版本有 291 个，差额为 Fork 自定义的 7 个 SQL 文件。Fork 曾将 154–195 区间中部分迁移顺延至 161–212，而双方在 213–241 的文件名再次趋同。为确保后续演进不直接复用上游迁移编号，本次把 8 个新增迁移分配到 249–256（原上游 242–248 前缀留空），未来新迁移从 **257** 起按 Fork 序列顺延，不应直接使用上游文件名。所有这 8 个 SQL 文件都只改名，字节及 SHA-256 与初次合入的上游版本完全一致。

迁移执行器增加了严格的旧文件名兼容表：若某数据库在初次未顺延版本中已经运行相同 SQL，且 schema_migrations 的旧文件名 checksum 与当前嵌入 SQL 一致，则仅添加保留 applied_at 的新文件名元数据记录，不重跑 SQL，也不删除旧记录；遇到未知 checksum、读写错误会终止并告警。未运行过旧文件名的数据库直接执行 249–256。此兼容逻辑经单元测试覆盖，但仍须在真实 PostgreSQL 备份副本上演练。

严禁改动已经发布并可能执行过的 Fork 旧迁移 001–241。部署前需要备份并确认历史 schema_migrations（尤其 242–248 的旧文件名）及 PostgreSQL 实际结构、数据；如发现不一致，停止部署并人工处理。
## 发布工件安全更正（P0）

本次对 AGENTS.md、note.md、README_CN.md、README.md、DEV_GUIDE.md、deploy/README.md 均增加强制数据库迁移编号与上线阻断提示，并解除根 AGENTS.md 的 Git 忽略以使它随版本传播。

先前生成的未顺延 Windows 工件（SHA-256 为 5ECD1F3C847A898AAD9DBAC67E5F0340C5D15DD0DABD39D0D0253595ADF179F6）已移至 D:\codes\2026\sub2api-upgrade-v0219-build\sub2api.UNSAFE_UNSHIFTED_DO_NOT_DEPLOY.blocked（仅作事故分析保留，不可部署）。

目前 D:\codes\2026\sub2api-upgrade-v0219-build\sub2api.exe 已重新构建，SHA-256 为 1C8CF72AC78611F4DA1AC6AFEAE6738FB7B5CEF602E68E16B5431738FC878D54，包含 Fork 249–256 迁移及 checksum 严格别名兼容。但此工件仍仅用于代码与测试验收，生产 PostgreSQL 升级模拟/回滚验证尚缺失，部署依然 BLOCKED。