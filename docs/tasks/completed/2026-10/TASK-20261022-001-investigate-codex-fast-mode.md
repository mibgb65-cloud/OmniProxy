# TASK-20261022-001: 排查 Codex 模型 Fast 闪电按钮缺失

## Status

Completed

## Created

2026-10-22

## Request

用户反馈通过新版 OmniProxy 使用 GPT 模型时，Codex 中可点击开启 Fast
模式的闪电按钮消失；旧版 v1 存在此按钮。明确不是推理强度滑杆。

## Goal

只读对比新旧实现及本机客户端能力契约，定位按钮缺失原因并说明修复范围。

## Scope

- 当前模型目录的能力采集、存储与 Codex 导出。
- 旧版 `E:/go/new/OmniProxy` 的对应实现。
- Fast 请求参数是否被网关剥离。

## Out of Scope

- 未经确认修改客户端配置、真实账号、运行程序或发布版本。
- 不相关工作区改动；本任务暂不实施功能修复。

# Analysis

## Root Cause

v2 的统一模型目录重建丢失了原生 GPT 模型的 Fast 服务档位信息。

- v1 `E:/go/new/OmniProxy/OmniProxyBackend/internal/proxy/codex_models.go:133`
  使用通用 JSON 对象读取官方模型目录，只追加缺失模型，原有 GPT 条目及未知字段保留；
  对应测试夹具明确包含 `service_tiers: [{"id":"priority","name":"Fast"}]`。
- v2 `internal/infrastructure/codex/models.go:21` 的采集结构与
  `internal/core/modelcatalog/model.go:22` 的元数据结构均没有服务档位；
  `internal/core/modelcatalog/codex.go:16` 的输出结构也没有 `service_tiers`。
- 本机只读核对：`C:/Users/mimanchi/.codex/models_cache.json` 中
  `gpt-6.1-sol` 带有 `priority / Fast`；同一模型在
  `C:/Users/mimanchi/.codex/omniproxy-models.json` 中没有 `service_tiers`。
  当前配置确实指定后一个受管目录。
- `internal/core/proxy/agent_request.go` 保留 Responses 原始请求体；
  `internal/provider/chatgpt/client.go:26` 以 JSON 字段映射转发，仅剥离
  `max_output_tokens`、`temperature`、`top_p`，没有删除 `service_tier`。
  因此已定位的缺陷在能力目录，不是此转发路径显式禁用了 Fast。

## Fix Scope

后续修复应在确认官方能力来源和客户端契约后，贯通服务档位的采集、
有界校验、持久化及 Codex 目录输出，并覆盖 GPT 原生条目和路由别名。
不能给未确认支持的第三方模型统一伪造 Fast，也不应直接修改用户受管 JSON
或全局强制开启 `service_tier`；这会被目录刷新覆盖或改变用户的额度消耗选择。

# Implementation Plan

1. 对照新旧模型目录与本机 Codex 协议。
2. 检查网关请求字段保留行为。
3. 记录证据、验证结果和修复建议，不修改产品代码。

# Implementation Result

完成新旧实现、本机官方／受管目录和原生 Responses 转发路径的只读对照。
未修改产品代码、客户端配置、账号、运行程序或发布版本；仅新增本任务记录。

# Verification

- PASS：本机目录断言脚本确认同一 `gpt-6.1-sol` 在官方缓存有
  `service_tiers.priority`，受管条目缺失 `service_tiers`。
- PASS：当前仓库执行
  `go test ./internal/core/modelcatalog ./internal/provider/chatgpt -count=1`。
- PASS：v1 后端目录执行
  `go test -mod=readonly ./internal/proxy -run '^(TestInjectCodexModelsAppendsMissingModel|TestInjectCodexModelsSkipsExistingSlugs|TestApplyCodexModelsInjectionRewritesResponse)$' -count=1`。
- N/A：未修改产品代码，仅记录排查结论，因此不运行全量 `task verify`。
- 限制：现有测试不等同于新增 Fast 专项回归；没有改目录后启动真实 Desktop
  验收闪电按钮，也没有发起真实 Fast 请求，不能据此证明上游实际提速、
  当前账号授权或额度消耗比例。

# Documentation

排查记录已归档。未更改 PRD／架构；未实施修复，不向 CURRENT 添加开发中任务，
也不覆盖该文件已有的其他任务改动。

# Git

`4420c051f45c3ecbbcd43f0b4c1189ff771e02a7`：排查记录随
TASK-20261022-002 的修复一同归档；本排查任务本身未修改产品代码。

# Result

这是 v2 重建模型目录时遗漏 Fast 服务档位的兼容性回归，和推理强度滑杆无关。
本轮已定位原因，功能修复尚未实施。

# Completed

2026-10-22
