# TASK-20261029-009: 构建本地 v2.1.1-beta.5 生产安装包

## Status

Completed

## Created

2026-10-29 14:40（Asia/Singapore）

## Request

交付可供本机覆盖更新的 Beta 生产版本，包含 TASK-20261029-008 的弹窗／浮层过渡统一。

## Goal

产出 Windows amd64 production 安装包 `v2.1.1-beta.5`，完成质量门、安装器元数据与发布资产校验，
本地提交并归档。

## Background

`v2.1.1-beta.4`（`c8231310`）包含「更新完成」提示与设置窗口／两个更新浮窗的退出动画；
`a226264d` 把其余 12 个弹窗、5 个浮层与全局提示也统一了，尚未出包。直接出 `beta.5`，
让用户一次覆盖 `beta.4` 未安装的部分。

## Scope

- 版本源递增至 `2.1.1-beta.5`，新增对应本地发布说明。
- 发布脚本构建、打包、生成 SHA256 并校验版本资源与资产目录。
- 同步 CURRENT 与本任务记录。

## Out of Scope

自动安装或停止用户运行中的应用、签名、打标签、推送、公开发布、修改业务代码或数据库迁移。

## Existing Behavior

上一份本地产物为 `v2.1.1-beta.4`（20,341,751 bytes）；公开正式版为 `2.1.0`。

---

# Implementation Plan

1. 版本源与 `frontend/package.json` 改为 `2.1.1-beta.5`，新增双语发布说明。
2. 运行 `scripts/build-release.ps1 -Version v2.1.1-beta.5 -OutputDirectory release-assets/v2.1.1-beta.5`。
3. 校验生产 EXE、安装器、SHA256、版本资源与目录内容，还原 `info.json` 为 LF。
4. 回填结果与验证、更新 CURRENT、提交并归档。

---

# Implementation Result

## 版本源

| 位置 | 值 |
| --- | --- |
| `internal/version/version.go` | `2.1.1-beta.5` |
| `frontend/package.json` | `2.1.1-beta.5` |
| `build/windows/info.json` | 文件版本 `2.1.1-beta.5`／产品数字版本 `2.1.1.0` |
| `build/config.yml` / `build/windows/nsis/wails_tools.nsh` | `2.1.1`（不含预发布标识，无需改） |

## 新增发布说明

`docs/releases/v2.1.1-beta.5.md`（中英双语）：全部弹窗与浮层的进出过渡、全局提示退出过渡、
重新打开保持空白草稿、更新完成提示，以及「常驻只影响渲染时机，未打开的弹窗不读设置／订阅会话」的边界说明。

## 构建与产物

命令：`scripts/build-release.ps1 -Version v2.1.1-beta.5 -OutputDirectory release-assets/v2.1.1-beta.5`
（日志 `tmp/release-v2.1.1-beta.5.log`，exit 0）：`set-release-version.ps1` → `task --force verify`
→ `task --force package`（NSIS）→ `test-windows-metadata.ps1` → 复制安装器与 `.sha256` → `verify-release-assets.ps1`。

| 文件 | 大小 |
| --- | --- |
| `release-assets/v2.1.1-beta.5/OmniProxy-Setup-v2.1.1-beta.5-windows-amd64.exe` | 20,343,683 bytes |
| 同上 `.sha256` | 114 bytes |

SHA256：`96df05939fe238f8c386d3d07969a07a1b9a3bfd6a7703b05fd8ca1f2376dfa4`

NSIS 统计：`Total size: 20343683 / 81111160 bytes (25.0%)`。与 `beta.4`（20,341,751 bytes）相差 1,932 bytes，
符合「只多一组过渡类」的预期。

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
| 发布脚本内 `task --force verify` | exit 0 |
| gofmt / prettier `format:check` | 通过 |
| `go vet` / `staticcheck` / `eslint` | 通过 |
| `tsc` typecheck | 通过 |
| Vitest | 145 文件／880 项全部通过 |
| Go 全量包测试 | 通过 |
| Windows production 构建 | 通过 |
| NSIS 打包 | 通过（25.0%） |
| `test-windows-metadata.ps1` | 「Windows 应用和安装器版本资源校验通过」 |
| `verify-release-assets.ps1` | 「发布资产校验通过：v2.1.1-beta.5」 |
| SHA256 独立复算 | 与 `.sha256` 一致（`96df0593…dfa4`） |

未覆盖（需用户实机确认）：原生 WebView2 下的过渡观感、140% 字号、真实覆盖安装，
以及本版首次出现的「更新完成」浮窗实际触发。

---

# Documentation

- 新增 `docs/releases/v2.1.1-beta.5.md`（中英双语）。
- 更新 `docs/plans/CURRENT.md`：顶部摘要、`## Current Version`、`## Recently Completed`。

---

# Deviations

1. 没有先出 `beta.4` 让用户安装：`beta.4` 与本次改动都在前端，直接递增到 `beta.5` 一次覆盖，避免连续安装。
2. Windows 数字版本仍为 `2.1.1.0`（发布脚本按 patch 段映射，预发布标识不进第四段）。
3. `build/windows/info.json` 被发布脚本改写为 CRLF，提交前还原为 LF。

---

# Git

## Commit

```text
80ba6efd chore(release): 构建 2.1.1-beta.5 本地生产安装包
```

归档与 CURRENT.md 回填为后续文档提交。未推送、未打标签、未签名。

---

# Result

`v2.1.1-beta.5` Windows amd64 production 安装包已产出并通过完整质量门（145 文件／880 项前端测试）、
NSIS 打包、Windows 版本资源与发布资产校验；安装器 20,343,683 bytes，
SHA256 `96df05939fe238f8c386d3d07969a07a1b9a3bfd6a7703b05fd8ca1f2376dfa4`。
内容为全部弹窗／浮层的进出过渡与「更新完成」提示，未签名、未自动安装、未打标签、未推送、未公开发布。

---

# Completed

2026-10-29 15:05
