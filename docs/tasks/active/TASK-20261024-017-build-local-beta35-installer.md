# TASK-20261024-017: 构建 Beta.35 本地生产安装包

## Status

In Progress（安装包已构建并校验，等待用户本地覆盖安装验收）

## Created

2026-10-24 18:35（Asia/Singapore）

## Request

用户要求：构建新的 Beta 生产版本，用于本机覆盖更新（承接 Codex 透明转发判定／归属与自动激活失败原因两批修复）。

## Goal

产出 `release-assets/v2.0.0-beta.35/` 安装器与 SHA256，通过完整质量门、Windows 版本资源与发布资产校验，供用户手动覆盖更新。

## Background

- 相对 Beta.34 的改动来自两个已记录任务，均已在源码中实现并通过定向验证：
  - [TASK-20261024-015](TASK-20261024-015-codex-passthrough-outcome-and-runtime-labels.md)：Codex 透明转发的结束分类按可证实信号（交付终结事件后断开不再记 `client_write_failed`）、未声明 `Content-Type` 的流也能统计用量、WebSocket 失败交回真实代码并按全部订阅账号取号，请求归属细分 `codex-cli`／`codex-desktop`／`codex-vscode`。
  - [TASK-20261024-016](TASK-20261024-016-activation-failure-code-and-websocket-toggle.md)：自动激活按账号保留具体错误码并写日志，面板／提示条显示原因；「Codex WebSocket」开关对已建立的透明转发连接逐帧复核。
- 无新增迁移（不改变数据库结构），`app.json` 结构未变；已安装用户可直接覆盖。

## Scope

- 版本同步至 `2.0.0-beta.35` 并新增 `docs/releases/v2.0.0-beta.35.md`。
- `scripts/build-release.ps1` 执行完整质量门 → 打包 → 版本资源 → 资产与 SHA256 校验。
- 更新 `docs/plans/CURRENT.md`。

## Out of Scope

- 不提交其他在办任务的改动；不自动安装、不打标签、不推送、不代码签名。
- 不在本任务内修复 `acct_bf1bf…` 的空模型目录（需用户同步或重新授权）。

---

# Implementation Result

- 版本同步至 `2.0.0-beta.35`（`internal/version/version.go`、`frontend/package.json`），
  `scripts/set-release-version.ps1` 同步 Windows 数字版本 `2.0.0.0`、
  `build/windows/info.json` 的 `ProductVersion` 与 `build/windows/nsis/wails_tools.nsh`。
- 新增发布说明 `docs/releases/v2.0.0-beta.35.md`（三件修复 + 归属细分 + 自动激活原因 + 已知边界）。
- `release-assets/v2.0.0-beta.35/`：
  - `OmniProxy-Setup-v2.0.0-beta.35-windows-amd64.exe` 19,831,297 bytes
  - SHA256 `be8ad7106e58d7d15652e8be4cca95072d004ffeb909bd5022519a18ca493a4f`
- 安装器 5 个页面、2 个语言表；比 Beta.34 的 19,811,654 bytes 增加 19,643 bytes。
- 发布目录总数 30（Beta.34 之前 29 个），历史资产未变。

# Verification

- PASS：`task --force verify` exit 0（日志 `tmp/verify-beta35.log`；格式／vet／staticcheck／ESLint／类型检查、
  全部 Go 测试、前端全部测试、Windows production 构建，输出 `bin/omniproxy.exe`）。
- PASS：`task --force package` exit 0（日志 `tmp/package-beta35.log`；NSIS 5 页、2 语言表，
  生成 `bin/OmniProxy-amd64-installer.exe` 19,831,297 bytes）。
- PASS：`scripts/test-windows-metadata.ps1 -Application bin\omniproxy.exe -Installer bin\OmniProxy-amd64-installer.exe -Version v2.0.0-beta.35`
  →「Windows 应用和安装器版本资源校验通过」。
- PASS：`scripts/verify-release-assets.ps1 -Version v2.0.0-beta.35 -AssetDirectory release-assets\v2.0.0-beta.35`
  →「发布资产校验通过：v2.0.0-beta.35」。
- 未执行：代码签名、自动安装、停止或替换生产进程、打标签、推送、真实向导点击验收。

# Remaining Risks

- `scripts/build-release.ps1` 在本机不能直接跑通：它用 `$ErrorActionPreference='Stop'` 调用
  `task --force verify`，而 Taskfile 的格式检查片段会打印 `uname`／`tail` 缺失的 stderr，
  PowerShell 把这段 stderr 当成终止错误，脚本在 verify 处中断（`task --force verify` 单独执行是 exit 0）。
  本轮按 Beta.34 记录的等价步骤逐步执行（verify → package → 版本资源 → 资产校验），产物与校验结果一致。
  该脚本兼容性问题需要单独修（例如把格式检查改成不依赖 Unix 工具的写法），不在本任务范围内。
- 打包需先把 `%ProgramFiles(x86)%\NSIS` 加入 PATH（`build-release.ps1` 本来会做，直接调 `task package` 不会）。
- `acct_bf1bf…`（x4107034@gmail.com）的空模型目录仍需用户同步或重新授权后才会激活成功；本版只让原因可见。
- 安装器未签名，Windows SmartScreen 仍可能提示；未自动安装、未打标签、未推送。

# Git

## Commit

```text
9c7d61b chore(release): 2.0.0-beta.35 版本与 Windows 安装包元数据
90681e0 docs: 同步 Codex 透明转发决策、产品文档与 2026-10 任务记录
```

版本映射随 9c7d61b 落库；安装包本身未签名、未推送、未发布。

---
