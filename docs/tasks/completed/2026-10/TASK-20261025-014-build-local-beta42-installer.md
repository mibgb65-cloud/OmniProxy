# TASK-20261025-014: 构建 Beta.42 本地生产安装包

## Status

Completed

## Created

2026-10-25 16:33（Asia/Shanghai）

## Request

> 代码都提交了吗？完成后，构建新的beta生产版本我进行本地更新

## Goal

产出 `release-assets/v2.0.0-beta.42/` 安装器与 SHA256，通过完整质量门、Windows 版本资源与发布资产校验，
供用户手动覆盖更新。

## Background

相对已完成的 Beta.41（`8f38269`）有两个已完成任务：

- [TASK-20261025-012](../completed/2026-10/TASK-20261025-012-background-quota-activation-refresh.md)
  （`4ec3667`、`f038213`）：自动激活常驻后台——开启后关闭主窗口留在托盘，滑动候选在启动／唤醒后
  约 35 秒补一次复检（连续最多 3 次），每轮结束推送 `subscription:activation-updated`，前端在根布局
  自动刷新并提示，不必打开设置面板。
- [TASK-20261025-013](../completed/2026-10/TASK-20261025-013-provider-card-open-accounts.md)
  （`d7a4cec`）：提供商页卡片可进入对应提供商的账号页面，卡片内按钮不受整卡点击影响。

无新增数据库迁移、无新增依赖；Wails 事件绑定已在 TASK-20261025-012 重新生成并提交
（`frontend/bindings/.../eventdata.d.ts` 增加 `subscription:activation-updated`）。

## Scope

- 版本同步至 `2.0.0-beta.42`（`internal/version/version.go`、`frontend/package.json`、
  `build/windows/info.json` 的 `ProductVersion`；Windows 数字版本仍 `2.0.0.0`）。
- 新增 `docs/releases/v2.0.0-beta.42.md` 发布说明。
- 完整质量门 → 打包 → Windows 版本资源 → 发布资产与 SHA256 校验。
- 更新 `docs/plans/CURRENT.md`（Current Version、Recently Completed）。

## Out of Scope

- 不自动安装、不打标签、不推送、不代码签名。
- 不停止或替换用户正在使用的生产程序；安装由用户手动覆盖执行。

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
- `build/windows/info.json` 只会改 `ProductVersion` 一行；用 `set-release-version.ps1` 会以
  `ConvertTo-Json` 重写整个文件造成格式噪音，因此按 Beta.41 的做法逐步手工执行等价步骤。
- 安装器未签名，覆盖安装时 SmartScreen 可能提示未知发布者。

---

# Implementation Plan

1. 版本提升到 `2.0.0-beta.42`（三个文件的最小改动）。
2. 新增 `docs/releases/v2.0.0-beta.42.md`。
3. `task --force verify`（日志 `tmp/verify-beta42.log`）。
4. `task --force package`（NSIS 加入 PATH，日志 `tmp/package-beta42.log`）。
5. `scripts/test-windows-metadata.ps1` 校验应用与安装器版本资源。
6. 产物复制到空的 `release-assets/v2.0.0-beta.42/`，生成 SHA256，`scripts/verify-release-assets.ps1` 校验。
7. 填写 Task Record、更新 `docs/plans/CURRENT.md`、提交并归档。

---

# Progress

- 版本同步至 `2.0.0-beta.42`，Windows 数字版本保持 `2.0.0.0`。
- `docs/releases/v2.0.0-beta.42.md` 已写。
- `task --force verify` exit 0（131 文件／792 项前端测试）。
- `task --force package` exit 0，安装器 20,103,644 bytes。
- Windows 版本资源、发布资产与 SHA256 校验通过。

---

# Implementation Result

- 版本同步至 `2.0.0-beta.42`：`internal/version/version.go`、`frontend/package.json`、
  `build/windows/info.json` 的 `ProductVersion`（Windows 数字版本仍 `2.0.0.0`；
  `build/config.yml` 与 `wails_tools.nsh` 数字版本未变，无需改）。
