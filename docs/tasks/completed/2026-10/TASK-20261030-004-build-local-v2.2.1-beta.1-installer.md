# TASK-20261030-004: 构建本地 v2.2.1-beta.1 生产安装包

## Status

Completed

## Created

2026-10-30 20:40（Asia/Singapore）

## Request

提交代码后，构建新的 Beta 生产版本供用户本地更新。

## Goal

产出 Windows amd64 production 安装包 `v2.2.1-beta.1`，包含 `v2.2.0` 之后的两项用户可见改动，
完成质量门、安装器元数据与发布资产校验，本地提交并归档。

## Background

公开正式版为 `v2.2.0`（标签源码 `fe450952`）。此后 `main` 上有两条用户可见改动尚未出包：

- `ab47f044` fix(account): 固定提供商切换栏并隔离详情滚动（TASK-20261030-002）。
- `c6c2f658` feat(dashboard): 右侧卡片自适应填充并补齐空态信息（TASK-20261030-003）。

本轮直接出 `v2.2.1-beta.1`，让用户一次覆盖安装即可拿到两项改动。

## Scope

- 版本源递增至 `2.2.1-beta.1`，新增对应本地发布说明。
- 发布脚本构建、打包、生成 SHA256 并校验版本资源与资产目录。
- 同步 CURRENT 与本任务记录。

## Out of Scope

自动安装或停止用户运行中的应用、签名、打标签、推送、公开发布、修改业务代码或数据库迁移。

## Existing Behavior

上一份本地产物为 `v2.1.1-beta.5`；公开正式版为 `v2.2.0`。

---

# Implementation Plan

1. 版本源与 `frontend/package.json` 改为 `2.2.1-beta.1`，新增双语发布说明。
2. 运行 `scripts/build-release.ps1 -Version v2.2.1-beta.1 -OutputDirectory release-assets/v2.2.1-beta.1`。
3. 校验生产 EXE、安装器、SHA256、版本资源与目录内容，还原 `info.json` 为 LF。
4. 回填结果与验证、更新 CURRENT、提交并归档。

---

# Implementation Result

## 版本源

| 位置                                                      | 值                                                  |
| --------------------------------------------------------- | --------------------------------------------------- |
| `internal/version/version.go`                             | `2.2.1-beta.1`                                      |
| `frontend/package.json`                                   | `2.2.1-beta.1`                                      |
| `build/windows/info.json`                                 | 数字版本 `2.2.1.0`／`ProductVersion` `2.2.1-beta.1` |
| `build/config.yml` / `build/windows/nsis/wails_tools.nsh` | `2.2.1`（不含预发布标识）                           |

## 新增发布说明

`docs/releases/v2.2.1-beta.1.md`（中英双语）：仪表盘右侧卡片自适应填充、待处理问题体检清单、
资源概况健康分布条与按供应商状态点阵、趋势图分档高度，以及账号页提供商切换栏固定与详情独立滚动；
同时记录「清单与问题列表共用过滤口径」「状态色点不单独承载信息」的一致性边界。

## 构建与产物

命令：`scripts/build-release.ps1 -Version v2.2.1-beta.1 -OutputDirectory release-assets/v2.2.1-beta.1`
（日志 `tmp/release-v2.2.1-beta.1.log`，exit 0）：`set-release-version.ps1` → `task --force verify`
→ `task --force package`（NSIS）→ `test-windows-metadata.ps1` → 复制安装器与 `.sha256` → `verify-release-assets.ps1`。

| 文件                                                                           | 大小             |
| ------------------------------------------------------------------------------ | ---------------- |
| `release-assets/v2.2.1-beta.1/OmniProxy-Setup-v2.2.1-beta.1-windows-amd64.exe` | 20,352,203 bytes |
| 同上 `.sha256`                                                                 | 114 bytes        |
| `bin/omniproxy.exe`（内嵌前端，production）                                    | 23,293,440 bytes |

