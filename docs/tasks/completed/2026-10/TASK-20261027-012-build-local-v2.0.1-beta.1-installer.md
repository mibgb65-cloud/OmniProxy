# TASK-20261027-012: 构建 v2.0.1-beta.1 本地生产安装包

## Status

Completed

## Created

2026-10-27 13:20（Asia/Shanghai）

## Request

> 构建新的beta生产版本我进行本地更新，目前应该构建的是v2.0.1的beta1了

## Goal

产出 `release-assets/v2.0.1-beta.1/` 安装器与 SHA256，版本源统一为 `2.0.1-beta.1`
（Windows 数字版本 `2.0.1.0`、NSIS `2.0.1`），通过完整质量门、Windows 版本资源与发布资产校验，
供用户手动覆盖更新。

## Background

`2.0.0` 已于 2026-10-27 公开发布，此后 `main` 上有两项会影响用户可见行为的改动：

- [TASK-20261027-003](TASK-20261027-003-fix-passthrough-token-accounting.md)（`04e053c4`）：
  修复 HTTP 透明转发提前结束尝试导致上游已报告的 Token 在持久化时整笔回滚。
- [TASK-20261027-011](TASK-20261027-011-refresh-openai-and-river-models.md)（`ba5ea28f`）：
  OpenAI 预设对齐官方 Featured models；River 预设四款 → 六款（新增 Qwen3.8-27B-FP8、Kimi-K3）。

另有 `5d9dac4d`（订阅额度测试屏障，仅测试）与若干文档／归档提交。

版本基线此前被 [TASK-20261027-010](TASK-20261027-010-revert-version-baseline.md) 退回 `2.0.0`，
并约定「版本号由用户决定」；本次用户明确要求 `v2.0.1` 的 beta1，故按 `2.0.1-beta.1` 构建。

## Scope

- 版本源：`internal/version/version.go`、`frontend/package.json` 改为 `2.0.1-beta.1`；
  `build/config.yml`、`build/windows/nsis/wails_tools.nsh` 改为数字版本 `2.0.1`；
  `build/windows/info.json` 的 `file_version`／`product_version` 为 `2.0.1.0`、`ProductVersion` 为 `2.0.1-beta.1`。
- 新增 `docs/releases/v2.0.1-beta.1.md` 发布说明。
- 完整质量门 → NSIS 打包 → Windows 版本资源校验 → 发布资产与 SHA256 校验。
- 更新 `docs/plans/CURRENT.md`（Current Version、Recently Completed）。

## Out of Scope

- 不打标签、不推送、不发布、不代码签名。
- 不停止、不替换用户正在使用的生产程序；覆盖安装由用户手动执行。
- 不改业务逻辑、数据库迁移、依赖。

## Existing Behavior

版本源为已发布的 `2.0.0`／`2.0.0.0`；`release-assets/` 最新目录为 `v2.0.0-beta.44`，
无 `v2.0.1-beta.1` 目录与发布说明。

## Expected Behavior

版本源 5 处一致为 `2.0.1-beta.1`／`2.0.1.0`；`docs/releases/v2.0.1-beta.1.md` 存在；
`release-assets/v2.0.1-beta.1/` 只含安装器与 `.sha256`，两者校验通过。

---

# Analysis

## Related Modules

- 版本源 5 处、`docs/releases/`、`release-assets/`
- `scripts/{build-release,set-release-version,test-windows-metadata,verify-release-assets}.ps1`

## Dependencies

- Go 1.26、Node 22、pnpm、Task、Wails beta.30、staticcheck、NSIS（`C:\Program Files (x86)\NSIS`）。

## Risks

- `set-release-version.ps1` 用 `ConvertTo-Json` 重写 `build/windows/info.json`：已实测当前文件
  往返序列化**逐字节一致**，不会产生格式噪音（Beta.42 记录中的旧噪音问题已不复现）。
- 安装器未签名，SmartScreen 可能提示未知发布者。
- 版本线从 `2.0.0-beta.44` 转到 `2.0.1-beta.1`：Windows 数字版本由 `2.0.0.0` 升到 `2.0.1.0`，
  属正常 patch 升级路径，安装身份与数据目录不变。

---

# Implementation Plan

1. 建立任务记录。
2. `internal/version/version.go`、`frontend/package.json` 改为 `2.0.1-beta.1`。
3. 新增 `docs/releases/v2.0.1-beta.1.md`。
4. `pwsh scripts/build-release.ps1 -Version v2.0.1-beta.1 -OutputDirectory release-assets/v2.0.1-beta.1`
   （内部依次执行版本校验／写入、`task --force verify`、`task --force package`、
   Windows 版本资源校验、资产复制与 SHA256、资产校验）。
5. 回填 Task Record、更新 CURRENT、提交并归档。

---

# Progress

- 已确认 `info.json` 的 `ConvertTo-Json` 往返输出与现有文件逐字节相同，可安全使用官方脚本。

---

# Implementation Result

## Changes

