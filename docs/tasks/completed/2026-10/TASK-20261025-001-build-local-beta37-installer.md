# TASK-20261025-001: 构建 Beta.37 本地生产安装包

## Status

Completed

## Created

2026-10-25 09:10（Asia/Singapore）

## Request

用户要求：构建新的 Beta 生产版本，用于本机覆盖更新。

## Goal

产出 `release-assets/v2.0.0-beta.37/` 安装器与 SHA256，通过完整质量门、Windows 版本资源与
发布资产校验，供用户手动覆盖更新。

## Background

- 相对 Beta.36 的改动只有已归档的
  [TASK-20261024-022](../completed/2026-10/TASK-20261024-022-passthrough-quota-failover.md)：
  Codex 透明入口（客户端实际使用的 `/codex/v1`）在执行前额度拒绝时换号。
- 无数据库迁移；Beta.36 的迁移 043（Oxion AI 内置服务商）已在用户当前版本中。
- 关键修复：额度拒绝判定上提到 `core/proxy` 共用、透明转首事件 4 KiB 有界预读命中即按真实
  `resets_at` 冷却换号、透明转发改尝试级记账（修掉「换号即 503 `accounting_unavailable`」）、
  冷却遵循上游 `Retry-After`、请求结束后按需去抖刷新额度观察。

## Scope

- 版本同步至 `2.0.0-beta.37`，新增 `docs/releases/v2.0.0-beta.37.md`。
- 完整质量门 → 打包 → Windows 版本资源 → 发布资产与 SHA256 校验。
- 更新 `docs/plans/CURRENT.md` 的当前版本段落。

## Out of Scope

- 不自动安装、不打标签、不推送、不代码签名。
- 不发送真实订阅账号请求；不验收原生窗口视觉与实机 Codex 客户端换号效果。

---

# Implementation Result

- 版本同步至 `2.0.0-beta.37`（`internal/version/version.go`、`frontend/package.json`），
  `scripts/set-release-version.ps1` 校验通过并同步 `build/windows/info.json` 的
  `ProductVersion`（Windows 数字版本保持 `2.0.0.0`；`build/config.yml` 与
  `wails_tools.nsh` 无需变更）。
- 新增发布说明 `docs/releases/v2.0.0-beta.37.md`：额度换号修复、尝试级记账修复、
  冷却边界、按需额度刷新、升级说明与已知边界。
- `release-assets/v2.0.0-beta.37/`：
  - `OmniProxy-Setup-v2.0.0-beta.37-windows-amd64.exe` 20,063,983 bytes
  - SHA256 `0c117ed5564903f7867b615b5308497c13017ba164e907c8047bf11c7e73f365`
- 相对 Beta.36 的 20,049,995 bytes 增加 13,988 bytes（只新增额度判定、预读与按需刷新代码）。
- 发布目录总数 32（Beta.36 之前 31 个），历史资产未变。

# Verification

- PASS：`task --force verify` exit 0（日志 `tmp/verify-beta37.log`；格式／vet／staticcheck／ESLint／
  类型检查、全部 Go 测试、前端 128 文件 / 760 项测试、Windows production 构建，输出 `bin/omniproxy.exe`）。
  首次运行时 pwsh 把 `uname`／`tail` 缺失的 stderr 当成 NativeCommandError，外层管道报 1；
  用 `cmd /c` 重跑拿到真实退出码 0，两轮日志内容一致。
- PASS：`task --force package` exit 0（日志 `tmp/package-beta37.log`；NSIS 5 页、2 语言表，
  最终安装器 20,063,983 bytes）。
- PASS：`scripts/test-windows-metadata.ps1 -Application bin\omniproxy.exe -Installer bin\OmniProxy-amd64-installer.exe -Version v2.0.0-beta.37`
  →「Windows 应用和安装器版本资源校验通过」。
- PASS：`scripts/verify-release-assets.ps1 -Version v2.0.0-beta.37 -AssetDirectory release-assets\v2.0.0-beta.37`
  →「发布资产校验通过：v2.0.0-beta.37」。
- 未执行：代码签名、自动安装、停止或替换生产进程、打标签、推送、真实向导点击验收、
  真实订阅账号的额度耗尽换号验收。

# Remaining Risks

- `scripts/build-release.ps1` 仍不能整体跑通：它用 `$ErrorActionPreference='Stop'` 调用
  `task --force verify`，Taskfile 格式检查片段打印的 `uname`／`tail` 缺失 stderr 会被当成终止错误。
  本轮按 Beta.34／35／36 记录的等价步骤逐步执行（verify → package → 版本资源 → 资产校验）。
- 打包需先把 `%ProgramFiles(x86)%\NSIS` 加入 PATH，直接调 `task package` 不会。
- 安装器未签名，Windows SmartScreen 仍可能提示；未自动安装、未打标签、未推送。
- 额度换号只在网关收到请求时生效；若客户端在发送前自行拦截额度，请求不会到达网关。

# Git

## Commit

```text
db6d331 chore(release): 2.0.0-beta.37 版本与 Windows 安装包元数据
```

版本映射与发布说明随本次 release 提交落库；安装包本身未签名、未推送、未发布。

---

# Result

Beta.37 本地生产安装包已构建并校验通过（20,063,983 bytes／SHA256 `0c117ed5…f365`），
相对 Beta.36 的差异只有 Codex 透明入口的额度换号修复（含尝试级记账与按需额度刷新）；
安装器未签名、未自动安装，供用户本地覆盖更新。

---

# Completed

2026-10-25 09:30
