# TASK-20261024-014: 构建 Beta.34 本地生产安装包

## Status

Completed（2026-10-27 随 v2.0.0 正式发布归档）

## Created

2026-10-24 16:55

## Request

用户要求：构建新的 Beta 生产版本，供本机覆盖更新（承接安装向导改造）。

## Goal

产出 `release-assets/v2.0.0-beta.34/` 安装器与 SHA256，通过完整质量门、Windows 版本资源与发布资产校验。

## Background

- 相对 Beta.33 的差异只有 TASK-20261024-013：向导加入语言选择、自定义文案双语、数据目录页读取当前实际位置并在改动前警告，写入改走辅助程序的 `--omniproxy-installer-data-dir-*` 模式。
- 无数据库迁移；`app.json` 结构未变。

## Scope

- 版本同步至 `2.0.0-beta.34` 并新增 `docs/releases/v2.0.0-beta.34.md`。
- 完整质量门 → 打包 → 版本资源 → 资产与 SHA256 校验。
- 更新 `docs/plans/CURRENT.md`。

## Out of Scope

- 不提交其他在办任务的改动；不自动安装、不打标签、不推送、不代码签名。

---

# Implementation Result

- 版本同步至 `2.0.0-beta.34`（`internal/version/version.go`、`frontend/package.json`，
  `scripts/set-release-version.ps1` 同步 Windows 数字版本 2.0.0.0 与
  `build/windows/info.json` 的 ProductVersion）。
- 新增发布说明 `docs/releases/v2.0.0-beta.34.md`。
- `release-assets/v2.0.0-beta.34/`：
  - `OmniProxy-Setup-v2.0.0-beta.34-windows-amd64.exe` 19,811,654 bytes
  - SHA256 `d338e703f4985654ac155494790f9acf8fb19e3ca5530f8207c6818f999b6f8d`
- 安装器 5 个页面、2 个语言表；比 Beta.33 的 19,796,083 bytes 增加 15,571 bytes。
- 发布目录总数 29（Beta.33 之前 28 个），历史资产未变。

# Verification

- PASS：`task --force verify` exit 0（126 文件 / 751 项前端测试、Go 全部包、Windows production 构建；日志 `tmp/verify-beta34.log`）。
- PASS：`task --force package` exit 0（日志 `tmp/package-beta34.log`）。
- PASS：`scripts/test-windows-metadata.ps1 -Application bin\omniproxy.exe -Installer release-assets\v2.0.0-beta.34\OmniProxy-Setup-v2.0.0-beta.34-windows-amd64.exe -Version v2.0.0-beta.34` →「Windows 应用和安装器版本资源校验通过」。
- PASS：`scripts/verify-release-assets.ps1 -Version v2.0.0-beta.34 -AssetDirectory release-assets\v2.0.0-beta.34` →「发布资产校验通过：v2.0.0-beta.34」。
- 未执行：代码签名、自动安装、停止或替换生产进程、打标签、推送、真实向导点击验收。

# Result

Beta.34 production 本地安装包已构建并校验通过，未签名、未自动安装，供用户本地覆盖更新。相对 Beta.33 的改动是安装向导（语言选择 + 数据目录页重做）。

---

# Archival (v2.0.0)

- 原状态：In Progress（安装包已构建并校验，等待用户本地覆盖安装验收）
- 2026-10-27：v2.0.0 正式版已公开（标签源码 `c9c48b4a31988a6cc3fa2f4b1d423e61bd10f71b`，
  Actions 完整质量门与 Windows 构建通过），本任务交付内容已随该版本交付。
- 归档依据：归档时工作区无未提交代码改动，本记录引用的实现提交均为上述发布提交的祖先提交。
- 未执行项（实机客户端／原生窗口视觉／真实覆盖安装验收等）不伪造为已验收，转为发布后
  跟进事项，见 `docs/plans/CURRENT.md` 的 Post-release Follow-ups。
