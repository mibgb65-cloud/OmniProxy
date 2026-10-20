# TASK-20261020-013: 构建 Beta.14 本地生产安装包

## Status

Completed

## Created

2026-10-20

## Request

用户要求构建新的 Beta 生产版本，供其自行本地更新。

## Goal

基于当前已提交功能构建 `v2.0.0-beta.14` Windows amd64 production 全量安装器与 SHA256 文件，交付独立资产，不自动安装或发布。

## Scope

- 同步应用、前端及 Windows 版本元数据，编写离线发布说明。
- 冻结本地提交并导出受跟踪源码，在独立目录执行原有发布脚本、完整质量门和 NSIS 打包。
- 校验安装器／应用原生版本、production 标签、摘要及构建来源。
- 保持旧发布资产、已安装程序、生产账号数据和无关草稿不变。

## Out of Scope

- 自动安装、运行安装器、停止用户应用、推送代码／标签或发布 GitHub Release。
- 将 DeepSeek 文本通道包装为支持图片，或将美元参考估值表述为官方余额。
- 修改未跟踪的 Atria 任务草稿或推进 TASK-008 的客户端试验验收。

# Analysis

当前最新本地安装器和应用／前端版本为 Beta.13，目标 Beta.14 尚不存在。
本次包含已提交的美元额度估算、图片历史安全诊断，以及整理到 main 的启动网关偏好；
旧版 Codex 登录同步试验工具不由安装包自动执行。

沿用 `scripts/build-release.ps1`：完整 `task --force verify`、production 构建、
NSIS 打包、原生版本和发布资产验证。完整版本改为 `2.0.0-beta.14`，
Windows 数字版本仍为 `2.0.0.0`，产品安装身份不变。

NSIS 已安装。只导出受跟踪源码到新的 `tmp/local-beta14-20261020/source/`；
不复制真实数据、不创建长期工作树，不包含未跟踪 Atria 草稿或前一任务的合成验收目录。

# Implementation Plan

1. 核对版本与发布脚本，记录原有资产／已安装程序保护基线。
2. 更新 Beta.14 版本及发布说明，通过版本映射检查并提交冻结源码。
3. 导出受跟踪源码至独立目录，运行原发布脚本完成完整验证和生产打包。
4. 核对版本、来源、SHA256、签名状态及保护基线，更新任务记录并交付安装器。

# Implementation Result

- 应用、前端及 Windows 完整版本更新为 Beta.14，保留数字版本 `2.0.0.0` 和既有安装身份。
- 新增随包发布说明及三种版本格式读取回归，准确说明美元估值、图片能力边界、启动偏好及迁移 037。
- 从冻结提交导出受跟踪源码，在独立目录通过原发布脚本完成完整质量门、Windows production 构建及 NSIS 打包。
- 安装器及 SHA256 独立输出到 `release-assets/v2.0.0-beta.14/`，未覆盖旧发布资产。
- 本任务未调用安装器、未停止生产进程、未读取真实凭据／账号数据、未推送或发布远端版本。

# Progress

- 已同步应用、前端和 Windows 完整版本为 Beta.14，数字资源与安装身份保持不变。
- 已新增随包发布说明，以及普通版本、带 v 前缀和本地构建元数据三种运行版本的说明读取回归。
- 已在忽略目录保存 19 个旧资产／已安装程序文件、1 个生产进程的保护基线。
- `scripts/set-release-version.ps1 -Version v2.0.0-beta.14` 与定向 Prettier 检查通过。
- 发行说明读取 35 项定向测试通过。
- 构建源码冻结为 `1c5c44f13668fbe270153dd4d0c0c52f1adeba68`；已核对美元估算、图片诊断和启动网关实现均为祖先提交。
- 仅通过 `git archive` 导出受跟踪源码；初次核对 1693 项受跟踪文本与 Git blob 完全匹配，无实际 `.env`、凭据、临时数据或未跟踪草稿。
- 原发布脚本已在独立源码目录运行，测试 TEMP／TMP 也位于独立构建根目录；额外的 `go test ./cmd/...` 和 `go vet ./cmd/...` 通过，未执行登录试验工具。
- 原发布脚本退出码 0，完整门禁／684 项前端测试、production 构建、NSIS 和内置版本／资产校验通过。
- 构建后核对 1693 项受跟踪文本，只有 `build/windows/info.json` 的 CRLF／LF 等价差异，没有内容差异或未跟踪数据混入。
- 最终严格保护基线检查未通过：观察到安装目录中的应用、卸载器及生产进程发生变化。进一步只读核对确认安装应用为 Beta.14，且 SHA256 与本次隔离构建完全一致；本任务没有执行安装动作，不能推断外部更新的操作者。
- 16 个旧发布资产全部保持不变，保护基线共 19 个文件中 17 个未变；没有回滚或覆盖已经更新的安装目录。

