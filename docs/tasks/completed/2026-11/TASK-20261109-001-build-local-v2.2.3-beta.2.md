# TASK-20261109-001: 构建本地 v2.2.3-beta.2 生产安装包

## Status

Completed

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
- 2026-11-09 09:11：版本映射、双语发行说明完成并提交 `46745cb1`。
- 2026-11-09 09:12：生产打包成功；首次因 NSIS 不在 PATH 失败，补齐 PATH 后生成新安装器。
- 2026-11-09 09:15：原生版本资源、生产标签、摘要与发布资产目录核验通过；任务归档。

---

# Implementation Result

- 应用版本、前端包版本与 Windows `ProductVersion` 已提升为 `2.2.3-beta.2`；Windows 数字版本保持 `2.2.3.0`，NSIS 基础版本保持 `2.2.3`。版本映射脚本校验通过，`build/config.yml` 与 NSIS 模板无需改变。
- 新增双语发行说明 `docs/releases/v2.2.3-beta.2.md`，说明 OpenAI 目录自动同步、兜底轮换、模型目录页只读状态、无迁移与未签名边界。
- 版本与文档已先提交为 `46745cb1`，随后执行生产打包；运行中的已安装实例未被启动、停止或覆盖。

## Build and Artifacts

- 构建命令（Git Bash，NSIS 加入 PATH）：
  ```text
  task --force package DEV=false BIN_DIR=tmp/local-v2.2.3-beta.2/bin
  ```
- 过程：Go 模块整理、绑定与图标生成、前端生产构建、`-tags production` Go 编译、NSIS 打包。
- 首次直接执行时 `makensis` 不在 PATH，未生成新安装器；将 `C:\Program Files (x86)\NSIS` 加入 PATH 后重跑成功。临时脚本位于忽略目录 `tmp/local-v2.2.3-beta.2/`。

| 产物 | 大小 |
| --- | --- |
| `release-assets/v2.2.3-beta.2/OmniProxy-Setup-v2.2.3-beta.2-windows-amd64.exe` | 20,572,398 bytes |
| 同目录 `.exe.sha256` | 114 bytes |
| `tmp/local-v2.2.3-beta.2/bin/omniproxy.exe` | 23,589,376 bytes |

安装器 SHA256：

```text
f0bd1994de7536cdde99468662e75a4ac5204f340e6b8a44f099fe7de5045770
```

生产应用 EXE SHA256：

```text
6c21c3e7fefa7d3aa2e683c07a109c86eaa97a045da4f82b133eeef5f0b79662
```

## Limits

- 安装包 `Get-AuthenticodeSignature` 状态为 `NotSigned`；未签名，Windows 可能提示未知发布者。
- 未安装、未停止或启动应用；用户应从托盘完全退出后再覆盖更新。
- 未打标签、未推送、未上传或公开发布，未修改证书信任和真实用户数据。
- 本次未重跑完整测试或 `task verify`；沿用 `TASK-20261107-001` 通过的 Go／前端测试与生产构建结果，打包本身执行了前端生产编译与 Go 生产编译。

---

# Verification

## Tests

- [ ] Unit Tests（本次仅版本元数据与文档，沿用上一任务结果）
- [ ] Integration Tests（同上）
- [ ] E2E Tests（不适用）

本次仅版本元数据与文档变化；沿用 `TASK-20261107-001` 已通过的 Go／前端测试结果，不重复运行完整测试，打包阶段执行生产编译和产物检查。

## Quality Checks

- [x] Format（gofmt、Prettier 检查通过）
- [ ] Lint（沿用上一任务结果）
- [ ] Type Check（生产前端构建包含 `tsc`）
- [ ] Test（沿用上一任务结果）
- [x] Build
- [ ] Verify（未重复完整质量门）

## Verification Result

```text
scripts/set-release-version.ps1 -Version v2.2.3-beta.2   PASS
task --force package DEV=false ...                      PASS
scripts/test-windows-metadata.ps1                       PASS
scripts/verify-release-assets.ps1                       PASS
go version -m <app>                                     production / trimpath / go1.26.2
Get-AuthenticodeSignature <installer>                   NotSigned
```

- 应用与安装器原生版本资源：应用 `ProductVersion=2.2.3-beta.2`，安装器基础版本 `2.2.3`，数字版本均为 `2.2.3.0`，公司与产品身份保持 OmniProxy。
- `release-assets/v2.2.3-beta.2` 仅包含安装器与 SHA256 侧车；独立复算摘要与侧车一致。
- 未重跑测试或 `task verify`；不把上一任务结果冒称为本版本重新通过完整质量门。

---

# Documentation

- [x] Release notes `docs/releases/v2.2.3-beta.2.md`
- [x] CURRENT
- [x] 本任务记录

---

# Deviations

- 首次打包因 `makensis` 不在 PATH 失败；仅把 NSIS 目录加入本次构建 PATH 后重跑，未修改项目构建脚本或签名设置。
- 按“仅版本元数据与文档”的范围未重复完整测试／`task verify`，沿用上一任务结果；打包本身仍执行前端与 Go 生产编译及产物校验。

---

# Git

## Commit

```text
46745cb1 chore(release): 准备本地 v2.2.3-beta.2 生产安装包
```

- 版本与文档先提交，再执行打包；未打标签、未推送远端。
- 无关未跟踪任务记录 `TASK-20261029-010-subq-provider.md` 保持原样，未纳入提交。
- 本任务记录归档与 CURRENT 收尾单独提交。

---

# Result

已提交 `v2.2.3-beta.2` 版本源码与发行说明，并生成本地 Windows amd64 生产安装包。安装器 20,572,398 bytes，SHA256 `f0bd1994…e5045770`；应用 EXE 23,589,376 bytes，SHA256 `6c21c3e7…f0b79662`。原生版本资源与发布资产校验通过，安装包未签名、未安装、未推送，供用户从托盘退出后手动覆盖更新。

---

# Completed

2026-11-09 09:15（Asia/Singapore）
