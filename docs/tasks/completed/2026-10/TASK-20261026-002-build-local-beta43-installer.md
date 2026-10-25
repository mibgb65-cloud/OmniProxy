# TASK-20261026-002: 构建 Beta.43 本地生产安装包

## Status

In Progress

## Created

2026-10-26 20:55（Asia/Shanghai）

## Request

> 代码提交后，构建新的beta生产版本我进行本地更新

## Goal

提交本次改动后产出 `release-assets/v2.0.0-beta.43/` 安装器与 SHA256，通过完整质量门、
Windows 版本资源与发布资产校验，供用户手动覆盖更新。

## Background

相对 Beta.42（安装器 20,103,644 bytes，SHA256 `8b2fe33f…3ca`）新增一个已完成任务：

- [TASK-20261026-001](../completed/2026-10/TASK-20261026-001-provider-outbound-proxy.md)：
  提供商出站代理（迁移 044、Core `internal/core/netproxy`、`internal/application/proxysettings`、
  11 个适配器与网关/探针/透明转发接入、Wails 绑定、提供商页「网络代理」弹窗与卡片徽标）。

新增一条数据库迁移（044），无新增 Go/前端依赖；Wails 绑定已重新生成并提交
（`frontend/bindings/omniproxy/internal/application/proxysettings/`、
`provider/models.ts` 增加 `proxyMode`、`appservice.ts` 增加 5 个方法）。

## Scope

- 提交 Beta.43 之前的全部改动（功能、绑定、测试、文档）。
- 版本同步至 `2.0.0-beta.43`（`internal/version/version.go`、`frontend/package.json`、
  `build/windows/info.json` 的 `ProductVersion`；Windows 数字版本仍 `2.0.0.0`）。
- 新增 `docs/releases/v2.0.0-beta.43.md`。
- 完整质量门 → 打包 → Windows 版本资源 → 发布资产与 SHA256 校验。
- 更新 `docs/plans/CURRENT.md`。

## Out of Scope

- 不自动安装、不打标签、不推送、不代码签名。
- 不停止或替换用户正在运行的生产程序；安装由用户手动覆盖执行。

---

# Analysis

## Related Modules

- `internal/version/version.go`、`frontend/package.json`、`build/windows/info.json`
- `scripts/test-windows-metadata.ps1`、`scripts/verify-release-assets.ps1`
- `docs/releases/`

## Dependencies

- Go 1.26、Node 22、pnpm、Task、Wails beta.30、staticcheck、NSIS（`C:\Program Files (x86)\NSIS`）。

## Risks

- `makensis` 不在 PATH：打包前把 NSIS 目录加入 PATH。
- `build/windows/info.json` 只改 `ProductVersion` 一行，避免 `ConvertTo-Json` 造成的格式噪音。
- 安装器未签名，覆盖安装时 SmartScreen 可能提示未知发布者。
- 迁移 044 会在首次启动新版本时应用，旧版本回退后数据库已升级；覆盖更新前提示用户备份无需强制。

---

# Implementation Plan

1. 提交功能、绑定、文档与任务记录。
2. 版本提升到 `2.0.0-beta.43`（三个文件的最小改动）并新增发布说明。
3. 提交版本与元数据。
4. `task --force verify`（日志 `tmp/verify-beta43.log`）。
5. `task --force package`（NSIS 加入 PATH，日志 `tmp/package-beta43.log`）。
6. `scripts/test-windows-metadata.ps1` 校验版本资源。
7. 产物复制到新的 `release-assets/v2.0.0-beta.43/`，生成 SHA256，
   `scripts/verify-release-assets.ps1` 校验。
8. 填写 Task Record、更新 `docs/plans/CURRENT.md`、提交并归档。

---

# Progress

- 功能、绑定、文档与 TASK-20261026-001 已提交（`8ad73c8`、`fd1ec5c`）。
- 版本同步至 `2.0.0-beta.43`，Windows 数字版本保持 `2.0.0.0`；发布说明已写，提交 `af2cba5`。
- `task --force verify` exit 0（132 文件／806 项前端测试）。
- `task --force package` exit 0，安装器 20,194,569 bytes。
- Windows 版本资源、发布资产与 SHA256 校验通过。

---

# Implementation Result

- 版本同步至 `2.0.0-beta.43`：`internal/version/version.go`、`frontend/package.json`、
  `build/windows/info.json` 的 `ProductVersion`（Windows 数字版本仍 `2.0.0.0`）。
- 新增发布说明 `docs/releases/v2.0.0-beta.43.md`（出站代理能力、mixed 自动识别、三态路由、
  失败语义、迁移 044 与已知边界）。
- `release-assets/v2.0.0-beta.43/`：
  - `OmniProxy-Setup-v2.0.0-beta.43-windows-amd64.exe` 20,194,569 bytes
  - SHA256 `968ca0822e4e8d3bf426363364436341d63e9b3b961695a54b63fe50cc33e7a8`
- 相对 Beta.42 的 20,103,644 bytes 增加 90,925 bytes（出站代理 Core／适配器包装、设置服务与绑定、
  提供商页弹窗与卡片徽标、迁移 044）。
- `bin/omniproxy.exe` 23,059,456 bytes；安装器 Total size 20,194,569／80,393,666 bytes（25.1%）。

# Verification

## Tests

- [x] Unit Tests（Go 全量）
- [x] Integration Tests（适配器／网关／透明转发既有集成用例）
- [ ] E2E Tests（无独立 E2E 套件）

## Quality Checks

- [x] Format
- [x] Lint
- [x] Type Check
- [x] Test
- [x] Build
- [x] Verify

## Verification Result

```text
task --force verify   exit 0（tmp/verify-beta43.log）
  format / vet / staticcheck / ESLint / tsc / go test / vitest / production build 全部通过
  前端 132 文件 / 806 项测试
task --force package  exit 0（tmp/package-beta43.log）
  安装器 20,194,569 bytes（Total size 20,194,569 / 80,393,666 bytes，25.1%）
scripts/test-windows-metadata.ps1   PASS「Windows 应用和安装器版本资源校验通过」
scripts/verify-release-assets.ps1   PASS「发布资产校验通过：v2.0.0-beta.43」
```

- 未执行：代码签名、自动安装、停止或替换生产进程、打标签、推送。

## Remaining Risks

- 安装器未签名，Windows SmartScreen 仍可能提示未知发布者。
- 迁移 044 会在首次启动新版本时应用；回退到 Beta.42 不删除新表，但旧版本不认识代理设置。
- 实机代理端口（mixed／纯 SOCKS／纯 HTTP）与浅色主题弹窗仍需安装后验收。

---

# Documentation

- [x] Release notes（`docs/releases/v2.0.0-beta.43.md`）
- [x] CURRENT（Current Version 与 In Progress）

---

# Deviations

None. 按 Beta.42 的做法手工提升版本，未使用会重写 `info.json` 格式的脚本。

---

# Git

## Commit

```text
8ad73c8 feat(providers): 提供商出站代理支持 mixed/SOCKS5/HTTP 端口
fd1ec5c docs(task): 记录 TASK-20261026-001 并归档
af2cba5 chore(release): 2.0.0-beta.43 版本与 Windows 安装包元数据
<docs-commit> docs(task): 记录 TASK-20261026-002 并归档
```

---

# Result

Beta.43 本地生产安装包已产出并通过质量门、Windows 版本资源、SHA256 与资产校验。用户运行
`release-assets/v2.0.0-beta.43/OmniProxy-Setup-v2.0.0-beta.43-windows-amd64.exe` 覆盖更新，
首次启动会应用迁移 044。

---

# Completed

2026-10-26 21:05