# Verification

- PASS：`scripts/set-release-version.ps1 -Version v2.0.0-beta.14`，完整版本及数字资源映射一致。
- PASS：版本相关前端定向 Prettier、发行说明读取 35 项测试。
- PASS：冻结源码内 `scripts/build-release.ps1 -Version v2.0.0-beta.14 -OutputDirectory <workspace>/release-assets/v2.0.0-beta.14`，退出码 0。
- PASS：脚本内 `task --force verify`，格式、Go vet／staticcheck、ESLint、TypeScript、全部 Go 包、前端 115 文件／684 用例和 Windows production 构建。
- PASS：额外 `go test ./cmd/...`、`go vet ./cmd/...`。
- PASS：`task --force package`，production 重建、官方 WebView2 bootstrapper、UTF-8 NSIS 安装器生成。
- PASS：独立 Windows 原生版本 API：应用 `2.0.0-beta.14`、安装器 `2.0.0`、数字资源 `2.0.0.0`、产品名／厂商 OmniProxy。
- PASS：`go version -m`：`production`、`windows/amd64`、`CGO_ENABLED=0`，无 `updateqa`。
- PASS：1693 项构建来源与冻结 Git blob 匹配，Windows 资源 JSON 仅换行等价。
- PASS：资产命名、目录仅包含安装器及 SHA256 文件，实际摘要匹配。
- PASS：16 个旧发布资产摘要及大小保持不变。
- CHANGED（不记为通过）：19 项保护基线中，已安装的 `OmniProxy.exe`、`uninstall.exe` 发生变化；生产进程由 PID 56552 变为 PID 37404，后者启动时间为 2026-10-20 20:01（新加坡时间）。
- CONFIRMED：核对时已安装应用为 Beta.14，摘要 `862428b28f23d51450d906fa77a5e48508f3b28764af9160693db1050a410e9b`，与本次构建应用完全一致。本任务没有调用安装器或停止原生产进程；不声称原安装状态未变，也不声称已验证真实账号数据保留。
- 安装包签名为 `NotSigned`，不将 SHA256 描述为数字签名。

## Artifacts

- 安装器：`E:/go/new/wails3-v/OmniProxy/release-assets/v2.0.0-beta.14/OmniProxy-Setup-v2.0.0-beta.14-windows-amd64.exe`
- 校验文件：安装器同名追加 `.sha256`。
- 大小：19,491,252 bytes。
- SHA256：`76d2f353b0d8442b3c22be4d58f39af2719de949d953cdfabaf9925031b07400`。
- 来源、构建日志、1693 项来源校验和保护基线差异保存在忽略目录 `tmp/local-beta14-20261020/`。

# Documentation

已更新 Beta.14 随包发布说明、CURRENT 及本任务记录；不改变架构或技术栈。

# Deviations

原计划同时确认旧发布资产和已安装程序／进程完全不变。实际 16 个旧发布资产未变，
但最终核对时安装目录和生产进程已经更新，且安装应用与本次 Beta.14 构建摘要一致。
保留最初严格检查失败与实际差异，不把环境变化包装为全部保护检查通过，不回滚现有安装。

# Git

`1c5c44f13668fbe270153dd4d0c0c52f1adeba68` — `chore(release): 准备 Beta.14 本地生产安装包`。

只提交本任务版本、发布说明、说明读取测试及任务记录，不推送、不创建发布标签。

# Result

Beta.14 Windows amd64 production 安装器与 SHA256 已生成并通过独立版本、来源和资产核对，
可交付本地更新／留存；未签名、未公开发布。
核对时发现当前安装目录已是同一 Beta.14 构建，但安装动作不是本任务执行；
真实使用、账号数据和覆盖安装过程仍由用户验收。

# Completed

2026-10-20（Asia/Singapore）
