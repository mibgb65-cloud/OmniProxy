# TASK-20261021-009: 汇总 main 并构建 Beta.18 本地生产安装包

## Status

Completed

## Created

2026-10-21（Asia/Singapore）

## Request

用户要求将内存优化及其他已修改的功能代码合并到 main，然后构建新的 Beta 生产版本，用于本地覆盖更新。

## Goal

确认所有现有功能代码已进入 main，在不影响真实数据、运行进程和历史发布资产的前提下，交付可追溯、经过质量验证的 Windows amd64 Beta.18 生产安装包。

## Scope

- 核对本地分支、工作树和 main 上尚未打包的功能提交。
- 按既有发布规范更新版本、构建并验证新的本地生产安装包。
- 校验安装资源／版本／SHA256／来源，更新任务记录和当前看板。

## Out of Scope

- 不自动安装、停止或重启用户正在运行的应用。
- 不修改真实账号、凭据、数据库、客户端配置或旧安装包。
- 不公开发布、推送远端或修改共享历史。
- 不将未完成的调查结论或历史任务草稿误当作已完成功能。

# Analysis

初查只有本地 `main` 分支，HEAD 为 `30884c1`；内存优化 `d59ba99` 和周额度功能 `007563d` 已在 main。工作区剩余改动暂时只有 CURRENT、Gemini 调查记录和 Atria 历史任务草稿，需继续核对其他工作树与发布步骤。

已核对：仅有一个注册工作树和一个本地 main，无其他待合并代码分支或未提交源码；相对 Beta.17 的业务代码差异为内存优化、周额度估算及卡片精简，其他差异为文档／测试。Atria 未跟踪文件是历史 Planned 调研草稿，不是功能实现；保留而不混入发布来源。main 已领先本地 origin/main 跟踪点，用户未要求推送或公开发布，不触发远端操作。

沿用既有发布脚本和 NSIS 身份。已保存 24 个旧发布文件、3 个安装文件及 1 个生产进程的只读保护基线；目标版本目录尚不存在。冻结本任务提交后用 git archive 导出受跟踪源码到 `tmp/local-beta18-20261021/source/`，不复制真实配置或凭据，独立 TEMP／TMP 避免路径别名干扰。

# Implementation Plan

1. 阅读既有 Beta 安装包任务、构建／版本／升级规范，核对代码来源与影响范围。
2. 确认 main 汇总状态，记录功能清单并更新完整 Beta 版本映射。
3. 对本次来源执行完整质量门，使用隔离输出生成 NSIS 安装包和来源清单。
4. 验证原生版本、安装元数据、来源、哈希及隔离运行时版本；不安装生产应用。
5. 更新文档、审查 diff、提交与归档，交付本地更新路径和风险说明。

# Implementation Result

- 已确认全部本地功能代码位于 main，包含内存优化 `d59ba99`、周额度及卡片精简 `007563d` 和既有 Gemini Schema 修复 `f4ae785`，没有待合并的功能分支。
- 完整版本更新为 `2.0.0-beta.18`，Windows 数字资源仍为 `2.0.0.0`，安装身份不变。
- 新增 Beta.18 发行说明及关于页版本对应回归；记录内存冷读取峰值、美元估算非官方余额及 Google 429 独立边界。
- 构建来源冻结为 main 提交 `c8df0c099adc419c181f2d1dc53e24e9dd34a856`，仅用 git archive 导出受跟踪源码；所有功能提交均为该来源的祖先，未混入未跟踪草稿、真实配置或凭据。
- 在 `tmp/local-beta18-20261021/source/` 完成完整质量门及 production NSIS 打包，最终产物存入新的独立 Beta.18 资产目录。
- 完成原生版本、编译标记、SHA256、1,753 项来源、官方工作因子内存专项及隔离 test 运行时验证；24 个旧发布文件、3 个安装文件和原生产进程保持不变。

## Artifacts

- 安装器：`E:/go/new/wails3-v/OmniProxy/release-assets/v2.0.0-beta.18/OmniProxy-Setup-v2.0.0-beta.18-windows-amd64.exe`。
- 校验文件：同名追加 `.sha256`，资产目录仅含这两个文件。
- 大小：19,551,074 bytes。
- SHA256：`f72b95a7e10f05edb0251fb67178d07c9a8e8773777471e1992360bc2b631ba2`。
- 签名：`NotSigned`。
- 源码归档／来源清单、构建日志、原生验收、性能专项及保护基线记录位于忽略目录 `tmp/local-beta18-20261021/`，不作为公开发布资产。

