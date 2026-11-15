# TASK-20261115-008: 构建本地 v2.3.1-beta.5 生产安装包

## Status

Completed

## Created

2026-11-15 18:20（Asia/Singapore）

## Request

延续 TASK-20261115-007：用户已安装 `v2.3.1-beta.4`，需要包含命名空间工具修复的新安装包做本地更新。

## Goal

- 版本提升为 `2.3.1-beta.5`；Windows 数字版本仍为 `2.3.1.0`，NSIS 基础版本 `2.3.1`。
- 交付 TASK-20261115-007（Codex 入口移除第三方服务商无法表达的命名空间工具）。
- 生产编译 + NSIS 打包，核对原生版本资源、SHA256 与发布资产目录。
- 新增双语发行说明 `docs/releases/v2.3.1-beta.5.md`。

## Background

- 实机请求 `1d7ddb7c13219589000807526fa08efb` 在 beta.4 上仍返回 `400 unsupported_content`；
  真实正文含 `type: "namespace"` 工具分组，River 只接受函数工具。
- 修复已实现并验证：单测／端到端通过，真实 River Key 下带真实工具形状的请求非流式
  200／2733 ms、流式 200／904 ms，上游回显只剩函数工具。

## Scope

- 版本映射、发行说明、生产打包、发布资产与摘要、任务记录与 CURRENT。

## Out of Scope

- 不新增功能、不修改其它业务代码。
- 不推送、不打标签、不公开发布、不签名、不自动安装、不启动。

---

# Implementation Plan

1. 提升 `version.go`、`frontend/package.json` 到 `2.3.1-beta.5`。
2. 运行 `scripts/build-release.ps1 -Version v2.3.1-beta.5 -OutputDirectory release-assets\v2.3.1-beta.5`。
3. 核对 `info.json` 换行与 diff 只含版本行，记录产物大小与 SHA256。
4. 回填 Task Record 与 CURRENT，提交并归档。

---

# Progress

- 2026-11-15 18:20：版本提升与发行说明就绪，开始生产打包。
- 2026-11-15 18:24：第一次打包在 `internal/provider/antigravity` 报 `[build failed]`；
  该包单独跑通过，属并行全量构建的偶发失败（与端口竞争同类），未改该包。
- 2026-11-15 18:17（脚本第二轮时间戳）：质量门、NSIS 打包、版本资源与发布资产校验全部通过，`BUILD_OK`。

---

# Implementation Result

## Changes

- 应用与前端版本 `2.3.1-beta.5`；Windows 数字资源保持 `2.3.1.0`，`ProductVersion` 为
  `2.3.1-beta.5`；`build/config.yml` 与 `wails_tools.nsh` 数字版本不变。
- 新增双语发行说明 `docs/releases/v2.3.1-beta.5.md`：命名空间工具根因、移除规则与代价、
  升级方式、验证边界。
- 版本包含实现提交（TASK-20261115-007，见该记录 Git 章节）。

## Build and Artifacts

| 产物 | 大小（bytes） | SHA256 |
| --- | ---: | --- |
| `release-assets/v2.3.1-beta.5/OmniProxy-Setup-v2.3.1-beta.5-windows-amd64.exe` | 21,021,154 | `99f2aa9392ed980a4c654817fe703c0a5ca48c9e85dcb0254489c5f7f0c90e53` |
| `bin/omniproxy.exe` | 24,212,480 | `e883fc9f440be72984d8a9f4a34a3e5d1535704aa875d476b91fd2be4ccdeaa9` |

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
- [x] 发布资产校验：`v2.3.1-beta.5` 通过。

## Verification Result

```text
build-release.ps1 -Version v2.3.1-beta.5 -OutputDirectory release-assets\v2.3.1-beta.5
  set-release-version            → 已校验 v2.3.1-beta.5，Windows 数字版本 2.3.1.0
  task --force verify            → 通过（154 files / 935 tests、全部 Go 包、Windows production build）
  task --force package           → 安装器 21,021,154 bytes
  test-windows-metadata          → Windows 应用和安装器版本资源校验通过
  verify-release-assets          → 发布资产校验通过：v2.3.1-beta.5
  BUILD_OK
```

第一次打包在 `internal/provider/antigravity` 报 `[build failed]`，该包单独执行通过、
本次未修改该包，判定为并行全量构建的偶发失败；第二轮通过。

未签名、未自动安装、未启动、未打标签、未推送。

---

# Documentation

- [x] 发行说明：`docs/releases/v2.3.1-beta.5.md`
- [x] CURRENT：已追加本版本摘要
- [x] 其余文档无需修改

---

# Git

## Commit

```text
<待回填>
```

---

# Result

已产出 `v2.3.1-beta.5` 本地 Windows amd64 生产安装包，供用户从托盘完全退出应用并备份数据后
覆盖安装。安装后 Codex 选择 River／DeepSeek 等非 OpenAI 模型时不再因 `type: "namespace"`
工具分组被本地拒绝：网关移除分组、保留函数工具，真实 River Key 下非流式与流式均 200。
MCP／多智能体工具在该模型上不可用，已写入发行说明。

---

# Completed

2026-11-15 18:32（Asia/Singapore）
