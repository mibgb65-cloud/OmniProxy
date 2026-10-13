# TASK-20261013-014: DeepSeek Claude Code 上下文变体与请求兼容

## Status

Completed

## Created

2026-10-13（Asia/Singapore）

## Request

用户要求补齐 Claude Code 的 DeepSeek 默认上下文与 `[1m]` 两种配置，例如 `deepseek-flash` / `deepseek-flash[1m]`，并排查截图中 Claude 桌面 Code 模式使用 `deepseek-v4-pro` 时出现的 `Invalid request`。

## Goal

保留原模型与路由语义，提供可选择的长上下文变体，定位并修复能由本地证据确认的真实请求兼容问题。

## Scope

- Claude CLI / Desktop 模型目录、受管配置和 Messages 模型解析。
- DeepSeek Messages 请求转换、外部输入校验及必要回归测试。
- 只读核对脱敏日志与客户端诊断，执行质量门并同步文档。

## Out of Scope

改变核心技术栈、清理会话、暴露密钥或提示词、无关重构、推送远端。

# Analysis

## Existing / Expected Behavior

既有 DeepSeek 同步只保存官方模型 ID，未保存已确认的 1M 元数据；客户端目录不生成长上下文变体，兼容 ID 将上下文标记一起散列，Messages 校验与解析也不支持该后缀。预期普通与长上下文模型均映射到同一真实上游模型，保留原默认与用户路由。

## Root Cause

1. 正式 Claude Code 2.1.295 请求为 `/v1/messages?beta=true`；网关只允许目录查询参数，读取正文与开始追踪前直接返回 `400 invalid_request`。原生健康探针不使用这一查询参数，因此成功探针不能证明正式对话可用。
2. 放行查询参数后的实机请求暴露第二个缺口：`messages` 中有附加的 `system` 文本消息，转换器原先只接受 user / assistant，返回 `unsupported_content`。
3. 原有 2 MiB 请求上限不足以容纳常见百万上下文的 JSON 文本，需要为 Messages 单独设置有界容量。

## Related Modules

`internal/core/modelcatalog`、`internal/core/agentprotocol`、`internal/core/proxy`、`internal/application/agentconfig`、`internal/application/gateway`、`internal/application/routing`。

## Dependencies / Risks

不新增依赖。保留已有兼容模型 ID、精确路由和原默认；未知能力不推断为 1M。真实截图根因未确认前，不将猜测写成结论。客户端配置存在工具与交互用户视图差异，实机验收单独记录。

# Implementation Plan

1. 核对官方 Claude Code / DeepSeek 协议说明、既有实现及脱敏本地失败记录。
2. 确认原因后补齐变体生成、配置选择和安全模型解析；修复有证据的请求转换缺陷。
3. 增加目录、配置、网关与协议回归测试，执行 `task verify`。
4. 更新产品说明、任务结果和 CURRENT，审查 diff 并提交当前任务文件，记录提交号后归档。

# Progress

- 已核对截图和此前 Desktop 接入任务。官方文档确认 `[1m]` 是客户端上下文标记，Claude Code 可在发送前去除。
- 本机 Claude Code 2.1.295 隔离抓取确认实际路径为 `/v1/messages?beta=true`，正文 max_tokens=32000；现有 validModelQuery 在鉴权／请求追踪前拒绝该参数。客户端日志的三次 `400 invalid_request` 与网关仅有成功探针记录吻合。根因已复现，无需记录提示词或凭据。
- DeepSeek 同步目前只存模型 ID，未记录上下文；为官方已确认的 Flash / Pro 补充 1M 元数据（已有快照在读取时兼容补齐），未知模型不推断能力。仅 Messages 接受精确 `[1m]` 后缀，不放宽普通路由模型语法。
- 真实 CLI 验证补充 system 提醒兼容后通过 Flash / Pro 四种上下文组合；Pro 还通过真实默认工具目录验证。
- 通过当前项目原生窗口重新读取、预览并应用 Desktop 配置，目录 8→10 项，普通默认保持不变；启动开发网关并验证实际 3001 端口的认证 Pro[1m] SSE 请求返回 200。
- Messages 容量提升为 16 MiB，其他协议保留 2 MiB；超过旧限制的文本完整转发，新上限与异常参数边界回归通过。

# Implementation Result

## Changes

- Messages 允许单个 `beta=true`，保留未知、重复、异常查询与接入 Key 校验。
- 保留 Claude Code 附加系统提醒的角色、位置与文本；系统工具／不支持的内容仍显式失败。
- 补齐 Flash / Pro 官方上下文元数据、旧快照读取兼容、普通与 `[1m]` 客户端目录，以及稳定兼容 ID 的可见后缀。
- 变体先解析原模型／用户路由，再沿用既有账号选择；精确目录 ID 优先，未知能力、已移除模型及不可用账号不绕过校验。
- 原默认不变；目录刷新保留两种上下文，应用与恢复继续使用既有加密备份和版本保护。
- Messages 使用 16 MiB 正文上限；在读取前鉴权并占用有界并发名额，JSON 重复字段、嵌套深度、UTF-8 与其余协议容量约束保留。

