# TASK-20261109-005: 构建本地 v2.2.3-beta.4 生产安装包

## Status

In Progress

## Created

2026-11-09 15:12（Asia/Singapore）

## Request

用户要求：提交代码，然后构建新的 Beta 生产版本供本地更新，并完成任务 Markdown 更新。

## Goal

把已完成的「请求明细记录并展示推理等级」（TASK-20261109-004，`e8dfc47e`）纳入下一本地 Beta：`v2.2.3-beta.4`。同步应用／前端版本、Windows 版本资源、NSIS 基础版本与双语发行说明，提交后构建 Windows amd64 生产安装包，记录实际产物、摘要与未签名／未安装边界。

## Scope

- 将应用版本从 `2.2.3-beta.3` 提升到 `2.2.3-beta.4`；Windows 数字版本保持 `2.2.3.0`，NSIS 基础版本保持 `2.2.3`。
- 新增双语发行说明 `docs/releases/v2.2.3-beta.4.md`，说明推理等级记录与展示、迁移 046 与本地覆盖更新方式。
- 提交版本与文档，再构建生产 EXE、NSIS 安装器及摘要侧车。
- 检查 Windows 原生版本资源、产物摘要与目录内容；补齐本任务记录和 CURRENT。

## Out of Scope

- 不推送远端、不打标签、不公开发布。
- 不自动安装、停止或启动用户当前实例，不修改真实用户数据。
- 不修改推理等级运行时代码、路由语义或新增依赖／迁移。

## Existing Behavior

- 当前应用与前端版本为 `2.2.3-beta.3`，Windows 数字版本 `2.2.3.0`，NSIS 基础版本 `2.2.3`。
- 代码工作区在 TASK-20261109-004 后干净（仅无关未跟踪文件 `docs/tasks/completed/2026-10/TASK-20261029-010-subq-provider.md`）。
- 本地发布资产目录 `release-assets/v2.2.3-beta.4` 尚不存在。

## Expected Behavior

- 版本源、前端包版本、Windows 资源与发行说明一致为 `2.2.3-beta.4`。
- 生产构建和 NSIS 打包成功，生成新的安装器及 SHA256 侧车；应用 EXE 使用 `production` 标签。
- 现有已安装实例不被启动、停止或覆盖；用户从托盘完全退出后可手动覆盖更新。

---

# Analysis

- 版本入口：`internal/version/version.go`、`frontend/package.json`、`build/config.yml`、`build/windows/info.json`、`build/windows/nsis/wails_tools.nsh`；由 `scripts/set-release-version.ps1` 校验并同步。
- 打包入口：`task --force package DEV=false BIN_DIR=tmp/local-v2.2.3-beta.4/bin`；依赖 Go 模块整理、绑定／图标、前端生产构建、`-tags production` 编译和 NSIS。
- 发布资产目录必须为空；安装器固定输出为 `bin/OmniProxy-amd64-installer.exe`，复制为 `OmniProxy-Setup-v2.2.3-beta.4-windows-amd64.exe` 并生成 `.sha256`。
- 质量门：本次代码改动已在实现任务中跑过完整 Go／前端测试、lint、typecheck 与生产编译；本任务只在版本元数据变更后重新打包并核对产物，不重复完整测试。

## Risks

- 版本资源与发行说明不一致会导致安装器显示错误版本；使用现有脚本并逐项核对。
- 打包过程若 Taskfile 依赖卡住，需要记录实际阶段并用等价命令完成，不隐瞒。
- NSIS 安装器未签名，Windows 可能提示未知发布者；不修改签名信任。

---

# Implementation Plan

1. 创建本任务记录并置为 In Progress。
2. 更新应用／前端版本，新增 `docs/releases/v2.2.3-beta.4.md`，运行版本映射脚本并检查差异。
3. 提交版本与文档。
4. 执行生产打包，核对产物大小、SHA256、Windows 版本资源和未签名状态。
5. 回填本任务记录与 CURRENT，归档并提交文档。

---

# Progress

- 2026-11-09 15:12：创建任务记录，Status = In Progress。

---

# Implementation Result

待填写。

---

# Verification

## Tests

- [ ] Unit Tests
- [ ] Integration Tests
- [ ] E2E Tests

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

- [ ] Release notes `docs/releases/v2.2.3-beta.4.md`
- [ ] CURRENT
- [ ] 本任务记录

---

# Deviations

待填写。

---

# Git

## Commit

```text
Not committed by Agent.
```

---

# Result

待填写。

---

# Completed

待填写。
