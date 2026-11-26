# TASK-20261126-003: 构建 v2.5.1-beta.1 本地生产安装包

## Status

Completed

## Created

2026-11-26 (Asia/Singapore)

## Request

提交 Windows 安装器修复，构建新的生产配置 Beta 供本地覆盖更新，完成任务记录并准备 v2.5.1 正式发布。

## Goal

交付带 Windows 原生版本资源和配对 SHA256 的 `v2.5.1-beta.1` 安装器；准备正式版发布说明，等待本地更新验收后再按正式流程发布。

## Analysis

- 基线为已公开的 `v2.5.0`，Beta 内容为 TASK-20261126-002 的安装器令牌权限与诊断修复（`7e81b8f6`）；没有新增数据库迁移。
- 当前无 `v2.5.1` 或本版本 Beta 标签，选 `v2.5.1-beta.1`。应用显示完整 SemVer，Windows 固定版本为 `2.5.1.0`，NSIS 基础版本为 `2.5.1`。
- 本次仅生产配置本地打包，不运行安装器、不启动应用，不签名、不打标签、不推送；Win10 实机首装仍是独立验收项。

## Implementation Plan

1. 添加中英 Beta 发布说明及 v2.5.1 正式发布草稿，同步版本并提交这些准备改动。
2. 使用仓库发布脚本运行完整质量门、NSIS 打包、版本资源和 SHA256 校验；以低并发运行前端测试，不改测试设置。
3. 将产物与结果记录回本任务及安装器修复任务，更新 CURRENT 并提交文档。

## 正式发布前验收

- 用户本地从原程序目录覆盖更新，保留原数据位置；确认「关于」显示 `2.5.1-beta.1`，原配置仍可用。本任务不代为执行。
- Windows 10 首装如仍失败，记录 `OMNI-DATA-*` 码。没有 Win10 实机结果时，在正式说明中保持验证边界。
- 本地验收后另行切换应用／前端版本到 `2.5.1`，复核 Windows 资源和正式说明；使用 GitHub Actions 先做不公开发布的构建、签名与资产复验。公开发布及标签推送须作为独立操作执行。

## Implementation Result

- 应用／前端版本已同步为 `2.5.1-beta.1`。Windows 固定版本为 `2.5.1.0`，NSIS `ProductVersion` 为 `2.5.1`。
- 新增 `docs/releases/v2.5.1-beta.1.md` 和正式版草稿 `docs/releases/v2.5.1.md`，并同步修复任务和 CURRENT。
- 本地产物：`release-assets/v2.5.1-beta.1/OmniProxy-Setup-v2.5.1-beta.1-windows-amd64.exe`（21,627,242 字节），附同名 `.sha256`。
- SHA256：`6907f3f04e337559ec511359b13e94c504e81f62e75fee08d0132a5998a300dc`；Authenticode：`NotSigned`。产物不入库。

## Verification

`scripts/build-release.ps1 -Version v2.5.1-beta.1 -OutputDirectory release-assets/v2.5.1-beta.1` 退出码 0。使用 PowerShell 7、Git `usr/bin` 及 `VITEST_MAX_WORKERS=4`；不修改测试配置。

- `task --force verify`：格式、Lint（两条既有前端警告）、类型检查、Go 全包、Vitest 164 个文件／1006 项和 Windows 生产构建通过。
- `task --force package`：NSIS 机器级安装器构建通过。再次独立运行 Windows 版本资源及发布资产校验通过。
- 静态核对实际安装包：包含 6 个 `OMNI-DATA-*` 诊断码，内嵌辅助程序与本次 Windows amd64 `production` 应用二进制一致。
- 本地更新由用户执行；没有运行安装程序、启动应用或完成 Win10 实机验收。日志：`tmp/release-v2.5.1-beta.1-20261126.log`。

## Deviations

无。仅本地 Beta；正式版说明仍为草稿。

## Git

- `7e81b8f6` — `fix(installer): 修正数据目录查询的 Windows 令牌权限`。
- `933457b4` — `chore(release): 准备 v2.5.1-beta.1 本地生产构建`。
- 构建结果随本任务记载提交；没有创建标签或推送。

## Result

`v2.5.1-beta.1` 本地生产安装包已交付。请确认覆盖更新后显示 `2.5.1-beta.1` 并能读取原有数据，再进行 v2.5.1 正式发布操作。

## Completed

2026-11-26 13:23 (Asia/Singapore)