SHA256（安装器，`.sha256` 与独立复算一致）：`8f14e3281b5984927ee9c2c251d5eb5d1294b2e29391f902f6970608dfa54cd1`
SHA256（`bin/omniproxy.exe`）：`a090e9799326421a65fe9fd62629986b5be33ce9ddf385f9f53e0acd2af53899`

NSIS 统计：`Total size: 20352203 / 81176962 bytes (25.0%)`。与 `v2.1.1-beta.5`（20,343,683 bytes）相差
8,520 bytes，与「两份前端改动 + 新增 i18n 文案」的量级相符。

安装器版本资源：`FileVersion` / `ProductVersion` 均为 `2.2.1`（NSIS 只取数字段；完整语义版本在应用内
`ProductVersion` 与 `internal/version` 中为 `2.2.1-beta.1`）。

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

| 项目                                | 结果                                                         |
| ----------------------------------- | ------------------------------------------------------------ |
| 发布脚本内 `task --force verify`    | exit 0                                                       |
| gofmt / prettier `format:check`     | 通过（`All matched files use Prettier code style!`）         |
| `go vet` / `staticcheck` / `eslint` | 通过                                                         |
| `tsc` typecheck                     | 通过                                                         |
| Vitest                              | 145 文件／884 项全部通过（123.48s）                          |
| Go 全量包测试                       | 通过                                                         |
| Windows production 构建             | 通过                                                         |
| NSIS 打包                           | 通过（25.0%）                                                |
| `test-windows-metadata.ps1`         | 「Windows 应用和安装器版本资源校验通过」（独立复跑同样通过） |
| `verify-release-assets.ps1`         | 「发布资产校验通过：v2.2.1-beta.1」                          |
| SHA256 独立复算                     | 与 `.sha256` 一致（`8f14e328…4cd1`）                         |

未覆盖（需用户实机确认）：真实覆盖安装（本次构建不自动安装、不停止正在运行的应用）、
原生 WebView2 下仪表盘卡片布局与账号页切换栏的实际观感、140% 字号。

---

# Documentation

- 新增 `docs/releases/v2.2.1-beta.1.md`（中英双语）。
- 更新 `docs/plans/CURRENT.md`：顶部摘要、`## Current Version`、`## Recently Completed`。

---

# Deviations

1. 没有为 `v2.2.0` 单独递增 patch 正式版，而是按既有习惯出本地预发布 `2.2.1-beta.1`，
   与 `2.1.1-beta.*` 的处理一致。
2. 数字版本为 `2.2.1.0`：发布脚本按 patch 段映射，预发布标识不进第四段，因此与 `2.2.0.0` 可区分。
3. `build/windows/info.json` 被发布脚本改写为 CRLF，提交前已还原为 LF（`CRLF=0`）。
4. 日志里 `+ & task --force verify` 是 Taskfile 的 `uname` 缺失提示写到 stderr 引起的
   NativeCommandError 渲染，`exit 1` 是 Taskfile gofmt 检查分支的脚本正文被回显，二者都不是失败；
   脚本随后完成 NSIS 打包与资产校验，说明质量门实际为 exit 0。

---

# Git

## Commit

```text
e4f37328 chore(release): 构建 2.2.1-beta.1 本地生产安装包
```

归档与 CURRENT.md 回填为后续文档提交。未推送、未打标签、未签名。

---

# Result

`v2.2.1-beta.1` Windows amd64 production 安装包已产出并通过完整质量门（145 文件／884 项前端测试）、
NSIS 打包、Windows 版本资源与发布资产校验；安装器 20,352,203 bytes，
SHA256 `8f14e3281b5984927ee9c2c251d5eb5d1294b2e29391f902f6970608dfa54cd1`。
内容为仪表盘卡片自适应填充与账号页提供商切换栏修复，未签名、未自动安装、未打标签、未推送、未公开发布。

---

# Completed

2026-10-30 21:05
