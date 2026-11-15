# TASK-20261115-006: 构建本地 v2.3.1-beta.4 生产安装包

## Status

Completed

## Created

2026-11-15 17:45（Asia/Singapore）

## Request

> 提交代码后，然后构建新的beta生产版本我进行本地更新，记得完成任务md更新

## Goal

- 版本由 `2.3.1-beta.3` 提升为 `2.3.1-beta.4`；Windows 数字版本仍为 `2.3.1.0`，
  NSIS 基础版本 `2.3.1`。
- 交付 TASK-20261115-005（Codex 入口把非 OpenAI 模型交回协议管线）。
- 生产编译 + NSIS 打包，核对原生版本资源、SHA256 与发布资产目录。
- 新增双语发行说明 `docs/releases/v2.3.1-beta.4.md`。

## Background

- 实现提交已入库：`74959333`（Codex 非 OpenAI 模型改走协议入口），记录回填提交 `93986422`。
- 实现任务质量门已通过：`task verify` EXIT=0（54 个 Go 包、前端 154 文件／935 项、
  format／lint／typecheck／Windows production 构建）。
- 真实链路已验证：只读生产库 + 真实 River Key，`/codex/v1/responses` +
  `DeepSeek-V4.1-Flash-1017` 非流式 200／1907 ms、流式 200／833 ms；修复前为 401×3 → 502。
- 用户当前运行 `v2.3.1-beta.3`（不含该修复），需要新安装包做本地覆盖更新。

## Scope

- 版本映射（`version.go`、`frontend/package.json`、`build/config.yml`、
  `build/windows/info.json`、`wails_tools.nsh`）、发行说明、生产打包、发布资产与摘要、
  任务记录与 CURRENT。

## Out of Scope

- 不重跑逐项功能开发；功能门禁沿用实现任务结果。
- 不推送、不打标签、不公开发布、不签名、不自动安装、不启动。
- 不新增功能、不修改业务代码。

## Existing Behavior

源码版本为 `2.3.1-beta.3`，Codex 选择非 OpenAI 模型时整池 401。

## Expected Behavior

覆盖安装后：Codex 选择 River／DeepSeek 等模型时由协议入口处理，模型与档位在网关内还原，
第三方凭据不再发往 `api.openai.com`；其余功能与 v2.3.1-beta.3 一致。

---

# Analysis

## Related Modules

- 版本：`internal/version/version.go`、`frontend/package.json`、`build/config.yml`、
  `build/windows/info.json`、`build/windows/nsis/wails_tools.nsh`
- 产物：`bin/`、`release-assets/v2.3.1-beta.4/`
- 文档：`docs/releases/`、`docs/plans/CURRENT.md`

## Dependencies

`scripts/build-release.ps1` → `set-release-version` → `task --force verify` → `task --force package`
→ 版本资源校验 → 发布资产复制／SHA256 → 资产校验。NSIS 位于 `C:\Program Files (x86)\NSIS`。
`set-release-version.ps1` 只校验并归一版本，要求源码版本已等于标签，因此需先手工提升
`version.go` 与 `frontend/package.json`。

## Risks

- 安装器未签名，可能触发未知发布者提示。
- 运行中的实例未完全退出时覆盖安装会被拒绝。
- `set-release-version.ps1` 用 `ConvertTo-Json` 重写 `info.json`，需把换行还原为 LF。
- 修复只覆盖 HTTP 路径；WebSocket 握手没有正文，非 OpenAI 模型仍靠客户端回退 HTTP。

---

# Implementation Plan

1. 创建本记录与发行说明。
2. 提升 `internal/version/version.go`、`frontend/package.json` 到 `2.3.1-beta.4`。
3. 运行 `scripts/build-release.ps1 -Version v2.3.1-beta.4 -OutputDirectory release-assets\v2.3.1-beta.4`。
4. 核对 `info.json` 换行与 diff 只含版本行，记录产物大小与 SHA256。
5. 回填 Task Record 与 CURRENT，提交并归档。

---

# Progress

- 2026-11-15 17:45：创建记录与发行说明，准备版本提升与生产打包。
- 2026-11-15 17:46：`version.go`、`frontend/package.json` 提升到 `2.3.1-beta.4`，
  `set-release-version` 校验通过；`build-release.ps1` 开始执行。
