# TASK-20261019-016: 优化订阅登录进度、登录链接复制及账号删除

## Status

In Progress

## Created

2026-10-19

## Request

1. 官方网页完成授权后，桌面弹窗应显示获取凭证、交换 Token 等实际进度，而不是一直停留在“等待官方页面完成授权”。
2. 提供复制登录链接按钮，支持用户在其他浏览器资料中完成授权。
3. 修复账号偶发无法删除的问题；删除账号不应退出官方登录，只有点击“退出登录”才执行退出。

## Goal

准确展示授权生命周期，提供可复制的授权链接，区分本地账号删除和官方退出登录语义并覆盖回归测试。

## Background

用户提供了 ChatGPT 订阅登录弹窗截图。现有工作区存在其他未跟踪任务记录，必须原样保留且不纳入本任务提交。

## Scope

- 订阅授权会话进度与桌面展示。
- 复制官方授权链接。
- 账号删除与退出登录的调用链、错误处理及回归验证。

## Out of Scope

- 更换 OAuth 协议、技术栈或凭证存储方式。
- 无关页面重构、安装包发布、修改真实账号或浏览器会话。

## Existing Behavior

- Codex 会话仅发布 waiting 与最终状态，官方完成通知之后的身份同步和保存没有进度事件；前端只依靠事件刷新。
- 删除 Codex 账号必须启动 app-server 并执行 Logout，官方组件不可用或注销失败会阻断删除。

## Expected Behavior

- 回调到达后展示真实处理阶段，授权失败时保留明确错误。
- 能复制当前有效会话的登录链接。
- 删除仅移除本地账号及其本地凭证，不主动撤销官方登录；退出登录保持独立操作。

# Analysis

## Related Modules

- `internal/application/subscription`、`internal/infrastructure/codex`：授权完成处理、本地凭据清理。
- `frontend/src/features/accounts`、订阅 i18n：进度、复制链接及缓存刷新。
- Google 共用登录 UI，需要同步其可观测的交换及保存状态与进行中状态判断。

## Dependencies

沿用现有 Go / Wails / React / TanStack Query / i18n。

## Risks

- 授权回调、取消和 UI 轮询并发可能造成状态覆盖。
- 删除失败可能涉及外键、凭证清理及后台刷新竞态，需要定位根因后处理。
- 认证链接及 Token 属敏感信息，不得写入日志或任务记录。

# Implementation Plan

1. 阅读相关规范，检查 OAuth 与账号操作的现有实现和测试。
2. 确认根因和影响范围，补全分析并进入 In Progress。
3. 实施最小闭环修改，补齐中英文文案与回归测试。
4. 运行针对性测试及 `task verify`，审查差异并记录实际结果。
5. 更新任务概览，按仓库规范提交本任务文件并归档记录。

# Progress

- 已创建任务记录；尚未修改业务代码。
- 已确认采用真实阶段通知与 1 秒轮询兜底，不改写官方 OAuth。Codex 仅提供交换完成通知，因此等待阶段明确涵盖官方交换，随后显示身份读取与本地保存，不伪造交换百分比。
- 删除改为受控本地凭据清理，不启动官方组件或调用 Logout；显式退出操作保持独立。
- 已实现真实阶段、处理中恢复／取消、复制成功及安全失败提示、中英文文案和账号相关缓存失效。
- 补充处理保存阶段持状态锁的问题：凭据／数据库 IO 不再阻塞状态读取，保存仍受操作门和会话取消控制。
- 定向 35 项前端测试及四个后端包通过；关键授权进度、取消、保存可读性与删除回归连续 10 轮通过。
- 第一轮及最终完整 `task verify` 均通过（625 项前端测试、全部 Go 检查及 Windows 构建）。
- 隔离真实 SQLite／DPAPI／Win32 本地删除验收通过，注入“调用即失败”的官方工厂证明删除不启动官方组件；测试夹具账号及授权目录已清理。

# Implementation Result

## Changes

