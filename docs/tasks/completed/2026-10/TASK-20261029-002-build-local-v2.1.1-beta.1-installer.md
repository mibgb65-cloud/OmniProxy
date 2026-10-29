# TASK-20261029-002: 构建本地 v2.1.1-beta.1 生产安装包

## Status

Completed

## Created

2026-10-29 10:20（Asia/Singapore）

## Request

用户要求提交代码，然后构建新的 Beta 生产版本用于本地更新，并完成任务 Markdown 更新。

## Goal

交付包含升级浮窗布局与发布说明 Markdown 渲染的 Windows amd64 production 安装包，
版本递增至 `v2.1.1-beta.1`，完成质量门、安装器元数据与发布资产校验，本地提交并归档任务记录。

## Background

`v2.1.0` 已于 2026-10-29 公开为正式 Latest。本次唯一用户可见改动为
[TASK-20261029-001](../completed/2026-10/TASK-20261029-001-update-dialog-layout-and-markdown.md)
（升级浮窗三段式布局 + 发布说明 Markdown 渲染），实现提交 `cd67d25b`、归档提交 `5f8e5c48`，
工作区干净。

## Scope

- 更新版本源与人工整理的发布说明。
- 使用既有发布脚本构建、打包并生成 SHA256 校验文件。
- 同步 CURRENT 与本任务记录。

## Out of Scope

自动安装或停止用户运行中的应用、签名、打标签、推送、公开发布、修改业务代码或数据库迁移。

## Existing Behavior

上一份产物为公开正式版 `v2.1.0`（安装器 20,362,016 bytes）；此后仅新增升级浮窗与
Markdown 渲染改动，未改动网关、账号、额度、代理或更新下载链路。

## Analysis

- `scripts/build-release.ps1` 依次执行：`set-release-version.ps1` 校验版本一致并映射 Windows
  数字资源 → 检查发布说明存在 → `task --force verify` → `task --force package` →
  `test-windows-metadata.ps1` → 复制安装器、写 SHA256 → `verify-release-assets.ps1`。
- `set-release-version.ps1` 要求 `internal/version/version.go` 与 `frontend/package.json`
  已是目标完整版本，因此先改版本源再运行脚本。
- 版本选择：`2.1.1-beta.1`，Windows 数字版本 `2.1.1.0`，NSIS 产品版本 `2.1.1`。
  相对公开的 `2.1.0` 为次版本递增后的首个预发布，符合 SemVer 预发布排序。

## Risks

- 发布脚本会用 `ConvertTo-Json` 重写 `build/windows/info.json`，产生无关格式差异
  → 构建后比对并还原原始排版，只保留版本值差异。
- 完整质量门约十余分钟，非零退出需先判定是否为瞬时工具故障再重跑。

---

# Implementation Plan

1. 创建本任务记录，更新 `internal/version/version.go` 与 `frontend/package.json` 为 `2.1.1-beta.1`。
2. 新增 `docs/releases/v2.1.1-beta.1.md`（中英双语，说明浮窗布局与 Markdown 渲染）。
3. 运行 `scripts/build-release.ps1 -Version v2.1.1-beta.1 -OutputDirectory release-assets/v2.1.1-beta.1`。
4. 校验生产 EXE、安装器、SHA256、版本资源与目录内容，还原 `info.json` 排版。
5. 回填结果与验证、更新 CURRENT、本地提交并归档。

---

# Implementation Result

- 版本完整字符串统一为 `2.1.1-beta.1`：`internal/version/version.go`、`frontend/package.json`、
  `build/windows/info.json` 的 `ProductVersion`。
- Windows 数字版本 `2.1.1.0`（`info.json` 的 `file_version`／`product_version`）；
  `build/config.yml` 与 NSIS `INFO_PRODUCTVERSION` 为 `2.1.1`；安装身份与路径语义不变。
- 新增本地发布说明 `docs/releases/v2.1.1-beta.1.md`（中英双语），随包进入「当前版本更新日志」。
- `bin/omniproxy.exe`：23,238,656 bytes，production 构建，产品版本 `2.1.1-beta.1`。
- 本地安装包：
  `release-assets/v2.1.1-beta.1/OmniProxy-Setup-v2.1.1-beta.1-windows-amd64.exe`
  （20,332,085 bytes）。
