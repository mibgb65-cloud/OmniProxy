# TASK-20261022-003: 构建 Beta.19 本地生产安装包

## Status

Completed

## Created

2026-10-22（Asia/Singapore）

## Request

用户要求继续构建新的 Beta 生产版本，用于自行本地更新。
上一轮只读核对后曾中断，尚未修改版本或启动构建。

## Goal

交付包含已提交 Fast 能力修复及关于页正式入口的 Windows amd64
`v2.0.0-beta.19` production 安装包和匹配 SHA256。

## Scope

- 核对既有发布脚本、版本映射和 Beta.18 产物。
- 同步完整版本、Windows 资源和离线更新日志。
- 在隔离产物目录执行完整门禁、生产打包和资产校验。
- 核对安装器／应用版本、来源、校验和与真实环境保护基线。

## Out of Scope

- 自动安装、重启或替换正在运行的程序。
- 修改真实账号、凭据、数据库和客户端配置。
- 创建／推送公开发布标签或发布 GitHub Release。
- 混入 Gemini 任务状态与 Atria 草稿等无关未提交改动。

# Analysis

当前源码与最新本地产物为 `2.0.0-beta.18`，新增功能已在 main。
标准发布脚本已核对：完整 SemVer 为 Beta.19，Windows 数字资源仍为
`2.0.0.0`，安装器数字字符串为 `2.0.0`，安装身份保持不变。
沿用 Beta.18 的冻结提交／`git archive` 隔离源码构建，执行标准脚本内的
`task --force verify` 和 `task --force package`，不将无关工作树改动混入来源。

本版新增内容是官方服务档位保留（Fast 入口）和关于页正式项目／协议入口；
继承 Beta.18 的内存与周额度修复，不宣称解决 Google 429 或完成真实模型调用验收。
新版本目录尚不存在，全部工具已安装；保留旧发布资产，不覆盖非空目录。

# Implementation Plan

1. 确认影响范围、保护基线和标准发布流程。
2. 升级到 Beta.19，补充真实变更日志并仅提交相关文件。
3. 基于精确源码完成完整门禁和 NSIS 生产安装器构建。
4. 验证原生版本、SHA256、源码来源及保护基线。
5. 归档记录并交付安装包路径和本地更新说明。

# Implementation Result

- 完整版本升级为 `2.0.0-beta.19`，Windows 数字资源保持 `2.0.0.0`，
  安装身份不变；新增发行说明和三个运行版本格式的离线日志回归。
- 源码冻结为 main 提交 `36144ece58fb59273ffbd86f9a94ee6d922cb80f`，
  使用 `git archive` 导出受跟踪源码到 `tmp/local-beta19-20261022/source/`，
  不混入其他任务的未提交文档、历史规划草稿、真实配置或凭据。
- 完成标准发布脚本中的强制完整门禁与 production NSIS 打包。
  新包包含 Fast 服务档位修复及关于页正式入口，继承 Beta.18 既有功能；
  没有新增依赖、数据库迁移或安装逻辑。
- 最终资产目录只含安装器与匹配 SHA256；旧发布资产、安装文件、
  Codex 配置及原生产进程均保持保护基线不变。

## Artifacts

- 安装器：
  `E:/go/new/wails3-v/OmniProxy/release-assets/v2.0.0-beta.19/OmniProxy-Setup-v2.0.0-beta.19-windows-amd64.exe`
- 校验文件：同名追加 `.sha256`。
- 大小：19561308 bytes。
- SHA256：`cc2db393a5557d03e8c961dc59bdc342949bf033c5d116513fb16604269bc47a`。
- 签名：`NotSigned`。
- 来源、构建日志、原生资源／运行时验证和保护基线位于
  `tmp/local-beta19-20261022/`，不作为公开发布资产。

# Verification

- PASS：`scripts/set-release-version.ps1 -Version v2.0.0-beta.19`，
  源码、前端及 Windows 数字／字符串版本映射一致。
- PASS：`go test ./internal/version -count=1`；发行说明 2 文件／44 项定向测试、
  新文件及 package.json 格式检查。
- PASS：Fast、关于页及原生验收提交均为冻结 main 提交的祖先；
  相对 Beta.18 没有 go.mod／go.sum 或数据库迁移变更。
- PASS：隔离源码执行
  `scripts/build-release.ps1 -Version v2.0.0-beta.19 -OutputDirectory <workspace>/release-assets/v2.0.0-beta.19`，
  退出码 0。脚本完整执行 `task --force verify` 与 `task --force package`。
- PASS：格式、Go vet／staticcheck／ESLint、类型检查、全部 Go 测试、
  120 文件／718 项前端测试、Windows production 构建及 NSIS 编译。
- PASS：隔离源码额外执行 `go test ./cmd/... -count=1`、`go vet ./cmd/...`；
  不执行真实登录或凭据修改命令。
- PASS：构建前后与冻结 Git blob 核对 1769 项源码／配置／文本；
  仅 info.json 有 CRLF／LF 等价差异，没有实际内容变化或私密／未跟踪文件混入。
- PASS：Windows 原生 API 核对应用字符串版本 `2.0.0-beta.19`、
  安装器字符串版本 `2.0.0`、双方数字资源 `2.0.0.0` 及产品／公司身份。
- PASS：`go version -m` 确认 `production`、windows/amd64、CGO_ENABLED=0，
  没有 `updateqa`；资产名称、数量、大小及 SHA256 完全匹配。
- PASS：production 二进制在独立 test profile、数据／配置／WebView／Codex
  目录与动态回环端口启动；healthz 确认实际版本 `2.0.0-beta.19`／test、
  监听进程归属及隔离数据库。仅查询健康信息，不发送推理或执行安装器，
  结束后按精确路径／启动时间关闭自己的进程。
- PASS：34 个保护文件和原生产进程 PID 52884 的路径／启动时间不变；
  没有安装、停止或替换用户程序。
- PASS：`git diff --check`。既有 jsdom scrollTo、Vite 大 chunk 和 Windows
  环境探测 tail 提示非阻断，未为打包引入无关修改。

## Remaining Limitations

未执行真实覆盖安装／卸载或完整真实模型调用；这些由用户本地更新后验收。
安装包未签名，机器级安装需管理员确认。冷解密峰值、Google 429 及真实
Codex 旧聊天兼容性不在已修复结论内。更新后需同步官方模型目录并由用户重载
Codex 模型菜单，不能将打包成功等同于真实 Fast 加速已经生效。

# Documentation

新增 Beta.19 发行说明并更新 CURRENT 和本任务记录。公开发布与自动更新服务端
状态未改，不将本地构建描述为已经公开发布。

# Git

Source Commit：`36144ece58fb59273ffbd86f9a94ee6d922cb80f`
`chore(release): 准备 Beta.19 本地生产安装包`。

仅提交本任务版本、发行说明、测试及记录；既有 Gemini 文档改动和 Atria 草稿
未混入来源或提交。未创建发布标签、推送远端或发布 GitHub Release。

# Result

已完成 Beta.19 Windows amd64 production 本地安装包及 SHA256 交付准备，
完整门禁、NSIS、源码／原生版本／运行时／哈希及保护基线验证通过。
由用户自行覆盖更新，构建过程没有自动安装或公开发布。

# Completed

2026-10-22（Asia/Singapore）
