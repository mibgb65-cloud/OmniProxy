# TASK-20261031-004: 构建本地 v2.2.1-beta.5 生产安装包

## Status

In Progress

## Created

2026-10-31 17:45

## Request

> 提交代码，然后构建新的 beta 生产版本我进行本地更新，记得完成任务 md 更新

## Goal

产出 Windows amd64 production 安装包 `v2.2.1-beta.5`，包含账号 Token 用量改为终身累计的改动
（TASK-20261031-003，`c2c538b4`／`bd1a9714`），完成质量门、安装器元数据与发布资产校验，本地提交并归档。

## Background

本地上一份产物为 `v2.2.1-beta.4`（TASK-20261031-002，安装器 20,357,883 bytes，SHA256 `85727445…5475`）。
此后 `main` 上新增的用户可见改动：`c2c538b4` 账号 Token 用量改为终身累计（迁移 045 + `usage_accounts`），
`bd1a9714` 归档该任务记录。代码已提交，尚未出包。

## Scope

- 版本源递增至 `2.2.1-beta.5`，新增对应双语发布说明。
- `scripts/build-release.ps1` 构建、打包、生成 SHA256 并校验版本资源与资产目录。
- 同步 CURRENT 与任务记录。

## Out of Scope

自动安装或停止用户运行中的应用、签名、打标签、推送、公开发布、修改业务代码或新增迁移。

## Existing Behavior

本地最新安装包为 `v2.2.1-beta.4`；公开正式版仍为 `v2.2.0`。

---

# Analysis

## Related Modules

- `internal/version/version.go`、`frontend/package.json`（人工改版本）
- `build/config.yml`、`build/windows/info.json`、`build/windows/nsis/wails_tools.nsh`（`set-release-version.ps1` 写数字版本）
- `docs/releases/v2.2.1-beta.5.md`（人工发布说明，脚本会校验存在）
- `scripts/build-release.ps1`

## Risks

同 TASK-20261031-002 已记录的三个坑：`frontend/package.json` 不能带 UTF-8 BOM（prettier 校验会失败）、
`build/windows/info.json` 会被写成 CRLF（提交前还原 LF）、日志里的 `exit 1` 是 Taskfile gofmt 分支回显而非失败。
本次另有数据库迁移 045，需在成果中说明升级后自动执行、无需手工操作。

---

# Implementation Plan

1. `internal/version/version.go` 与 `frontend/package.json` 改为 `2.2.1-beta.5`（无 BOM、保持 LF）。
2. 新增 `docs/releases/v2.2.1-beta.5.md`（中英双语）。
3. 运行 `scripts/build-release.ps1 -Version v2.2.1-beta.5 -OutputDirectory release-assets/v2.2.1-beta.5`。
4. 校验 EXE、安装器、SHA256、版本资源与目录内容，恢复 `build/windows/info.json` 的 LF。
5. 回填结果与验证、更新 CURRENT、提交并归档。

---

# Progress

- 版本源与发布说明就绪后运行 `scripts/build-release.ps1`，一次通过：质量门 → NSIS 打包 → 版本资源 → 资产校验，exit 0。
- `build/windows/info.json` 被脚本写成 CRLF，提交前还原为 LF（`CR=0`）。

---

# Implementation Result

## 版本源

| 位置                                                      | 值                                                  |
| --------------------------------------------------------- | --------------------------------------------------- |
| `internal/version/version.go`                             | `2.2.1-beta.5`                                      |
| `frontend/package.json`                                   | `2.2.1-beta.5`（无 BOM、LF）                        |
| `build/windows/info.json`                                 | 数字版本 `2.2.1.0`／`ProductVersion` `2.2.1-beta.5` |
| `build/config.yml` / `build/windows/nsis/wails_tools.nsh` | `2.2.1`（不含预发布标识，本次未变）                 |

## 新增发布说明

`docs/releases/v2.2.1-beta.5.md`（中英双语）：账号卡片／配额页 Token 用量改为终身累计、
迁移 045 升级即回填、身份口径与「用量」页统一、缓存明细不再重复计入或降级为未知；
并写明首次启动自动执行迁移、请求明细上限不再影响卡片数字、升级前已淘汰明细的缓存部分只显示下界。

## 构建与产物

