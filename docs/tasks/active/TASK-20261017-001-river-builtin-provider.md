# TASK-20261017-001: River 内置服务商接入

## Status

In Progress

## Created

2026-10-17（Asia/Singapore）

## Request

用户希望将 https://docs.river.ai/ 对应的 River 服务商内置接入 OmniProxy，包括 DeepSeek V4.1 Flash 及其他官方可用模型。

## Goal

依据官方文档确认接口、鉴权和模型，接入现有服务商、账号、模型目录、路由和网关链路。

## Scope

- 核对 River 官方推理文档及模型能力。
- 复用现有适配架构完成最小闭环，不新增大型依赖。
- 补齐必要回归测试、产品文档并运行 `task verify`。

## Out of Scope

- 训练、微调和部署自定义模型。
- 未获证实的余额、套餐和模型能力。

# Analysis

- 官方 https://docs.river.ai/guides/inference/ 确认 Base URL `https://api.river.ai/v1`、Bearer API Key、Chat／Responses／GET models。
- https://docs.river.ai/guides/inference-models/ 列出 DeepSeek V4.1 Flash、Nemotron 3.5 Super VL、GLM 5.3 Flash，统一 1048576 上下文；输出分别 384000、128K、128K。
- 原生转发比跨协议转发更能保留思考、图片、工具及流式事件；Messages 无官方端点，复用既有文本／工具转换，输出 SSE 需要完整结果缓冲。
- 扩展现有追加迁移、provider、modelcatalog、routing、gateway 及前端名单；沿用 DPAPI、账号版本与 usage 观察。不新增依赖或修改 Core 协议行为。
- 静态预设与账号发现区分；发现失败保留旧目录，未知模型不自动宣称支持；已有网关请求容量／超时保持现有边界，1M 元数据不代表完成长输入验收。

# Implementation Plan

1. 核查官方协议与现有内置服务商实现。
2. 确认范围后接入 River 及官方模型目录。
3. 验证关键路径、运行质量门、同步文档并提交本任务变更。

# Progress

- 已确认工作区有其他任务的未跟踪记录，将保持原样。
- 在原工作目录创建 `codex/river-builtin-provider`，基于现有分支提交继续，未重置或修改其他任务记录。
- 完成 River 内置迁移、预设／发现、路由、原生 Chat／Responses 和 Messages 转换。
- 定向验证发现并修正新增服务商后旧测试数量预期，以及新夹具模型名／账号时间格式；最终定向 Go 和前端路径均已纳入质量门。
- 使用独立 `BIN_DIR` 运行完整质量门，避免覆盖正在运行的开发二进制。

# Implementation Result

## Changes

- 迁移 029 追加第十一家默认启用的 River 内置服务商；沿用 API Key 单个／批量管理与 DPAPI，不新增依赖。
- 三款官方精确模型 ID、上下文／输出上限和已公布推理档位集中定义；仅标记已接入的文本输入能力。
- 无凭据读取的账号预设与有界 `GET /models` 发现区分，去重、过滤未知型号；发现失败保留旧数据，凭据版本变化使旧发现失效。
- 默认映射、精确别名、固定账号、网关模型列表及客户端目录一致；停用账号／服务商后不发布可调用模型。
- 原生 Chat／Responses JSON／SSE 保留思考和函数工具正文，校验输出、推理档位、无状态历史参数及函数工具边界；禁止重定向和本地认证头透传。
- Messages 复用既有文本／工具转换，去除转换器的 DeepSeek 专有 thinking 字段；保留工具调用 ID 与结果关联，先缓冲后输出。
- 用量观察、错误分类、取消与受保护凭据回调复用现有实现；前端新增 River 名称、同步选项、路由保存与网关检查名单。

## Files Changed

- `internal/provider/river/`、`migrations/029_builtin_river.sql`。
- `internal/application/{provider,modelcatalog,routing,gateway}/` 及测试、`main_test.go`。
- 前端服务商／路由／网关名单与中英翻译、行为测试。
- PRD、RIVER、GATEWAY、ROUTING、LOCAL_AGENTS、OVERVIEW、TECH_STACK、CURRENT 与本记录。

# Verification

- 定向 Go 验证覆盖：升级保留旧配置／凭证引用／开关且重复启动不重置；预设不读凭证、账号发现替换预设、失败保留旧目录、换 Key 失效、路由与发现一致；默认／显式／固定账号路由及停用资格；原生 Chat／Responses 各两种 JSON／SSE、首事件先于终态、Messages 文本／工具历史／工具结果关联及两种下游输出；原上游输入 12／输出 4 Token 统计、凭证字节清零。
- River 契约覆盖 URL／Bearer、函数工具与思考参数保留、输出类型和上限、GLM 不可关闭思考、未确认型号／多模态／状态续接拒绝、发现容量／去重／未知过滤、无效 Key、401／403／402／429／5xx／重定向不自动跟随、Retry-After、取消。
- `task verify BIN_DIR=tmp/river-20261017/verify-bin`：PASS（退出码 0），格式、Go vet／staticcheck／ESLint、类型检查、Go 全套、前端 88 文件／472 测试与 Windows 生产构建全部通过。绑定重新生成后无差异；无依赖变化。
- 原生窗口：独立 test 配置、数据、45930／45931 端口与质量门二进制，核对 River 卡片、模型浮层三款精确 ID、账号详情及添加账号入口；没有添加真实凭据或发起推理，没有覆盖运行中的开发程序。
- 本机证据：`tmp/river-20261017/task-verify.log`、`native-verification.json`、原生截图；均为忽略的验收文件，不提交。
- `git diff --check`：PASS。新增及修改文件使用 UTF-8 无 BOM／LF，其他任务的五份未跟踪记录保持原样。

## Remaining Risks

未提供真实 River API Key，账号权限、实际计费／推理、真实客户端长会话、思考工具续接及 1M 长输入尚待实测。首版仅文本，Messages 使用缓冲转换；余额、官方上游 WebSocket 不宣称已支持。保留现有本地容量与超时，详见 `docs/product/RIVER.md`。

# Documentation

已同步 PRD、CURRENT、架构／技术栈以及网关／路由／客户端文档，新增 `docs/product/RIVER.md`；沿用现有架构，无新 ADR。

# Deviations

官方支持图片，但现有 Core Chat 校验及跨协议转换以文本为边界。此次不扩大公共协议解析器，三款模型先接入文本／函数工具，目录不宣称图片已可用；未提供真实 Key，真实上游与长窗口仍待验证。Messages 使用现有缓冲转换，不假设 River 有原生 Messages 端点。

# Git

待提交。

# Result

待完成。
