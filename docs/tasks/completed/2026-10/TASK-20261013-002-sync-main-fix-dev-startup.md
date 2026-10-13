# TASK-20261013-002: 同步主分支并修复开发启动时序

## Status

Completed

## Created

2026-10-13（Asia/Singapore）

## Request

我需要你处理一下这个 git，如果文档能提交就提交吧，先同步到最新分支，然后修复 `task dev`。

## Goal

将现有安全、相关的文档提交到 Git，使本地 `main` 包含最新代码与文档；永久修复普通 `task dev` 在前端就绪前启动原生应用的问题，并完成验证和提交。

## Scope

- 检查并提交当前工作区的既有文档。
- 将本地 `main` 快进到最新实现，保留其他分支及 worktree。
- 修复开发启动等待逻辑，覆盖延迟就绪、超时和错误路径。
- 更新构建说明和任务看板，运行 `task verify` 与实际开发启动验收。

## Out of Scope

新增远端、推送、改写提交历史、删除旧分支或 worktree、修复业务功能中的独立问题。

## Analysis

当前分支为 `codex/account-details-layout`，最新代码提交为 `44fdbc4`；本地 `main` 为其祖先，落后 33 个提交。未配置远端。70 项未提交内容均为 Markdown 文档，用户已授权纳入本次 Git 整理。

原有 Wails 配置在开发构建后并发启动 Vite 和原生应用；Wails 应用连接前端仅重试 10 次，每次等待 500 毫秒，今天两次启动因前端尚未就绪而失败。临时等待脚本已经证明先确认前端可用再启动应用能恢复开发环境。

方案比较：仓库 PowerShell 启动脚本可复用临时方案，但会增加独立的脚本验证链路。选择在 Go 桌面适配层、创建 Wails 应用前等待前端就绪，使用现有标准库与 Go 测试；通过构建标签使生产版保持直接启动。等待仅允许本机开发地址，整体上限 120 秒，单次请求有超时且不跟随重定向。

## Implementation Plan

1. 审查文档、编码、敏感信息和 Git 差异，提交现有文档及本任务记录。
2. 切换到本地 `main`，使用 `--ff-only` 同步最新实现和文档。
3. 比较启动等待方案，选择与当前 Windows 技术栈兼容的最小实现，保留 Wails 热更新及进程生命周期。
4. 验证延迟就绪、连接失败及超时行为，运行完整 `task verify`。
5. 用普通 `task dev` 启动并检查窗口、HMR、Control API；更新相关文档并提交修复，归档记录。

## Risks

开发进程需要退出以释放端口和二进制；必须按项目路径识别，保留其他应用实例。等待机制必须有明确超时和失败输出，不能无限等待或泄露凭据。

## Implementation Result

- 已提交 71 个 Markdown 文件（含本任务记录），规范化一份既有任务记录的 BOM／换行；现有文档通过敏感模式及相对链接检查。
- 本地 `main` 从 `c9fc9a8` 快进到 `fdfd29e`，包含昨日 `44fdbc4` 实现快照及现有文档；未改写历史，其他分支和 worktree 保留。
- 新增 Go 桌面适配层的开发服务等待逻辑，在创建 Wails 应用前轮询 Vite 首页，最长 120 秒，单次请求上限 2 秒，重试间隔 500 毫秒，仅 HTTP 200 表示就绪。
- 开发地址限制为无凭据、路径、查询参数和片段的本机 HTTP/HTTPS 地址；不使用系统代理或跟随重定向。地址无效或等待超时会返回明确错误。
- 使用 `production` 构建标签区分实现，生产版直接使用内嵌前端。原有 Taskfile、Wails 配置、源码 watcher 和 Vite 热更新流程保持现有入口。
- 普通 `task dev` 已在 `main` 启动：任务 PID 17520，桌面应用 PID 37392，Vite PID 37376。窗口路径为当前仓库的 `bin/omniproxy.exe`，可见且正常响应。
- 本次运行日志、元数据和验证快照位于 `C:/Users/mimanchi/AppData/Local/Temp/omniproxy-fixed-dev-20261013-85dfca28/`。

## Root Cause

原有开发流程在构建完成后并发启动前端和原生应用，未等待前端返回可用页面；Wails 框架的短重试窗口不能覆盖整个前端启动链路，应用失败后 watcher 按 primary 的 shutdown 策略退出并终止前端。

## Regression Prevention

Go 回归测试覆盖超过 5 秒的前端就绪等待、持续 HTTP 503、连接失败、重定向拒绝、请求取消和开发地址边界；独立的 production 标签测试验证生产版忽略开发服务环境变量。

## Verification

- 文档提交前检查：71 项 Markdown 未发现敏感凭据模式，相对文件链接均有效；既有编码问题已规范化为 UTF-8 无 BOM、LF。
- `go test ./internal/transport/wails -run 'Test(ParseDevServerURL|WaitFor)' -count=1`：PASS，慢启动回归实际等待超过 5 秒。
- `go test -tags production ./internal/transport/wails -run '^TestProductionIgnoresDevServer$' -count=1`：PASS。
- `task verify`：PASS（退出码 0），格式检查、Go vet／staticcheck／ESLint、TypeScript、全部 Go 测试、64 个前端测试文件／340 项测试及生产构建通过。
- 普通 `task dev`：开发构建与运行成功，使用仓库原有 `build/config.yml`，无临时等待配置。
- `http://127.0.0.1:9245/` 和 `/@vite/client` 均返回 HTTP 200。
- `http://127.0.0.1:3891/healthz` 返回 `status=ok`、`profile=development`、`controlPort=3891`。
- 日志包含中文的等待／就绪信息及 Wails `Connected to frontend dev server!`，无 FATAL。
- 当前仓库原生窗口标题为 OmniProxy，句柄 1248868，`IsWindowVisible=true`，`Responding=true`；9245 和 3891 均为回环监听，未主动启动网关。
- 构建未产生额外的已跟踪源码／绑定差异；新文件均为 UTF-8 无 BOM、LF，文件长度符合现有规范。

## Remaining Risks

本地没有远端配置，因此 Git 同步仅涉及本地分支。Wails 工具链缺少 uname／tail、jsdom scrollTo 和客户端模型目录刷新警告仍存在，但未影响本次完整质量门和开发窗口启动；这些独立问题未纳入本任务。

## Documentation

已更新 `docs/development/BUILD.md` 的等待时限、输入边界和故障排查说明；本记录已归档，`docs/plans/CURRENT.md` 已同步完成状态。

## Deviations

None.

## Git

- 文档提交：`fdfd29ec1a935f1d106dacd4c3c550bc3d6e2f29`，`docs: 同步网关客户端实现文档与任务记录`，71 个 Markdown 文件。
- 已切换到本地 `main`，通过 `git merge --ff-only codex/account-details-layout` 同步到文档提交；旧分支及其他 worktree 保留。
- 修复提交：`1ecb6bba28952faeab13e58d8be5ed71a2f22fc5`，`fix(wails): 等待前端就绪后启动开发窗口`，提交于 `main`。

## Result

代码和文档已同步到本地 `main`，永久启动修复通过完整质量门与普通 `task dev` 原生验收并完成提交；开发环境保持运行，任务记录已归档。

## Completed

2026-10-13（Asia/Singapore）
