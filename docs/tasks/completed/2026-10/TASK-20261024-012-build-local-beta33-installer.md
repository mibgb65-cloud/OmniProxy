# TASK-20261024-012: 构建 Beta.33 本地生产安装包

## Status

Completed（2026-10-27 随 v2.0.0 正式发布归档）

## Created

2026-10-24 14:25

## Request

用户要求：构建新的 Beta 生产版本，供本机覆盖更新。

## Goal

产出 `release-assets/v2.0.0-beta.33/` 安装器与 SHA256，通过完整质量门、Windows 版本资源与发布资产校验。

## Background

- 相对 Beta.32（2026-10-24 11:49 构建）只有两项代码变更，均为本轮已提交任务：
  - TASK-20261024-010：通用设置「桌面行为」两个开关统一使用共享 `Switch`（颜色、尺寸、滑块一致）。
  - TASK-20261024-011：安装向导新增「数据存储位置」页；数据目录解析顺序改为 `OMNIPROXY_DATA_DIR` > `app.json` > 安装选择（`data-dir.txt`）> 默认。
- 按文件修改时间核对：11:49:59 之后除上述两处代码与文档外，无其他源码改动，其余在办任务改动均已包含在 Beta.32 中。
- 无新增数据库迁移；`app.json` 结构未变。

## Scope

- 版本同步至 `2.0.0-beta.33` 并新增 `docs/releases/v2.0.0-beta.33.md`。
- 完整质量门 → 打包 → 版本资源 → 资产与 SHA256 校验。
- 更新 `docs/plans/CURRENT.md`。

## Out of Scope

- 不提交其他在办任务的改动；不自动安装、不打标签、不推送、不代码签名。

---

# Implementation Result

- 版本同步至 `2.0.0-beta.33`：`internal/version/version.go`、`frontend/package.json`，
  随后由 `scripts/set-release-version.ps1 -Version v2.0.0-beta.33` 同步 Windows
  数字版本（`build/config.yml` 与 `build/windows/nsis/wails_tools.nsh` 的数字版本
  仍为 2.0.0，未变；`build/windows/info.json` 的 ProductVersion 更新为
  `2.0.0-beta.33`，并按脚本 `ConvertTo-Json` 格式重写）。
- 新增发布说明 `docs/releases/v2.0.0-beta.33.md`：安装时可选数据存储位置、
  桌面行为开关统一两节，含解析顺序、静默升级不改写、多用户与既有 `app.json`
  选择的已知边界。
- `release-assets/v2.0.0-beta.33/`：
  - `OmniProxy-Setup-v2.0.0-beta.33-windows-amd64.exe` 19,796,083 bytes
  - SHA256 `0b6a18bce6c3e74000be86a877a917dce744db1efe3cb6be48b2211a828332cf`
  - 同名 `.sha256` 内容为 `<hash>  <filename>`（无尾随换行），共两个文件。
- 安装器仍为 5 个向导页（Welcome／安装目录／数据存储位置／安装／完成），比 Beta.32
  的 19,779,433 bytes 增加 16,650 bytes。
- 发布目录总数 28（Beta.32 之前 27 个），历史资产未变。

# Verification

- PASS：`task --force verify` exit 0（126 文件 / 751 项前端测试、Go 全部包、
  Windows production 构建；日志 `tmp/verify-beta33.log`）。
- PASS：`task --force package` exit 0（日志 `tmp/package-beta33.log`）。
- PASS：`scripts/test-windows-metadata.ps1 -Application bin\omniproxy.exe
  -Installer release-assets\v2.0.0-beta.33\OmniProxy-Setup-v2.0.0-beta.33-windows-amd64.exe
  -Version v2.0.0-beta.33` 输出「Windows 应用和安装器版本资源校验通过」。
- PASS：`scripts/verify-release-assets.ps1 -Version v2.0.0-beta.33
  -AssetDirectory release-assets\v2.0.0-beta.33` 输出「发布资产校验通过：v2.0.0-beta.33」。
- 未执行：代码签名、自动安装、停止或替换生产进程、打标签、推送、真实安装向导点击验收。
- 环境说明：本机需要 `C:\Program Files\Git\usr\bin`（`uname`）、
  `C:\Program Files (x86)\NSIS`（`makensis`）与 `E:\nodejs`（`pnpm`）在 PATH 中，
  否则 `format:check` 报 `uname` 缺失、`package` 以 127 退出。

# Result

Beta.33 production 本地安装包已构建并校验通过，未签名、未自动安装，供用户本地覆盖更新。相对 Beta.32 仅有安装时数据位置选择与桌面行为开关统一两处变化。

---

# Archival (v2.0.0)

- 原状态：In Progress（安装包已构建并校验，等待用户本地覆盖安装验收）
- 2026-10-27：v2.0.0 正式版已公开（标签源码 `c9c48b4a31988a6cc3fa2f4b1d423e61bd10f71b`，
  Actions 完整质量门与 Windows 构建通过），本任务交付内容已随该版本交付。
- 归档依据：归档时工作区无未提交代码改动，本记录引用的实现提交均为上述发布提交的祖先提交。
- 未执行项（实机客户端／原生窗口视觉／真实覆盖安装验收等）不伪造为已验收，转为发布后
  跟进事项，见 `docs/plans/CURRENT.md` 的 Post-release Follow-ups。