## Files Changed

- `internal/core/modelcatalog/claude.go`、`claude_desktop.go`、`model.go` 及测试。
- `internal/core/proxy/` 查询、模型、正文边界与回归测试；`internal/core/agentprotocol/request.go` 及系统提醒测试。
- `internal/provider/deepseek/metadata.go`、`internal/application/modelcatalog/service.go`、`internal/application/routing/model_entries.go` 及路由元数据测试。
- `internal/application/agentconfig/client_entries.go` 及应用／刷新／恢复测试。
- `internal/application/gateway/` 模型解析、跨协议测试与显式启用的真实 CLI 集成测试。

## Technical Notes

`[1m]` 是客户端上下文标记，不是新的 DeepSeek 上游模型。客户端会在发送前去掉该标记，因此 Desktop 两种选项共享基础摘要 ID，长上下文变体仅追加可见后缀。网关直接收到带后缀请求时也能去除并验证目标能力。既有用户精确路由、提供商与账号优先级保持有效。

# Verification

## Tests

- PASS：Core / 网关 / 路由 / 配置服务针对性单测与 HTTP 集成回归。
- PASS：本机 Claude Code 2.1.295、隔离临时配置与真实 DeepSeek，Flash / Pro 普通与 1M 共四组合完成对话；客户端报告普通 200000、长上下文 1000000。Pro 普通模式另带真实默认工具目录完成回复。
- PASS：真实 DeepSeek 原生 Chat Completions JSON / SSE，返回 200。
- PASS：当前仓库原生开发窗口预览、应用，Desktop 目录为 10 项，含四个 DeepSeek 标签，后缀仅在两个长上下文兼容 ID 上。
- PASS：当前开发网关 3001 使用实际受管 Desktop 接入 Key，Pro[1m] + beta=true + system 提醒返回 HTTP 200、回复事件与 message_stop；不输出授权或回复原文。
- PASS：超过 2 MiB 文本完整跨协议转发；16 MiB 上限接收、超限拒绝，Responses / Chat 容量不放宽。
- PASS：未知／重复／畸形 beta 参数、缺失／错误 Key、错误上下文后缀、未知能力、移除条目、停用账号，以及配置应用／刷新／恢复边界。

## Quality Checks

PASS：最终 `VITEST_MAX_WORKERS=4 task verify BIN_DIR=E:/go/new/wails3-v/OmniProxy/tmp/claude-deepseek/quality-bin` 退出码 0。格式、Go vet / staticcheck、ESLint、TypeScript、全部 Go 测试、69 个前端测试文件 / 366 项测试和 Windows 生产构建通过。正文容量改动后重新运行最终门，不沿用旧结果代替最终验证。

`git diff --check` 与本任务 26 个变更文件 UTF-8 无 BOM / LF 检查通过；手写 Go 文件均未超过 200 行。

## Verification Result

完整质量门与上述真实调用、原生配置和合成回归均通过。质量日志与临时原生操作脚本仅保留在忽略目录 `tmp/claude-deepseek/`；既有 jsdom scrollTo 提示不影响质量门通过。

# Documentation

已同步 `docs/product/LOCAL_AGENTS.md` 的上下文选项、正文容量、查询与系统消息兼容说明，以及 CURRENT。无技术栈／架构边界改变，无新依赖或迁移。

# Deviations

实机请求发现了第二个系统提醒兼容缺口，并在评估实际 1M 使用后补齐正文容量；均直接属于原需求。隔离实机测试辅助函数初版提前关闭数据库，产生仅测试中的 503，已修正清理生命周期；该测试缺陷不是产品根因。

# Git

Branch: `codex/deepseek-claude-context`。最终质量门已通过，提交本任务文件后补记实现提交号；不推送远端。

# Result

已修复两个真实 Claude Code 请求兼容缺口，普通／1M 选项与实际调用通过；当前 Desktop 受管配置已更新，开发网关已启动。最终质量门通过，任务完成。

## Remaining Risks

未发送真实百万 token 长提示进行压测，避免额外 API 消耗；已验证客户端实际上下文数值及合成大正文边界。并发大正文可能增加内存占用，容量和并发均有上限。Claude Desktop 模型菜单在完全退出并重启后读取更新目录，本次保留当前正在打开的客户端和用户会话。

## Completed

2026-10-13 20:48（Asia/Singapore）
