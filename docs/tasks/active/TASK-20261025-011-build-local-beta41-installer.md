# TASK-20261025-011: 构建 Beta.41 本地生产安装包

## Status

Completed

## Created

2026-10-25 12:56（Asia/Singapore）

## Request

> 构建新的beta生产版本我进行本地更新

## Goal

产出 `release-assets/v2.0.0-beta.41/` 安装器与 SHA256，通过完整质量门、Windows 版本资源与发布资产校验，供用户手动覆盖更新。

## Background

相对已发布的 Beta.40（`33a382e`）有两个已完成的界面任务：

- [TASK-20261025-009](../completed/2026-10/TASK-20261025-009-sidebar-full-height-brand-header.md)（`7ae0f22`）：侧栏改为一体式通顶布局，标题栏不再横跨全屏左侧品牌，全窗口只在左上角保留一个品牌标识，收展按钮移到品牌栏右侧。
- [TASK-20261025-010](../completed/2026-10/TASK-20261025-010-account-in-use-indicator.md)（`a3b288f`）：River 账号卡片用「正在使用」圆环替代会闪烁的「包含进行中或用量未完整返回的请求…」说明行，卡片高度稳定；判据换成新增的 `pendingRequests`。

无新增数据库迁移、无新增依赖；Wails 绑定已随 TASK-20261025-010 重新生成并提交。

## Scope

- 版本同步至 `2.0.0-beta.41`（`internal/version/version.go`、`frontend/package.json`，并同步 `build/windows/info.json` 元数据）。
- 新增 `docs/releases/v2.0.0-beta.41.md` 发布说明。
- 完整质量门 → 打包 → Windows 版本资源 → 发布资产与 SHA256 校验。
- 更新 `docs/plans/CURRENT.md`（Current Version、Recently Completed）。

## Out of Scope

- 不自动安装、不打标签、不推送、不代码签名。
- 不停止或替换用户正在使用的生产程序；安装由用户手动覆盖执行。

---

# Analysis

## Related Modules

- `internal/version/version.go`、`frontend/package.json`、`build/config.yml`、`build/windows/info.json`、`build/windows/nsis/wails_tools.nsh`
- `scripts/set-release-version.ps1`、`scripts/test-windows-metadata.ps1`、`scripts/verify-release-assets.ps1`
- `docs/releases/`

## Dependencies

- Go 1.26、Node 22、pnpm、Task、Wails beta.30、staticcheck、NSIS（`makensis`）。

## Risks

- 打包需要 `makensis` 在 PATH；历史记录中需手动加入 `%ProgramFiles(x86)%\NSIS`。
- 版本号散落在 4 个文件，漏改会导致版本资源或前端显示不一致；用 `set-release-version.ps1` 校验。
- 安装器未签名，覆盖安装时 SmartScreen 可能提示未知发布者。

---

# Implementation Plan

1. 版本提升到 `2.0.0-beta.41`，运行 `scripts/set-release-version.ps1 -Version v2.0.0-beta.41` 校验并同步元数据。
2. 新增 `docs/releases/v2.0.0-beta.41.md`。
3. `task --force verify`（完整质量门，日志写 `tmp/`）。
4. `task --force package` 生成安装器（`makensis` 加入 PATH）。
5. `scripts/test-windows-metadata.ps1` 校验应用与安装器版本资源。
6. 把产物复制到空的 `release-assets/v2.0.0-beta.41/`，生成 SHA256，`scripts/verify-release-assets.ps1` 校验。
7. 填写 Task Record、更新 `docs/plans/CURRENT.md`、提交并归档。

---

# Progress

- 版本同步至 `2.0.0-beta.41`，Windows 数字版本保持 `2.0.0.0`。
- `docs/releases/v2.0.0-beta.41.md` 已写。
- `task --force verify` exit 0。
- `task --force package` exit 0，安装器 20,093,224 bytes。
- Windows 版本资源、发布资产与 SHA256 校验通过。

---

# Implementation Result

- 版本同步至 `2.0.0-beta.41`（`internal/version/version.go`、`frontend/package.json`，UTF-8 无 BOM），
  `scripts/set-release-version.ps1` 校验通过并同步 `build/windows/info.json` 的 `ProductVersion`
  （Windows 数字版本仍 `2.0.0.0`；`build/config.yml` 与 `wails_tools.nsh` 无需变化）。
- 新增发布说明 `docs/releases/v2.0.0-beta.41.md`（侧栏通顶与「正在使用」圆环两项界面改动、升级说明、已知边界）。
- `release-assets/v2.0.0-beta.41/`：
  - `OmniProxy-Setup-v2.0.0-beta.41-windows-amd64.exe` 20,093,224 bytes
  - SHA256 `f78bbef2e5536ddab8466e6638ae681563d2b48bad2ad2bb8c8e3f00f156f8d4`
- 相对 Beta.40 的 20,090,750 bytes 增加 2,474 bytes（侧栏布局合并、圆环组件与共享用量 Hook）。
- `bin/omniproxy.exe` 22,908,416 bytes；安装器 20,093,224 bytes（`makensis` 报告 Total size 20,093,224／79,889,727 bytes，25.1%）。

# Verification

- PASS：`task --force verify` exit 0（日志 `tmp/verify-beta41.log`；格式／vet／staticcheck／ESLint／
  类型检查、全部 Go 测试、前端 129 文件／785 项测试、Windows production 构建）。
- PASS：`task --force package` exit 0（日志 `tmp/package-beta41.log`；安装器 20,093,224 bytes）。
- PASS：`scripts/test-windows-metadata.ps1 -Application bin/omniproxy.exe -Installer bin/OmniProxy-amd64-installer.exe -Version v2.0.0-beta.41`
  →「Windows 应用和安装器版本资源校验通过」。
- PASS：`scripts/verify-release-assets.ps1 -Version v2.0.0-beta.41 -AssetDirectory release-assets\v2.0.0-beta.41`
  →「发布资产校验通过：v2.0.0-beta.41」。
- 未执行：代码签名、自动安装、停止或替换生产进程、打标签、推送。

# Remaining Risks

- 安装器未签名，Windows SmartScreen 仍可能提示未知发布者。
- 打包需先把 `%ProgramFiles(x86)%\NSIS` 加入 PATH（本次按此执行）；`scripts/build-release.ps1` 仍按等价步骤逐步手动执行，未整脚本跑通。
- Beta.41 的两项改动只在隔离数据目录的原生窗口中做过视觉核对，未覆盖浅色主题与 80%／140% 字号；覆盖安装后的真实数据表现需用户验收。

# Git

## Commit

```text
待填写
```

仅本地提交，未推送；安装包本身未签名、未发布。

---

# Deviations

- `build/windows/info.json` 经 `set-release-version.ps1`（Windows PowerShell 5.1 的 `ConvertTo-Json`）写回后带 CRLF 与 BOM 风险，已手工归一为 UTF-8 无 BOM + LF，避免 `.gitattributes` 归一化产生额外改动噪音；脚本未修改（超出本任务范围，属既有行为）。
- `release-assets/*.sha256` 首次写出带 UTF-8 BOM 与 CRLF（`Out-File` 默认），已按既有资产格式（无 BOM、LF）重写后再校验。

---

# Result

Beta.41 本地生产安装包已构建并校验通过（20,093,224 bytes／SHA256 `f78bbef2…8d4`），
相对 Beta.40 只含两项界面改动：侧栏一体通顶与单一品牌、River 账号卡片「正在使用」圆环替代闪烁提示；
安装器未签名、未自动安装，供用户本地覆盖更新。

---

# Completed

2026-10-25 12:58（Asia/Singapore）
