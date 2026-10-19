# TASK-20261019-013: 构建 Codex 兼容修复的本地 Beta.9 生产安装包

## Status

Completed

## Created

2026-10-19（Asia/Singapore）

## Request

用户要求完成后在本地构建生产环境的 Beta 安装包，由用户自行安装测试。

## Goal / Scope

将已完成的 Codex Desktop 文件变更与上下文压缩修复生成 Windows amd64 本地生产 Beta 安装包，核对完整版本、Windows 元数据与 SHA256，交付安装路径。不自动安装、不发布 GitHub、不创建／推送标签，不修改真实账号、数据、客户端配置或生产实例。

## Analysis

当前最新本地资产为 Beta.8，Codex 兼容修复提交 697942b 已完成质量门与双核心离线验收。本次递增为 2.0.0-beta.9，保留 Beta.6／7／8 资产和安装身份。现有 build-release.ps1 校验源码／前端版本并映射 Windows 数字版本，强制运行 verify、package、原生元数据和 SHA256 资产校验。

为避免覆盖主工作区 bin/，从仅暂存本任务版本、随包说明及回归的确定 Git tree 导出忽略目录中的隔离源码；在该源码中执行原有发布脚本。构建完成后提交相同 tree，再单独更新结果与归档。不引入构建捷径、业务依赖、数据库迁移或 updateqa 标签。

## Implementation Plan

1. 同步 Go／前端 Beta.9 版本及 Windows 元数据，补充文件变更／压缩修复的本地发布说明和随包说明回归。
2. 保存生产／开发二进制与真实 Codex 配置只读基线，仅暂存本任务文件并导出确定 Git tree。
3. 在隔离源码执行完整生产质量门与 NSIS 打包，输出到独立空 Beta.9 资产目录；校验产品版本、SHA256 和签名状态。
4. 核对构建来源与现有生产实例，记录未执行的安装／真实账号／GUI 验收，提交版本与任务归档。

## Implementation Result

- Go 与前端完整版本、Windows 应用 ProductVersion 同步为 2.0.0-beta.9；Windows 数字文件／产品版本保持 2.0.0.0，NSIS 产品版本为 2.0.0，安装身份与范围沿用既有 machine 安装器。
- 新增随包 Beta.9 发布说明，说明原生文件变更、V2／legacy 压缩、受管目录刷新及真实验收边界；增加普通版本／v 前缀／构建元数据三种随运行版本读取的回归。
- 从确定 Git tree `f7ac5bae598024929b33a947b7464d94cec9e41b` 导出独立源码至忽略目录 `tmp/local-beta9-20261019/clean-source`，通过原有 build-release.ps1 执行完整质量门、NSIS 打包与资源校验。未跟踪任务、凭据、用户配置、数据库与日志未进入源码快照。
- 本地 Beta.9 安装器与同名 SHA256 文件输出至独立目录，Beta.6／7／8 资产保持原样。未修改构建脚本、业务依赖或数据库迁移，不含 updateqa 标签。
- 版本提交的 Git tree 与构建树完全相同；任务结果／归档与当前看板由后续文档提交保存，不改变交付二进制。

## Verification

- PASS：主工作区 Prettier 定向检查，以及 `pnpm exec vitest run src/features/about/current-release.test.ts`，20 项通过。
- PASS：隔离源码运行 `scripts/build-release.ps1 -Version v2.0.0-beta.9 -OutputDirectory <主工作区>/release-assets/v2.0.0-beta.9`，退出码 0。
- PASS：脚本内 `task --force verify`，gofmt／Prettier、Go vet／staticcheck、ESLint、TypeScript、全部 Go 测试、107 文件／616 项前端测试、绑定生成、前端生产资源及 Windows 构建通过。
- PASS：脚本内 `task --force package`，NSIS UTF-8 machine 安装器打包通过。
- PASS：原生 Windows 版本 API 两次检查应用／安装器身份及完整／数字资源；资产脚本两次确认最终目录仅安装器及同名 SHA256 两文件，格式与摘要匹配。
- PASS：独立 `Get-FileHash` 与 SHA256 文件一致；`Get-AuthenticodeSignature` 为 NotSigned。未签名是本地测试包的已披露边界，不冒充可信签名发行。
- PASS：`go version -m` 确认应用为 `-tags=production`、GOOS=windows、GOARCH=amd64、CGO_ENABLED=0，不含 updateqa。
- PASS：构建后核对 go.mod／go.sum、Windows 元数据和版本配置与冻结 tree 一致；实现提交 `d00dc593feb029334419c7518c2601054728aad7` 的 tree 为 `f7ac5bae598024929b33a947b7464d94cec9e41b`，与导出源码完全相同。
- PASS：真实 Codex 配置 SHA256、生产 PID 5336／启动时间／生产二进制 SHA256 保持构建前基线；主工作区开发二进制哈希未被本地发布脚本改变。未主动安装、启动或停止应用。
- PASS：提交前 `git diff --check`、本任务文件白名单和文档 UTF-8 无 BOM／LF 检查。
- 非阻塞既有告警：jsdom 的 Window.scrollTo 未实现提示、Vite 主包超过 500 KiB及构建性能建议；未隐藏或绕过告警。

未执行：实际安装／覆盖升级、管理员确认、真实账号调用、Desktop GUI 视觉验收及真实超长会话自动压缩。用户将自行安装测试；不把打包与离线契约验证冒充真实安装验收。

## Local Artifact

- 安装器：`release-assets/v2.0.0-beta.9/OmniProxy-Setup-v2.0.0-beta.9-windows-amd64.exe`。
- 校验文件：同目录 `OmniProxy-Setup-v2.0.0-beta.9-windows-amd64.exe.sha256`。
- 大小：19360936 bytes。
- SHA256：`ce6b5ba081878ea6ff5c49bdbe91c7d587a8f51d9ec0dcc76f9c2ed74fb0957b`。
- 签名状态：NotSigned。
- 构建日志、独立源码、资产摘要和只读基线保存在忽略目录 `tmp/local-beta9-20261019/`，均不提交。

## Git

`d00dc593feb029334419c7518c2601054728aad7` — `chore(release): 准备 Beta.9 本地生产安装包`。

包含 Codex 兼容修复 `697942bc512737413955375e6b983f07ae0b153b`。仅本地提交，未创建／推送版本标签或发布 GitHub。其他任务已有未提交内容未纳入版本提交。

## Result

Beta.9 本地 Windows amd64 生产安装包及 SHA256 已生成并核验，可交付用户自行安装测试。现有生产、账号和真实客户端配置保持原样；安装后需通过正常接入流程刷新受管目录并重新加载 Codex，不能据打包成功声称生产中原问题已经验收通过。

## Completed

2026-10-19 20:51（Asia/Singapore）
