# TASK-20261027-016: 构建 v2.0.1-beta.2 本地生产安装包

## Status

Completed

## Created

2026-10-27 20:35（Asia/Shanghai）

## Request

> 构建新的beta生产版本我进行本地更新

## Goal

产出 `release-assets/v2.0.1-beta.2/` 安装器与 SHA256，版本源统一为 `2.0.1-beta.2`
（Windows 数字版本 `2.0.1.0`、NSIS `2.0.1`），通过完整质量门、Windows 版本资源与发布资产校验，
供用户手动覆盖更新。

## Background

自本地构建的 `2.0.1-beta.1`（`3bc4f850`）之后，`main` 上有三项用户可见改动：

- [TASK-20261027-015](TASK-20261027-015-reset-credits-http-display.md)（`edc2edaa`）：
  修复重置卡面板恒显示「官方暂未返回重置卡信息」——额度改走官方 HTTP 接口后重置卡没有被读取。
- [TASK-20261027-013](../completed/2026-10/TASK-20261027-013-request-table-horizontal-scrollbar.md)（`d9a30e6a`）：
  请求列表横向滚动条改为统一细滑块。
- [TASK-20261027-014](../completed/2026-10/TASK-20261027-014-requests-view-internal-scroll.md)（`82c28ed3`）：
  请求页改为表格内部滚动。

另有两笔文档／归档提交（`e5d62944`、`0ebfca15`、`e3ff9a7c`、`c4955868`）。

## Scope

- 版本源：`internal/version/version.go`、`frontend/package.json` 改为 `2.0.1-beta.2`；
  `build/windows/info.json` 的 `ProductVersion` 改为 `2.0.1-beta.2`；
  `build/config.yml`、`build/windows/nsis/wails_tools.nsh` 与 `file_version`／`product_version`
  保持数字版本 `2.0.1`／`2.0.1.0`（同一 patch 的 beta 递增不改数字版本）。
- 新增 `docs/releases/v2.0.1-beta.2.md` 发布说明。
- 完整质量门 → NSIS 打包 → Windows 版本资源校验 → 发布资产与 SHA256 校验。
- 更新 `docs/plans/CURRENT.md`（Current Version 本地构建段、Recently Completed）。

## Out of Scope

- 不打标签、不推送、不发布、不代码签名。
- 不停止、不替换用户正在使用的生产程序；覆盖安装由用户手动执行。
- 不改业务逻辑、数据库迁移、依赖。

## Existing Behavior

版本源为本地构建的 `2.0.1-beta.1`／`2.0.1.0`；`release-assets/` 最新目录为 `v2.0.1-beta.1`，
无 `v2.0.1-beta.2` 目录与发布说明。

## Expected Behavior

版本源一致为 `2.0.1-beta.2`／`2.0.1`／`2.0.1.0`；`docs/releases/v2.0.1-beta.2.md` 存在；
`release-assets/v2.0.1-beta.2/` 只含安装器与 `.sha256`，两者校验通过。

---

# Analysis

## Related Modules

- 版本源 5 处、`docs/releases/`、`release-assets/`
- `scripts/{build-release,set-release-version,test-windows-metadata,verify-release-assets}.ps1`

## Dependencies

- Go、Node 22、pnpm、Task、Wails beta.30、staticcheck、NSIS（`C:\Program Files (x86)\NSIS`）。

## Risks

- 数字版本与 beta.1 相同（`2.0.1.0`）：安装器按同一产品标识覆盖升级，符合此前 beta 递增做法；
  SmartScreen 仍可能提示未知发布者。
- `set-release-version.ps1` 会用 `ConvertTo-Json` 重写 `build/windows/info.json` 并产生 CRLF，
  需按 `.gitattributes` 还原为 LF。

---

# Implementation Plan

1. 建立任务记录。
2. `internal/version/version.go`、`frontend/package.json` 改为 `2.0.1-beta.2`。
3. `pwsh scripts/set-release-version.ps1 -Version v2.0.1-beta.2` 写入数字版本与 `ProductVersion`，
   并把 `info.json` 换行还原为 LF。
4. 新增 `docs/releases/v2.0.1-beta.2.md`。
5. `task --force verify` → `task --force package` → `scripts/test-windows-metadata.ps1` →
   资产复制与 SHA256 → `scripts/verify-release-assets.ps1`。
6. 回填 Task Record、更新 CURRENT、提交并归档。

---

# Progress

- 已确认当前版本源为 `2.0.1-beta.1`／`2.0.1`／`2.0.1.0`，`release-assets/` 最新为 `v2.0.1-beta.1`。

---

# Implementation Result

## Changes

