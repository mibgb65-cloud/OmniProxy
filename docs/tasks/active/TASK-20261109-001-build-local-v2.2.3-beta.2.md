# TASK-20261109-001: 构建本地 v2.2.3-beta.2 生产安装包

## Status

In Progress

## Created

2026-11-09 09:10（Asia/Singapore）

## Request

用户要求：提交代码，然后构建新的 Beta 生产版本供本地更新，并完成任务 Markdown 更新。

## Goal

把刚完成的 OpenAI 目录自动同步与兜底轮换纳入下一本地 Beta：`v2.2.3-beta.2`。同步应用／前端版本、Windows 版本资源、NSIS 基础版本与双语发行说明，提交后再构建 Windows amd64 生产安装包，记录实际产物、摘要与未签名／未安装边界。

## Scope

- 将应用版本从 `2.2.3-beta.1` 提升到 `2.2.3-beta.2`；Windows 数字版本保持 `2.2.3.0`，NSIS 基础版本保持 `2.2.3`。
- 新增双语发行说明 `docs/releases/v2.2.3-beta.2.md`，说明目录自动同步、兜底轮换、模型目录页只读状态，以及无迁移、未签名和本地覆盖更新方式。
- 提交版本与文档，再构建生产 EXE、NSIS 安装器及摘要侧车。
- 检查 Windows 原生版本资源、产物摘要与目录内容；补齐本任务记录和 CURRENT。

## Out of Scope

- 不推送远端、不打标签、不公开发布。
- 不自动安装、停止或启动用户当前实例，不修改真实用户数据。
- 不修改目录版本语义、路由行为或新增依赖／迁移。
- 除版本元数据与打包外，不扩展为其他功能或重构任务。

## Existing Behavior

- 当前应用与前端版本为 `2.2.3-beta.1`，Windows 数字版本 `2.2.3.0`，NSIS 基础版本 `2.2.3`。
- 上一任务 `TASK-20261107-001` 已提交目录自动同步与兜底轮换，实现提交 `d6f413b0`；完整 Go／前端测试与生产构建已通过。
- 本地发布资产目录 `release-assets/v2.2.3-beta.2` 尚不存在。

## Expected Behavior

- 版本源、前端包版本、Windows 资源与发行说明一致为 `2.2.3-beta.2`。
- 生产构建和 NSIS 打包成功，生成新的安装器及 SHA256 侧车；应用 EXE 使用 `production` 标签。
- 现有已安装实例不被启动、停止或覆盖；用户从托盘完全退出后可手动覆盖更新。

---

# Analysis

- 版本入口：`internal/version/version.go`、`frontend/package.json`、`build/config.yml`、`build/windows/info.json`、`build/windows/nsis/wails_tools.nsh`；由 `scripts/set-release-version.ps1` 校验并同步。
- 打包入口：`task --force package DEV=false BIN_DIR=tmp/local-v2.2.3-beta.2/bin`；依赖 Go 模块整理、绑定／图标、前端生产构建、`-tags production` 编译和 NSIS。
- 发布资产目录必须为空，避免覆盖既有安装包；安装器固定输出后复制为 `OmniProxy-Setup-v2.2.3-beta.2-windows-amd64.exe` 并生成 `.sha256`。
- 本次只改版本与文档，不改变运行时代码；质量门沿用上一任务结果，打包前仍执行版本一致性、生产编译和产物检查。

## Risks

- 版本资源与发行说明不一致会导致安装器显示错误版本；使用现有脚本并逐项核对。
- 打包过程若 Taskfile 依赖卡住，需要记录实际阶段并用等价命令完成，不隐瞒。
- NSIS 安装器未签名，Windows 可能提示未知发布者；不修改签名信任。

---

# Implementation Plan

1. 创建本任务记录并置为 In Progress。
2. 更新应用／前端版本，新增 `docs/releases/v2.2.3-beta.2.md`，运行版本映射脚本并检查差异。
3. 提交版本与文档。
4. 执行生产打包，核对产物大小、SHA256、Windows 版本资源和未签名状态。
5. 回填本任务记录与 CURRENT，归档并提交文档。

---

# Progress

- 2026-11-09 09:10：创建任务记录，Status = In Progress；尚未修改版本或构建。

---

# Implementation Result

待填写。

---

# Verification

## Tests

- [ ] Unit Tests
- [ ] Integration Tests
- [ ] E2E Tests

本次仅版本元数据与文档变化；沿用 `TASK-20261107-001` 已通过的 Go／前端测试结果，不重复运行完整测试，打包阶段执行生产编译和产物检查。

## Quality Checks

- [ ] Format
- [ ] Lint
- [ ] Type Check
- [ ] Test
- [ ] Build
- [ ] Verify

## Verification Result

待填写。

---

# Documentation

- [ ] Release notes `docs/releases/v2.2.3-beta.2.md`
- [ ] CURRENT
- [ ] 本任务记录

---

# Deviations

None.

---

# Git

## Commit

```text
Not committed by Agent.
```

---

# Result

待完成。

---

# Completed

Pending
