# TASK-20261014-008: 修复 Claude Desktop 一键配置写入位置

## Status

Completed

## Created

2026-10-14 11:24（Asia/Singapore）

## Request

用户反馈 Claude Desktop 一键接入未成功，实际仍使用旧项目的配置；旧项目 `E:\go\new\OmniProxy` 可以成功。排查是否读取配置位置错误，或开发版未安装造成。

## Goal

确认客户端实际配置入口，修复错误定位，使一键配置写入与 Claude 使用的配置位置一致，保留原配置及恢复保护。

## Scope

- 对比新旧项目路径及激活实现，只读核对本机客户端清单和脱敏配置状态。
- 修复 Claude 第三方配置位置并增加路径回归验证。
- 执行质量门，更新相关文档及记录。
- 核对运行中的开发版是否使用修复，验证实际配置位置。

## Out of Scope

修改旧项目、删除包缓存、清理用户会话、修改上游凭据或模型路由、变更技术栈。

## Existing Behavior / Root Cause

当前路径选择只要在商店包 LocalCache 发现 `_meta.json`，就改写该包缓存；共享 `Local\Claude-3p` 不再更新。该逻辑由 `cbd7176` 引入。

本机 Claude `2.31226.1.0` 的安装清单明确将 `LocalAppData\Claude-3p` 排除出文件系统虚拟化；实际运行日志使用共享目录及旧项目的 3000 地址。截图中的 `D:\WpSystem\...\LocalCache\Local\Claude-3p` 是包缓存，而不是该客户端的第三方配置入口。

两处配置确实同时存在，共享配置 9 个模型、包缓存 10 个模型。开发网关 3001 正常监听，读取两份配置中的接入 Key 请求 `/v1/models` 均返回 200；两份 Key 一致。没有证据说明必须安装 OmniProxy 才能接入。

## Analysis

采用客户端清单和旧项目已验证的共享入口；不复制写入两份目录，不依据残留包缓存猜测当前入口。路径变化后沿用现有重新读取和备份归档保护。

运行中的开发版位于 `E:\go\new\wails3-v\OmniProxy-main\bin\omniproxy.exe`，当前工作区为另一个分支；交付需明确运行版与源码的区别。

## Implementation Plan

1. 恢复统一共享第三方配置目录，验证空目录、双目录、损坏包缓存都不误选。
2. 检查路径变更后的重新读取、激活及恢复逻辑，执行针对测试及 `task verify`。
3. 更新接入说明与 CURRENT，核对 Git 差异，仅提交本任务改动。
4. 检查实际运行版并记录原生验证结果与限制。

## Progress

- 已完成只读定位；用户进一步确认未写入实际配置，仍显示旧软件配置。
- 已修复路径选择；配置服务全部测试通过，包括激活、恢复、重新读取及备份归档。任务编号按 main 已存在的 001–007 调整为 008，避免与并行工作记录重复。
- 新增路径纠正后的应用／恢复回归测试：未重新读取时拒绝覆盖，重新读取后激活共享配置，恢复使用共享目录的原状态，旧包缓存保持不变；测试通过。
- 源工作区完整 `task verify` 退出码 0；接下来同步至实际运行的 main worktree，并执行该版本的质量门及原生验收。
- 修复提交以 cherry-pick 同步至干净的 main worktree，未混入其他功能；现有 Wails 开发 watcher 自动重建并重启，实际新进程为 13064。
- 已通过实际原生窗口重新读取、预览并应用；共享配置更新至 3001 和 10 个模型，旧包缓存时间保持不变。main 完整质量门通过。

## Implementation Result

- 已恢复共享第三方配置入口，删除错误的包缓存自动选择，保留现有路径安全检查。
- 覆盖新旧包名、共享配置存在／尚未创建、损坏包缓存，以及路径纠正后的安全应用和恢复。
- 修复同步至 main 实际开发环境；恢复重启前运行中的 3001 网关，通过原生 UI 完成重新读取、预览和应用。
- 实际共享配置于 11:33:53 更新，`appliedId` 指向 `309006`，两份启动模式为 `3p`，`disableDeploymentModeChooser=true`，网关地址为 `http://127.0.0.1:3001`，包含 10 个模型。
- 未修改旧项目及生产进程，未删除或复制写入包缓存。原配置备份继续由现有 DPAPI 流程保护；只读核对数据库存在 1 份当前备份、4 份历史备份及 1 份激活状态。

主要修改文件：`paths.go`、`paths_windows_test.go`、`desktop_path_rebase_test.go`；删除不再适用的 `paths_windows.go` 和 `paths_other.go`。

## Verification

- PASS：`go test ./internal/application/agentconfig -count=1`。
- PASS：`go test ./internal/application/agentconfig -run 'TestDesktop(Path|Paths)' -count=1`，包含新增路径恢复测试。
- PASS：源工作区 `VITEST_MAX_WORKERS=4 task verify BIN_DIR=E:/go/new/wails3-v/OmniProxy/tmp/claude-desktop-shared-config/quality-bin`，退出码 0；完整检查与生产构建通过。
- PASS：`git diff --check`。
- PASS：main 的 `VITEST_MAX_WORKERS=4 task verify BIN_DIR=E:/go/new/wails3-v/OmniProxy-main/tmp/claude-desktop-shared-config/quality-bin`，退出码 0；格式、Go vet／staticcheck、ESLint、TypeScript、全部 Go 测试、70 个前端测试文件／384 项测试和 Windows 生产构建通过。源工作区为 69 文件／379 项测试，新增的路径恢复测试另行通过，main 完整检查已包含该测试。
- PASS：实际 `OmniProxy-main/bin/omniproxy.exe` 原生窗口应用配置，共享文件模型数由 9 更新至 10；包缓存仍为 11:19:54 的文件，未被本次应用改写。
- PASS：实际写入的接入 Key 请求 3001 `/v1/models` 返回 HTTP 200（56 个可用目录条目）；3001／3891 均归属于开发进程 13064，生产进程 31320 保持运行。
- 验证日志位于两个 worktree 的忽略目录 `tmp/claude-desktop-shared-config-verify.log`，原生截图和辅助脚本位于源工作区 `tmp/claude-desktop-shared-config/`。
- 既有工具探测、jsdom scrollTo 及打包体积提示未影响质量门。

## Documentation

已更新 LOCAL_AGENTS 的实际配置入口、开发版端口和错误位置恢复步骤；CURRENT 同步完成状态。没有新增产品或架构范围，无需 ADR。

## Deviations

为使用户当前运行版本生效，将同一修复同步至 main worktree，额外执行 main 完整质量门；不把源分支构建当作运行版验收。未跟踪的其他任务记录保持原样。

## Git

- 源分支实现：`1aaf76d0b26d93294b2ef335959064d3273a606e fix(config): 恢复 Claude Desktop 共享配置入口`。
- main 同步实现：`6285c0cbcc8299faa3bb0f530ad01ffa19a9f8d7 fix(config): 恢复 Claude Desktop 共享配置入口`。
- 仅提交本任务代码与文档，未纳入既有未跟踪的 Atria 任务记录；不推送远端。

## Result

已确认根因为误选包缓存，与 OmniProxy 是否安装无关。修复后实际开发窗口成功改写共享配置，3001 网关正常，质量门和路径纠正恢复测试通过。

## Remaining Risks

本轮未关闭用户运行中的 Claude，也未重新启动其会话或发送新推理请求。实际配置已经写入，客户端需完全退出并重开后加载；不能把本轮文件写入和目录鉴权验证表述为重启后的完整对话验收。

## Completed

2026-10-14 11:36（Asia/Singapore）
