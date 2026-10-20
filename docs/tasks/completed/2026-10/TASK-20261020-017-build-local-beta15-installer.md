# TASK-20261020-017: 构建 Beta.15 本地 production 安装包

## Status

Completed

## Created

2026-10-20

## Request

完成当前修复后构建新的 Beta production 版本，供用户自行本地更新。

## Goal

交付包含当前已提交改动的 Windows amd64 production Beta 安装包，验证版本、来源和 SHA256；不自动安装或公开发布。

## Scope

版本递增、发布构建与质量门、资产／来源／版本校验、隔离启动验证及交付记录。

## Out of Scope

- 不安装、替换或停止用户当前生产程序，不修改真实账号、凭据、聊天历史或客户端配置。
- 不上传发布、不推送 Git、不覆盖旧版本发布资产。
- 不提交既有 Atria 草稿或其他无关变更。

# Analysis

- 当前最新本地包为 Beta.14；Beta.15 目录不存在，沿用完整 SemVer／数字 Windows 版本映射，保留安装身份。
- 使用既有发布脚本、NSIS、原生版本与资产校验，源码先提交冻结，再用 `git archive` 导出受跟踪源码到独立目录。
- 纳入 TASK-014／015／016：额度展示与 Token 块、Google 额度持久化、River 队列／均衡与失败冷却、Claude Messages 长会话修复。
- 初步确认 18 个旧发布资产及一个在运行的 production 进程；记录保护基线后不停止、不覆盖或回滚该进程／安装目录。
- 真实安装覆盖及原聊天续接不冒充构建验证；本次仅交付用户手动更新。

# Implementation Plan

1. 核对版本／构建规则，保护旧资产和当前安装状态。
2. 更新版本并提交明确的构建源，使用现有 production 流程构建和验证。
3. 校验安装包、摘要和来源，进行隔离启动验证，交付文件并归档任务。

# Implementation Result

## Changes

- 应用、前端和 Windows 完整版本同步为 `2.0.0-beta.15`；数字版本仍为 `2.0.0.0`，安装身份不变。
- 新增随包发布说明，覆盖额度／Token 展示、Google 额度留存、River 使用方式及 Claude 长会话修复，并明确实际能力和安装后验收边界。
- 源码冻结为 `e188d1cd4252d2de03fcce84d7dbe4eb56a79424`，核对 TASK-014／015／016 的实现提交均为祖先。
- 仅导出受跟踪源码至 `tmp/local-beta15-20261020/source/`，通过原发布脚本完成完整质量门、production 构建和 NSIS 打包；未创建长期工作树或复制真实数据。
- 安装器与 SHA256 输出至独立 `release-assets/v2.0.0-beta.15/`，未覆盖旧资产。

## Artifacts

- 安装器：`E:/go/new/wails3-v/OmniProxy/release-assets/v2.0.0-beta.15/OmniProxy-Setup-v2.0.0-beta.15-windows-amd64.exe`
- 校验文件：同名追加 `.sha256`
- 大小：19,533,652 bytes
- SHA256：`99617167f1168f557eca05da61407eca45c5463ecf33f51e41de76fb69653d06`
- 数字签名：`NotSigned`
- 构建日志、来源和核对记录在忽略目录 `tmp/local-beta15-20261020/`，不作为公开发布文件。

# Verification

- 保护基线已记录：18 个旧发布资产、3 个当前安装文件及 1 个 production 进程；均不由构建任务更改。
- `scripts/set-release-version.ps1 -Version v2.0.0-beta.15` 通过。
- 定向 Prettier 检查和 38 项当前发行说明测试通过，`internal/version` 测试通过。
- PASS：冻结源码内 `scripts/build-release.ps1 -Version v2.0.0-beta.15 -OutputDirectory <workspace>/release-assets/v2.0.0-beta.15`，日志完整结束，包含质量门、production 重建、NSIS、版本和资产检查成功信息。中断后未盲目重跑或覆盖产物。
- PASS：完整质量门包含格式、Go vet／staticcheck、ESLint、TypeScript、全部 Go 测试、118 个前端文件／703 项测试及实际 Windows 构建。
- PASS：额外 `go test ./cmd/...`、`go vet ./cmd/...`；未执行登录试验命令。
- PASS：1,725 项受跟踪源码文本与构建配置逐一核对 Git blob；无内容差异，只有 Windows `info.json` 的 CRLF／LF 等价差异。未混入 `.env`、凭据、临时数据或未跟踪 Atria 草稿。
- PASS：独立校验安装器名称／两文件目录／SHA256，原生版本 API 确认应用完整版本 Beta.15、安装器数字版本 `2.0.0`、双方数字资源 `2.0.0.0` 及产品身份。
- PASS：`go version -m` 确认 `production`、`windows/amd64`、`CGO_ENABLED=0`，没有 `updateqa`。
- PASS：最终产物的隔离 test profile 启动、关于页 `2.0.0-beta.15` 及本地健康信息校验；使用独立数据／配置／WebView 目录与端口，验收进程已按路径核对后关闭。未执行安装器。
- PASS：18 个旧发布资产摘要及大小保持不变。
- CHANGED（不记为未变）：保护基线 21 个文件中 19 个未变；安装目录 `OmniProxy.exe`／`uninstall.exe` 和原 production 进程发生变化。
- CONFIRMED：中断后只读核对显示已安装应用为 Beta.15，SHA256 为 `a18a9272c594a6110e00f47a9fda4619a7913347bf1d9f1bb38c60fe1476de06`，与隔离构建应用完全一致；当前 PID 34940，启动时间 2026-10-21 00:40:18（Asia/Singapore）。本任务未调用安装器或停止原进程，不能推断外部更新操作者，也未回滚安装。
- 真实覆盖安装、账号数据保留及原聊天续接仍由用户本地验收，不用构建／合成测试代替。
- 既有 jsdom `scrollTo` 未实现、Vite 大 chunk 和构建插件耗时提示为非阻断项。

# Documentation

已同步随包发布说明、说明读取测试、CURRENT 和本任务；无技术栈、架构或发布流程变更。

# Deviations

构建任务开始于 2026-10-20，中断后于 2026-10-21 继续核对已完成的后台产物，没有重复构建。观察到安装／进程在此期间改变，已保留严格保护检查失败结果，并单独核对版本与摘要，不将原安装状态未变包装为通过。

# Git

`e188d1cd4252d2de03fcce84d7dbe4eb56a79424` — `chore(release): 准备 Beta.15 本地生产安装包`

仅提交本任务版本、发布说明、对应测试和文档；未提交 Atria 草稿，未创建发布标签、推送或公开发布。

# Result

Beta.15 Windows amd64 production 安装包已生成并验证，可交付本地更新／留存。当前安装目录在核对时已经是同一构建，但本任务没有执行安装动作；不把这项观察当作真实数据保留和原聊天续接已验收。安装包未签名，未公开发布。

# Completed

2026-10-21（Asia/Singapore）