命令：`scripts/build-release.ps1 -Version v2.2.1-beta.5 -OutputDirectory release-assets/v2.2.1-beta.5`
（日志 `tmp/release-v2.2.1-beta.5.log`，exit 0）：`set-release-version.ps1` → `task --force verify`
→ `task --force package`（NSIS）→ `test-windows-metadata.ps1` → 复制安装器与 `.sha256` → `verify-release-assets.ps1`。

| 文件                                                                           | 大小             |
| ------------------------------------------------------------------------------ | ---------------- |
| `release-assets/v2.2.1-beta.5/OmniProxy-Setup-v2.2.1-beta.5-windows-amd64.exe` | 20,366,155 bytes |
| 同上 `.sha256`                                                                 | 114 bytes        |
| `bin/omniproxy.exe`（内嵌前端与 migrations，production）                        | 23,313,408 bytes |

SHA256（安装器，`.sha256` 与独立复算一致）：`10d3c0664ca7424c6bef4b1a77f5c27ca5f94e1383567eafbc5bafb7f433ef38`
SHA256（`bin/omniproxy.exe`）：`5ecd713e49c65f3b69102f1f665097869274c78e6be0109882bea39dd3006581`

NSIS 统计：`Total size: 20366155 / 81243687 bytes (25.0%)`。比 `v2.2.1-beta.4`（20,357,883 bytes）增加
8,272 bytes，与「新增迁移 + 一条 upsert + 读路径改写 + 一项新测试（不进产物）+ 新增发布说明（不进产物）」的量级相符。

安装器版本资源：`FileVersion` / `ProductVersion` 均为 `2.2.1`（NSIS 只取数字段；完整语义版本在应用内为 `2.2.1-beta.5`）。

迁移随包下发：`main.go` 用 `//go:embed all:migrations` 内嵌迁移，安装后首次启动自动执行 045。

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

| 项目                                | 结果                                                 |
| ----------------------------------- | ---------------------------------------------------- |
| 发布脚本内 `task --force verify`    | exit 0                                               |
| gofmt / prettier `format:check`     | 通过                                                 |
| `go vet` / `staticcheck` / `eslint` | 通过                                                 |
| `tsc` typecheck                     | 通过                                                 |
| Vitest                              | 146 文件／890 项全部通过                             |
| Go 全量包测试                       | 51 个包 ok                                           |
| Windows production 构建             | 通过                                                 |
| NSIS 打包                           | 通过（25.0%）                                        |
| `test-windows-metadata.ps1`         | 「Windows 应用和安装器版本资源校验通过」             |
| `verify-release-assets.ps1`         | 「发布资产校验通过：v2.2.1-beta.5」                  |
| SHA256 独立复算                     | 与 `.sha256` 一致（`10d3c066…ef38`）                 |

未覆盖（需用户实机确认）：真实覆盖安装（本次不自动安装、不停止正在运行的应用）、
首次启动执行迁移 045 后的账号卡片数值、原生 WebView2 观感。

---

# Documentation

- 新增 `docs/releases/v2.2.1-beta.5.md`（中英双语）。
- 更新 `docs/plans/CURRENT.md`：顶部摘要、`## Current Version`、`## Recently Completed`（补 TASK-20261031-004 与 TASK-20261031-003）。
- 归档 TASK-20261031-003 并回填提交号（`bd1a9714`）。

---

# Deviations

1. `build/windows/info.json` 被 `set-release-version.ps1` 按 CRLF 重写（15 处 CR），提交前已还原为 LF（`CR=0`），diff 只剩 `ProductVersion` 一行。
2. 数字版本仍为 `2.2.1.0`（patch 段不变），与 `2.2.1-beta.4` 相同，符合既有映射规则；Windows 升级判定依赖应用内完整版本。
3. 本次构建一次通过，未复现 beta.4 的 `package.json` BOM 与日志 `exit 1` 假阳性问题（版本改写使用 `UTF8Encoding($false)`）。

---

# Git

## Commit

（回填）

---

# Result

`v2.2.1-beta.5` Windows amd64 production 安装包已产出并通过完整质量门（146 文件／890 项前端测试、51 个 Go 包）、
NSIS 打包、Windows 版本资源与发布资产校验；安装器 20,366,155 bytes，
SHA256 `10d3c0664ca7424c6bef4b1a77f5c27ca5f94e1383567eafbc5bafb7f433ef38`。
内容为账号 Token 用量改为终身累计（迁移 045），未签名、未自动安装、未打标签、未推送、未公开发布，等待用户本地覆盖更新。

---

# Completed

（回填）
