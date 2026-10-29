# TASK-20261029-007: 构建本地 v2.1.1-beta.4 生产安装包

## Status

Completed

## Created

2026-10-29 13:05（Asia/Singapore）

## Request

交付可供本机覆盖更新的 Beta 生产版本，包含刚完成的「更新完成」提示（TASK-20261029-005）
与弹窗退出动画（TASK-20261029-006）。

## Goal

产出 Windows amd64 production 安装包 `v2.1.1-beta.4`，完成质量门、安装器元数据与发布资产校验，
本地提交并归档。

## Background

`v2.1.1-beta.3`（TASK-20261029-005，`75832d06`）包含「更新完成」提示，但用户尚未安装；
`7d4ff154`（TASK-20261029-006）修复了弹窗关闭没有退出动画的问题，尚未出包。
为避免用户安装两次，直接出 `beta.4`，一次覆盖两处改动。

## Scope

- 版本源递增至 `2.1.1-beta.4`，新增对应本地发布说明。
- 发布脚本构建、打包、生成 SHA256 并校验版本资源与资产目录。
- 同步 CURRENT 与本任务记录。

## Out of Scope

自动安装或停止用户运行中的应用、签名、打标签、推送、公开发布、修改业务代码或数据库迁移。

## Existing Behavior

上一份本地产物为 `v2.1.1-beta.3`（20,340,868 bytes）；公开正式版为 `2.1.0`。

---

# Implementation Plan

1. 版本源与 `frontend/package.json` 改为 `2.1.1-beta.4`，新增双语发布说明。
2. 运行 `scripts/build-release.ps1 -Version v2.1.1-beta.4 -OutputDirectory release-assets/v2.1.1-beta.4`。
3. 校验生产 EXE、安装器、SHA256、版本资源与目录内容，还原 `info.json` 为 LF。
4. 回填结果与验证、更新 CURRENT、提交并归档。

---

# Implementation Result

## 版本源

| 位置 | 值 |
| --- | --- |
| `internal/version/version.go` | `2.1.1-beta.4` |
| `frontend/package.json` | `2.1.1-beta.4` |
| `build/windows/info.json` | 文件版本 `2.1.1-beta.4`／产品数字版本 `2.1.1.0` |
| `build/config.yml` / `build/windows/nsis/wails_tools.nsh` | `2.1.1`（不含预发布标识，无需改） |

## 新增发布说明

`docs/releases/v2.1.1-beta.4.md`（中英双语）：更新完成浮窗、弹窗淡入淡出、Markdown 排版与安全边界，
并注明本版取代 `beta.1`／`beta.2`／`beta.3`；末尾说明「首次触发更新完成提示」的判定条件，
便于用户在本地更新时区分「没弹」是缺陷还是设计。

## 构建与产物

命令：

```text
scripts/build-release.ps1 -Version v2.1.1-beta.4 -OutputDirectory release-assets/v2.1.1-beta.4
```

过程（日志 `tmp/release-v2.1.1-beta.4.log`，exit 0）：`set-release-version.ps1` → `task --force verify`
→ `task --force package`（NSIS）→ `test-windows-metadata.ps1` → 复制安装器与 `.sha256` → `verify-release-assets.ps1`。

| 文件 | 大小 |
| --- | --- |
| `release-assets/v2.1.1-beta.4/OmniProxy-Setup-v2.1.1-beta.4-windows-amd64.exe` | 20,341,751 bytes |
| 同上 `.sha256` | 114 bytes |

SHA256：`c5078af748d3b7dc47bb86297441075af610f1314a9c4707012ec6f67f6e0a93`

NSIS 统计：`Total size: 20341751 / 81082669 bytes (25.0%)`。与 `beta.3`（20,340,868 bytes）相差 883 bytes，
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
| `verify-release-assets.ps1` | 「发布资产校验通过：v2.1.1-beta.4」 |
| SHA256 独立复算 | 与 `.sha256` 一致（`c5078af7…0a93`） |

已知无害噪声：Vite 大 chunk 警告、`tail`／`uname` 在 Windows 缺失；不影响结果。

未覆盖（需用户实机确认）：原生 WebView2 下的弹窗过渡观感、140% 字号、真实覆盖安装，
以及本版首次出现的「更新完成」浮窗实际触发。

---

# Documentation

- 新增 `docs/releases/v2.1.1-beta.4.md`（中英双语）。
- 更新 `docs/plans/CURRENT.md`：顶部摘要、`## Current Version`、`## Recently Completed`。

---

# Deviations

1. 没有先出 `beta.3` 让用户安装再出 `beta.4`：用户尚未安装 `beta.3`，直接递增到 `beta.4`
   一次覆盖两处改动，避免连续安装两次。
2. Windows 数字版本仍为 `2.1.1.0`（发布脚本按 patch 段映射，预发布标识不进第四段），与 `beta.1`–`beta.3` 相同。
3. `build/windows/info.json` 被发布脚本改写为 CRLF，提交前还原为 LF，最终 diff 只含版本值。

---

# Git

## Commit

```text
c8231310 chore(release): 构建 2.1.1-beta.4 本地生产安装包
```

归档与 CURRENT.md 回填为后续文档提交。未推送、未打标签、未签名。

---

# Result

`v2.1.1-beta.4` Windows amd64 production 安装包已产出并通过完整质量门（145 文件／880 项前端测试）、
NSIS 打包、Windows 版本资源与发布资产校验；安装器 20,341,751 bytes，
SHA256 `c5078af748d3b7dc47bb86297441075af610f1314a9c4707012ec6f67f6e0a93`。

内容为「更新完成」提示与弹窗淡入淡出，相对公开 `2.1.0` 与本地 `beta.1`–`beta.3` 均为递增。
未签名、未自动安装、未打标签、未推送、未公开发布，供用户本地覆盖更新。

---

# Completed

2026-10-29 13:35
