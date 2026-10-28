# TASK-20261028-006: 构建 v2.0.1-beta.5 本地生产安装包

## Status

Completed

## Created

2026-10-28 16:00（Asia/Shanghai）

## Request

> 提交代码，然后构建新的beta生产版本我进行本地更新，记得完成任务md更新

## Goal

产出 `release-assets/v2.0.1-beta.5/` 安装器与 SHA256，版本源统一为 `2.0.1-beta.5`
（Windows 数字版本 `2.0.1.0`、NSIS `2.0.1`），通过完整质量门、Windows 版本资源与发布资产校验，
供用户手动覆盖更新；同时更新发布说明、CURRENT 与任务记录。

## Background

自本地构建的 `2.0.1-beta.4`（`60143132`）之后，`main` 上只有一项用户可见改动：

- [TASK-20261028-005](../completed/2026-10/TASK-20261028-005-accounts-bento-expand-dock.md)（`34953623`）：
  账号概览页改为 Bento 原位展开 + 底部切换 Dock（600ms FLIP）。

另有归档提交（`e3ceff78`）。

## Scope

- 版本源：`internal/version/version.go`、`frontend/package.json` 改为 `2.0.1-beta.5`；
  `build/windows/info.json` 的 `ProductVersion` 同步；`build/config.yml`、
  `build/windows/nsis/wails_tools.nsh` 与 `file_version`／`product_version`
  保持数字版本 `2.0.1`／`2.0.1.0`。
- 新增 `docs/releases/v2.0.1-beta.5.md` 发布说明。
- 完整质量门 → NSIS 打包 → Windows 版本资源校验 → 发布资产与 SHA256 校验。
- 更新 `docs/plans/CURRENT.md`（顶部状态行、Current Version 本地构建段、Recently Completed）。

## Out of Scope

- 不打标签、不推送、不发布、不代码签名。
- 不停止、不替换用户正在使用的生产程序；覆盖安装由用户手动执行。
- 不改业务逻辑、数据库迁移、依赖。

## Existing Behavior

版本源为本地构建的 `2.0.1-beta.4`／`2.0.1`／`2.0.1.0`；`release-assets/` 最新目录为
`v2.0.1-beta.4`，无 `v2.0.1-beta.5` 目录与发布说明。

## Expected Behavior

版本源一致为 `2.0.1-beta.5`／`2.0.1`／`2.0.1.0`；`docs/releases/v2.0.1-beta.5.md` 存在；
`release-assets/v2.0.1-beta.5/` 只含安装器与 `.sha256`，两者校验通过。

---

# Analysis

## Related Modules

- 版本源 5 处、`docs/releases/`、`release-assets/`
- `scripts/{build-release,set-release-version,test-windows-metadata,verify-release-assets}.ps1`

## Dependencies

- Go 1.26.2、Node 22.19.0、pnpm 12.6.0、Task 3.53.1、Wails beta.30、staticcheck、
  NSIS（`C:\Program Files (x86)\NSIS`，脚本自动前置到 PATH）。

## Risks

- 数字版本与 beta.4 相同（`2.0.1.0`）：安装器按同一产品标识覆盖升级；SmartScreen 仍可能提示未知发布者。
- `set-release-version.ps1` 重写 `build/windows/info.json` 会产生 CRLF，需按 `.gitattributes` 还原为 LF。
- `build-release.ps1` 默认 `-OutputDirectory` 是仓库根 `release-assets/`（非空即拒绝），必须显式指定版本目录。
- 上次全量测试在并行负载下偶发过与业务无关的 `stepfun.test.tsx` 超时（重跑通过）；若本次质量门重现，
  按同样方式复核是否为既有偶发。

---

# Implementation Plan

1. 建立任务记录。
2. `internal/version/version.go`、`frontend/package.json` 改为 `2.0.1-beta.5`。
3. 新增 `docs/releases/v2.0.1-beta.5.md`。
4. `scripts/build-release.ps1 -Version v2.0.1-beta.5 -OutputDirectory release-assets/v2.0.1-beta.5`。
5. 核对版本源、`info.json` 换行与 `frontend/bindings` 无 diff。
6. 回填 Task Record、更新 CURRENT、提交并归档。

---

# Progress

- 已确认工作区干净、基线为 `2.0.1-beta.4`，自其之后只有 Bento 账号概览一项用户可见改动。

---

# Implementation Result

## Changes

