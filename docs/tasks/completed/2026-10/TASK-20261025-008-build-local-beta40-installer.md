# TASK-20261025-008: 构建 Beta.40 本地生产安装包

## Status

Completed

## Created

2026-10-25 11:55（Asia/Singapore）

## Request

> 构建新的beta生产版本我进行本地更新

## Goal

产出 `release-assets/v2.0.0-beta.40/` 安装器与 SHA256，通过完整质量门、Windows 版本资源与发布资产校验，供用户手动覆盖更新。

## Background

相对 Beta.39 的改动是 [TASK-20261025-007](../completed/2026-10/TASK-20261025-007-codex-quota-http-fast-path-and-retention.md)（实现提交 `5ce4bfc`）：

- Codex 订阅额度优先用官方 HTTP 额度接口（`/backend-api/wham/usage`）读取，不再为每次刷新启动 `codex app-server` 子进程；401/403 直接返回未授权，传输／协议失败与需要续期才回退 stdio。
- 显示快照写入失败降级为告警，不再否决已读到的额度。
- 前端额度改为 keep-last-good：瞬时失败在最近一次成功后 10 分钟内继续显示上次结果并标注，确定性失败立即透出。

无新增数据库迁移、无新增依赖、无绑定签名变化。

## Scope

- 版本同步至 `2.0.0-beta.40`，新增 `docs/releases/v2.0.0-beta.40.md`。
- 完整质量门 → 打包 → Windows 版本资源 → 发布资产与 SHA256 校验。
- 更新 `docs/plans/CURRENT.md`。

## Out of Scope

- 不自动安装、不打标签、不推送、不代码签名。
- 不做真实账号的额度接口联调（需要用户在本机验收）。

---

# Implementation Result

- 版本同步至 `2.0.0-beta.40`（`internal/version/version.go`、`frontend/package.json`，UTF-8 无 BOM），
  `scripts/set-release-version.ps1` 校验通过并同步 `build/windows/info.json` 的 `ProductVersion`
  （Windows 数字版本仍 `2.0.0.0`；`build/config.yml` 与 `wails_tools.nsh` 无需变化）。
- 新增发布说明 `docs/releases/v2.0.0-beta.40.md`。
- `release-assets/v2.0.0-beta.40/`：
  - `OmniProxy-Setup-v2.0.0-beta.40-windows-amd64.exe` 20,090,750 bytes
  - SHA256 `b9000218bfb7360f77797acfb561d9c51fe10e17f387a9cf9737b6f697cccb3d`
- 相对 Beta.39 的 20,075,276 bytes 增加 15,474 bytes（HTTP 额度客户端、keep-last-good 展示策略及其测试）。
- `bin/omniproxy.exe` 22,904,320 bytes；安装器 20,090,750 bytes（`makensis` 报告 Total size 20,090,750／79,875,991 bytes，25.1%）。

# Verification

- PASS：`task --force verify` exit 0（日志 `tmp/verify-beta40.log`；格式／vet／staticcheck／ESLint／
  类型检查、全部 Go 测试、前端 129 文件／780 项测试、Windows production 构建）。
- PASS：`task --force package` exit 0（日志 `tmp/package-beta40.log`；安装器 20,090,750 bytes）。
- PASS：`scripts/test-windows-metadata.ps1 -Application bin/omniproxy.exe -Installer bin/OmniProxy-amd64-installer.exe -Version v2.0.0-beta.40`
  →「Windows 应用和安装器版本资源校验通过」。
- PASS：`scripts/verify-release-assets.ps1 -Version v2.0.0-beta.40 -AssetDirectory release-assets\v2.0.0-beta.40`
  →「发布资产校验通过：v2.0.0-beta.40」。
- 未执行：代码签名、自动安装、停止或替换生产进程、打标签、推送、真实账号额度接口联调。

# Remaining Risks

- 安装器未签名，Windows SmartScreen 仍可能提示未知发布者。
- 打包需先把 `%ProgramFiles(x86)%\NSIS` 加入 PATH（本次按此执行）。
- `scripts/build-release.ps1` 仍按等价步骤逐步手动执行，未整脚本跑通。
- 官方 HTTP 额度接口的返回形状需要用户在本机用真实账号验收；失败时按设计回退原有链路。

# Git

## Commit

```text
33a382ef717cd604132a2b5007dce9076ee7a458 chore(release): 2.0.0-beta.40 版本与 Windows 安装包元数据
```

仅本地提交，未推送；安装包本身未签名、未发布。本记录随后归档到 `docs/tasks/completed/2026-10/`。

---

# Result

Beta.40 本地生产安装包已构建并校验通过（20,090,750 bytes／SHA256 `b9000218…b3d`），
相对 Beta.39 只有额度链路改动（官方 HTTP 接口快路径 + keep-last-good 展示）；安装器未签名、
未自动安装，供用户本地覆盖更新。
