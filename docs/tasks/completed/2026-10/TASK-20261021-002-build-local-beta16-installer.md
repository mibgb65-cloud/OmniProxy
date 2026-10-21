# TASK-20261021-002: 构建 Beta.16 本地 production 安装包

## Status

Completed

## Created

2026-10-21（Asia/Singapore）

## Request

用户要求完成修复后构建新的 Beta 生产版本，由用户自行进行本地更新。

## Goal

交付包含 Google 订阅错误分类修复的 Windows amd64 production Beta.16 安装包，验证质量门、来源、原生版本及 SHA256。

## Scope

- 完整版本递增至 `2.0.0-beta.16`，保持 Windows 数字版本及安装身份兼容。
- 新增准确的发行说明，冻结源码后隔离构建、打包并校验新资产。
- 记录产物、验证结果及已知限制，更新当前版本看板。

## Out of Scope

- 不自动安装、停止或替换现有生产应用，不修改真实账号、凭据、聊天或客户端配置。
- 不创建／推送发布标签，不公开发布，不覆盖旧发布资产。
- 不提交既有 Atria 草稿或其他无关变更，不执行真实 Google 推理。

# Analysis

- 当前版本和最新本地包为 Beta.15，目标 Beta.16 尚不存在。
- Google 修复提交为 `f214c79`，只纠正普通端点 HTTP 400 被误报成 401，保留真实认证和限流语义；不宣称真实 Google 授权或推理已恢复。
- 沿用 `scripts/build-release.ps1`：版本核对 → `task --force verify` → production 构建／NSIS → 原生元数据与资产校验。
- 源码先提交冻结，仅通过 `git archive` 导出受跟踪文件到独立目录；不复制真实配置、凭据或无关未跟踪文件。
- 构建前保存旧发布资产、当前安装文件及进程的只读基线，完成后复核；生产安装验收留给用户。

# Implementation Plan

1. 保存保护基线，更新版本、发行说明并提交明确的构建源。
2. 从冻结源码隔离构建，执行既有完整发布脚本和必要补充验证。
3. 核对新资产来源／摘要／原生版本及 production 编译参数，复核保护基线，归档并交付。

# Implementation Result

- 版本与 Windows 完整资源已更新至 `2.0.0-beta.16`，数字版本保持 `2.0.0.0`，安装身份未变。
- 已补充发行说明，准确区分 Google 错误分类修复与真实认证／限流恢复，保留数据和手动更新说明。
- 构建保护基线已保存至忽略目录 `tmp/local-beta16-20261021/`：20 个旧发布资产、3 个安装文件及 1 个运行中的生产进程。
- 源码冻结为 `eeb7188cc2f2c23d3979eb67aba85486e89dc9fd`，已核对 Google 修复 `f214c79` 为其祖先。
- 通过 `git archive` 仅导出受跟踪源码至 `tmp/local-beta16-20261021/source/`，独立 TEMP／TMP 避免 Windows 8.3 路径别名，已完成完整质量门、production 构建及 NSIS 打包。
- 不复制真实配置或凭据；旧 Atria 未跟踪草稿不进入构建。
- 新资产写入独立 `release-assets/v2.0.0-beta.16/`，未覆盖旧发布资产。

## Artifacts

- 安装器：`E:/go/new/wails3-v/OmniProxy/release-assets/v2.0.0-beta.16/OmniProxy-Setup-v2.0.0-beta.16-windows-amd64.exe`
- 校验文件：同名追加 `.sha256`
- 大小：19,536,016 bytes
- SHA256：`0e91ae41849e072bd79b92c71805349c2a3d6fa6ba78cd582ae30aff949fca0d`
- 数字签名：`NotSigned`
- 构建日志、源码归档摘要、来源核对、原生启动与保护基线记录位于忽略目录 `tmp/local-beta16-20261021/`，不作为公开发布资产。

# Verification

- `scripts/set-release-version.ps1 -Version v2.0.0-beta.16`：PASS。
- `go test ./internal/version`：PASS。
- 前端 `package.json` Prettier 检查及发行说明 38 项测试：PASS。
- `git diff --check`：PASS；Windows 版本资源脚本的 CRLF／LF 提示不涉及内容差异。
- 隔离源码内 `go test ./cmd/...`、`go vet ./cmd/...`：PASS，未执行登录试验命令。
- 冻结源码的完整版本／数字版本及 UTF-8 无 BOM 核对：PASS。
- PASS：冻结源码内 `scripts/build-release.ps1 -Version v2.0.0-beta.16 -OutputDirectory <workspace>/release-assets/v2.0.0-beta.16` 正常完成。
- PASS：`task --force verify` 的格式、Go vet／staticcheck、ESLint、TypeScript、全部 Go 测试、118 个文件／703 项前端测试及 Windows 构建。
- PASS：`task --force package` 的 production 重建与 NSIS 打包；未执行安装器。
- PASS：1,729 项源码文本／配置与冻结 Git blob 核对，仅 `build/windows/info.json` 存在 CRLF／LF 等价差异；无内容差异，无 `.env`、真实凭据、临时目录或未跟踪 Atria 草稿。
- PASS：资产目录仅含新安装器及同名 SHA256，名称、摘要和文件大小正确。
- PASS：Windows 原生版本 API 校验应用完整版本 Beta.16、安装器数字版本 `2.0.0`、双方数字资源 `2.0.0.0` 和产品身份。
- PASS：`go version -m` 确认 `production`、`windows/amd64`、`CGO_ENABLED=0`，不含 `updateqa`。
- PASS：最终 production 二进制使用独立 test profile、数据／配置／WebView 目录及 43116／43916 端口启动，运行时 `/healthz` 返回 Beta.16／test；验收后仅关闭路径匹配的自有进程。未验收真实覆盖安装或原聊天续接。
- PASS：20 个旧发布资产及 3 个安装文件的大小／SHA256 均未变化；原生产 PID 34940 的路径及启动时间未变。
- 非失败提示：既有 jsdom `scrollTo` 未实现及产物分块／插件耗时提示，不引入无关修改。

# Git

## Source Commit

`eeb7188cc2f2c23d3979eb67aba85486e89dc9fd chore(release): 准备 Beta.16 本地生产安装包`

# Result

已交付包含 Google 订阅错误分类修复的 Beta.16 Windows amd64 production 安装包及 SHA256。完整质量门、打包、来源／原生版本／摘要与隔离启动验收通过，旧资产和真实安装／进程未变。

未安装、公开发布、推送 Git 或创建发布标签；未修改真实账号／客户端配置，未发送真实推理。安装包未签名，真实覆盖安装、数据保留和 Google 认证／限流恢复仍由用户更新后验收。

# Completed

2026-10-21（Asia/Singapore）
