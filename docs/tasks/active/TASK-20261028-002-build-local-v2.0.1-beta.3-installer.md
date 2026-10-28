# TASK-20261028-002: 构建 v2.0.1-beta.3 本地生产安装包

## Status

Completed

## Created

2026-10-28 10:20（Asia/Shanghai）

## Request

> 提交代码，然后构建新的beta生产版本我进行本地更新，记得完成任务md更新

## Goal

产出 `release-assets/v2.0.1-beta.3/` 安装器与 SHA256，版本源统一为 `2.0.1-beta.3`
（Windows 数字版本 `2.0.1.0`、NSIS `2.0.1`），通过完整质量门、Windows 版本资源与发布资产校验，
供用户手动覆盖更新；同时更新发布说明、CURRENT 与任务记录。

## Background

自本地构建的 `2.0.1-beta.2`（`f2627495`）之后，`main` 上只有一项用户可见改动：

- [TASK-20261028-001](../completed/2026-10/TASK-20261028-001-theme-circular-reveal-transition.md)（`63c8e3df`）：
  标题栏亮暗模式按钮改为基于点击位置的圆形扩散（View Transitions API）过渡，不支持时优雅降级。

另有文档／归档提交（`4efca14e`、`6efa367b`）。

## Scope

- 版本源：`internal/version/version.go`、`frontend/package.json` 改为 `2.0.1-beta.3`；
  `build/windows/info.json` 的 `ProductVersion` 同步；`build/config.yml`、
  `build/windows/nsis/wails_tools.nsh` 与 `file_version`／`product_version`
  保持数字版本 `2.0.1`／`2.0.1.0`（同一 patch 的 beta 递增不改数字版本）。
- 新增 `docs/releases/v2.0.1-beta.3.md` 发布说明。
- 完整质量门 → NSIS 打包 → Windows 版本资源校验 → 发布资产与 SHA256 校验。
- 更新 `docs/plans/CURRENT.md`（顶部状态行、Current Version 本地构建段、Recently Completed、
  In Progress 中主题过渡任务的归档状态）。

## Out of Scope

- 不打标签、不推送、不发布、不代码签名。
- 不停止、不替换用户正在使用的生产程序；覆盖安装由用户手动执行。
- 不改业务逻辑、数据库迁移、依赖。

## Existing Behavior

版本源为本地构建的 `2.0.1-beta.2`／`2.0.1`／`2.0.1.0`；`release-assets/` 最新目录为
`v2.0.1-beta.2`，无 `v2.0.1-beta.3` 目录与发布说明。

## Expected Behavior

版本源一致为 `2.0.1-beta.3`／`2.0.1`／`2.0.1.0`；`docs/releases/v2.0.1-beta.3.md` 存在；
`release-assets/v2.0.1-beta.3/` 只含安装器与 `.sha256`，两者校验通过。

---

# Analysis

## Related Modules

- 版本源 5 处、`docs/releases/`、`release-assets/`
- `scripts/{build-release,set-release-version,test-windows-metadata,verify-release-assets}.ps1`

## Dependencies

- Go 1.26.2、Node 22.19.0、pnpm 12.6.0、Task 3.53.1、Wails beta.30、staticcheck、
  NSIS（`C:\Program Files (x86)\NSIS`，脚本自动前置到 PATH）。

## Risks

- 数字版本与 beta.2 相同（`2.0.1.0`）：安装器按同一产品标识覆盖升级，与之前 beta 递增做法一致；
  SmartScreen 仍可能提示未知发布者。
- `set-release-version.ps1` 用 `ConvertTo-Json` 重写 `build/windows/info.json`，可能产生 CRLF，
  需按 `.gitattributes` 还原为 LF。

---

# Implementation Plan

1. 建立任务记录。
2. `internal/version/version.go`、`frontend/package.json` 改为 `2.0.1-beta.3`。
3. 新增 `docs/releases/v2.0.1-beta.3.md`。
4. `pwsh scripts/build-release.ps1 -Version v2.0.1-beta.3`
   （内含 `set-release-version.ps1`、`task --force verify`、`task --force package`、
   `test-windows-metadata.ps1`、资产复制与 SHA256、`verify-release-assets.ps1`）。
5. 核对版本源、`info.json` 换行与 `frontend/bindings` 无 diff。
6. 回填 Task Record、更新 CURRENT、提交并归档。

---

# Progress

- 已确认 `main` 工作区干净、`v2.0.1-beta.2` 为基线，自其之后只有主题过渡一项用户可见改动。
- 工具链核对：Go 1.26.2、Node 22.19.0、pnpm 12.6.0、Task 3.53.1、Wails beta.30、
  staticcheck 均在 PATH；`makensis` 不在 PATH，但 NSIS 目录存在，脚本会自动补。

---

# Implementation Result

## Changes

- 版本源同步为 `2.0.1-beta.3`：`internal/version/version.go`、`frontend/package.json`、
  `build/windows/info.json` 的 `ProductVersion`；`build/config.yml`、`build/windows/nsis/wails_tools.nsh`
  与 `file_version`／`product_version` 保持 `2.0.1`／`2.0.1.0`（同一 patch 的 beta 递增不改数字版本，
  脚本输出「已校验 v2.0.1-beta.3，Windows 数字版本 2.0.1.0」）。
