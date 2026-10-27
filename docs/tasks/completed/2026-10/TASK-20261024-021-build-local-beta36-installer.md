# TASK-20261024-021: 构建 Beta.36 本地生产安装包

## Status

Completed（2026-10-27 随 v2.0.0 正式发布归档）

## Created

2026-10-24 22:35（Asia/Singapore）

## Request

用户要求：构建新的 Beta 生产版本，用于本机覆盖更新（承接 Oxion AI 内置服务商接入）。

## Goal

产出 `release-assets/v2.0.0-beta.36/` 安装器与 SHA256，通过完整质量门、Windows 版本资源与
发布资产校验，供用户手动覆盖更新。

## Background

- 相对 Beta.35 的改动来自已归档的
  [TASK-20261024-020](TASK-20261024-020-builtin-oxionai-provider.md)：
  新增第 12 个内置服务商 Oxion AI（`https://oxionai.com/v1`，`cc_` Bearer Key，
  只有 OpenAI 兼容 Chat Completions，目录按账号从 `GET /v1/models` 同步）。
- 新增迁移 `043_builtin_oxionai.sql`：只追加 `builtin_oxionai` 预设，不改动既有服务商开关、
  账号与凭证引用；已安装用户覆盖后自动执行。
- Beta.35 → Beta.36 之间没有其他代码改动。

## Scope

- 版本同步至 `2.0.0-beta.36`，新增 `docs/releases/v2.0.0-beta.36.md`。
- 完整质量门 → 打包 → Windows 版本资源 → 发布资产与 SHA256 校验。
- 更新 `docs/plans/CURRENT.md` 的当前版本段落。

## Out of Scope

- 不自动安装、不打标签、不推送、不代码签名。
- 不使用真实 Oxion AI Key 发请求；不验收原生窗口视觉。

---

# Implementation Result

- 版本同步至 `2.0.0-beta.36`（`internal/version/version.go`、`frontend/package.json`），
  `scripts/set-release-version.ps1` 校验通过并同步 `build/windows/info.json` 的
  `ProductVersion`（Windows 数字版本保持 `2.0.0.0`，`build/config.yml` 与
  `wails_tools.nsh` 无需变更）。
- 新增发布说明 `docs/releases/v2.0.0-beta.36.md`：Oxion AI 接入、迁移 043、升级说明与已知边界。
- `release-assets/v2.0.0-beta.36/`：
  - `OmniProxy-Setup-v2.0.0-beta.36-windows-amd64.exe` 20,049,995 bytes
  - SHA256 `f9baf279b402865b0a5095ddc406ecf92076936e2289c52c4730f841a4100bab`
- 相对 Beta.35 的 19,831,297 bytes 增加 218,698 bytes（新增 provider 包、测试与品牌图标资源）。
- 发布目录总数 31（Beta.35 之前 30 个），历史资产未变。

# Verification

- PASS：`task --force verify` exit 0（日志 `tmp/verify-beta36.log`；格式／vet／staticcheck／ESLint／
  类型检查、全部 Go 测试、前端 128 文件 / 760 项测试、Windows production 构建，输出 `bin/omniproxy.exe`）。
- PASS：`task --force package` exit 0（日志 `tmp/package-beta36.log`；NSIS 5 页、2 语言表，
  最终安装器 20,049,995 bytes）。
- PASS：`scripts/test-windows-metadata.ps1 -Application bin\omniproxy.exe -Installer bin\OmniProxy-amd64-installer.exe -Version v2.0.0-beta.36`
  →「Windows 应用和安装器版本资源校验通过」。
- PASS：`scripts/verify-release-assets.ps1 -Version v2.0.0-beta.36 -AssetDirectory release-assets\v2.0.0-beta.36`
  →「发布资产校验通过：v2.0.0-beta.36」。
- 未执行：代码签名、自动安装、停止或替换生产进程、打标签、推送、真实向导点击验收、
  真实 Oxion AI Key 请求。

# Remaining Risks

- `scripts/build-release.ps1` 仍不能整体跑通：它用 `$ErrorActionPreference='Stop'` 调用
  `task --force verify`，Taskfile 格式检查片段打印的 `uname`／`tail` 缺失 stderr 会被当成终止错误。
  本轮按 Beta.34／35 记录的等价步骤逐步执行（verify → package → 版本资源 → 资产校验），
  产物与校验结果一致。该脚本兼容性问题需要单独修，不在本任务范围内。
- 打包需先把 `%ProgramFiles(x86)%\NSIS` 加入 PATH，直接调 `task package` 不会。
- 安装器未签名，Windows SmartScreen 仍可能提示；未自动安装、未打标签、未推送。
- Oxion AI 默认启用但没有模型，用户需添加 API Key 并同步目录；真实 Key 与原生窗口视觉未验收。

# Git

## Commit

```text
896b0e7 chore(release): 2.0.0-beta.36 版本与 Windows 安装包元数据
```

版本映射与发布说明随本次 release 提交落库；安装包本身未签名、未推送、未发布。

---

# Result

Beta.36 本地生产安装包已构建并校验通过（20,049,995 bytes／SHA256 `f9baf279…0bab`），
相对 Beta.35 的差异只有 Oxion AI 内置服务商（含迁移 043）；安装器未签名、未自动安装，
供用户本地覆盖更新。

---

# Archival (v2.0.0)

- 原状态：In Progress（安装包已构建并校验，等待用户本地覆盖安装验收）
- 2026-10-27：v2.0.0 正式版已公开（标签源码 `c9c48b4a31988a6cc3fa2f4b1d423e61bd10f71b`，
  Actions 完整质量门与 Windows 构建通过），本任务交付内容已随该版本交付。
- 归档依据：归档时工作区无未提交代码改动，本记录引用的实现提交均为上述发布提交的祖先提交。
- 未执行项（实机客户端／原生窗口视觉／真实覆盖安装验收等）不伪造为已验收，转为发布后
  跟进事项，见 `docs/plans/CURRENT.md` 的 Post-release Follow-ups。
