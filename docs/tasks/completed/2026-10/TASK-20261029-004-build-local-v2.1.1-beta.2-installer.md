# TASK-20261029-004: 构建本地 v2.1.1-beta.2 生产安装包

## Status

Completed

## Created

2026-10-29 10:55（Asia/Singapore）

## Request

在 TASK-20261029-003（设置页当前版本更新日志改用 Markdown 渲染）之后，产出可供本机覆盖更新的
Beta 生产版本。

## Goal

交付同时包含升级浮窗布局（TASK-20261029-001）与设置页日志 Markdown 渲染
（TASK-20261029-003）的 Windows amd64 production 安装包，版本 `v2.1.1-beta.2`，
完成质量门、安装器元数据与发布资产校验，本地提交并归档。

## Background

`v2.1.1-beta.1` 已构建（TASK-20261029-002，`9e15ae01`），但不含 TASK-20261029-003
（`ae97d7b8`）的设置页修复。为避免用户安装后仍看到原始 Markdown 记号，重新出包递增至 `beta.2`。

## Scope

- 版本源递增至 `2.1.1-beta.2`，新增对应本地发布说明。
- 发布脚本构建、打包、生成 SHA256 并校验版本资源与资产目录。
- 同步 CURRENT 与本任务记录。

## Out of Scope

自动安装或停止用户运行中的应用、签名、打标签、推送、公开发布、修改业务代码或数据库迁移。

## Existing Behavior

上一份本地产物为 `v2.1.1-beta.1`（20,332,085 bytes）；公开正式版为 `v2.1.0`。

---

# Implementation Plan

1. 版本源与 `frontend/package.json` 改为 `2.1.1-beta.2`，新增双语发布说明。
2. 运行 `scripts/build-release.ps1 -Version v2.1.1-beta.2 -OutputDirectory release-assets/v2.1.1-beta.2`。
3. 校验生产 EXE、安装器、SHA256、版本资源与目录内容，还原 `info.json` 为 LF。
4. 回填结果与验证、更新 CURRENT、提交并归档。

---

# Implementation Result

## 版本源

| 位置 | 值 |
| --- | --- |
| `internal/version/version.go` | `2.1.1-beta.2` |
| `frontend/package.json` | `2.1.1-beta.2` |
| `build/windows/info.json` | 文件版本 `2.1.1-beta.2`／产品数字版本 `2.1.1.0` |
| `build/config.yml` / `build/windows/nsis/wails_tools.nsh` | `2.1.1`（数字段不含预发布标识，与 `beta.1` 相同，无需改） |

`scripts/set-release-version.ps1` 校验 `version.go` 与 `package.json` 一致，并按
`major.minor.patch.N` 生成 Windows 数字版本；`N` 取 0，因此 **`2.1.1.0` 与 `beta.1`
相同**，覆盖安装由 NSIS 的版本号与文件替换完成，不依赖数字版本递增。

## 新增发布说明

`docs/releases/v2.1.1-beta.2.md`（中英双语）：升级浮窗三段式布局与设置页「当前版本更新日志」
共用 Markdown 渲染、链接白名单与 HTML 只作纯文本的安全边界，并注明本版本高于公开 `2.1.0`
与本地 `2.1.1-beta.1`。

## 构建与产物

命令：

```text
scripts/build-release.ps1 -Version v2.1.1-beta.2 -OutputDirectory release-assets/v2.1.1-beta.2
```

过程（日志 `tmp/release-v2.1.1-beta.2.log`，exit 0）：

1. `set-release-version.ps1` 写入版本源（`info.json` 被改写为 CRLF，随后还原为 LF，只留版本值差异）。
2. `task --force verify`：`format:check`（gofmt／prettier）→ `lint`（go vet、staticcheck、eslint）
   → `typecheck` → 测试（Go 全量包 + Vitest）→ Windows production 构建，全部通过。
3. `task --force package`：`wails3 generate syso` → `go build -tags production -trimpath -buildvcs=false -ldflags="-w -s -H windowsgui"`
   → NSIS `makensis` 生成安装器。
4. `test-windows-metadata.ps1` 校验应用与安装器版本资源。
5. 复制安装器与 `.sha256` 到输出目录，`verify-release-assets.ps1` 校验资产。

产物：