- 配套 `.exe.sha256`：
  `b029b3742df30bd6d253738891a5e9ff6bb5abdf8a8bdd82293d5e4f80a53a5c`。
- 安装器签名状态为未签名（NotSigned）；未使用项目私钥、未改变用户信任库。
- 相对公开 `2.1.0` 的用户可见改动只有 TASK-20261029-001（升级浮窗三段式布局 + 发布说明
  Markdown 渲染），无新增数据库迁移、无新增依赖。

### Files Changed

- `internal/version/version.go`、`frontend/package.json`、`build/config.yml`、
  `build/windows/info.json`、`build/windows/nsis/wails_tools.nsh`
- `docs/releases/v2.1.1-beta.1.md`
- `docs/plans/CURRENT.md`
- 本任务记录

### Technical Notes

`set-release-version.ps1` 用 `ConvertTo-Json` 重写 `build/windows/info.json`，写出的文件为 CRLF；
已按 `.gitattributes`（`eol=lf`）还原为 LF，最终差异只剩三个版本值。
`bin/`、`release-assets/` 与 `tmp/` 日志按既有规则忽略，不纳入 Git。

---

# Verification

## Quality Checks

- [x] Format
- [x] Lint
- [x] Type Check
- [x] Test
- [x] Build
- [x] Verify

## Verification Result

```text
scripts/build-release.ps1 -Version v2.1.1-beta.1 -OutputDirectory release-assets/v2.1.1-beta.1
PASS（exit 0；日志 tmp/release-v2.1.1-beta.1.log）

task --force verify（由发布脚本执行）
PASS：format / go vet / staticcheck / ESLint / TypeScript / Go tests
Frontend：142 files / 869 tests PASS
Vite production / Windows amd64 production build：PASS

task --force package（由发布脚本执行）
PASS：NSIS UTF-8 打包，Total size 20,332,085 / 81,002,699 bytes（25.1%）

scripts/test-windows-metadata.ps1
PASS：应用完整产品版本 2.1.1-beta.1，安装器产品版本 2.1.1
两者固定 FileVersionNumber / ProductVersionNumber 均为 2.1.1.0

scripts/verify-release-assets.ps1
PASS：目录只含本次安装器及匹配的 SHA256 文件

独立复核
PASS：安装器 SHA256 重新计算为 b029b374…a5c，与 .sha256 文件一致
PASS：对 bin/omniproxy.exe 与发布目录中的安装器重跑版本资源校验

git diff --check
PASS
```

`build/windows/info.json` 已还原 LF；版本源不存在旧 `2.1.0` 字符串。
保留既有 jsdom `window.scrollTo`、Vite 大 chunk 及 Windows 上 `uname`／`tail` 探测提示，
均未导致质量门或打包失败，不做无关修正。

未执行自动安装、真实覆盖安装验收、签名、打标签、推送或公开发布。

---

# Documentation

- [x] 已新增 `docs/releases/v2.1.1-beta.1.md`
- [x] 已更新 `docs/plans/CURRENT.md`
- 无产品范围、架构、技术栈、数据库或构建流程变更；`docs/product/UPDATE.md` 的浮窗描述已在
  TASK-20261029-001 同步，本次不改动。

---

# Deviations

None。发布脚本一次执行通过；仅把脚本写出的 `info.json` CRLF 还原为仓库约定的 LF，
避免无关行尾差异。

---

# Git

## Commit

```text
9e15ae01 chore(release): 构建 2.1.1-beta.1 本地生产安装包
```

任务归档与 CURRENT.md 回填为后续独立文档提交；构建产物、`bin/` 与 `tmp/` 日志未提交。

---

# Result

包含升级浮窗布局与发布说明 Markdown 渲染的 `v2.1.1-beta.1` 本地生产安装包已构建并校验通过，
覆盖更新由用户执行；当前运行中的安装版及真实用户数据未被本任务操作。

---

# Completed

2026-10-29 10:12（Asia/Singapore）