- 新增发布说明 `docs/releases/v2.0.1-beta.3.md`：主题切换圆形扩散过渡、升级说明与已知边界。
- `bin/omniproxy.exe` 23,154,176 bytes（production，`2.0.1-beta.3` 资源）；
  `bin/OmniProxy-amd64-installer.exe` 20,278,174 bytes（`makensis` Total size 20,278,174／
  80,722,111 bytes，25.1%）。
- `release-assets/v2.0.1-beta.3/`：
  - `OmniProxy-Setup-v2.0.1-beta.3-windows-amd64.exe` 20,278,174 bytes
  - SHA256 `b167f19b5017754b3ec8e9e45b0614ef2033ec15f657d47b5aae7f8dbe972a9c`

## Files Changed

- `internal/version/version.go`、`frontend/package.json`、`build/windows/info.json`
- `docs/releases/v2.0.1-beta.3.md`（新增）
- `docs/plans/CURRENT.md`
- 本记录

## Technical Notes

- `build/windows/info.json` 由 `set-release-version.ps1` 经 `ConvertTo-Json` 重写后产生 CRLF，
  已按 `.gitattributes`（`*.json text eol=lf`）还原为 LF，diff 只有 `ProductVersion` 一行。
- `task --force package` 复用 `wails3 generate bindings -clean`，构建后 `frontend/bindings` 无 diff
  （本次无接口变更）；`build/config.yml` 与 `wails_tools.nsh` 数字版本与上一版相同，因此无 diff。
- 安装器内容只有主题过渡这一项前端改动，未引入依赖或迁移。

---

# Verification

## Tests

- [x] Unit Tests（`task --force verify` 内 `go test . ./internal/...`）
- [x] Integration Tests（同上，含 routing／modelcatalog／gateway／agentconfig）
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
scripts/build-release.ps1 -Version v2.0.1-beta.3 -OutputDirectory release-assets/v2.0.1-beta.3
                                        exit 0（日志 tmp/release-v2.0.1-beta.3.log）
  set-release-version.ps1              → 「已校验 v2.0.1-beta.3，Windows 数字版本 2.0.1.0」
  task --force verify                  → format:check / go vet / staticcheck / ESLint / tsc PASS
                                          go test ./internal/... PASS
                                          vitest Test Files 134 passed，Tests 819 passed
  task --force package                 → bin/omniproxy.exe 23,154,176 bytes
                                          makensis Total size 20,278,174／80,722,111 bytes（25.1%）
  test-windows-metadata.ps1            → 「Windows 应用和安装器版本资源校验通过」
  verify-release-assets.ps1            → 「发布资产校验通过：v2.0.1-beta.3」
```

版本源核对：`2.0.1-beta.3`（version.go、package.json、info.json `ProductVersion`）、
`2.0.1`（config.yml、NSIS `INFO_PRODUCTVERSION`）、`2.0.1.0`（info.json
`file_version`／`product_version`），无 `2.0.1-beta.2` 残留。

未执行：代码签名、自动安装、停止或替换正在运行的生产程序、打标签、推送、公开发布；
未做原生窗口内的主题扩散动画人工验收。

---

# Documentation

- [x] CURRENT（顶部状态行、Current Version 本地构建段、Recently Completed、In Progress）
- [x] 发布说明 `docs/releases/v2.0.1-beta.3.md`
- [ ] PRD／Architecture／Tech Stack／Coding Rules／Testing／Build／ADR（无产品范围、架构与流程变化）

---

# Deviations

- 计划一次跑完 `build-release.ps1`；第一次未指定 `-OutputDirectory`，脚本默认写入仓库根的
  `release-assets/`（已存在其它版本目录），在资产复制前以「发布目录非空」拒绝覆盖而退出
  （该轮 verify、package 与 Windows 版本资源校验均已通过）。改为
  `-OutputDirectory release-assets/v2.0.1-beta.3` 重跑完整脚本，未跳过任何校验步骤。

---

# Git

## Commit

```text
<filled after commit>
```

（仅本地提交，未推送；安装包未签名、未打标签、未发布，`release-assets/` 与 `bin/` 按 `.gitignore` 不入库。）

---

# Result

- `v2.0.1-beta.3` 本地生产安装包已构建并校验通过：20,278,174 bytes，
  SHA256 `b167f19b5017754b3ec8e9e45b0614ef2033ec15f657d47b5aae7f8dbe972a9c`，
  位于 `release-assets/v2.0.1-beta.3/OmniProxy-Setup-v2.0.1-beta.3-windows-amd64.exe`。
- 相对 beta.2 只含一项用户可见改动：主题切换圆形扩散过渡（TASK-20261028-001，`63c8e3df`）。
- 安装器未签名、未自动安装、未打标签、未推送；覆盖安装与真实窗口动画观感由用户执行。

---

# Completed

2026-10-28 10:40

