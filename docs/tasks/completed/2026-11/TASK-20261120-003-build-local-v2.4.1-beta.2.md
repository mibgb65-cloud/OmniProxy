# TASK-20261120-003: 构建 v2.4.1-beta.2 本地安装包

## Status

Completed

## Created

2026-11-20 08:50

## Request

用户在 TASK-20261120-001（「记录」侧边栏页面）与 TASK-20261120-002（取消 256 KiB 截断、完整压缩保存正文）完成后继续本地更新流程，要求提交代码并构建新的 Beta 生产版本。

## Goal

产出 `v2.4.1-beta.2` Windows amd64 生产安装包（未签名、本地使用），供本机覆盖更新后验证：

- 侧边栏「记录」集中浏览已保存正文的请求。
- 请求正文完整保存（gzip 压缩，总预算 512 MiB），不再按 256 KiB 截断。
- 迁移 054 在首次启动自动把旧明文正文迁移到新表，历史截断行保留标记。

## Background

- 版本基线 `v2.4.1-beta.1`（`b338fca1` 构建准备、`0ebf30ea` 记录）。
- 用户反馈「记录不了」后核查确认记录已生效，但 413 条中 252 条被 256 KiB 截断，且没有集中入口。
- 两个功能任务完成并通过质量门后进入构建；`docs/releases/v2.4.1-beta.2.md` 与 `release-assets/v2.4.1-beta.2` 尚不存在。

## Scope

- `docs/releases/v2.4.1-beta.2.md` 版本说明（中英）。
- 版本号 → `2.4.1-beta.2`（`internal/version/version.go`、`frontend/package.json`）。
- `scripts/build-release.ps1 -Version v2.4.1-beta.2 -OutputDirectory release-assets\v2.4.1-beta.2`。
- 记录安装器大小、SHA256、Windows 数字版本与 NSIS 基础版本。

## Out of Scope

- 打标签、推送、公开 Release、代码签名、自动安装或启动应用。
- 真实模型请求与桌面端实机目视检查（由用户安装后验证）。

## Existing Behavior

`v2.4.1-beta.1`：正文按 256 KiB 截断；没有「记录」页面；正文只散落在请求详情里。

## Expected Behavior

安装后：侧边栏出现「记录」；开启记录后新请求完整保存；旧记录迁移后仍可读，历史截断行标为「已截断」；设置页显示正文压缩占用与 512 MiB 预算。

---

# Analysis

## Related Modules

- `internal/version/version.go`、`frontend/package.json`、`build/config.yml`、`build/windows/info.json`、`build/windows/nsis/wails_tools.nsh`
- `docs/releases/v2.4.1-beta.2.md`
- `scripts/build-release.ps1`、`scripts/set-release-version.ps1`

## Dependencies

NSIS、`task`、`wails3`、Go 1.26、pnpm —— 已验证可用。

## Risks

- 构建脚本内部运行 `task --force verify` 与 `task --force package`；构建前后核对 `git status`。
- `release-assets/v2.4.1-beta.2` 必须为空或不存在。
- 首次启动执行迁移 054（重建 `request_contents`）；旧数据量约 100 MB，迁移在事务中完成。

---

# Implementation Plan

1. 写版本说明与版本号，提交 `chore(release)`。
2. 运行 `scripts/build-release.ps1`。
3. 校验安装器、版本资源与发布资产。
4. 回填任务记录、归档、更新 CURRENT，提交文档。

---

# Progress

- 2026-11-20 08:50 创建记录。
- 2026-11-20 09:05 版本号 → `2.4.1-beta.2`，同步 Windows 资源。
- 2026-11-20 09:06 直接运行 `.ps1` 时发现当前 shell 为 Windows PowerShell 5.1、代码页 936，无 BOM UTF-8 脚本中的中文被误读导致解析失败；改用临时 UTF-8 BOM 副本运行构建脚本（仓库文件未改动，副本已删除）。
- 2026-11-20 09:12 构建 EXIT=0：质量门、NSIS 打包、版本资源与发布资产校验全部通过。

---

# Implementation Result

## Changes

