# TASK-20261126-002: 修正 Windows 安装器数据目录查询

## Status

Completed

## Created

2026-11-26 (Asia/Singapore)

## Request

GitHub v2.4.0 在一台 Windows 10 上首次安装时，默认程序路径点下一步即出现无法确认数据位置的提示。无法取得该机器的更多信息或进行 Win10 实机测试，要求按已发现的可能问题修复。

## Goal

按 Windows API 契约查询使用者的 AppData，并在安装向导中提供能分辨失败阶段的安全诊断。保留实际使用者身份与原数据保护，不在读取失败时猜测默认位置。

## Scope

- Windows 安装辅助程序的数据目录读取与相关令牌权限。
- NSIS 失败提示、针对性回归验证与开发文档。

## Out of Scope

- 重打公开的 v2.4.0、发布新版本、更改发布签名流程或降低 Windows 应用控制。
- 在提权桌面无法确认普通使用者时回退为管理员配置。

## Existing Behavior

安装器把启动失败、用户身份确认失败、路径解析失败和临时结果失败显示为同一提示。公开包已核实包含未单独签名的辅助程序；不能据此确认反馈机器有拦截策略。

## Analysis

`SHGetKnownFolderPath` 文档要求显式用户令牌具有 `TOKEN_QUERY` 和 `TOKEN_IMPERSONATE` 权限，当前数据目录查询使用的是 `TOKEN_QUERY | TOKEN_DUPLICATE`。本机 Win11 下旧权限仍可能查询成功，反馈机器实际失败点未确认。只调整数据目录所需权限，不扩大安装完成后启动界面所用令牌权限。

## Implementation Plan

1. 按 API 权限契约打开令牌，保持原有身份校验、线程模拟和失败即停语义。
2. 区分辅助程序启动、身份／目录解析、结果文件各阶段；只显示固定诊断码和文案，不显示原始路径或系统输出。
3. 跑 Windows 相关测试、编译验证原生 NSIS 页面，并执行 `task verify`；记录实机边界。

## Implementation Result

- 数据目录读写的用户令牌现在包含 `TOKEN_IMPERSONATE`；安装完成后启动界面的权限要求不变。
- NSIS 用固定的 `OMNI-DATA-*` 代码区分助手无法启动、用户身份、AppData、配置、临时结果及异常退出。空结果或仅含换行的结果也不能继续；安装器仍以 `67` 退出，不改已有数据。
- 保留中文路径的 UTF-16LE 协议、普通命令行读取模式及静默更新行为。内嵌辅助程序签名状态不变；无应用控制拦截的实机证据，不降低安全策略。

## Verification

- `go test -count=1 ./internal/infrastructure/appinstall ./internal/platform/profile`：沙箱原子重命名返回无效句柄；沙箱外重跑通过。新增测试使用实际 Windows 令牌验证权限，并检查助手退出码。用 Go overlay 去掉新增权限后，定向回归测试按预期在 `SetThreadToken` 返回拒绝访问；没有修改仓库源文件。
- 原生 NSIS 用隔离的 x64 夹具编译通过。Win11 上中文／英文共 18 例通过：助手缺失、各类失败码、无结果、空结果及中文路径成功；未安装或启动真实应用。
- `task verify BIN_DIR=tmp/verify-installer-data-directory` 首次通过格式、Lint、类型检查及 Go 全部测试；默认并发下 Vitest 出现 11 个超时／断言失败，未进入构建阶段。
- 以 `VITEST_MAX_WORKERS=4` 重跑：前四阶段通过（含前端 164 文件、1006 项测试；ESLint 有 2 条既有警告）。进入构建后 Task 长时间无输出且无构建子进程，停止该校验进程；完整命令未返回通过。
- 单独执行 `task --force build DEV=true BIN_DIR=tmp/verify-installer-data-directory`，开发版前端与 Windows amd64 编译通过；不运行可执行文件，不生成发布安装包。
- `git diff --check` 通过。未在 Win10 或真实跨账号 UAC 环境复测，尚不能确认原反馈机器的根因。

## Documentation

- `docs/development/BUILD.md` 同步目录查询权限及诊断码，`docs/plans/CURRENT.md` 记录修复与实机验证边界。

## Deviations

- 完整质量门的构建环节在本机未完成，改为单独验证开发版编译及隔离的 NSIS 脚本，不将其记为一次完整质量门通过。

## Git

- `7e81b8f6` — `fix(installer): 修正数据目录查询的 Windows 令牌权限`。

## Result

权限修复与可诊断失败提示已在当前开发分支本地提交；原公开安装包未变。后续如复现，依据新包的 `OMNI-DATA-*` 诊断区分原因，不放宽原数据保护。

后续生产验证：[TASK-20261126-003](TASK-20261126-003-build-local-v2.5.1-beta.1.md) 已交付包含本修复的 `v2.5.1-beta.1` 本地安装包，完整 `task --force verify`、NSIS 打包、版本资源和 SHA256 校验均通过。未代用户覆盖安装，Win10 首装仍待实机验证。

## Completed

2026-11-26 12:13 (Asia/Singapore)
