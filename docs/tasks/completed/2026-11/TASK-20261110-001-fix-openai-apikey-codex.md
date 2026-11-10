# TASK-20261110-001: 排查并修复 OpenAI API Key 的 Codex 调用

## Status

Completed

## Created

2026-11-10（Asia/Singapore）

## Request

用户反馈昨天测试 OpenAI API Key 时无法在 Codex 中正常调用，要求检查接入是否成功并修复问题。

## Goal

定位 API Key 账号经 OmniProxy 接入 Codex 的实际故障，最小化修复并补充回归验证，明确本地验证与真实上游验收的边界。

## Background

前置任务 TASK-20261109-002 已实现 API Key 账号转发，但真实 Key 验收留给用户测试。

## Scope

- 检查现有 Codex 接入、API Key 转发、模型路由及必要的脱敏运行证据。
- 为已确认的根因增加回归测试和最小修复。
- 更新任务记录及受影响文档，执行适当验证。

## Out of Scope

- 不打印或提交密钥，不擅自改写用户 Codex 配置或生产数据。
- 不启动应用、不打生产安装包。
- 未经确认不发送会消耗官方额度的真实推理请求。

## Existing Behavior

代码已有 OpenAI API Key 支持；用户实测 Codex 调用失败，具体原因待排查。

## Expected Behavior

API Key 账号使用正确的官方 API 协议和认证，Codex 请求能正确转发，失败时保留可诊断且不泄露凭据的错误。

# Analysis

## Related Modules

- `internal/application/gateway`
- `internal/application/agentconfig`
- `internal/core/proxy`
- OpenAI API Key 适配器与模型目录

## Dependencies

复用既有实现，不新增依赖。

## Risks

不能影响 ChatGPT 订阅通道、跨账号计费边界或凭据保密；真实调用可能产生费用。

# Implementation Plan

1. 阅读相关实现与前置任务，收集最小脱敏证据，确认故障条件。
2. 先增加失败场景的回归测试，再修复根因。
3. 运行定向测试和质量门；如质量门包含生产打包，遵循用户开发工作流限制并说明替代验证。
4. 记录实际结果、限制与后续验收方式。

# Progress

- 已确认工作区有一份无关的未跟踪历史任务文件；保持原样，不纳入本任务。
- 只读检查生产 SQLite：API Key 账号已保存且启用，但模型快照为 0；昨天没有由该账号处理的请求记录，现有 Codex 请求实际仍由订阅账号处理。
- 已发现两个接入缺口：API Key 透明转发直接使用内置提供商的空基址；OpenAI 后台模型同步仅选择订阅账号，漏掉 API Key。
- 接下来仅执行官方 `GET /v1/models` 检查认证和目录，不发送计费推理，不改生产数据库。
- 官方只读检查成功：HTTP 200，96 个模型、无非法模型 ID，包含 Codex 当前模型。证明 Key 可认证及读取目录，不代表模型推理／计费已验收。
- 两个回归测试在修复前均准确失败：空基址；API Key 同步请求数为 0。修复已完成。
- 新建／批量新增 API Key 账号没有触发后台同步，已补通知及桌面绑定回归测试。
- 首次全量质量门发现新增桌面夹具受同包其他应用初始化测试留下的进程级代理 Resolver 影响（其数据库已关闭）；已在本夹具内保存／清空／恢复全局代理状态，不修改生产行为。完整桌面绑定测试重跑通过。

# Implementation Result

## Changes

- 透明转发的 API Key 账号在提供商基址为空时使用 `https://api.openai.com/v1`，保留自定义基址与订阅通道原行为；覆盖 HTTP、压缩路径与 WebSocket 地址。
- OpenAI 后台目录同步纳入启用的 API Key 账号，不再要求其拥有订阅认证状态；继续按账号凭据版本去重，换 Key 后重新读取。
- 单个／批量新增账号后通知后台目录同步；不启动网关，不写入客户端上游 Key。
- 新增默认基址、混合订阅／API Key 同步、换 Key 重同步、单个／批量创建绑定回归测试。

## Technical Notes

不改变订阅与按量账号的计费池隔离，也不自动把已存在的订阅模型路由切到按量计费。需明确测试 API 额度时，由用户在路由页固定 API Key 账号。无新增依赖、迁移或架构边界变化。

# Verification

## Tests

