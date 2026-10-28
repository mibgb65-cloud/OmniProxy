# TASK-20261028-009: 构建 v2.0.1-beta.6 本地生产安装包

## Status

Completed

## Created

2026-10-28 22:45（Asia/Shanghai）

## Request

> 提交代码，然后构建新的beta生产版本我进行本地更新，记得完成任务md更新

## Goal

产出 `release-assets/v2.0.1-beta.6/` 安装器与 SHA256，版本源统一为 `2.0.1-beta.6`
（Windows 数字版本 `2.0.1.0`、NSIS `2.0.1`），通过完整质量门、Windows 版本资源与发布资产校验，
供用户手动覆盖更新；同时更新发布说明、CURRENT 与任务记录。

## Background

`2.0.1-beta.5`（`17298620`）之后，`main` 上新增一项用户可见功能改动：

- [TASK-20261028-008](../completed/2026-10/TASK-20261028-008-configurable-gateway-port.md)（`4f09490a`）：
  本地网关端口可配置，端口被占用时自动避让，受管客户端地址随端口改写。
  回填提交 `eb023a5f`。

## Scope

- 版本源：`internal/version/version.go`、`frontend/package.json` 改为 `2.0.1-beta.6`；
  `build/windows/info.json` 的 `ProductVersion` 同步；`build/config.yml`、
  `build/windows/nsis/wails_tools.nsh` 与 `file_version`／`product_version`
  保持数字版本 `2.0.1`／`2.0.1.0`。
- 新增 `docs/releases/v2.0.1-beta.6.md` 发布说明。
- 完整质量门 → NSIS 打包 → Windows 版本资源校验 → 发布资产与 SHA256 校验。
- 更新 `docs/plans/CURRENT.md`（顶部状态行、Current Version 本地构建段、Recently Completed）。

## Out of Scope

- 不打标签、不推送、不发布、不代码签名。
- 不停止、不替换用户正在使用的生产程序；覆盖安装由用户手动执行。
- 不改业务逻辑、数据库迁移、依赖。

## Existing Behavior

版本源为本地构建的 `2.0.1-beta.5`／`2.0.1`／`2.0.1.0`；`release-assets/` 最新目录为
`v2.0.1-beta.5`，无 `v2.0.1-beta.6` 目录与发布说明。

## Expected Behavior

版本源一致为 `2.0.1-beta.6`／`2.0.1`／`2.0.1.0`；`docs/releases/v2.0.1-beta.6.md` 存在；
`release-assets/v2.0.1-beta.6/` 只含安装器与 `.sha256`，两者校验通过。

---

# Analysis

## Related Modules

- 版本源 5 处、`docs/releases/`、`release-assets/`
- `scripts/{build-release,set-release-version,test-windows-metadata,verify-release-assets}.ps1`

## Dependencies

- Go、Node 22.x、pnpm 12.6.0、Task、Wails beta.30、staticcheck、NSIS
  （`C:\Program Files (x86)\NSIS`，脚本自动前置到 PATH）。

## Risks

- 数字版本与 beta.5 相同（`2.0.1.0`）：安装器按同一产品标识覆盖升级；SmartScreen 仍可能提示未知发布者。
- `set-release-version.ps1` 重写 `build/windows/info.json` 会产生 CRLF，需按 `.gitattributes` 还原为 LF。
- 用户已在本机运行旧版生产程序：构建与打包不停止、不替换该进程，覆盖安装由用户执行。
- 端口可配置改动会改写受管客户端地址：本任务只产出安装包，不改真实客户端配置。

---

# Implementation Plan

1. 建立任务记录。
2. `internal/version/version.go`、`frontend/package.json` 改为 `2.0.1-beta.6`。
3. 新增 `docs/releases/v2.0.1-beta.6.md`。
4. `scripts/build-release.ps1 -Version v2.0.1-beta.6 -OutputDirectory release-assets/v2.0.1-beta.6`。
5. 核对版本源、`info.json` 换行与 `frontend/bindings` 无 diff。
6. 回填 Task Record、更新 CURRENT、提交并归档。

---

# Progress

- 已确认工作区干净，`2.0.1-beta.5` 之后只有端口可配置一项用户可见改动。

---

# Implementation Result

## Changes

- 版本源同步为 `2.0.1-beta.6`：`internal/version/version.go`、`frontend/package.json`、
  `build/windows/info.json` 的 `ProductVersion`；`build/config.yml`、
  `build/windows/nsis/wails_tools.nsh` 与 `file_version`／`product_version` 保持
  `2.0.1`／`2.0.1.0`（脚本输出「已校验 v2.0.1-beta.6，Windows 数字版本 2.0.1.0」，两文件无 diff）。
- 新增发布说明 `docs/releases/v2.0.1-beta.6.md`：端口可配置、冲突自动避让、受管客户端地址同步、
  升级说明与已知边界。
