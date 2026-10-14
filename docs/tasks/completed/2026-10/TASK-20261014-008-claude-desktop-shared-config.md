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

本机 Claude `2.31226.1.0` 的安装清单明确将 `LocalAppData\Claude-3p` 排除出文件系统虚拟化；正常桌面上下文的文件句柄证实入口是共享目录。

另一个独立根因是启动上下文：工具及其直接启动的开发版访问共享路径时，实际被重定向到 `Packages\OpenAI.Codex_2p2nqsd0c76g0\LocalCache\Local\Claude-3p`。第一轮看到的 3001／10 模型属于该缓存；用户重开后的 Claude 仍读取真实共享文件的旧项目 3000／4 模型。同一用户名、相同路径字符串和未打包进程身份不能排除该重定向。与 OmniProxy 是否安装无关。

恢复逻辑还存在相关缺口：接受重新读取的主配置时，只重建主备份，没有同步更新启动模式和选中配置的恢复基准。同一逻辑路径跨视图后 Scope 相同，可能恢复错误配置选择。

## Analysis

采用真实共享入口，从正常桌面上下文启动 `task dev`；不复制写入多个缓存。已接受当前配置且主备份实际重建时，同步以当前启动状态为恢复基准，仍在既有事务内提交。

运行中的开发版位于 `E:\go\new\wails3-v\OmniProxy-main\bin\omniproxy.exe`，当前工作区为另一个分支；交付需明确运行版与源码的区别。

## Implementation Plan

1. 恢复统一共享第三方配置目录，验证空目录、双目录、损坏包缓存都不误选。
2. 检查路径变更后的重新读取、激活及恢复逻辑，执行针对测试及 `task verify`。
3. 更新接入说明与 CURRENT，核对 Git 差异，仅提交本任务改动。
4. 检查实际运行版并记录原生验证结果与限制。

## Progress

- 11:40 用户完全退出重开 Claude 后反馈仍使用旧配置。本机新 Claude 进程始于 11:39，启动日志仍使用 `3000/claude-desktop`，而开发窗口与工具读取到的共享配置为 3001。前轮未验证客户端真实读取，不能据写入侧结果宣布问题已解决；本任务重新打开，继续核对实际读取路径和执行环境。
- 已通过文件句柄确认第二个根因：工具与其直接启动的开发版读取 `Local\Claude-3p` 时，被重定向到 `D:\WpSystem\...\Packages\OpenAI.Codex_2p2nqsd0c76g0\LocalCache\Local\Claude-3p`。`GetPackageFullName` 对工具和开发版仍返回未打包，因此不能靠包身份或相同用户名排除重定向。
- 两种启动方式的只读核对已区分视图：COM ShellExecute 启动仍读取 Codex 缓存；显式资源管理器打开隐藏只读启动器后，句柄指向真实 `C:\Users\mimanchi\AppData\Local\Claude-3p`，选中旧项目 `157210`、3000、4 个模型。输出仅包含路径、配置 ID、公开地址和模型数，不含 Key。
- 已通过精确匹配开发进程的托盘 Quit 正常退出 PID 13064，生产进程 31320 保留；通过资源管理器启动 main 的普通 `task dev`，准备验证真实共享配置。
- 进一步审查恢复保护发现：同一逻辑路径跨文件视图时，现有 Scope 哈希相同。重新读取会更新主配置备份，却保留旧视图的启动模式／appliedId 基准。需要在已接受当前配置、且主备份实际重建时，同步使用当前激活状态作为恢复基准；增加同路径外部配置变更的回归测试，再应用真实配置。
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
- 第一轮 11:33:53 的“共享文件已更新”判断仅覆盖 Codex 虚拟视图，不能作为客户端已接入的证据。用户于 11:40 反馈后重开本任务，继续验证。
- 通过精确托盘 Quit 正常退出旧开发进程；资源管理器打开隐藏启动器，在 main 运行普通 `task dev`。保留原开发数据库、开发与生产隔离；修复后的最终开发进程为 34660。
- 在正常开发窗口完成重新读取／预览／应用。11:54:42 的正常桌面只读核对确认文件实体是 `C:\Users\mimanchi\AppData\Local\Claude-3p\configLibrary\_meta.json`，选中 `309006`，网关为 `http://127.0.0.1:3001`，10 个模型。
- 补齐重新读取后的激活恢复基准，同事务保存；只读核对恢复选择为旧项目 `157210`，两份原模式为 `3p`，历史加密主备份累计 5 份。旧项目及其生产进程未修改，未删除包缓存或用户会话。
- Claude 经自己的 File → Exit 正常退出，再由正常开发窗口的「打开 Claude Code」重启。新进程 36884 始于 11:55:40；实际日志使用 3001，`ConfigHealth={state:'healthy',provider:'gateway'}`，原生 Code 入口显示 `deepseek-flash[1m]`。

