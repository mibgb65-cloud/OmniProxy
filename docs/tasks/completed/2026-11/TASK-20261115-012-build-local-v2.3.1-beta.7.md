# TASK-20261115-012: 构建本地 v2.3.1-beta.7 生产安装包

## Status

Completed

## Created

2026-11-15 21:40（Asia/Singapore）

## Request

用户要求：提交 HowToFix 接入代码后，构建新的 Beta 生产版本用于本地更新，并更新任务记录。
原话：

> 提交代码后，然后构建新的beta生产版本我进行本地更新，记得完成任务md更新

## Goal

- 版本提升为 `2.3.1-beta.7`；Windows 数字版本仍为 `2.3.1.0`，NSIS 基础版本 `2.3.1`。
- 交付 TASK-20261115-011（HowToFix 内置服务商、迁移 049、余额面板、前端标签与图标）。
- 生产编译 + NSIS 打包，核对原生版本资源、SHA256 与发布资产目录。
- 新增双语发行说明 `docs/releases/v2.3.1-beta.7.md`，更新 CURRENT。

## Background

- 实现提交已落地：`974afdaa`（接入 HowToFix）、`69503c66`（TASK-20261115-011 归档）。
- 用户当前运行 `v2.3.1-beta.6`，本版本是它的直接后继；首次启动会执行迁移 049，
  追加 `builtin_howtofix` 预设（默认启用，不改动已有服务商开关、账号与凭据引用）。

## Scope

- 版本映射、发行说明、生产打包、发布资产与摘要、任务记录与 CURRENT。

## Out of Scope

- 不新增功能、不修改业务代码、不改动迁移内容。
- 不推送、不打标签、不公开发布、不签名、不自动安装、不启动。

---

# Implementation Plan

1. `internal/version/version.go` 与 `frontend/package.json` 提升到 `2.3.1-beta.7`。
2. 新增 `docs/releases/v2.3.1-beta.7.md`（`build-release.ps1` 会校验该文件存在）。
3. 运行 `scripts/build-release.ps1 -Version v2.3.1-beta.7 -OutputDirectory release-assets\v2.3.1-beta.7`。
4. 还原 `build/windows/info.json` 的换行为 LF，确认 diff 只含版本行。
5. 记录产物大小与 SHA256，回填 Task Record 与 CURRENT，提交并归档。

---

# Progress

- 2026-11-15 21:40：建立任务记录，开始提升版本与准备发行说明。
- 2026-11-15 21:48：`build-release.ps1` EXIT=0——`task --force verify`、`task --force package`、
  原生版本资源校验、发布资产校验全部通过；安装器 21,075,467 bytes。

---

# Implementation Result

## Changes

- 应用与前端版本 `2.3.1-beta.7`；Windows 数字资源保持 `2.3.1.0`，`ProductVersion` 为
  `2.3.1-beta.7`；`build/config.yml` 与 `wails_tools.nsh` 数字版本不变。
- 新增双语发行说明 `docs/releases/v2.3.1-beta.7.md`：HowToFix 内置服务商、迁移 049、
  目录按账号分组同步、Chat 原生与 Responses／Messages 转换、预付余额面板、边界与未验证项。
- 版本包含实现提交 `974afdaa`（TASK-20261115-011）与归档提交 `69503c66`。

## Build and Artifacts

| 产物 | 大小（bytes） | SHA256 |
| --- | ---: | --- |
| `release-assets/v2.3.1-beta.7/OmniProxy-Setup-v2.3.1-beta.7-windows-amd64.exe` | 21,075,467 | `ea3b699e608a4b969996cd2dd29bed7c25f5a504ed576a1acb89ff35bae80940` |
| `bin/omniproxy.exe` | 24,277,504 | `3fec9c7aea8ef16094b61fd6711e5f7308464e94bacd50e2afa94107ce8f6388` |

摘要侧车与安装器同目录；应用与安装器均未签名。

## Technical Notes

- `set-release-version.ps1` 只校验并归一版本，因此先手工提升 `version.go` 与
  `frontend/package.json`，再由脚本写入 `info.json`／`config.yml`／`wails_tools.nsh`。
- 写版本时 PowerShell 的 `Set-Content -Encoding utf8` 会在文件开头写入 BOM，导致
  `frontend/package.json` 变成非法 JSON、`gofmt` 报格式不符；本轮改用
  `[IO.File]::WriteAllText` 配 `UTF8Encoding($false)` 修正，最终两个文件都只有版本行 diff。
- 打包后 `info.json` 的 CRLF（15 处）已还原为 LF（UTF-8 无 BOM），diff 只剩 `ProductVersion` 一行。

---

# Verification

## Quality Checks

- [x] Format：`info.json` 换行还原为 LF，diff 只含 `ProductVersion` 行；`gofmt`／Prettier 无差异。
- [x] Lint／Type Check：脚本内 `task --force verify` 通过（go vet、staticcheck、ESLint、tsc）。
- [x] Test：Go 全部包通过；前端 155 文件／940 项测试通过。
- [x] Build：`task --force package` 随脚本执行成功，NSIS 输出 21,075,467 bytes。
- [x] Verify：脚本内 `task --force verify` 通过。
- [x] 原生版本资源校验：`test-windows-metadata.ps1` 对应用与安装器均通过。
- [x] 发布资产校验：`verify-release-assets.ps1` 对 `v2.3.1-beta.7` 通过。

## Verification Result

```text
build-release.ps1 -Version v2.3.1-beta.7 -OutputDirectory release-assets\v2.3.1-beta.7
  set-release-version            → 已校验 v2.3.1-beta.7，Windows 数字版本 2.3.1.0
  task --force verify            → 通过（155 files / 940 tests、全部 Go 包、Windows production build）
  task --force package           → 安装器 21,075,467 bytes
  test-windows-metadata          → Windows 应用和安装器版本资源校验通过
  verify-release-assets          → 发布资产校验通过：v2.3.1-beta.7
SCRIPT_EXIT=0
```

未签名、未自动安装、未启动、未打标签、未推送。

---

# Documentation

- [x] 发行说明：`docs/releases/v2.3.1-beta.7.md`
- [x] CURRENT：已追加本版本摘要
- [x] 其余文档无需修改

---

# Git

Not committed by Agent.

---

# Result

已产出 `v2.3.1-beta.7` 本地 Windows amd64 生产安装包，供用户从托盘完全退出应用并备份数据后
覆盖安装。版本内容为 HowToFix 内置服务商接入（迁移 049、按账号分组同步目录、Chat 原生透传与
Responses／Messages 转换、预付美元余额面板、前端图标与文案）。首次启动执行迁移 049 追加
`builtin_howtofix` 预设，不改动已有服务商开关、账号、凭据引用与统计。

---

# Completed

2026-11-15 21:50（Asia/Singapore）
