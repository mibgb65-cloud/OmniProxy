# TASK-20261118-002: 构建本地 v2.3.1-beta.23 生产安装包

## Status

Completed

## Created

2026-11-18 09:30（Asia/Singapore）

## Request

套餐徽章改动（TASK-20261118-001，`2b995410`）已提交，用户要求构建新的 Beta 生产版本用于本地更新，
并同步更新任务文档。

## Goal

产出 `v2.3.1-beta.23` Windows amd64 本地 Beta 生产安装包，内容为套餐徽章「无套餐」文案与
Antigravity 档位名／配色映射。

## Background

`v2.3.1-beta.22` 之后唯一的源码变更是 TASK-20261118-001：徽章在账号没有套餐字段时显示「无套餐」，
Antigravity 的 `g1-ultra-tier`／`g1-pro-tier`／`g1-free-tier`／`free-tier` 显示为 Ultra／Pro／Free
并分别配色。本版本无数据库迁移、无后端改动。

## Scope

- 版本号 `2.3.1-beta.23`：`internal/version/version.go`、`frontend/package.json`；
  `build/windows/info.json` 的 `ProductVersion` 由构建脚本同步（数字版本仍 `2.3.1.0`）。
- `docs/releases/v2.3.1-beta.23.md`（构建前必须存在，脚本会校验）。
- `scripts/build-release.ps1 -Version v2.3.1-beta.23 -OutputDirectory release-assets\v2.3.1-beta.23`。
- 校验安装器大小／SHA256、原生版本资源与发布资产；记录到 Task Record 与 `docs/plans/CURRENT.md`。

## Out of Scope

- 不打标签、不推送、不自动安装、不启动安装后的程序、不修改用户数据。
- 不修改功能代码；不新增迁移。

---

# Implementation Result

## Changes

- 版本号 `2.3.1-beta.23`：`internal/version/version.go`、`frontend/package.json`、
  `build/windows/info.json` 的 `ProductVersion`；Windows 数字版本仍 `2.3.1.0`，
  NSIS 基础版本仍 `2.3.1`（`build/config.yml` 与 `wails_tools.nsh` 无差异）。
- 新增 `docs/releases/v2.3.1-beta.23.md`（中英成对：问题修复、不兼容变更、本地更新、验证边界）。
- 运行 `scripts/build-release.ps1 -Version v2.3.1-beta.23 -OutputDirectory release-assets\v2.3.1-beta.23`
  （内部 `task --force verify` → `task --force package` → 版本资源与资产校验）。

## Artifacts

- `release-assets/v2.3.1-beta.23/OmniProxy-Setup-v2.3.1-beta.23-windows-amd64.exe`
  **21,383,326 bytes**
  SHA256 `9f75ca1a81e9f72ce96593967867b7c09d3917a3c53c7c5c86555589eef6ce99`
- `release-assets/v2.3.1-beta.23/OmniProxy-Setup-v2.3.1-beta.23-windows-amd64.exe.sha256`
  （内容与上一致）
- `release-assets/` 被 `.gitignore` 忽略。

## Technical Notes

- 相对 `v2.3.1-beta.22`（21,380,531 bytes）增加 2,795 bytes：徽章空值文案、Google 档位对照表、
  `ultra` 配色与 7 项徽章测试。
- 本版本无数据库迁移；覆盖安装即可，无需重新登录账号。
- `set-release-version.ps1` 用 `ConvertTo-Json` 重写 `build/windows/info.json` 时带 CRLF，
  构建后已恢复为 LF（仓库 `.gitattributes` 要求）；文件内只有 `ProductVersion` 变化。
- 构建脚本首轮在 `task --force verify` 的前端测试阶段失败于
  `agent-detail-workspace.test.tsx > 右侧展示实际已接入模型`（1 秒 `findByRole` 超时，页面仍停在
  「读取中…」）；该用例单独连跑 3 次与整套 159 文件重跑均通过，属既有异步用例的时序抖动，
  与本次改动无关。重跑构建脚本后全绿。

---

# Verification

## Tests

- [x] `scripts/build-release.ps1` 内 `task --force verify` 通过（format／lint／typecheck、
      Go 全包、前端 159 文件／976 项测试、Windows production 构建）
- [x] `scripts/test-windows-metadata.ps1`：应用与安装器版本资源校验通过（`2.3.1-beta.23`／`2.3.1.0`）
- [x] `scripts/verify-release-assets.ps1`：发布资产校验通过（`v2.3.1-beta.23`）
- [x] 安装器大小／SHA256 与 `.sha256` 一致（21,383,326 bytes／`9f75ca1a…`）
- [ ] 安装后界面徽章配色未做截图核对（需用户安装后确认）

## Quality Checks

- [x] Format
- [x] Lint
- [x] Type Check
- [x] Test
- [x] Build
- [x] Verify
- [x] Packaging

## Verification Result

```text
已校验 v2.3.1-beta.23，Windows 数字版本 2.3.1.0
task --force verify   → 通过（Test Files 159 passed / Tests 976 passed）
task --force package  → 通过（NSIS）
Windows 应用和安装器版本资源校验通过
发布资产校验通过：v2.3.1-beta.23
BUILD_EXIT=0
```

---

# Documentation

- [x] `docs/releases/v2.3.1-beta.23.md`
- [x] `docs/plans/CURRENT.md`

---

# Deviations

- 首轮构建在 `task --force verify` 因既有异步用例 `agent-detail-workspace.test.tsx` 的 1 秒超时失败；
  未修改该测试，确认可稳定复跑后重试构建成功。实现计划本身无偏离。

---

# Git

## Commit

（回填后填写）

---

# Result

已产出 `v2.3.1-beta.23` Windows amd64 本地 Beta 生产安装包（21,383,326 bytes，
SHA256 `9f75ca1a81e9f72ce96593967867b7c09d3917a3c53c7c5c86555589eef6ce99`），内容为套餐徽章
「无套餐」文案与 Antigravity（Ultra／Pro／Free）档位名与配色。未签名、未公开发布、未打标签、
未推送、未自动安装。覆盖安装后打开 Google／Antigravity 账号页即可看到新的档位徽章。

---

# Completed

2026-11-18 10:05（Asia/Singapore）