- 版本源同步为 `2.0.1-beta.5`：`internal/version/version.go`、`frontend/package.json`、
  `build/windows/info.json` 的 `ProductVersion`；`build/config.yml`、`build/windows/nsis/wails_tools.nsh`
  与 `file_version`／`product_version` 保持 `2.0.1`／`2.0.1.0`（脚本输出
  「已校验 v2.0.1-beta.5，Windows 数字版本 2.0.1.0」，两文件无 diff）。
- 新增发布说明 `docs/releases/v2.0.1-beta.5.md`：Bento 原位展开与 Dock、升级说明与已知边界。
- `bin/omniproxy.exe` 23,165,952 bytes（production，`2.0.1-beta.5` 资源）；
  `bin/OmniProxy-amd64-installer.exe` 20,283,688 bytes（`makensis` Total size
  20,283,688／80,760,437 bytes，25.1%）。
- `release-assets/v2.0.1-beta.5/`：
  - `OmniProxy-Setup-v2.0.1-beta.5-windows-amd64.exe` 20,283,688 bytes
  - SHA256 `3f71ca546726f212945c0cc6aab772bbdb7a6a705a35ba57a969c209f6cf2a7e`

## Files Changed

- `internal/version/version.go`、`frontend/package.json`、`build/windows/info.json`
- `docs/releases/v2.0.1-beta.5.md`（新增）
- `docs/plans/CURRENT.md`
- 本记录

## Technical Notes

- `build/windows/info.json` 经 `set-release-version.ps1` 重写后为 CRLF，已按 `.gitattributes`
  还原为 LF，diff 只有 `ProductVersion` 一行。
- 一次跑通：显式传 `-OutputDirectory release-assets/v2.0.1-beta.5`，未出现重跑。
- `task --force package` 重新生成 `frontend/bindings` 后无 diff（本次无接口变更）。

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
scripts/build-release.ps1 -Version v2.0.1-beta.5 -OutputDirectory release-assets/v2.0.1-beta.5
                                        exit 0（日志 tmp/release-v2.0.1-beta.5.log）
  set-release-version.ps1              → 「已校验 v2.0.1-beta.5，Windows 数字版本 2.0.1.0」
  task --force verify                  → format:check / go vet / staticcheck / ESLint / tsc PASS
                                          go test ./internal/... PASS
                                          vitest Test Files 135 passed，Tests 826 passed
  task --force package                 → bin/omniproxy.exe 23,165,952 bytes
                                          makensis Total size 20,283,688／80,760,437 bytes（25.1%）
  test-windows-metadata.ps1            → 「Windows 应用和安装器版本资源校验通过」
  verify-release-assets.ps1            → 「发布资产校验通过：v2.0.1-beta.5」
```

版本源核对：`2.0.1-beta.5`（version.go、package.json、info.json `ProductVersion`）、
`2.0.1`（config.yml、NSIS `INFO_PRODUCTVERSION`）、`2.0.1.0`（info.json
`file_version`／`product_version`），无 `2.0.1-beta.4` 残留。

未执行：代码签名、自动安装、停止或替换正在运行的生产程序、打标签、推送、公开发布；
未做原生窗口内的 Bento 展开观感人工验收。

---

# Documentation

- [x] CURRENT（顶部状态行、Current Version 本地构建段、Recently Completed）
- [x] 发布说明 `docs/releases/v2.0.1-beta.5.md`
- [ ] PRD／Architecture／Tech Stack／Coding Rules／Testing／Build／ADR（无产品范围、架构与流程变化）

---

# Deviations

None.（按计划一次跑通 `build-release.ps1`；本次全量质量门未重现上次的 `stepfun.test.tsx` 偶发超时。）

---

# Git

## Commit

```text
<filled after commit>
```

（仅本地提交，未推送；安装包未签名、未打标签、未发布，`release-assets/` 与 `bin/` 按 `.gitignore` 不入库。）

---

# Result

- `v2.0.1-beta.5` 本地生产安装包已构建并校验通过：20,283,688 bytes，
  SHA256 `3f71ca546726f212945c0cc6aab772bbdb7a6a705a35ba57a969c209f6cf2a7e`，
  位于 `release-assets/v2.0.1-beta.5/OmniProxy-Setup-v2.0.1-beta.5-windows-amd64.exe`。
- 相对 beta.4 只含一项用户可见改动：账号概览页 Bento 原位展开与底部切换 Dock（TASK-20261028-005，`34953623`）。
- 安装器未签名、未自动安装、未打标签、未推送；覆盖安装与真实窗口观感由用户执行。

---

# Completed

2026-10-28 16:20