- 新增 `docs/releases/v2.4.1-beta.2.md`（中英版本说明：记录页、完整正文与压缩预算、迁移 054、本地更新与验证边界）。
- 版本号：`internal/version/version.go`、`frontend/package.json` → `2.4.1-beta.2`；`build/windows/info.json` 的 `ProductVersion` → `2.4.1-beta.2`；`fixed` 仍为 `2.4.1.0`，`build/config.yml` 与 NSIS `INFO_PRODUCTVERSION` 仍为 `2.4.1`（数字版本未变）。
- 安装包：`release-assets/v2.4.1-beta.2/OmniProxy-Setup-v2.4.1-beta.2-windows-amd64.exe`
  （21,555,718 bytes，SHA256 `e4d690d0a933bbe9fd08a22103cff3d37d1ab1e1b4551bf38d73fb885508eb77`），附 `.sha256`。
- 产物目录被 `.gitignore` 忽略，不入库。

## Files Changed

- `docs/releases/v2.4.1-beta.2.md`
- `internal/version/version.go`、`frontend/package.json`、`build/windows/info.json`
- 产物目录 `release-assets/v2.4.1-beta.2/`

## Technical Notes

- 安装包未签名（本地 Beta）、未自动安装、未启动、未打标签、未推送。
- 本版本包含迁移 054：首次启动重建 `request_contents` 并迁移旧明文正文；正文记录开关仍默认关闭。
- 构建脚本在构建前已产出安装器，随后提交的 `738a8936` 只包含版本号与版本说明，源码与构建内容一致。

---

# Verification

## Tests

- [x] 构建脚本内置的 `task --force verify`（格式、lint、`tsc --noEmit`、Go 全包、前端 164 文件／1005 项测试、Windows 生产构建）
- [x] Windows 应用与安装器版本资源校验
- [x] 发布资产（数量、命名、SHA256）校验
- [ ] 真实安装与启动：由用户执行
- [ ] 桌面端实机目视检查：未执行

## Quality Checks

- [x] Format
- [x] Lint（2 条既有 `react-refresh` 警告，0 error）
- [x] Type Check
- [x] Test
- [x] Build
- [x] Verify

## Verification Result

```text
scripts\build-release.bom.ps1 -Version v2.4.1-beta.2 -OutputDirectory release-assets\v2.4.1-beta.2
BUILD_EXIT=0
已校验 v2.4.1-beta.2，Windows 数字版本 2.4.1.0
前端 164 文件 / 1005 项测试通过
Windows 应用和安装器版本资源校验通过
发布资产校验通过：v2.4.1-beta.2
安装器：21,555,718 bytes
SHA256：e4d690d0a933bbe9fd08a22103cff3d37d1ab1e1b4551bf38d73fb885508eb77
Windows 数字版本：2.4.1.0；NSIS 基础版本：2.4.1
```

---

# Documentation

- [x] 版本说明 `docs/releases/v2.4.1-beta.2.md`（会被前端 `import.meta.glob` 收录）
- [x] CURRENT
- [x] 功能文档由 TASK-20261120-001／002 更新，本轮无新增

---

# Deviations

- 当前执行环境为 Windows PowerShell 5.1、代码页 936；仓库 `.ps1` 是无 BOM UTF-8，直接运行会把中文误读为乱码并解析失败。构建改用临时 UTF-8 BOM 副本完成，仓库脚本未修改，副本已删除。后续在 PowerShell 5.1 环境重复构建时需要同样处理，或改用 PowerShell 7。

---

# Git

## Commit

```text
738a8936 chore(release): 准备 v2.4.1-beta.2 本地构建
```

本任务记录与 `docs/plans/CURRENT.md` 的更新随 `chore(release): 记录 v2.4.1-beta.2 构建产物` 一并提交。

---

# Result

`v2.4.1-beta.2` 本地 Windows amd64 生产安装包已产出并校验通过，内容为「记录」页面、完整请求正文的 gzip 存储与 512 MiB 总预算、迁移 054；未签名、未安装、未启动、未打标签、未推送。

---

# Completed

2026-11-20 09:14
