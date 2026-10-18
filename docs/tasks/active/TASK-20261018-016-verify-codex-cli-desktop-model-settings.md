# TASK-20261018-016: 验证 Codex CLI 与 Desktop 模型设置

## Status

In Progress

## Created

2026-10-18 21:54

## Request

用户询问 Codex CLI 与 Desktop 的模型设置是否不同，要求检查能否成功配置。

## Goal

核对官方配置契约、现有受管配置实现及本机版本，在隔离环境验证 CLI 与 Desktop 后端的模型目录、默认模型与实际请求；发现可复现兼容缺陷时最小修复，不把后端协议通过等同于 Desktop UI 已验收。

## Scope

- 现有 Codex 接入／模型目录／默认模型／Desktop 激活实现与相关测试。
- 本机 CLI、Desktop 内置引擎及配置位置的只读识别。
- 临时配置、模拟上游和独立端口的可重复验证，必要的回归修复。

## Out of Scope

- 不修改当前 Agent 的 config.toml、登录、会话、用户首选项或真实模型目录。
- 不重启／退出当前 Codex Desktop，不发真实付费模型请求，不自动发布新版本。
- 不擅自改变 CLI／Desktop 共用配置的产品规则，不推测未验证的 UI 行为。

# Analysis

CLI 与 Desktop 现由同一 codex 客户端配置入口管理，不另写 Desktop 登录／会话。已只读确认本机 CLI 0.163.0，Desktop 包 26.1013.4894.0 内置／实际本地引擎为 0.163.0-alpha.10.2，不能只测 PATH 稳定版就宣称 Desktop 成功。现有安装核心目录契约测试在稳定版通过。将使用真实 Service.Apply 生成临时配置，分别验证两种引擎的 model/list、config/read、thread/start 默认模型和实际请求；已有会话及文件热更新单独观察，不启动／重启真实 Desktop UI。model/list 的 isDefault 是推荐默认，不应未经验证当作当前会话选择。

# Implementation Plan

1. 确认 CLI／Desktop 共享与差异配置层次，以及现有实现的检测规则。
2. 在独立 CODEX_HOME／临时工作目录验证实际模型列表、默认模型及请求模型。
3. 对发现的兼容问题增加回归并最小修复，按风险执行质量门。
4. 复核外部配置基线不变，记录真实验证层级与限制。

# Progress

- 原本安装核心目录契约在 CLI 稳定版通过，随后确认 Desktop 实际引擎为独立 alpha 版本。
- 使用真正 Service.Apply 创建隔离配置，CLI／Desktop 内核各测内置 openai 和已有自定义提供商，共四组；原始矩阵及显式覆盖对照矩阵全部通过。
- 默认选择 qa-second 后，目录推荐项仍为 qa-first，但 config/read、新会话和实际请求均为 qa-second；因此推荐标记不能当作当前默认。
- 将临时共享默认改为 qa-first 后，新会话及请求转为 qa-first，已有会话请求仍为 qa-second；重启隔离后端也读取 qa-first。CLI -m 和 thread/start 显式模型都可覆盖文件默认。
- 模拟 ChatGPT 身份的额外 account/read 探测收到 workspace routing discovery unauthorized 401；模拟身份不能证明真实订阅授权，不将其误报为模型设置缺陷，也不尝试绕过登录。主矩阵采用隔离假 API 凭据，仅验证模型设置及本地协议，未使用真实登录。
- 已只读核对真实共享配置：默认为 openai／gpt-6.1-sol，目录 11 项且包含当前模型，但文件的内置入口未指向 loopback。没有替用户切换，不能仅凭目录存在宣称已接入 OmniProxy；运行时覆盖仍可能优先。

# Implementation Result

未发现当前两种后端版本的模型配置格式不兼容，因此不修改业务适配逻辑。
增强既有可选原生核心契约，支持显式选择 Desktop 内核，并断言 config/read、thread/start 默认与显式覆盖；测试内还隔离用户根目录和缓存。
补充 LOCAL_AGENTS 的目录／默认／会话三层、推荐标记语义、双内核差异和验证边界。


# Verification

- PASS：CLI 稳定版与 Desktop alpha 的增强原生契约各连续三次成功。
- PASS：真实 Service.Apply 生成配置的两内核 × 两提供商隔离矩阵，以及 CLI／会话显式覆盖对照；请求均由 loopback 模拟服务记录模型字段，不记录提示词／授权值。
- PASS：热更新默认、新／旧会话差异及隔离后端重启后模型一致性。
- PASS：真实用户配置 SHA256 与排查前一致，没有重启当前 Desktop 或生产应用。
- PASS：go test ./internal/infrastructure/codex ./internal/application/agentconfig ./internal/core/modelcatalog -count=1。
- PASS：完整 task --force verify，格式、lint、类型、Go 全包、104 文件／575 项前端测试与 Windows production 构建成功；日志 tmp/codex-model-settings-20261018/verify.log。既有 Vite 包体积和 jsdom scrollTo 提示不影响通过。
- PASS：以 codex_desktop 客户端标识重新执行完整对照矩阵，两内核／两提供商仍全部成功。
- PASS：最终逐项核对三份结果矩阵及真实用户配置 SHA256，无当前配置写入或真实 Desktop 前端重启。

## Limits

没有操作真实 Desktop 界面或修改真实配置，没有使用真实订阅账号／真实模型调用。Desktop UI 的可见选项、缓存和用户手动选中状态仍不能由后端协议测试替代。单凭本地配置不能排除进程／会话级覆盖，未宣称当前运行 Agent 已切入本地网关。


# Git

待本地提交；本任务不推送或发布新版本。

# Result

配置能够被 CLI 稳定版及 Desktop alpha 后端成功读取，且按共享默认／显式会话模型发出正确的本地模拟请求。
已有会话不会因更改共享默认而切换；模型推荐标记不是当前选择。没有把后端通过冒充真实 Desktop 界面验收，也没有把本地目录存在当作当前 Agent 已接入网关。
仅增强测试和文档，不修改业务实现、不改当前用户配置、不发布新版本。