- 两个根因回归在修复前均失败，在修复后通过。
- `go test ./internal/application/gateway ./internal/application/modelcatalog ./internal/transport/wails -run 'TestPassthroughAPIKey|TestOpenAIAutoSync|TestCreateOpenAIAPIKeyAccountsRequestsModelSync' -count=1`：通过。
- `go test ./internal/transport/wails -count=1`：通过（17.459 秒），覆盖完整同包顺序及新增绑定测试。
- 质量门重跑的 `go test . ./internal/...`：全部通过，完整桌面绑定测试 19.445 秒。
- 官方 `GET https://api.openai.com/v1/models`：HTTP 200，96 个模型，当前 Codex 模型存在；凭据仅在受保护读取回调内使用，未输出任何 Key 或原始响应。
- `task verify DEV=true BIN_DIR=tmp/openai-apikey-verify`：第二轮格式、Go vet、staticcheck、ESLint（0 error／2 条原有 Fast Refresh warning）、类型检查与全量 Go 测试通过；前端默认并行在 `minimax.test.tsx`、`mimo.test.tsx`、`accounts-view.test.tsx` 出现失败（含约 5 秒超时）及持续 jsdom `scrollTo` 未实现输出，未完成全套质量门。
- 另以 `pnpm exec vitest run --maxWorkers=2 --testTimeout=15000` 尝试前端全量复验，持续无单文件完成结果；已只终止本次 Vitest 进程树，未删除测试或改动不相关前端代码。改为记录限制并执行 OpenAI 定向前端测试及独立开发编译。
- `pnpm exec vitest run src/features/accounts/openai-account-settings.test.tsx src/features/accounts/openai-account-rotation.test.tsx --maxWorkers=1 --testTimeout=15000`：2 文件／10 项全部通过（14.06 秒）。
- `task build DEV=true BIN_DIR=tmp/openai-apikey-verify`：未完成；`go mod tidy` 后长期没有进入后续命令，Task 持续占用 CPU，已停止本次构建进程树，不修改无关构建配置。
- 按 Windows Taskfile 的开发参数执行 `go build -buildvcs=false -gcflags=all="-l" -o tmp/openai-apikey-verify/omniproxy.exe .`：exit 0，开发模式后端编译通过，复用未修改的既有前端嵌入资源；未启动、不打生产包。
- `git diff --check`：通过。没有新增依赖或生成文件差异。

## Verification Result

关键路径与后端全量验证通过，OpenAI 前端定向 10 项通过，开发模式 Go 编译通过。**完整 `task verify` 未通过／未完成**：前端默认并行失败、低并发全量未完成；独立 Task 构建也未完成。未通过删除测试、放宽断言或改动无关前端／构建配置来隐藏此限制。

## Remaining Boundaries

- 未发送会计费的真实 Responses 推理，不能把模型目录认证成功当作完整 Codex 端到端成功。
- 未更改生产账号、模型快照、路由或 Codex 配置；运行中的已安装版本不会因源码变更自动获得修复。
- 旧记录中的订阅流中断不属于本次 API Key 故障，未做无证据归因或无关修改。

# Documentation

- 更新 `docs/product/GATEWAY.md`：默认官方 API 基址、API Key 自动目录同步、明确固定按量账号的验收方式。
- 更新 `docs/plans/CURRENT.md`：任务概览。

# Git

代码提交：`a175b8ca4693bc60f99791cd651b4f3c16120d3c`（`fix(gateway): 修复 OpenAI API Key 的 Codex 接入`）。

本次仅提交修复代码、回归测试及任务／产品文档；无关未跟踪任务文件保持原样。未推送远端。

# Result

已证实现有 Key 可向官方认证并读取模型，已修复阻断接入的三个代码缺口并完成关键路径验证。完整质量门的限制已如实记录；未替换已安装版本或改写生产数据。真实 Codex 计费推理仍需用户确认后，在修复版本中固定 API Key 账号进行验收。

## Delivery Follow-up

2026-11-10 用户明确要求无需再测试，提交并构建新的 Beta 生产版本供本地更新。后续交付由 [TASK-20261110-002](TASK-20261110-002-build-local-v2.2.3-beta.5.md) 记录，目标为 `v2.2.3-beta.5`；不将用户自行验收或本地打包视为已通过真实 Codex 计费调用。