- Codex 官方完成通知后发布 verifying、saving 真实阶段，完成后清除授权链接和设备码；Google 回调发布 exchanging、saving 阶段。
- 前端统一处理进行中状态，关闭后可恢复，不展示重复授权入口；原生事件配合 1 秒状态读取，最终状态停止轮询。
- 新增“复制登录链接”按钮，可用于本机其他浏览器资料；打开浏览器失败仍可复制，提供复制成功与安全错误提示。
- ChatGPT 删除改用 Win32 本地 keyring 定向删除，兼容旧 Token bundle、新加密存储口令、缺失条目和目录；不启动官方组件，不调用 account/logout 或远端撤销。
- keyring 清理失败保留账号和定位供重试，已有其他账号不受影响；删除后刷新账号、模型目录与默认路由缓存。

## Files Changed

- `internal/application/subscription/`：真实进度、保存状态锁、本地删除及回归。
- `internal/infrastructure/codex/credential_delete_*`：本地系统凭据清理与 Windows 边界验证。
- `internal/application/googlesubscription/`：共享进行中状态及进度信号。
- `internal/application/account/mutations.go`：托管删除注释与实际语义一致。
- `frontend/src/features/accounts/`、`frontend/src/services/subscription.ts`、订阅双语言字典。
- `docs/product/SUBSCRIPTION.md`、`docs/plans/CURRENT.md` 和本记录。

## Technical Notes

不新增依赖、数据库迁移或 Wails 接口。原有显式退出、重新授权身份检查、临时授权取消／失败清理继续保留。删除只影响 OmniProxy 隔离凭据，不访问默认 Codex 授权、网页 Cookie 或其他客户端。

# Verification

## Tests

- PASS：`go test ./internal/application/subscription ./internal/application/googlesubscription ./internal/infrastructure/codex ./internal/transport/wails -count=1`。
- PASS：保存锁与取消补充后，前三个后端包再次 `-count=1`。
- PASS：`go test ./internal/application/subscription -run 'TestAuthorizationReportsActualProgress|TestCancelDuringVerification|TestSavingProgress|TestDelete' -count=10`。
- PASS：四个前端测试文件共 35 项，覆盖无事件轮询、处理阶段恢复／取消、最终停止轮询、复制失败重试及现有账号删除／退出。
- PASS：Windows adapter 使用随机临时路径验证真实 CredDeleteW 对缺失条目的幂等语义；未读取真实授权。

## Quality Checks

- 第一轮 `task verify BIN_DIR=tmp/auth-progress-20261019/verify-bin`：PASS，108 文件／625 项前端测试、全部 Go 测试、format/lint/typecheck、Wails 绑定生成与 Windows amd64 production 构建通过。
- PASS：最终 `task verify BIN_DIR=tmp/auth-progress-20261019/verify-bin`，exit 0；108 文件／625 项前端测试、全部 Go 测试、格式／lint／类型检查、Wails 绑定生成、实际前端生产构建与 Windows amd64 production 构建通过。
- PASS：隔离真实 SQLite／DPAPI／Win32 删除验收，不启动官方组件；无真实授权 Token，账号记录、DPAPI 定位及随机授权目录均已清理。
- 未通过原生 UI 自动验收：隔离窗口未暴露可操作的 UIA 控件，无法可靠导航到本次弹窗。已尝试恢复窗口，仍无可靠结果，未把其他同名开发窗口截图当作通过证据；已关闭仅本任务启动的隔离进程。界面交互有组件测试覆盖，但仍需人工原生及真实 OAuth 验收。
- 已执行 `git diff --check`，未发现空白错误。
- 现有 jsdom scrollTo 未实现和 Vite 大 chunk 提示不影响质量门通过；无依赖、生成绑定或数据库迁移改动。

## Limitations

真实上游 OAuth、Token 交换耗时、原生弹窗交互及真实浏览器账号验收未完成；不接触用户实际账号与生产数据，不将离线回归冒充真实授权成功。官方 Codex 无细分 Token 交换通知，该阶段只能明确说明仍在等待官方交换结果。未打包安装器、发布或替换生产程序。

# Documentation

- 任务记录与 `docs/plans/CURRENT.md`。
- 其他受影响文档根据实际变更同步。

# Git

待审查与提交。

# Deviations

为避免将“删除不退出”实现为遗留无引用的系统凭据，采用直接清理本地隔离 keyring，而不是仅删数据库记录。远端会话和其他客户端保持不变。Google 已有删除仅清理本地凭据，本次未改写其删除机制。
