# TASK-20261019-015: 构建 Beta.10 本地生产安装包

## Status

Completed

## Created

2026-10-19

## Request

构建新的 Beta 生产版本，供用户自行进行本地更新。

## Goal

交付包含额度查询／凭据锁修复的 Windows amd64 production Beta.10 安装包及校验信息，不自动安装或远端发布。

## Scope

- 按现有发布约定统一 Beta.10 版本和本地更新说明。
- 从已提交源码隔离构建，执行 `task verify`、NSIS 打包与资产／版本校验。
- 输出到 `release-assets/v2.0.0-beta.10/`，记录来源提交及 SHA256。

## Out of Scope

- 不自动更新、停止或替换现有生产／开发程序，不改动真实配置与凭据。
- 不推送、不打远端标签或创建公开发行；无证书时不宣称已签名。

# Analysis

当前本地安装包最新为 2.0.0-beta.9，本轮修复提交为 `ab6f5a6`（任务归档 `422fe4c`）。递增为 2.0.0-beta.10，沿用既有机器级安装身份和目录；Go／前端／ProductVersion 使用完整版本，Windows 数字版本与 NSIS 保持 2.0.0.0／2.0.0。

采用现有 `scripts/build-release.ps1`，从本任务版本提交导出的隔离源码运行，不覆盖主工作区 bin，不引入新脚本、依赖或打包捷径。脚本强制运行完整门禁与 NSIS，检查原生元数据并生成 SHA256；本机无签名时明确标记未签名。只提交本任务文件，已有未跟踪任务记录不纳入。

# Implementation Plan

1. 检查版本映射、更新说明及本地隔离打包流程。
2. 更新版本、测试预期及内置更新说明，提交本次构建源。
3. 隔离执行完整门禁、NSIS 打包和资产校验。
4. 同步 CURRENT、实际结果与任务归档，交付安装包路径。

# Implementation Result

- Go／前端／Windows ProductVersion 同步为 2.0.0-beta.10，Windows 数字版本与 NSIS 为 2.0.0.0／2.0.0；机器级安装身份不变。
- 新增随包发行说明和普通版本、v 前缀、构建元数据三种读取回归，说明额度查询、并发锁、自动激活及本地验收边界。
- 从版本提交 `21b5e2aa544b630283f1763f4ae3e06df98edb0a`（tree `186a03580668521dd19a83190a8a14c863ae17db`）导出隔离源码，沿用现有发布脚本完成完整门禁和 NSIS 打包。构建源包含额度修复 `ab6f5a6`。
- 安装器与同名 SHA256 文件输出到独立 Beta.10 目录，旧 Beta.6／7／8／9 资产保持原样。未推送、创建标签、发布、运行安装器或主动停止／替换当前程序。
- 构建日志、源码快照、只读基线和独立校验摘要保存在忽略的 `tmp/local-beta10-20261019/`，不提交构建产物、用户配置或凭据。

# Verification

- PASS：`scripts/set-release-version.ps1 -Version v2.0.0-beta.10`，完整／数字版本映射一致。
- PASS：前端版本文件及随包说明回归的定向 Prettier 检查。
- PASS：`pnpm exec vitest run src/features/about/current-release.test.ts`，23 项测试。
- PASS：隔离源码运行 `scripts/build-release.ps1 -Version v2.0.0-beta.10 -OutputDirectory <主工作区>/release-assets/v2.0.0-beta.10`，退出码 0。
- PASS：脚本内 `task --force verify`，格式、Go vet／staticcheck、ESLint、TypeScript、全部 Go 测试、107 文件／620 项前端测试、Wails 绑定生成、前端生产构建及 Windows 构建通过。
- PASS：脚本内 `task --force package`，NSIS UTF-8 机器级安装器生成成功；未使用 updateqa 或其他测试构建入口。
- PASS：原生 Windows 版本 API 在打包脚本及独立核对中验证应用完整版本、数字版本、产品名、公司名与安装器身份。
- PASS：资产验证脚本两次核对目录仅含安装器和同名 SHA256 文件；独立 SHA256 复核一致，签名状态 NotSigned。
- PASS：`go version -m` 确认 Go 1.26.2、`-tags=production`、GOOS=windows、GOARCH=amd64、CGO_ENABLED=0，不含 updateqa。
- PASS：冻结提交与隔离源码的 11 个关键源码／依赖／版本文件通过 Git blob 核对；构建过程中未引入依赖或版本漂移。
- PASS：11 项只读文件基线（生产／开发二进制、Codex 配置、旧版资产）摘要保持一致；生产 PID 18200 和启动时刻保持原样。
- 观察到开发进程由 PID 41092 自动重载为 22844，开发二进制摘要未变；未主动调用启动／停止命令。独立进程核对先因时间字符串类型／开发版重载中断，改用 UTC 时刻并区分生产和开发核对，未将开发进程也记为“保持不变”。未修改现有 watcher 配置。
- PASS：`git diff --check`、任务及发行说明 UTF-8 无 BOM、来源提交与修复提交祖先关系检查。
- 非阻塞既有告警：jsdom scrollTo 未实现、Vite 大 chunk 与 Tailwind 构建性能提示；未隐藏或跳过检查。

未执行：真实覆盖安装／管理员确认、生产新版本启动、真实过期授权／上游网络与自动激活、GUI 外壳验收。打包成功不等同于这些场景通过。

## Artifact

- 安装器：`E:/go/new/wails3-v/OmniProxy/release-assets/v2.0.0-beta.10/OmniProxy-Setup-v2.0.0-beta.10-windows-amd64.exe`。
- 校验文件：同目录 `OmniProxy-Setup-v2.0.0-beta.10-windows-amd64.exe.sha256`。
- 大小：19370853 bytes（约 18.5 MiB）。
- SHA256：`d91f4459a9cd641b7f2f4e30a17e6d2f1dbb24d85476e18385e2ee5586a41025`。
- 签名状态：NotSigned。

# Documentation

已新增 Beta.10 发行说明、更新 CURRENT 最新本地版本与本任务结果；不改动历史发行说明或现有构建／升级规范。

# Git

`21b5e2aa544b630283f1763f4ae3e06df98edb0a chore(release): 准备 Beta.10 本地生产安装包`。

仅本地提交，不推送；结果与归档由后续文档提交保存。已有不属于本任务的未跟踪任务记录保持不变。

# Result

Beta.10 Windows amd64 production 本地安装包与 SHA256 已生成并验证，可由用户自行覆盖更新测试。生产程序与真实配置未替换，安装包未签名，未发布远端。

## Completed

2026-10-19（Asia/Singapore）