# Verification

- `scripts/set-release-version.ps1 -Version v2.0.0-beta.18`：通过，完整版本和数字映射一致。
- `go test ./internal/version`：通过。
- 前端 package／发行说明测试格式检查：通过；发行说明与关于页 2 文件／48 项定向测试通过。
- 必需功能提交的 `git merge-base --is-ancestor <commit> main`：全部通过。
- `scripts/build-release.ps1 -Version v2.0.0-beta.18 -OutputDirectory <workspace>/release-assets/v2.0.0-beta.18`：退出码 0。
- 隔离源码 `task --force verify`：通过格式、Go vet／staticcheck／ESLint、TypeScript、Go 测试、118 文件／706 项前端测试及 Windows production 构建。
- `task --force package`：production 重建与 NSIS 打包通过，未执行安装器。
- 隔离源码 `go test ./cmd/...`／`go vet ./cmd/...`：通过，未执行登录试验命令。
- 隔离源码官方 factor 20 假数据专项：通过；cold 分配约 1 GiB，完成后进程私有提交 49.99 MiB；cached 分配 12,752 B、完成后私有提交 50.08 MiB。本次临时路径及单次测量与前次约 10.6 KiB 有差异，均未重复 GiB 分配，不作为完整应用内存保证。
- 构建前后源码／配置／文本与冻结 Git blob 核对：1,753 项通过；仅 `build/windows/info.json` 出现 CRLF／LF 等价差异，无实际内容变化，没有 `.env`、凭据目录、临时源码或 Atria 未跟踪草稿混入。
- Windows 原生 API：应用字符串版本 `2.0.0-beta.18`、安装器字符串数字版本 `2.0.0`、双方数字资源 `2.0.0.0`，产品／公司身份一致，通过。
- `go version -m`：`production`、`windows/amd64`、`CGO_ENABLED=0`，无 `updateqa`，通过。
- 资产文件名／数量／大小及 SHA256：通过。
- 隔离原生启动：使用独立数据／配置／WebView 目录和 43118／43918 端口，确认测试 DB、健康端口归属和运行时 `2.0.0-beta.18`／test；只读取 healthz，不读取真实账号或发送推理。验收进程 PID 31200 已按精确路径／启动时间关闭。
- 保护基线复核：27 个文件内容与大小均未变；原生产进程 PID 14716、路径和启动时间未变，没有安装或停止生产应用。
- `git diff --check`／UTF-8 无 BOM：通过。既有 jsdom `scrollTo`、Vite 大 chunk／插件耗时和 Windows 环境探测提示非阻断，未引入无关修复。

# Documentation

更新 Beta.18 发行说明、当前版本看板和本记录。发行说明明确冷解密峰值、美元估算非官方余额、真实安装／模型调用待验收及 Google 429 独立边界；不扩大产品功能承诺。

# Deviations

没有执行无意义的合并提交：核对时只有 main，全部功能代码已经位于该分支；在 main 正常提交版本更新并冻结来源后构建。原有调查文档与历史规划草稿保持未提交，不当作额外功能打包。无发布流程偏离。

# Git

Source Commit：`c8df0c099adc419c181f2d1dc53e24e9dd34a856` — `chore(release): 汇总 main 并准备 Beta.18 本地生产安装包`。

只提交本任务版本、发行说明、测试、记录及看板内容；不覆盖或提交既有 Gemini 调查改动和 Atria 草稿。未推送、创建发布标签或公开发布。

# Result

已交付汇总 main 全部功能代码的 Beta.18 Windows amd64 production 本地安装包及 SHA256，包含内存优化、独立周额度估算和卡片精简，继承此前功能。完整质量门、NSIS、来源／版本／哈希和隔离运行时验证通过，历史资产及生产文件／进程未变。

未自动安装或公开发布；安装包未签名。真实覆盖更新、数据保留、完整应用内存走势与模型调用由用户本地验收，冷读取的瞬时算法峰值和 Google 上游 429 不宣称完全消除。

# Completed

2026-10-21（Asia/Singapore）