- 版本源 5 处同步为 `2.0.1-beta.1`：`internal/version/version.go`、`frontend/package.json` 为
  完整 SemVer；`build/config.yml` 与 `build/windows/nsis/wails_tools.nsh` 为数字版本 `2.0.1`；
  `build/windows/info.json` 的 `file_version`／`product_version` 为 `2.0.1.0`、
  `ProductVersion` 为 `2.0.1-beta.1`（`scripts/set-release-version.ps1` 校验并写入）。
- 新增发布说明 `docs/releases/v2.0.1-beta.1.md`：透明转发 Token 漏记修复、OpenAI 预设与
  River 六款模型、测试屏障、升级说明与已知边界。
- `release-assets/v2.0.1-beta.1/`：
  - `OmniProxy-Setup-v2.0.1-beta.1-windows-amd64.exe` 20,260,640 bytes
  - SHA256 `bbca554220f277ee6bf4d2e4eae747582a20a523da0d18bc5de5ad3aee7dc11c`
- `bin/omniproxy.exe` 23,130,112 bytes（production，`2.0.1-beta.1` 资源）；
  `bin/OmniProxy-amd64-installer.exe` 20,260,640 bytes（`makensis` Total size 20,260,640／
  80,640,988 bytes，25.1%）。

## Files Changed

- `internal/version/version.go`、`frontend/package.json`、`build/config.yml`、
  `build/windows/info.json`、`build/windows/nsis/wails_tools.nsh`
- `docs/releases/v2.0.1-beta.1.md`（新增）
- `docs/plans/CURRENT.md`
- 本记录

## Technical Notes

- `build/windows/info.json` 的 `ConvertTo-Json` 往返序列化与现有文件逐字节一致，因此直接使用
  官方 `set-release-version.ps1` 写入，未采用 Beta.42 的手工替代；写入后该脚本会产生 CRLF，
  已按 `.gitattributes`（`eol=lf`）还原为 LF，diff 只有 3 行版本值。
- 打包仍分步执行（`task --force verify` → `task --force package` → 版本资源校验 → 资产复制与
  SHA256 → 资产校验），未跑 `scripts/build-release.ps1`：该脚本会用默认 `release-assets/` 根目录
  并要求目录为空，分步执行与它等价且能保留每一步日志。
- 版本线从 `2.0.0-beta.44` 转到 `2.0.1-beta.1`：Windows 数字版本 `2.0.0.0` → `2.0.1.0`，
  属正常 patch 升级；安装身份、安装目录与数据目录不变，无新增迁移。

---

# Verification

## Tests

- [x] Unit Tests（`task --force verify` 内 `go test`）
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
task --force verify                     exit 0（日志 tmp/verify-v2.0.1-beta.1.log）
  format:check / go vet / staticcheck / ESLint / tsc → PASS
  go test ./internal/...                → PASS
  vitest                                → Test Files 132 passed，Tests 809 passed
  wails3 production build               → bin/omniproxy.exe 23,130,112 bytes
task --force package                    exit 0（日志 tmp/package-v2.0.1-beta.1.log）
  makensis Total size 20,260,640／80,640,988 bytes（25.1%）
scripts/test-windows-metadata.ps1       → 「Windows 应用和安装器版本资源校验通过」
scripts/verify-release-assets.ps1       → 「发布资产校验通过：v2.0.1-beta.1」
```

版本源核对：`2.0.1-beta.1`（version.go、package.json、info.json `ProductVersion`）、
`2.0.1`（config.yml、NSIS `INFO_PRODUCTVERSION`）、`2.0.1.0`（info.json
`file_version`／`product_version`），三者一致，无 `2.0.0` 残留。

未执行：代码签名、自动安装、停止或替换正在运行的生产程序、打标签、推送、公开发布。
未使用真实 River／OpenAI Key 验证新模型权限与计费，也未做原生窗口视觉验收。

---

# Documentation

- [x] CURRENT（Current Version 增加本地构建段、Recently Completed 追加本条）
- [x] 发布说明 `docs/releases/v2.0.1-beta.1.md`（同时作为应用内「当前版本发行说明」的来源）
- [ ] PRD／Architecture／Tech Stack／Coding Rules／Testing／Build／ADR（无产品范围、架构与流程变化）

---

# Deviations

- 计划用 `scripts/build-release.ps1` 一条命令完成；实际改为分步执行（理由见 Technical Notes），
  校验项与脚本完全一致，没有跳过任何检查。
- 额外把 `build/windows/info.json` 的换行从 CRLF 还原为 LF（脚本写入副作用，`.gitattributes`
  要求 LF），避免后续整文件 diff 噪音。

---

# Git

## Commit

3bc4f850 chore(release): 2.0.1-beta.1 版本与本地 Windows 安装包
（仅本地提交，未推送；安装包未签名、未打标签、未发布。）

---

# Result

- `v2.0.1-beta.1` 本地生产安装包已构建并校验通过：20,260,640 bytes，
  SHA256 `bbca554220f277ee6bf4d2e4eae747582a20a523da0d18bc5de5ad3aee7dc11c`。
- 相对已发布的 2.0.0 含两项用户可见改动：透明转发 Token 漏记修复、OpenAI 预设与 River 六款模型。
- 安装器未签名、未自动安装、未打标签、未推送；覆盖安装与真实账号验收由用户执行。

---

# Completed

2026-10-27 13:20