| 文件 | 大小 |
| --- | --- |
| `release-assets/v2.1.1-beta.2/OmniProxy-Setup-v2.1.1-beta.2-windows-amd64.exe` | 20,332,270 bytes |
| 同上 `.sha256` | 114 bytes |
| `bin/omniproxy.exe`（production 构建） | 23,241,216 bytes |

SHA256：`2067937c9f62899870fe875287bc3db88c17242a25011b118d260aab3332a07f`

NSIS 统计：`Total size: 20332270 / 81010656 bytes (25.0%)`，数据块优化节省 17976 KiB（约 47.5%）。

与 `beta.1`（20,332,085 bytes）相比仅增加 185 bytes，符合「只多一处前端渲染改动」的预期。

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

| 项目 | 结果 |
| --- | --- |
| `task --force verify` | exit 0 |
| gofmt / prettier `format:check` | 通过 |
| `go vet` / `staticcheck` / `eslint` | 通过 |
| `tsc` typecheck | 通过 |
| Vitest | 143 文件／873 项全部通过（含 Markdown 解析 12 项、渲染 3 项、设置页日志卡片 3 项） |
| Go 全量包测试 | 通过（无测试文件的包仅有 `[no test files]` 提示） |
| Windows production 构建 | 通过 |
| NSIS 打包 | 通过（25.0%） |
| `test-windows-metadata.ps1` | 「Windows 应用和安装器版本资源校验通过」 |
| `verify-release-assets.ps1` | 「发布资产校验通过：v2.1.1-beta.2」 |
| SHA256 独立复算 | 与 `.sha256` 一致（`2067937c…a07f`） |

已知无害噪声：Vite 大 chunk 警告、jsdom `window.scrollTo` 未实现、`tail`／`uname`
在 Windows 缺失；均不影响结果，未跳过或放宽任何检查。

未覆盖（需用户实机确认）：native WebView2 下的浮窗观感、140% 字号、真实覆盖安装与
主客户端联调。隔离 Chromium 截图只代表前端渲染，不作为原生验收结论。

---

# Documentation

- [x] `docs/releases/v2.1.1-beta.2.md`（中英双语）。
- [x] `docs/plans/CURRENT.md`：顶部摘要、`## Current Version`、`## Recently Completed`。
- [ ] PRD / Architecture / Tech Stack / Coding Rules / Testing / Build / ADR：无变更
  （只出包，不改接口、依赖与构建流程）。
- `docs/product/UPDATE.md` 已在 TASK-20261029-001／003 中说明浮窗与设置页共用 Markdown 渲染，
  本任务无产品行为变更，未再改。

---

# Deviations

1. 首次 `task verify` 在末尾 `powershell Remove-item *.syso` 处返回 exit 1 且无错误文本；
   以相同命令重跑得到 exit 0。按瞬时失败记录，未跳过或放宽任何检查项。
2. Windows 数字版本 `2.1.1.0` 与 `beta.1` 相同。发布脚本按 `patch` 位映射，预发布标识不递增
   第四段；保持与既有 Beta 一致，不做手工偏移。
3. `build/windows/info.json` 被发布脚本改写为 CRLF，提交前还原为 LF（仓库 `.gitattributes`
   要求 `eol=lf`），最终 diff 只含版本值。

---

# Git

## Commit

```text
e38fbf96 chore(release): 构建 2.1.1-beta.2 本地生产安装包
```

前置实现提交：`cd67d25b`（TASK-20261029-001）、`ae97d7b8`（TASK-20261029-003）。

任务归档与 CURRENT.md 回填为后续独立文档提交；构建产物、`bin/` 与 `tmp/` 日志未提交。

---

# Result

`v2.1.1-beta.2` Windows amd64 production 安装包已产出并通过完整质量门（143 文件／873 项前端测试）、
NSIS 打包、Windows 版本资源与发布资产校验；安装器 20,332,270 bytes，
SHA256 `2067937c9f62899870fe875287bc3db88c17242a25011b118d260aab3332a07f`。

内容为升级浮窗三段式布局与发布说明／设置页日志的共享 Markdown 渲染，相对公开 `2.1.0`
与本地 `2.1.1-beta.1` 均为递增。未签名、未自动安装、未打标签、未推送、未公开发布，供用户本地覆盖更新。

---

# Completed

2026-10-29 11:20
