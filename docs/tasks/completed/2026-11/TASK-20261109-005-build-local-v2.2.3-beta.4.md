# TASK-20261109-005: 构建本地 v2.2.3-beta.4 生产安装包

## Status

Completed

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
- 2026-11-09 15:14：版本映射与双语发行说明完成，提交 `60dda7a3`。
- 2026-11-09 15:17：生产打包成功，生成新安装器与应用 EXE。
- 2026-11-09 15:19：原生版本资源、发布资产与摘要校验通过；任务归档。

---

# Implementation Result

- 应用版本、前端包版本与 Windows `ProductVersion` 已提升为 `2.2.3-beta.4`；Windows 数字版本保持 `2.2.3.0`，NSIS 基础版本保持 `2.2.3`。版本映射脚本校验通过，`build/config.yml` 与 NSIS 模板无需改变。
- 新增双语发行说明 `docs/releases/v2.2.3-beta.4.md`，说明推理等级记录与展示、迁移 046、历史记录显示 `—`、不参与聚合与未签名边界。
- 版本与文档已先提交为 `60dda7a3`，随后执行生产打包；运行中的已安装实例未被启动、停止或覆盖。

## Build and Artifacts

- 构建命令（NSIS 加入本次 PATH）：
  ```text
  task --force package DEV=false BIN_DIR=tmp/local-v2.2.3-beta.4/bin
  ```
- 过程：Go 模块整理、绑定与图标生成、前端生产构建（含 `tsc`）、`-tags production` Go 编译、NSIS 打包。

| 产物 | 大小 |
| --- | --- |
| `release-assets/v2.2.3-beta.4/OmniProxy-Setup-v2.2.3-beta.4-windows-amd64.exe` | 20,594,633 bytes |
| 同目录 `.exe.sha256` | 114 bytes |
| `tmp/local-v2.2.3-beta.4/bin/omniproxy.exe` | 23,635,968 bytes |

安装器 SHA256：

```text
367a8d4240656e3d1d7eb3bc325b6cf8b3c867fdd5e1900e3fa1b9a1f7bce4f7
```

生产应用 EXE SHA256：

```text
95c69d33e32e89044113d83112e4f8a54bb9781fdb00dc6b49a67393498bfdf6
```

## Limits

- 安装包 `Get-AuthenticodeSignature` 状态为 `NotSigned`；未签名，Windows 可能提示未知发布者。
- 未安装、未停止或启动应用；用户应从托盘完全退出后再覆盖更新。
- 未打标签、未推送、未上传或公开发布，未修改证书信任和真实用户数据。
- 本次未重跑完整测试；沿用 TASK-20261109-004 的 Go／前端测试结果，打包本身执行了前端生产编译与 Go 生产编译。

# Verification

## Tests

- [ ] Unit Tests（本次仅版本元数据与文档，沿用 TASK-20261109-004 结果）
- [ ] Integration Tests（同上）
- [ ] E2E Tests（不适用）

## Quality Checks

- [x] Format（版本脚本与文档变更；生产前端构建包含 `tsc`）
- [ ] Lint（沿用 TASK-20261109-004 结果）
- [ ] Type Check（生产前端构建包含 `tsc`）
- [ ] Test（沿用 TASK-20261109-004 结果）
- [x] Build
- [ ] Verify（未重复完整质量门）

## Verification Result

```text
scripts/set-release-version.ps1 -Version v2.2.3-beta.4   PASS
task --force package DEV=false ...                       PASS
scripts/test-windows-metadata.ps1                        PASS
scripts/verify-release-assets.ps1                        PASS
go version -m <app>                                      -tags=production -trimpath
Get-AuthenticodeSignature <installer>                    NotSigned
```

- 应用原生版本资源：`ProductVersion=2.2.3-beta.4`、数字版本 `2.2.3.0`；安装器：`ProductVersion=2.2.3`、数字版本 `2.2.3.0`；产品与公司身份保持 OmniProxy。
- `release-assets/v2.2.3-beta.4` 仅包含安装器与 SHA256 侧车；独立复算摘要与侧车一致。
- 生产前端产物包含新「推理等级」文案（`frontend/dist` 于 15:16:37 重建，应用 EXE 于 15:16:40 编译）。
- 未重跑测试或 `task verify`；不把上一任务结果冒称为本版本重新通过完整质量门。

# Documentation

- [x] Release notes `docs/releases/v2.2.3-beta.4.md`
- [x] CURRENT
- [x] 本任务记录

# Deviations

- 本次仅版本元数据与文档变化，未重复完整测试／`task verify`，沿用 TASK-20261109-004 结果；打包本身仍执行前端与 Go 生产编译及产物校验。
- 未使用 `scripts/build-release.ps1`（其中会串行执行 `task verify`），改为直接 `task --force package`，原因同上一条。
- 未推送远端、未打标签、未公开发布，符合本地 Beta 交付范围。

# Git

## Commit

```text
60dda7a3 chore(release): prepare local v2.2.3-beta.4
```

- 版本与文档先提交，再执行打包；未打标签、未推送远端。
- 无关未跟踪任务记录 `TASK-20261029-010-subq-provider.md` 保持原样，未纳入提交。
- 本任务记录归档与 CURRENT 收尾单独提交。

# Result

已提交 `v2.2.3-beta.4` 版本源码与发行说明，并生成本地 Windows amd64 生产安装包。安装器 20,594,633 bytes，SHA256 `367a8d42…bce4f7`；应用 EXE 23,635,968 bytes，SHA256 `95c69d33…8bfdf6`。原生版本资源与发布资产校验通过，安装包未签名、未安装、未推送，供用户从托盘退出后手动覆盖更新。

# Completed

2026-11-09 15:20（Asia/Singapore）