- 2026-11-15 17:52：质量门、NSIS 打包、原生版本资源与发布资产校验全部通过，`BUILD_OK`。
- 2026-11-15 17:54：`info.json` 换行还原为 LF，diff 只含 `ProductVersion` 行。

---

# Implementation Result

## Changes

- 应用与前端版本 `2.3.1-beta.4`；Windows 数字资源保持 `2.3.1.0`，`ProductVersion` 为
  `2.3.1-beta.4`；`build/config.yml` 与 `wails_tools.nsh` 的数字版本不变（脚本校验后无差异）。
- 新增双语发行说明 `docs/releases/v2.3.1-beta.4.md`：Codex 非 OpenAI 模型的
  401 根因、改走协议入口后的行为、与透明转发的边界、升级方式与验证边界。
- 版本包含实现提交 `74959333`（Codex 非 OpenAI 模型改走协议入口）。

## Build and Artifacts

| 产物 | 大小（bytes） | SHA256 |
| --- | ---: | --- |
| `release-assets/v2.3.1-beta.4/OmniProxy-Setup-v2.3.1-beta.4-windows-amd64.exe` | 21,015,871 | `290712bd211b9059f65ee4545e07a0e55d768539d0f413abf00e734329c2670b` |
| `bin/omniproxy.exe` | 24,203,264 | `75d692297711c0cdc88900638832fca4c2754420a5de100072482581edd78800` |

摘要侧车（`*.sha256`）与安装器同目录；应用与安装器均未签名。

## Technical Notes

- `set-release-version.ps1` 只校验并归一版本，要求源码版本已等于标签；因此先手工提升
  `version.go` 与 `frontend/package.json`，再由脚本写入 `info.json`／`config.yml`／
  `wails_tools.nsh` 的数字版本。
- 脚本用 `ConvertTo-Json` 重写 `info.json` 会带回 CRLF；打包完成后已还原为 LF
  （UTF-8 无 BOM），diff 只剩 `ProductVersion` 一行。

---

# Verification

## Quality Checks

- [x] Format：`info.json` 换行还原为 LF，diff 只含 `ProductVersion` 行。
- [x] Lint／Type Check：脚本内 `task --force verify` 通过（lint 0 errors，2 条既有 react-refresh 警告）。
- [x] Test：Go 全部包通过；前端 154 文件／935 项测试通过。
- [x] Build：`task --force package` 随脚本执行成功。
- [x] Verify：脚本内 `task --force verify` 通过。
- [x] 原生版本资源校验：Windows 应用与安装器版本资源通过。
- [x] 发布资产校验：`v2.3.1-beta.4` 通过。

## Verification Result

```text
build-release.ps1 -Version v2.3.1-beta.4 -OutputDirectory release-assets\v2.3.1-beta.4
  set-release-version            → 已校验 v2.3.1-beta.4，Windows 数字版本 2.3.1.0
  task --force verify            → 通过（154 files / 935 tests、全部 Go 包、Windows production build）
  task --force package           → 安装器 21,015,871 bytes
  test-windows-metadata          → Windows 应用和安装器版本资源校验通过
  verify-release-assets          → 发布资产校验通过：v2.3.1-beta.4
  BUILD_OK
```

未签名、未自动安装、未启动、未打标签、未推送。

---

# Documentation

- [x] 发行说明：`docs/releases/v2.3.1-beta.4.md`
- [x] CURRENT：已追加本版本摘要并更新 TASK-005 的真机复测状态
- [x] 其余文档无需修改

---

# Git

## Commit

```text
5faa2662 chore(release): 准备 v2.3.1-beta.4 本地构建
```
实现与发布准备的提交；本记录的提交号由后续文档提交回填。

---

# Result

已产出 `v2.3.1-beta.4` 本地 Windows amd64 生产安装包，供用户从托盘完全退出应用并备份数据后
覆盖安装。安装后 Codex 选择 River／DeepSeek 等非 OpenAI 模型时由协议入口处理：
公共模型 ID 与档位在网关内还原，第三方凭据不再发往 `api.openai.com`，原先的整池 401 消失。
安装后需重启 Codex 重新加载模型目录；WebSocket 模式仍只覆盖 OpenAI 模型（握手没有正文）。

---

# Completed

2026-11-15 17:56（Asia/Singapore）
