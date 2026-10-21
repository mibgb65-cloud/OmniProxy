# TASK-20261021-003: 排查 Claude Code Desktop 使用 Gemini 报错

## Status

Completed

## Created

2026-10-21（Asia/Singapore）

## Request

用户在 Claude Code Desktop 上尝试使用 Gemini 模型仍然报错，要求定位原因并说明、实施适当修复。

## Goal

结合实际请求记录、客户端安全诊断及现有协议适配确定故障边界；对确认的代码缺陷做最小修复和回归验证，不把离线验证等同真实桌面验证。

## Scope

- 只读检查本机相关运行状态及经过筛选、脱敏的错误元数据。
- 检查 Claude Messages 到 Gemini 的模型路由、工具与内容适配。
- 必要实现、回归测试、质量门和文档更新。

## Out of Scope

- 输出账号密钥、用户消息正文或完整客户端会话。
- 未经确认改写真实客户端认证、自动重放用户消息、安装／替换生产程序。
- 无关模块重构及用户已有修改。

# Analysis

- 初始工作区存在无关未跟踪任务记录，保持不动。
- 生产记录：2026-10-21 11:07:46（新加坡时间），Messages → `gemini-3.8-flash-high` 返回 `400 / invalid_request`；11:07:43 的另一请求返回 200。
- Claude `cli-diagnostics.jsonl` 的 03:07:47Z `cli_api_error` 确认 400，模型别名与网关一致；未输出会话正文或完整诊断。
- 先前 TASK-20261021-001 只修正上游 400 的错误映射；不能将该项修复当作实际请求兼容性已经修好。
- 历史上游正文未保存，继续以隔离合成请求／结构检查寻找可复现原因，不凭状态码猜测唯一根因。

## Root Cause

- 使用 Desktop 自带的 Claude Code 2.1.299 内核，在独立配置目录、假 Key、本机 mock、
  `--bare`、无会话持久化模式捕获合成请求。自带工具 Schema 包含 `$schema`、
  `exclusiveMinimum`、`additionalProperties` 等 JSON Schema 关键字。
- `toolDeclarations` 原实现将完整 JSON Schema 放入 Google 的受限 `parameters` 字段。
  真实 Google 对照中，该格式 HTTP 400／INVALID_ARGUMENT，安全白名单标记确认
  `Unknown name`、`$schema`、`exclusiveMinimum`；只替换为 `parametersJsonSchema`
  后相同工具和合成提示 HTTP 200，适配器成功读取结果。
- 原现场未保存完整请求，不能证明唯一参数组合；现已复现同内核默认工具的同类确定缺陷，
  而非将旧 401／429、历史工具签名问题一并宣称修复。
- 真实验证使用既有受控凭据回调，不输出秘密或上游原始正文；诊断进程只读生产 SQLite。
  首次检查发现 token 已过期，由运行中的网关按正常请求路径续期，不由诊断进程修改凭据。
  无工具短合成请求第一次 502、第二次 200，保留该偶发边界，不作为本次 Schema 缺陷结论。

# Implementation Plan

1. 读取相关规范、既有任务和实现，筛选实际错误证据。
2. 确认根因、影响范围与最小修复；先补回归测试。
3. 运行针对性测试及适用质量门，记录结果与真实环境验证边界。

# Progress

- 已确认生产 OmniProxy 和 Claude Desktop 正在运行；尚未读取或修改真实凭据。
- 已完成隔离内核请求结构捕获及真实 Google Schema 字段 A/B 对照。
- 先补适配器与三协议 JSON／SSE 回归：修复前 Gemini 2 组适配器、6 组跨协议按预期失败；
  第三方通道原格式与输入边界保持通过。

# Implementation Result

## Changes

- Gemini 工具声明使用 `parametersJsonSchema`，保留完整原 Schema；不裁剪约束、不降级工具能力。
- Google 订阅内 Claude／GPT-OSS 继续使用原 `parameters` 字段；没有新增依赖、迁移或公开接口变化。
- 增加 Schema 字段互斥、内容保真、无 Schema／错误类型边界及三协议 JSON／SSE 回归。
- 更新 Google 订阅能力／排查说明和 CURRENT 看板。

## Files Changed

- `internal/provider/antigravity/chat_options.go`
- `internal/provider/antigravity/chat_request.go`
- `internal/provider/antigravity/tool_schema_test.go`
- `internal/application/gateway/google_tool_schema_test.go`
- `docs/product/GOOGLE_SUBSCRIPTION.md`
- `docs/plans/CURRENT.md`

## Remaining Limitations

- 未自动替换、重启或安装生产程序；已安装旧构建不会因源码更新立即生效。
- 使用 Desktop 2.1.299 内核、独立目录及修复后的本机网关进一步验证时，真实上游返回
  `429 / rate_limited`。不能将 Schema A/B 成功等同真实 Desktop GUI 或默认请求形态全部恢复。
- 隔离内核会自行重试；有日志的一轮共 8 条 429，90 秒预算后退出，未取得 `OK`。
  已停止进一步真实请求，不将 SDK 重试混同网关重放或成功验收。
- 不根据 429 猜测实时余额或绕过 Google 的资源限制；不重放真实用户消息。

# Verification

- 修复前回归：Gemini Schema 方言断言按预期失败；非 Gemini 通道及输入边界通过。
- 修复后：
  `go test ./internal/provider/antigravity ./internal/application/gateway ./internal/core/agentprotocol -count=1`
  PASS，包含真实回环网关既有工具签名续接和新增三协议／JSON／SSE 覆盖。
- 真实 Google 受控合成对照：同内核默认三工具的旧 `parameters` 格式 400，
  仅换为 `parametersJsonSchema` 后 200；没有输出原始上游错误正文或真实凭据。
- 完整质量门 `task --force verify`：PASS，退出码 0。
  格式、Go vet／staticcheck、ESLint、TypeScript、Go 测试、118 文件／703 项前端测试及
  Windows amd64 production 构建全部通过；`--force` 只关闭增量缓存，不跳过检查。
  保留既有 jsdom `scrollTo` 提示及 Vite 大 chunk 警告，未将其误记为失败。
- 本次全部交付文件 UTF-8 无 BOM 校验：PASS。
- `git diff --check`：PASS（最终提交前复核）。

# Documentation

- 已更新 Google 订阅产品说明和 CURRENT。
- 无产品范围、技术栈、依赖方向、数据库或架构决策变更，不新增 ADR。

# Deviations

- 除离线回归，增加受控真实 Schema 字段 A/B；进一步隔离 Desktop 内核验收遇到 429，
  不将该限制掩盖为完全验收通过。

# Git

`f4ae7859cb17fc24b2931dbd87b84352b0b134c5 fix(google): 修复 Gemini 工具 JSON Schema 方言适配`

仅提交本任务文件；初始无关未跟踪任务记录保持原样，未推送。

# Result

已修复并验证 Gemini 工具 JSON Schema 方言不匹配导致的可复现 400。
真实 A/B、离线回归及完整质量门通过；真实 Desktop 内核进一步验收因 429 未通过，
需更新含修复的网关构建并在资源恢复后继续验证，不能宣称已安装应用或全部调用恢复。

# Completed

2026-10-21（Asia/Singapore）