主要修改文件：`paths.go`、`paths_windows_test.go`、`desktop_path_rebase_test.go`、`desktop_activation.go` 及 Apply／Preview／Restore 调用；删除不再适用的 `paths_windows.go` 和 `paths_other.go`。

## Verification

- PASS：`go test ./internal/application/agentconfig -count=1`。
- PASS：`go test ./internal/application/agentconfig -run 'TestDesktop(Path|Paths)' -count=1`，包含新增路径恢复测试。
- PASS：源工作区 `VITEST_MAX_WORKERS=4 task verify BIN_DIR=E:/go/new/wails3-v/OmniProxy/tmp/claude-desktop-shared-config/quality-bin`，退出码 0；完整检查与生产构建通过。
- PASS：`git diff --check`。
- PASS：main 的 `VITEST_MAX_WORKERS=4 task verify BIN_DIR=E:/go/new/wails3-v/OmniProxy-main/tmp/claude-desktop-shared-config/quality-bin`，退出码 0；格式、Go vet／staticcheck、ESLint、TypeScript、全部 Go 测试、70 个前端测试文件／384 项测试和 Windows 生产构建通过。源工作区为 69 文件／379 项测试，新增的路径恢复测试另行通过，main 完整检查已包含该测试。
- 第一轮写入侧检查：虚拟视图文件 9 → 10 模型，Key 请求返回 HTTP 200；之后发现不能代表真实 Claude 文件或加载结果。PID 13064 已正常退出，不再使用该上下文验收。
- PASS：新增同路径重新读取恢复测试，修复前明确失败，修复后 `go test ./internal/application/agentconfig -count=1` 全部通过。
- PASS：最终 main `VITEST_MAX_WORKERS=4 task verify BIN_DIR=E:/go/new/wails3-v/OmniProxy-main/tmp/claude-desktop-shared-config/normal-quality-bin` 退出码 0；全部 Go 测试、70 个前端文件／384 项测试、格式、lint、类型检查和生产构建通过。日志为 `tmp/claude-desktop-shared-config-normal-verify.log`。
- PASS：正常桌面上下文的只读核对 `normal-read.json`，实体路径为真实共享目录，选中 309006、3001、10 模型；恢复基准为实际旧项目 157210，保留 5 份历史备份。
- PASS：正常启动的最终开发进程 34660，9245 前端服务连接成功，3001／3891 监听；生产进程 31320 保留。
- PASS：真实 Claude 36884 重启后进入 Code，原生页面显示 DeepSeek Flash 1M，实际启动日志报告 3001 和连接健康；没有继续要求用户重复重启来代替验收。
- 验证日志位于两个 worktree 的忽略目录 `tmp/claude-desktop-shared-config-verify.log`，原生截图和辅助脚本位于源工作区 `tmp/claude-desktop-shared-config/`。
- 既有工具探测、jsdom scrollTo 及打包体积提示未影响质量门。

## Documentation

已更新 LOCAL_AGENTS 的配置入口、正常启动要求与恢复基准；BUILD 记录实际句柄核对、Codex 重定向和正常桌面启动／客户端验收要求；CURRENT 同步最终完成状态。没有新增产品或架构范围，无需 ADR。

## Deviations

第一轮未验证 Claude 读取就宣布完成，用户复测发现失败；已重新打开任务并澄清该轮只是虚拟视图写入侧结果。最终增加正常桌面启动、客户端重启及恢复基准修复。两次实现均同步到 main，最终重新执行 main 全量质量门；未跟踪的其他任务保持原样。

## Git

- 源分支实现：`1aaf76d0b26d93294b2ef335959064d3273a606e fix(config): 恢复 Claude Desktop 共享配置入口`。
- main 同步实现：`6285c0cbcc8299faa3bb0f530ad01ffa19a9f8d7 fix(config): 恢复 Claude Desktop 共享配置入口`。
- 第二轮源分支实现：`678e4ae2d64304dcb619bf99a53731de127c546c fix(config): 重新读取时同步 Claude 启动恢复基准`。
- 第二轮 main 同步实现：`fa69dc6 fix(config): 重新读取时同步 Claude 启动恢复基准`。
- 仅提交本任务代码与文档，未纳入既有未跟踪的 Atria 任务记录；不推送远端。

## Result

已修复误选 Claude 包缓存和重新读取恢复状态缺口，并从正常桌面上下文启动开发版，消除本轮 Codex 虚拟视图影响。真实共享配置已更新；实际 Claude 重启后读取 3001、进入 Code、连接健康，恢复备份保留，最终质量门通过。

## Remaining Risks

已验证真实配置加载、原生连接探针和 Code 页面，未发送新编程任务或逐一验证 10 个模型。今后从 Codex 工具直接启动开发版仍可能继承虚拟视图，必须遵循 BUILD 的正常桌面启动要求；实际接入验收不能仅以工具侧读取结果为准。

## Completed

2026-10-14（Asia/Singapore；第二轮最终验收）