- 新增发布说明 `docs/releases/v2.0.0-beta.42.md`（自动激活后台常驻与自动刷新、提供商卡片进入
  账号页、升级说明与已知边界）。
- `release-assets/v2.0.0-beta.42/`：
  - `OmniProxy-Setup-v2.0.0-beta.42-windows-amd64.exe` 20,103,644 bytes
  - SHA256 `8b2fe33feac3f68151a7188e9adb98f28adc983106b1334ca541b826d757a3ca`
- 相对 Beta.41 的 20,093,224 bytes 增加 10,420 bytes（后台补检与状态事件、应用级监视器、
  卡片入口与提示文案）。
- `bin/omniproxy.exe` 22,923,776 bytes；安装器 20,103,644 bytes
  （`makensis` 报告 Total size 20,103,644／79,940,802 bytes，25.1%）。

# Verification

- PASS：`task --force verify` exit 0（日志 `tmp/verify-beta42.log`；格式／vet／staticcheck／ESLint／
  类型检查、全部 Go 测试、前端 131 文件／792 项测试、Windows production 构建）。
- PASS：`task --force package` exit 0（日志 `tmp/package-beta42.log`；安装器 20,103,644 bytes）。
- PASS：`scripts/test-windows-metadata.ps1 -Application bin/omniproxy.exe -Installer bin/OmniProxy-amd64-installer.exe -Version v2.0.0-beta.42`
  →「Windows 应用和安装器版本资源校验通过」。
- PASS：`scripts/verify-release-assets.ps1 -Version v2.0.0-beta.42 -AssetDirectory release-assets\v2.0.0-beta.42`
  →「发布资产校验通过：v2.0.0-beta.42」。
- 未执行：代码签名、自动安装、停止或替换生产进程、打标签、推送。

# Remaining Risks

- 安装器未签名，Windows SmartScreen 仍可能提示未知发布者。
- 打包需先把 `C:\Program Files (x86)\NSIS` 加入 PATH（本次按此执行），未整脚本跑
  `scripts/build-release.ps1`。
- 本版两项改动的真实行为仍需覆盖安装后验收：真实账号的后台自动激活、关闭窗口后托盘常驻的长周期
  检测、浅色主题与 80%／140% 字号下的卡片悬停／焦点。

# Git

## Commit

```text
677b243b904b215b9d4365fc9a8d0dbc0a90b84d chore(release): 2.0.0-beta.42 版本与 Windows 安装包元数据
```

仅本地提交，未推送；安装包本身未签名、未发布。

---

# Deviations

- 未运行 `scripts/build-release.ps1`：该脚本会先用 `ConvertTo-Json` 重写
  `build/windows/info.json`，产生整文件格式噪音（Beta.41 记录中已遇到）。本次按等价步骤手工执行，
  只改 `ProductVersion` 一行，diff 保持最小；`set-release-version.ps1` 的校验逻辑由三处版本值
  手工核对替代（`version.go`／`package.json`／`info.json` 一致，Windows 数字版本不变）。

---

# Documentation

- [ ] PRD（本版无产品行为新增，两项改动已在各自任务记录中更新）
- [ ] Architecture（无变化）
- [ ] Tech Stack（无变化）
- [ ] Coding Rules（无变化）
- [ ] Testing（无变化）
- [ ] Build（无变化）
- [ ] ADR（无架构决策变更）
- [x] CURRENT（Current Version 与 Recently Completed）
- [x] 发布说明 `docs/releases/v2.0.0-beta.42.md`

---

# Result

Beta.42 本地生产安装包已构建并校验通过（20,103,644 bytes／SHA256 `8b2fe33f…3ca`），
相对 Beta.41 只含两项改动：自动激活常驻后台并自动刷新结果、提供商卡片可进入对应账号页面；
安装器未签名、未自动安装，供用户本地覆盖更新。

---

# Completed

2026-10-25 16:45（Asia/Shanghai）