- 版本源同步为 `2.0.1-beta.2`：`internal/version/version.go`、`frontend/package.json`、
  `build/windows/info.json` 的 `ProductVersion`；`build/config.yml`、`build/windows/nsis/wails_tools.nsh`
  与 `file_version`／`product_version` 保持 `2.0.1`／`2.0.1.0`（同一 patch 的 beta 递增不改数字版本，
  脚本校验通过：「已校验 v2.0.1-beta.2，Windows 数字版本 2.0.1.0」）。
- 新增发布说明 `docs/releases/v2.0.1-beta.2.md`：重置卡信息恢复读取、请求页内部滚动与细横向滚动条、
  升级说明与已知边界。
- `bin/omniproxy.exe` 23,149,568 bytes（production，`2.0.1-beta.2` 资源）；
  `bin/OmniProxy-amd64-installer.exe` 20,275,456 bytes（`makensis` Total size 20,275,456／
  80,707,115 bytes，25.1%）。
- `release-assets/v2.0.1-beta.2/`：
  - `OmniProxy-Setup-v2.0.1-beta.2-windows-amd64.exe` 20,275,456 bytes
  - SHA256 `9ebea510011723c6def363fca1062b4b535fc3fb7006113f0bb83f199ce3b15a`

## Files Changed

- `internal/version/version.go`、`frontend/package.json`、`build/windows/info.json`
- `docs/releases/v2.0.1-beta.2.md`（新增）
- `docs/plans/CURRENT.md`
- 本记录

## Technical Notes

- `build/windows/info.json` 由 `scripts/set-release-version.ps1` 写入后产生 CRLF，已按 `.gitattributes`
  还原为 LF，diff 只有 `ProductVersion` 一行。
- 打包首次因 `makensis` 不在 PATH 失败（exit 127）；按 `scripts/build-release.ps1` 的做法把
  `C:\Program Files (x86)\NSIS` 前置到 PATH 后重新打包成功，并删除了上一次遗留的旧安装器。
- `task --force package` 会用 `wails3 generate bindings -clean` 重新生成绑定，构建后
  `frontend/bindings` 无 diff（本次无接口变更）。

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
task --force verify                     exit 0（日志 tmp/verify-v2.0.1-beta.2.log）
  format:check / go vet / staticcheck / ESLint / tsc → PASS
  go test ./internal/...                → PASS
  vitest                                → Test Files 132 passed，Tests 811 passed
  wails3 production build               → bin/omniproxy.exe 23,149,568 bytes
task --force package                    exit 0（日志 tmp/package-v2.0.1-beta.2.log）
  makensis Total size 20,275,456／80,707,115 bytes（25.1%）
scripts/test-windows-metadata.ps1       → 「Windows 应用和安装器版本资源校验通过」
scripts/verify-release-assets.ps1       → 「发布资产校验通过：v2.0.1-beta.2」
```

版本源核对：`2.0.1-beta.2`（version.go、package.json、info.json `ProductVersion`）、
`2.0.1`（config.yml、NSIS `INFO_PRODUCTVERSION`）、`2.0.1.0`（info.json
`file_version`／`product_version`），无 `2.0.1-beta.1` 残留。

未执行：代码签名、自动安装、停止或替换正在运行的生产程序、打标签、推送、公开发布；
未做原生窗口视觉验收与真实账号上的重置卡刷新验收。

---

# Documentation

- [x] CURRENT（Current Version 本地构建段、顶部状态行）
- [x] 发布说明 `docs/releases/v2.0.1-beta.2.md`
- [ ] PRD／Architecture／Tech Stack／Coding Rules／Testing／Build／ADR（无产品范围、架构与流程变化）

---

# Deviations

- 计划直接 `task --force package`；实际首次因 `makensis` 不在 PATH 失败，改为前置 NSIS 目录后重跑，
  与 `scripts/build-release.ps1` 的处理一致，没有跳过任何校验。

---

# Git

## Commit

```text
f2627495 chore(release): 2.0.1-beta.2 版本与本地 Windows 安装包
```

（仅本地提交，未推送；安装包未签名、未打标签、未发布，`release-assets/` 与 `bin/` 按 `.gitignore` 不入库。）

---

# Result

- `v2.0.1-beta.2` 本地生产安装包已构建并校验通过：20,275,456 bytes，
  SHA256 `9ebea510011723c6def363fca1062b4b535fc3fb7006113f0bb83f199ce3b15a`，
  位于 `release-assets/v2.0.1-beta.2/OmniProxy-Setup-v2.0.1-beta.2-windows-amd64.exe`。
- 相对 beta.1 含三项用户可见改动：重置卡信息恢复读取、请求列表细横向滚动条、请求页内部滚动。
- 安装器未签名、未自动安装、未打标签、未推送；覆盖安装与真实账号验收由用户执行。

---

# Completed

2026-10-27 20:52
