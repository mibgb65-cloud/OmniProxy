# TASK-20261023-008: 构建 Beta.24 本地生产安装包（含工作区全部改动）

## Status

In Progress

## Created

2026-10-23（Asia/Singapore）

## Request

用户要求把当前全部代码（包括其在其他位置完成的改动）打成 Beta.24 生产安装包，供本机手动更新。

## Goal

基于当前工作区完整状态（HEAD `b2a2fc8` + 未提交的 Beta.22／23 图片能力改动 + TASK-20261023-007 推理档位修复）产出 `release-assets/v2.0.0-beta.24/` 安装器与 SHA256，并通过完整质量门、版本资源与资产校验。

## Scope

- 版本同步至 `2.0.0-beta.24`。
- 新增 `docs/releases/v2.0.0-beta.24.md`。
- 完整质量门、NSIS 打包、Windows 元数据、SHA256、资产验证。

## Out of Scope

- 不提交他人未提交的改动（用户未要求），只按工作区现状构建。
- 不自动安装、不停止或替换正在运行的生产进程、不打标签、不推送。

## Background

工作区包含上一轮会话之外完成的改动：River／Claude 图片输入转发（TASK-003）、客户端目录按确认能力声明图片模态（TASK-005），以及对应的 Beta.22／23 构建记录。这些改动尚未提交，但用户明确要求一并打包。

---

# Implementation Plan

1. 备份并同步 Beta.24 版本号。
2. 撰写发布说明，覆盖 Beta.22／23／24 的净变化。
3. 依次执行 `set-release-version` → `task --force verify` → `task --force package` → 元数据校验 → 资产复制与 SHA256 → 资产验证。
4. 记录结果。

---

# Progress

- 已确认 HEAD 版本号为 `2.0.0-beta.23`，工作区有 24 个已跟踪文件修改与 14 个未跟踪文件。
- 首次元数据校验失败：只改了 `version.go` 与 `package.json`，未同步 `build/windows/info.json`，应用二进制仍带 `2.0.0-beta.23` 资源。改用 `scripts/set-release-version.ps1` 正确同步后重新打包并通过。
- `set-release-version.ps1` 再次用 `ConvertTo-Json` 重排 `info.json` 并引入 CRLF，已归一化回原缩进与 LF，diff 仅 `ProductVersion` 一行。
- 版本资源改动发生在首次 `task --force package` 之后，因此归档前重新执行了一次打包，确保安装器与最终资源一致。

---

# Implementation Result

## Changes

- 版本同步至 `2.0.0-beta.24`：`internal/version/version.go`、`frontend/package.json`、`build/windows/info.json`（仅 `0409.ProductVersion` 一行，Windows 数字版本维持 `2.0.0.0`）。
- 新增 `docs/releases/v2.0.0-beta.24.md`，覆盖 Beta.22／23 图片能力与本次推理档位修复。
- 产出 `release-assets/v2.0.0-beta.24/` 两个文件，未覆盖任何历史发布目录。

## Files Changed

- `internal/version/version.go`、`frontend/package.json`、`build/windows/info.json`
- `docs/releases/v2.0.0-beta.24.md`（新增）
- `docs/tasks/active/TASK-20261023-008-build-local-beta24-installer.md`（本记录）

## Technical Notes

本版按用户要求包含工作区**全部**改动，包括尚未提交的 River／Claude 图片输入转发（TASK-20261023-003）与客户端图片模态声明（TASK-20261023-005）。构建使用工作区现状，未替用户提交这些改动。

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

- PASS：`task --force verify` exit 0 —— 格式 → lint → 类型 → Go 全套 → 121 文件／724 项前端测试 → Windows production 构建。
- PASS：`task --force package` exit 0（makensis 3.12）。
- PASS：`scripts/test-windows-metadata.ps1 -Version v2.0.0-beta.24` 输出「Windows 应用和安装器版本资源校验通过」（应用与安装器均经原生版本 API 核对）。
- PASS：`scripts/verify-release-assets.ps1 -Version v2.0.0-beta.24` 输出「发布资产校验通过」。
- PASS：安装器 19,580,773 bytes，SHA256 `cd9dcaacd8de71fa4ba6ff1329ab91eb3f39468caceade0b9aaa154d2f5e91ec`。
- PASS：`release-assets/` 下 19 个目录各 2 个文件，Beta.6–Beta.23 历史资产未变；Beta.24 为新建目录。
- PASS：`build/windows/info.json` 无 CRLF，diff 仅一行；`gofmt -l` 与 `go vet` 无输出。
- 未执行：代码签名、自动安装、停止或替换生产进程。生产应用仍为 PID 65100／`E:\codexproxy\OmniProxy\OmniProxy.exe`（启动于 2026-10-22 20:08:21），本任务未触碰；未打标签、未推送。
- 未执行：真实覆盖安装、数据保留、Claude 强度控件与 Harness Effort 菜单的界面验收，由用户本地确认。

---

# Documentation

- [x] `docs/releases/v2.0.0-beta.24.md`
- [ ] CURRENT：未修改——`docs/plans/CURRENT.md` 含用户其他未提交改动，避免混入本任务内容

---

# Deviations

- `scripts/build-release.ps1` 未整体执行（PATH 缺 `uname`、`task` 把 stderr 当终止错误），改为按该脚本的相同步骤逐项执行，全部返回 0，未修改脚本与质量门。
- 首次构建的安装器因版本资源未同步而作废，已在同步后重新打包并重新校验。

---

# Git

## Commit

```text
90681e0 docs: 同步 Codex 透明转发决策、产品文档与 2026-10 任务记录
```

版本映射随后续构建演进为 2.0.0-beta.35（9c7d61b）。

---

# Result

Beta.24 production 本地安装包已构建，包含工作区全部代码（Beta.22／23 图片能力 + 推理档位修复），通过完整质量门、Windows 版本资源与发布资产校验。未签名、未自动安装、未打标签、未推送，供用户本地覆盖更新。

---

# Remaining Risks

- 工作区改动未提交即构建，若后续提交时被修改，此安装包的来源将无法与某次提交对应。
- 安装包未签名，Windows 会提示未知发布者。
- 升级后需重新应用各客户端配置并完全退出重启，否则客户端可能沿用缓存菜单。
- 真实覆盖安装、数据保留与两个客户端的强度 UI 表现未经本任务验证。
