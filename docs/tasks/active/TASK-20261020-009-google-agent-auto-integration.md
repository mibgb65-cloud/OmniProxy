# TASK-20261020-009: Google 模型自动接入 AI Agent

## Status

In Progress

## Created

2026-10-20

## Request

用户反馈 Google 账号模型目前不会自动接入，希望自动将 Google 的模型提供给不同 AI Agent，并核查如何实现。

## Goal

查明账号模型发现、网关目录和各客户端配置之间的缺口，在现有边界内实现可验证的自动接入，保留用户已有客户端配置。

## Scope

- 核查 Google 模型发现和手动同步的实际行为。
- 核查不同 AI Agent 的协议、目录和配置能力。
- 修复可自动完成的模型目录／接入同步，并增加针对性测试。

## Out of Scope

- 不替用户登录或读取真实凭据。
- 不未经许可接管未受 OmniProxy 管理的客户端，不更换仍可用的用户默认模型。
- 不修改工作区已有的其他任务改动。

# Analysis

## Root Cause

- `GoogleSubscriptionModels` 只读取官方选择器，不保存网关的 `account_models`；此前只有手动 `ModelSync` 完成目录持久化和受管客户端刷新。
- `agentconfig.RefreshCatalog` 已具备协议适配、备份修订保护和显式模型限选，无需重写客户端配置层。

## Decision

采用单个可取消、合并通知的后台同步器，在应用启动、Google 授权状态变化和成功读取模型后触发；每账号有超时，失败保留旧快照。复用现有同步事务和 `RefreshCatalog`，不新增依赖或数据库迁移。

不采用前端挂载即重复提交 mutation：那样离开账号页面就不自动同步，而且容易因缓存失效产生循环。首次接入仍需用户确认，显式限选不扩张，运行中的客户端可能需要重新加载。

## Risks

- 已有其他任务的未提交改动；只作必要的独立增量，不提交既有改动。
- 官方接口失败或授权并发变化必须保持上次有效目录；单账号失败不能中止其他账号。
- 后台事件只刷新目录相关缓存，不反向重新读取 Google 模型，避免同步循环。

# Implementation Plan

1. 阅读相关产品文档、任务历史和代码，明确自动同步与客户端接管的边界。
2. 实现启动／授权／刷新触发的串行后台同步，复用受管客户端更新；增加前端目录事件及自动同步说明。
3. 执行针对性检查及 `task verify`，回填实际结果并检查 diff。

# Implementation Result

## Changes

- 新增可取消的单后台同步器：应用启动、Google 授权变更、模型读取成功后自动发现已启用订阅账号的实际适配模型。
- 合并在途通知，每账号独立 30 秒超时；不启动网关、不请求推理，不自行持续重试。一个账号失败仍继续同步其他账号，复用版本绑定事务保留有效快照。
- 调用现有受管客户端目录刷新，保留仍可用默认模型、显式模型限选、备份和外部文件修订保护，不接管未确认的客户端。
- 全局目录事件只失效模型目录、默认路由、路由、客户端及实际受管菜单缓存；不反向失效 Google 模型读取查询。
- 卡片澄清“网关支持”和真实客户端接入的区别，展示自动同步边界，保留“重新同步到 AI Agent”手动重试；手动同步也刷新实际受管菜单缓存。

## Files Changed

- `internal/application/modelcatalog/google_auto*.go`
- `internal/transport/wails/google_model_sync*.go`、`google_subscription.go`、`subscription.go`，以及 `app.go` 的单行启动调用。
- `internal/application/agentconfig/google_auto_test.go`
- `frontend/src/app/providers.tsx`
- `frontend/src/features/gateway/use-model-catalog-events*`
- `frontend/src/features/google/google-model-sync.tsx`、对应模型卡片测试及 Google 中英文字典。
- Wails 自动生成的事件类型、Google 订阅／Agent 使用说明和架构概览。

## Technical Notes

复用原有协议适配，不扩张 Google 普通 API Key 的协议能力；未知／未适配型号不因自动化而被虚报支持。
后台错误仅记录固定错误码，事件不携带账号身份或凭据。没有新增依赖、数据库迁移、客户端登录改写或生产安装替换。

# Verification

## Tests

- [x] 后台启动、逐账号失败隔离／旧快照保留、禁用账号／提供商／其他通道过滤。
- [x] 通知合并、单账号超时、在途取消、重复关闭和客户端文件冲突报告。
- [x] Windows 隔离 DPAPI／契约传输验证启动、模型读取、授权刷新触发；不访问真实凭据。
- [x] 13 种受管客户端配置，共 39 个子用例：实际菜单更新、显式限选、默认保留、未接管与外部改动保护。
- [x] 前端事件缓存失效／取消订阅／无反向发现循环，以及应用级 Provider 内挂载。
- [x] 模型卡片的数量文案、自动说明、手动失败重试和额度独立性。
- [x] 原生开发窗口 UIA 与截图人工核对，确认自动说明、重试入口和“网关支持”计数；未点击配置应用或启停网关。
- 真正运行各 Agent 的推理调用：本次未执行，不能用菜单／配置测试替代。

## Verification Result

- 定向 Go 测试通过；后台同步用例连续 5 轮，Wails 与跨客户端回归连续 2 轮。
- 初轮完整门禁发现根布局隔离测试缺少 QueryClient；已将事件订阅放到应用级 Provider 内，不修改或绕过原滚动测试，并增加挂载回归测试。定向 3 文件 8 用例通过。
- 后续完整 Go 测试及前端 114 文件／673 用例通过；普通构建的增量源文件扫描耗时过大，只停止本任务的验证进程，未删除历史临时目录或影响生产程序。
- `task --force verify BIN_DIR=tmp/verify-google-auto`：PASS（退出码 0）。格式检查、Go vet／staticcheck／ESLint、TypeScript、全部 Go 测试、前端 114 文件／673 用例、Wails 事件绑定再生成和 Windows amd64 production 构建全部通过。
- 产物：`tmp/verify-google-auto/omniproxy.exe`；完整日志：`tmp/verify-google-auto-force.log`。保留已有的 jsdom scrollTo 提示和 Vite 大 chunk 提示，不将非阻断告警伪装成失败或顺带修改无关模块。

## Quality Checks

- [x] Format
- [x] Lint
- [x] Type Check
- [x] Test
- [x] Build
- [x] Verify

# Documentation

已更新 Google 订阅、客户端接入和架构概览；CURRENT 仅添加本任务状态，不覆盖其他任务记录。技术栈、数据库和 ADR 无变更。

# Deviations

为避免根布局引入不必要的查询依赖，全局事件订阅实际挂在 `AppProviders` 内部，保持纯布局可独立测试。其余方案不变。

# Git

待提交本任务的独立改动，既有其他任务的文件保持未暂存。

# Result

实现与必要验证已完成，待回填提交并归档。真实 Agent 重载／推理、生产安装升级不在本次自动执行范围内。
