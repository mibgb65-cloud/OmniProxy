# TASK-20261115-004: 构建本地 v2.3.1-beta.3 生产安装包

## Status

Completed

## Created

2026-11-15 13:16（Asia/Singapore）

## Request

> 提交代码后，然后构建新的beta生产版本我进行本地更新，记得完成任务md更新

## Goal

- 版本由 `2.3.1-beta.2` 提升为 `2.3.1-beta.3`；Windows 数字版本仍为 `2.3.1.0`，
  NSIS 基础版本 `2.3.1`。
- 交付 TASK-20261115-003（网关回填被 Antigravity 丢弃的工具结果图片）。
- 生产编译 + NSIS 打包，核对原生版本资源、SHA256 与发布资产目录。
- 新增双语发行说明 `docs/releases/v2.3.1-beta.3.md`。

## Background

- 实现提交已入库：`f85af40b`（工具结果图片回填），记录提交 `af61023e`。
- 实现任务质量门已通过：`task verify` EXIT=0（Go 全包、前端 154 文件／935 项、
  format／lint／typecheck／Windows production 构建）。
- 证据链：客户端 `read_file` 确实产出 media 并落盘，但网关收到的同轮请求没有图片
  （14137 → 14466 Token，只多 329）；同一张图直发网关占 1173 prompt tokens 且模型答对内容。
- v2.3.1 尚无公开正式版；本次是同一数字版本上的下一个本地 Beta。

## Scope

- 版本映射（`version.go`、`frontend/package.json`、`build/config.yml`、
  `build/windows/info.json`、`wails_tools.nsh`）、发行说明、生产打包、发布资产与摘要、
  任务记录与 CURRENT。

## Out of Scope

- 不重跑逐项功能开发；功能门禁沿用实现任务结果。
- 不推送、不打标签、不公开发布、不签名、不自动安装、不启动。
- 不新增功能、不修改业务代码。

## Existing Behavior

源码版本为 `2.3.1-beta.2`，Antigravity 网关模式下读图仍只有占位文本。

## Expected Behavior

覆盖安装后：`read_file` 读到图片时，网关按同一 `tool_call_id` 的路径回读文件并补成
`inlineData`，Gemini 族模型能看到像素；其余功能与 v2.3.1-beta.2 一致。

---

# Analysis

## Related Modules

- 版本：`internal/version/version.go`、`frontend/package.json`、`build/config.yml`、
  `build/windows/info.json`、`build/windows/nsis/wails_tools.nsh`
- 产物：`bin/`、`release-assets/v2.3.1-beta.3/`
- 文档：`docs/releases/`、`docs/plans/CURRENT.md`

## Dependencies

`scripts/build-release.ps1` → `set-release-version` → `task --force verify` → `task --force package`
→ 版本资源校验 → 发布资产复制／SHA256 → 资产校验。NSIS 位于 `C:\Program Files (x86)\NSIS`。

## Risks

- 安装器未签名，可能触发未知发布者提示。
- 运行中的实例未完全退出时覆盖安装会被拒绝。
- `set-release-version.ps1` 用 `ConvertTo-Json` 重写 `info.json`，需把换行还原为 LF。
- 回填只覆盖 `read_file` 形态（参数里带绝对路径）；浏览器截图仍不在覆盖范围。

---

# Implementation Plan

1. 创建本记录。
2. 新增双语发行说明 `docs/releases/v2.3.1-beta.3.md`。
3. 运行 `scripts/build-release.ps1 -Version v2.3.1-beta.3 -OutputDirectory release-assets\v2.3.1-beta.3`。
4. 核对 `info.json` 换行与 diff 只含版本行，记录产物大小与 SHA256。
5. 回填 Task Record 与 CURRENT，提交并归档。

---

# Progress

- 2026-11-15 13:16：创建记录与发行说明，准备开始生产打包。

---

# Implementation Result

## Changes

- 应用与前端版本 `2.3.1-beta.3`；Windows 数字资源保持 `2.3.1.0`，`ProductVersion` 为
  `2.3.1-beta.3`；`build/config.yml` 与 `wails_tools.nsh` 的数字版本不变，diff 只有版本行。
- 新增双语发行说明 `docs/releases/v2.3.1-beta.3.md`：客户端丢图根因、网关回填规则与边界、
  升级方式、无迁移与验证边界。
- 版本包含实现提交 `f85af40b`（工具结果图片回填）。

## Build and Artifacts

| 产物 | 大小（bytes） | SHA256 |
| --- | ---: | --- |
| `release-assets/v2.3.1-beta.3/OmniProxy-Setup-v2.3.1-beta.3-windows-amd64.exe` | 21,010,954 | `d380fd1ba0c9740192ea628998a79c70a3bd6ff59b3b5e6cd15bd634ffa50221` |
| `bin/omniproxy.exe` | 24,192,512 | `3a7693fb4752105d9d91c0f7376e9cc91c79308deb4dbd825ccc2547ee976e07` |

摘要侧车与安装器同目录；应用与安装器均未签名。

---

# Verification

## Quality Checks

- [x] Format：`info.json` 换行还原为 LF，diff 只含 `ProductVersion` 行。
- [x] Build：`task --force package` 随 `build-release.ps1` 执行成功。
- [x] Verify：脚本内的 `task --force verify` 通过（154 文件／935 项前端测试、全部 Go 包、
  Windows production 构建）。
- [x] 原生版本资源校验：Windows 应用与安装器版本资源通过。
- [x] 发布资产校验：`v2.3.1-beta.3` 通过。

## Verification Result

```text
build-release.ps1 -Version v2.3.1-beta.3 -OutputDirectory release-assets\v2.3.1-beta.3
  set-release-version            → 已校验 v2.3.1-beta.3，Windows 数字版本 2.3.1.0
  task --force verify            → 通过（154 files / 935 tests，Windows production build）
  task --force package           → 安装器 21,010,954 bytes
  test-windows-metadata          → Windows 应用和安装器版本资源校验通过
  verify-release-assets          → 发布资产校验通过：v2.3.1-beta.3
  EXIT=0
```

未签名、未自动安装、未启动、未打标签、未推送。

---

# Documentation

- [x] 发行说明：`docs/releases/v2.3.1-beta.3.md`
- [x] CURRENT：已追加本版本摘要
- [x] 其余文档无需修改

---

# Git

## Commit

```text
6d2d124b chore(release): 准备 v2.3.1-beta.3 本地构建
```

---

# Result

已产出 `v2.3.1-beta.3` 本地 Windows amd64 生产安装包，供用户从托盘退出应用并备份数据后覆盖安装；
安装后 Antigravity 的 `read_file` 读图由网关回填，模型能看到像素。Antigravity 真机复测已通过：工具结果仍只有占位文本，但模型答出「浅绿底 / Z3ZVB9 / 深蓝正方形、红色菱形、绿色圆形」，与图片一致。

---

# Completed

2026-11-15 13:24（Asia/Singapore）