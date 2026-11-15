# TASK-20261115-010: 构建本地 v2.3.1-beta.6 生产安装包

## Status

Completed

## Created

2026-11-15 19:30（Asia/Singapore）

## Request

延续 TASK-20261115-009：用户当前运行 `v2.3.1-beta.5`，需要包含流式失败分类改进的新安装包做本地更新。

## Goal

- 版本提升为 `2.3.1-beta.6`；Windows 数字版本仍为 `2.3.1.0`，NSIS 基础版本 `2.3.1`。
- 交付 TASK-20261115-009（`upstream_error_event` 分类与 `upstream_stream_kind` /
  `upstream_event_type` / `upstream_error_code` 日志字段）。
- 生产编译 + NSIS 打包，核对原生版本资源、SHA256 与发布资产目录。
- 新增双语发行说明 `docs/releases/v2.3.1-beta.6.md`。

## Background

- 协议路径的流式失败原来只有一个 `upstream_stream_failed`，日志也没有可核实的原因字段。
- 改进已实现并验证：`task verify` EXIT=0，新增分类、短码净化与网关端到端用例通过。

## Scope

- 版本映射、发行说明、生产打包、发布资产与摘要、任务记录与 CURRENT。

## Out of Scope

- 不新增功能、不修改其它业务代码。
- 不推送、不打标签、不公开发布、不签名、不自动安装、不启动。

---

# Implementation Plan

1. 提升 `version.go`、`frontend/package.json` 到 `2.3.1-beta.6`（随 TASK-20261115-009 提交）。
2. 运行 `scripts/build-release.ps1 -Version v2.3.1-beta.6 -OutputDirectory release-assets\v2.3.1-beta.6`。
3. 核对 `info.json` 换行与 diff 只含版本行，记录产物大小与 SHA256。
4. 回填 Task Record 与 CURRENT，提交并归档。

---

# Progress

- 2026-11-15 19:36：开始生产打包。
- 2026-11-15 19:46：NSIS 打包与发布资产校验输出通过；单独复跑 `verify-release-assets.ps1`
  与 `test-windows-metadata.ps1` 均 EXIT=0（外层管道的退出码为 1，不影响产物校验）。

---

# Implementation Result

## Changes

- 应用与前端版本 `2.3.1-beta.6`；Windows 数字资源保持 `2.3.1.0`，`ProductVersion` 为
  `2.3.1-beta.6`；`build/config.yml` 与 `wails_tools.nsh` 数字版本不变。
- 新增双语发行说明 `docs/releases/v2.3.1-beta.6.md`：失败分类规则、日志字段、调查结论、
  升级方式与验证边界。
- 版本包含实现提交 `0bbe5b59`（TASK-20261115-009）。

## Build and Artifacts

| 产物 | 大小（bytes） | SHA256 |
| --- | ---: | --- |
| `release-assets/v2.3.1-beta.6/OmniProxy-Setup-v2.3.1-beta.6-windows-amd64.exe` | 21,029,561 | `f593af13b41d1427c814f3f54816a321efa04233d0b2af735b6fe3fc73004659` |
| `bin/omniproxy.exe` | 24,224,768 | `90a96143390456f4811a0357872bae1c708aab5a8688512c7e15a58ba5bc3e1` |

摘要侧车与安装器同目录；应用与安装器均未签名。

## Technical Notes

- `set-release-version.ps1` 只校验并归一版本，因此先手工提升 `version.go` 与
  `frontend/package.json`，再由脚本写入 `info.json`／`config.yml`／`wails_tools.nsh`。
- 打包后 `info.json` 的 CRLF 已还原为 LF（UTF-8 无 BOM），diff 只剩 `ProductVersion` 一行。

---

# Verification

## Quality Checks

- [x] Format：`info.json` 换行还原为 LF，diff 只含 `ProductVersion` 行。
- [x] Lint／Type Check：脚本内 `task --force verify` 通过。
- [x] Test：Go 全部包通过；前端 154 文件／935 项测试通过。
- [x] Build：`task --force package` 随脚本执行成功。
- [x] Verify：脚本内 `task --force verify` 通过。
- [x] 原生版本资源校验：Windows 应用与安装器版本资源通过。
- [x] 发布资产校验：`v2.3.1-beta.6` 通过。

## Verification Result

```text
build-release.ps1 -Version v2.3.1-beta.6 -OutputDirectory release-assets\v2.3.1-beta.6
  set-release-version            → 已校验 v2.3.1-beta.6，Windows 数字版本 2.3.1.0
  task --force verify            → 通过（154 files / 935 tests、全部 Go 包、Windows production build）
  task --force package           → 安装器 21,029,561 bytes
  test-windows-metadata          → Windows 应用和安装器版本资源校验通过（复跑 EXIT=0）
  verify-release-assets          → 发布资产校验通过：v2.3.1-beta.6（复跑 EXIT=0）
```

外层管道捕获到的退出码为 1，两个校验脚本单独复跑均 EXIT=0、产物与版本资源一致，
判定为管道封装层的退出码，不影响打包结果。

未签名、未自动安装、未启动、未打标签、未推送。

---

# Documentation

- [x] 发行说明：`docs/releases/v2.3.1-beta.6.md`
- [x] CURRENT：已追加本版本摘要
- [x] 其余文档无需修改

---

# Git

## Commit

```text
9ccceaf8 chore(release): 准备 v2.3.1-beta.6 本地构建
```

---

# Result

已产出 `v2.3.1-beta.6` 本地 Windows amd64 生产安装包，供用户从托盘完全退出应用并备份数据后
覆盖安装。版本包含流式失败分类改进：上游在 SSE 里主动报错单独记为 `upstream_error_event`，
日志带可核实的分类与上游短码，说明文字既不回显也不落日志；非法 JSON 与非对象形状仍为
`upstream_stream_failed`。原始 6 条失败的上游原因事后无法复现，本版本不声称修复它，
只让下一次发生能直接判断原因。

---

# Completed

2026-11-15 19:52（Asia/Singapore）
