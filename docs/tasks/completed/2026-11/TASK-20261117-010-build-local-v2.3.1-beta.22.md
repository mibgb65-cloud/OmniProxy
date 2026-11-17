# TASK-20261117-010: 构建本地 v2.3.1-beta.22 生产安装包

## Status

Planned

## Created

2026-11-17 20:05（Asia/Singapore）

## Request

受管客户端目录停更的可见性修复（TASK-20261117-009）需要装到用户机器上，用户才能一键恢复
Codex／Claude／DSH 的自动模型刷新，进而在 DeepSeek Harness 里选到 WorkBuddy 国内版模型。

## Goal

产出 `v2.3.1-beta.22` Windows amd64 本地 Beta 生产安装包，内容为目录刷新失败可见与
「重新读取并应用」恢复路径。

## Background

`v2.3.1-beta.21` 已包含 WorkBuddy 国内版接入；本机 6 个受管客户端（codex、claude、
claude-desktop、harness-web、harness-desktop、pi）的配置都被客户端自己改过，
后台目录刷新按安全策略停摆，DSH 的 OmniProxy 模型清单停在旧版本。本版本无数据库迁移。

## Scope

- 版本号 `2.3.1-beta.22`：`internal/version/version.go`、`frontend/package.json`；
  `build/windows/info.json` 的 `ProductVersion` 同步（数字版本仍 `2.3.1.0`）。
- `docs/releases/v2.3.1-beta.22.md`（已随实现提交）。
- `scripts/build-release.ps1 -Version v2.3.1-beta.22 -OutputDirectory release-assets\v2.3.1-beta.22`。
- 校验安装器大小／SHA256、原生版本资源与发布资产；记录到 Task Record 与 `docs/plans/CURRENT.md`。

## Out of Scope

- 不打标签、不推送、不自动安装、不启动安装后的程序、不修改用户数据。
- 不改前端；不新增迁移。

---

# Implementation Result

## Changes

- 版本号 `2.3.1-beta.22`：`internal/version/version.go`、`frontend/package.json`、
  `build/windows/info.json` 的 `ProductVersion`；Windows 数字版本仍 `2.3.1.0`，
  NSIS 基础版本仍 `2.3.1`。
- `docs/releases/v2.3.1-beta.22.md` 已随实现提交（`e2e53b7d`）。
- 运行 `scripts/build-release.ps1 -Version v2.3.1-beta.22 -OutputDirectory release-assets\v2.3.1-beta.22`。

## Artifacts

- `release-assets/v2.3.1-beta.22/OmniProxy-Setup-v2.3.1-beta.22-windows-amd64.exe`
  **21,380,531 bytes**
  SHA256 `4481fffdb95277a7b3c3f64930882ccfa950f39ef9323fe35b124db720b05cf4`
- `release-assets/v2.3.1-beta.22/OmniProxy-Setup-v2.3.1-beta.22-windows-amd64.exe.sha256`
- `release-assets/` 被 `.gitignore` 忽略。

## Technical Notes

- 相对 `v2.3.1-beta.21`（21,372,038 bytes）增加 8,493 bytes：刷新失败码记录、
  「重新读取并应用」入口、列表页汇总提示与前后端测试。
- 构建脚本内部的 `task --force verify` 覆盖本次实现（前端 158 files / 969 tests）。
- 本版本无数据库迁移；`set-release-version.ps1` 重写 `build/windows/info.json` 带 CRLF，已恢复 LF。

---

# Verification

## Tests

- [x] `scripts/build-release.ps1` 内 `task --force verify` 通过（format／lint／typecheck、
      Go 全包、前端 158 文件／969 项测试、Windows production 构建）
- [x] `scripts/test-windows-metadata.ps1`：应用与安装器版本资源校验通过（`2.3.1-beta.22`／`2.3.1.0`）
- [x] `scripts/verify-release-assets.ps1`：发布资产校验通过（`v2.3.1-beta.22`）
- [x] 安装器大小／SHA256 与 `.sha256` 一致（21,380,531 bytes／`4481fffd…`）
- [ ] 安装后对 DSH／Codex／Claude 执行「重新读取并应用」未验证（需用户执行）
- [ ] DSH 里调用 WorkBuddy 国内版模型未验证（需用户执行）

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
已校验 v2.3.1-beta.22，Windows 数字版本 2.3.1.0
task --force verify   → 通过（Test Files 158 passed / Tests 969 passed）
task --force package  → 通过（NSIS，Total size 21,380,531 / 85,998,258 bytes）
Windows 应用和安装器版本资源校验通过
发布资产校验通过：v2.3.1-beta.22
BUILD_EXIT=0
```

---

# Documentation

- [x] `docs/releases/v2.3.1-beta.22.md`
- [x] `docs/plans/CURRENT.md`

---

# Deviations

- 无偏离：沿用既有 Beta 构建流程。`build/windows/info.json` 由脚本重写为 CRLF，已恢复 LF。

---

# Git

## Commit

TBD

---

# Result

已产出 `v2.3.1-beta.22` Windows amd64 本地 Beta 生产安装包（21,380,531 bytes，
SHA256 `4481fffdb95277a7b3c3f64930882ccfa950f39ef9323fe35b124db720b05cf4`），内容为
受管客户端目录停更可见与「重新读取并应用」恢复路径。未签名、未公开发布、未打标签、未推送、
未自动安装。覆盖安装后，在「网关 → 客户端接入」对 DeepSeek Harness Desktop（与 Web、
Codex、Claude 等）点「重新读取并应用」，DSH 的 OmniProxy 提供商即可出现最新目录。

---

# Completed

2026-11-17 20:18（Asia/Singapore）