- `bin/omniproxy.exe` 23,215,616 bytes（production，`2.0.1-beta.6` 资源）；
  `bin/OmniProxy-amd64-installer.exe` 20,315,659 bytes（`makensis` Total size
  20,315,659／80,925,522 bytes，25.1%）。
- `release-assets/v2.0.1-beta.6/`：
  - `OmniProxy-Setup-v2.0.1-beta.6-windows-amd64.exe` 20,315,659 bytes
  - SHA256 `3fee16a3f6d79358c4a1bfdd9da662455b527de435d224a97fe58fa686eb7aa3`

## Files Changed

- `internal/version/version.go`、`frontend/package.json`、`build/windows/info.json`
- `docs/releases/v2.0.1-beta.6.md`（新增）
- `docs/plans/CURRENT.md`
- 本记录

## Technical Notes

- 第一次运行 `build-release.ps1` 在 `format:check` 失败：用 PowerShell 改写 `version.go`／
  `package.json` 时写入了 UTF-8 BOM，`gofmt -l` 与 `prettier --check` 均报错；去掉 BOM 后重跑一次通过。
- `build/windows/info.json` 经 `set-release-version.ps1` 重写后为 CRLF，已按 `.gitattributes`
  还原为 LF，diff 只有 `ProductVersion` 一行。
- `task --force package` 重新生成 `frontend/bindings` 后无 diff（本任务无接口变更）。
- 构建过程不停止、不替换正在运行的生产程序；端口改写只在用户保存端口时发生，本次构建未触碰真实客户端配置。

---

# Verification

## Tests

- [x] Unit Tests（`task --force verify` 内 `go test . ./internal/...`）
- [x] Integration Tests（同上，含 gateway／agentconfig／wails 绑定层）
- [ ] E2E Tests：N/A（本任务只改版本源与发布说明）

## Quality Checks

- [x] Format
- [x] Lint
- [x] Type Check
- [x] Test
- [x] Build
- [x] Verify

## Verification Result

```text
scripts/build-release.ps1 -Version v2.0.1-beta.6 -OutputDirectory release-assets/v2.0.1-beta.6
                                        exit 0（日志 tmp/release-v2.0.1-beta.6.log）
  set-release-version.ps1              → 「已校验 v2.0.1-beta.6，Windows 数字版本 2.0.1.0」
  task --force verify                  → format:check / go vet / staticcheck / ESLint / tsc PASS
                                          go test ./internal/... PASS
                                          vitest Test Files 139 passed，Tests 849 passed
  task --force package                 → bin/omniproxy.exe 23,215,616 bytes
                                          makensis Total size 20,315,659／80,925,522 bytes（25.1%）
  test-windows-metadata.ps1            → 「Windows 应用和安装器版本资源校验通过」
  verify-release-assets.ps1            → 「发布资产校验通过：v2.0.1-beta.6」
```

版本源核对：`2.0.1-beta.6`（version.go、package.json、info.json `ProductVersion`）、
`2.0.1`（config.yml、NSIS `INFO_PRODUCTVERSION`）、`2.0.1.0`（info.json
`file_version`／`product_version`），无 `2.0.1-beta.5` 残留。

未执行：代码签名、自动安装、停止或替换正在运行的生产程序、打标签、推送、公开发布；
未做端口设置卡片的原生窗口人工验收。

---

# Documentation

- [x] CURRENT（顶部状态行、Current Version 本地构建段、Recently Completed）
- [x] 发布说明 `docs/releases/v2.0.1-beta.6.md`
- [ ] PRD／Architecture／Tech Stack／Coding Rules／Testing／Build／ADR（本任务无产品范围、架构与流程变化；
      端口相关文档已由 TASK-20261028-008 更新）

---

# Deviations

原计划一次跑通 `build-release.ps1`。实际第一次因改写版本文件时引入 UTF-8 BOM 导致
`format:check` 失败，修正 BOM 后重跑一次通过；构建与校验逻辑本身未改动。

---

# Git

## Commit

```text
88f83d2a chore(release): 2.0.1-beta.6 版本与本地 Windows 安装包
```

（仅本地提交，未推送；安装包未签名、未打标签、未发布，`release-assets/` 与 `bin/` 按 `.gitignore` 不入库。
commit 字段由后续 docs 提交回填。）

---

# Result

- `v2.0.1-beta.6` 本地生产安装包已构建并校验通过：20,315,659 bytes，
  SHA256 `3fee16a3f6d79358c4a1bfdd9da662455b527de435d224a97fe58fa686eb7aa3`，
  位于 `release-assets/v2.0.1-beta.6/OmniProxy-Setup-v2.0.1-beta.6-windows-amd64.exe`。
- 相对 beta.5 只含一项用户可见改动：本地网关端口可配置 + 端口冲突自动避让
  （TASK-20261028-008，`4f09490a`）。
- 安装器未签名、未自动安装、未打标签、未推送；覆盖安装与真实窗口观感由用户执行。

---

# Completed

2026-10-28 23:15
