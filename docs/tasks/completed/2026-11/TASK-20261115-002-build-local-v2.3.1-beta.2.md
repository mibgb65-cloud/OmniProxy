# TASK-20261115-002: 构建本地 v2.3.1-beta.2 生产安装包

## Status

Completed

## Created

2026-11-15 10:05（Asia/Singapore）

## Request

> 提交代码后，然后构建新的beta生产版本我进行本地更新，记得完成任务md更新

## Goal

- 版本由 `2.3.1-beta.1` 提升为 `2.3.1-beta.2`；Windows 数字版本仍为 `2.3.1.0`，
  NSIS 基础版本 `2.3.1`。
- 交付 TASK-20261115-001（Google 订阅 Gemini 声明图片输入、Chat 接受工具结果图片）。
- 生产编译 + NSIS 打包，核对原生版本资源、SHA256 与发布资产目录。
- 新增双语发行说明 `docs/releases/v2.3.1-beta.2.md`。

## Background

- 实现提交已入库：`1d7120c4`（Gemini 图片能力），记录提交 `3fb99247`。
- 实现任务质量门已通过：`task --force verify` EXIT=0（Go 全包、前端 154 文件／935 项、
  format／lint／typecheck／Windows production 构建）。
- 隔离实例真实请求已验证用户消息与工具结果图片均 200（`Pink`），对照组证明图片确实到达模型。
- v2.3.1 尚无公开正式版；本次是同一数字版本上的下一个本地 Beta。

## Scope

- 版本映射（`version.go`、`frontend/package.json`、`build/config.yml`、
  `build/windows/info.json`、`wails_tools.nsh`）、发行说明、生产打包、发布资产与摘要、
  任务记录与 CURRENT。

## Out of Scope

- 不重跑逐项功能开发；功能门禁沿用实现任务结果。
- 不推送、不打标签、不公开发布、不签名、不自动安装、不启动。
- 不新增功能、不修改业务代码。

## Existing Behavior

源码版本为 `2.3.1-beta.1`，不含 TASK-20261115-001 的 Gemini 图片能力。

## Expected Behavior

覆盖安装后：Antigravity 的 Gemini 模型目录发布 `supportsImages: true`，agent 工具结果里的
图片转成上游 `inlineData`；其余功能与 v2.3.1-beta.1 一致。

---

# Analysis

## Related Modules

- 版本：`internal/version/version.go`、`frontend/package.json`、`build/config.yml`、
  `build/windows/info.json`、`build/windows/nsis/wails_tools.nsh`
- 产物：`bin/`、`release-assets/v2.3.1-beta.2/`
- 文档：`docs/releases/`、`docs/plans/CURRENT.md`

## Dependencies

`scripts/build-release.ps1` → `set-release-version` → `task --force verify` → `task --force package`
→ 版本资源校验 → 发布资产复制／SHA256 → 资产校验。NSIS 位于 `C:\Program Files (x86)\NSIS`。

## Risks

- 安装器未签名，可能触发未知发布者提示。
- 运行中的实例未完全退出时覆盖安装会被拒绝。
- `set-release-version.ps1` 用 `ConvertTo-Json` 重写 `info.json`，需把换行还原为 LF。
- 用户侧目录能力存在 `account_models` 里，安装后仍需重新「同步官方模型」并重连桌面端。

---

# Implementation Plan

1. 创建本记录。
2. 提升 `version.go` 与 `frontend/package.json` 到 `2.3.1-beta.2`。
3. 新增双语发行说明。
4. 运行 `scripts/build-release.ps1 -Version v2.3.1-beta.2`。
5. 核对 `info.json` 换行，确认 diff 只含版本行。
6. 回填 Task Record 与 CURRENT，提交并归档。

---

# Progress

- 2026-11-15 10:05：创建记录；`version.go`、`frontend/package.json` 提升到 `2.3.1-beta.2`。
- 2026-11-15 10:12：`set-release-version` 完成版本映射；`info.json` 换行还原为 LF。
- 2026-11-15 10:16：首次 `build-release.ps1` 完成 verify 与打包，但资产复制因默认
  `release-assets` 目录非空（已有 beta.1）被拒绝；改用版本专属目录重跑。
- 2026-11-15 10:19：重新打包完成（21,007,062 bytes），版本资源与发布资产校验通过。

---

# Implementation Result

## Changes

- 应用与前端版本 `2.3.1-beta.2`；Windows 数字资源保持 `2.3.1.0`，`ProductVersion` 为
  `2.3.1-beta.2`；`build/config.yml` 与 `wails_tools.nsh` 的数字版本不变，diff 只有版本行。
- 新增双语发行说明 `docs/releases/v2.3.1-beta.2.md`：Gemini 图片输入声明、工具结果图片转换、
  升级后需同步官方模型并重启桌面端、无迁移与验证边界。
- 版本包含实现提交 `1d7120c4`（Gemini 图片能力）。

## Build and Artifacts

| 产物 | 大小（bytes） | SHA256 |
| --- | ---: | --- |
| `release-assets/v2.3.1-beta.2/OmniProxy-Setup-v2.3.1-beta.2-windows-amd64.exe` | 21,007,062 | `12ea539d3c689967fde1ace883deb5046397c92478bdc0b6e8eaf3b9326b6968` |
| `bin/omniproxy.exe` | 24,181,248 | `7d57ef0a20fda7753792b2dc34976004e006cf55ab99b8c8ac49a36703e9e72e` |

摘要侧车与安装器同目录；应用与安装器均未签名。

## Deviations

`build-release.ps1` 的默认 `OutputDirectory` 是仓库级 `release-assets`，该目录已有 beta.1 资产，
脚本按设计拒绝非空目录。未改动脚本：用 `-OutputDirectory release-assets\v2.3.1-beta.2` 重跑，
完成校验与资产复制。

---

# Verification

## Quality Checks

- [x] Format：`info.json` 保持 LF，diff 只含 `ProductVersion` 行。
- [x] Build：`task --force package` 随 `build-release.ps1` 执行成功。
- [x] Verify：脚本内的 `task --force verify` 通过（154 文件／935 项前端测试、全部 Go 包、
  Windows production 构建）。
- [x] 原生版本资源校验：Windows 应用与安装器版本资源通过。
- [x] 发布资产校验：`v2.3.1-beta.2` 通过。

## Verification Result

```text
build-release.ps1 -Version v2.3.1-beta.2 -OutputDirectory release-assets\v2.3.1-beta.2
  set-release-version            → 已校验 v2.3.1-beta.2，Windows 数字版本 2.3.1.0
  task --force verify            → 通过（154 files / 935 tests，Windows production build）
  task --force package           → 安装器 21,007,062 bytes
  test-windows-metadata          → Windows 应用和安装器版本资源校验通过
  verify-release-assets          → 发布资产校验通过：v2.3.1-beta.2
  EXIT=0
```

未签名、未自动安装、未启动、未打标签、未推送。

---

# Documentation

- [x] 发行说明：`docs/releases/v2.3.1-beta.2.md`
- [x] CURRENT：已追加本版本摘要
- [x] 其余文档无需修改

---

# Git

## Commit

```text
668122a5 chore(release): 准备 v2.3.1-beta.2 本地构建
```

---

# Result

已产出 `v2.3.1-beta.2` 本地 Windows amd64 生产安装包，供用户退出应用并备份数据后覆盖安装。

---

# Completed

2026-11-15 10:20（Asia/Singapore）

