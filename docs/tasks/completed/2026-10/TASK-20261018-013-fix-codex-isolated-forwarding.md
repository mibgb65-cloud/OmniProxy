# TASK-20261018-013: 隔离修复 Codex 转发与错误诊断

## Status

In Progress

## Created

2026-10-18 20:12

## Request

用户要求研究并修复 Codex 接入转发失败，使用沙盒／隔离环境，避免修改外部配置影响当前运行 Agent。

## Goal

在不接触当前 Agent 配置、真实凭据和生产进程的前提下，复现并修复可证实的协议问题及错误信息失真，补充安全诊断及回归验证。

## Background

接续 TASK-20261018-012：历史上游 400、400、429，WebSocket 统一显示 502，随后本地冷却。上游 400 的具体原因尚未确认。

## Scope

- 仓库内转发协议、错误映射与相应测试。
- 临时数据库／配置／模拟上游及独立本地端口，不改生产运行状态。
- 必要的安全错误诊断与项目文档。

## Out of Scope

- 不修改用户 .codex 配置、凭据、登录状态或真实账号。
- 不启动／停止／替换生产应用，不监听生产 3000／3890 端口。
- 不发真实付费模型请求，不发布新版本。

# Analysis

方案选择：HTTP 与 WebSocket 共用非 200 状态分类，保留传输特定错误帧，不透传任意上游错误正文。已用本机 Codex CLI 0.163.0、独立 CODEX_HOME／工作目录、假密钥、loopback 模拟服务器捕获到首帧 generate=false 预热，随后真正创建响应引用预热 ID。现有网关把预热作为真实模型请求转发，且保留仅属于 WebSocket 的 generate 字段；应本地完成无生成预热，保留会话上下文，不消耗上游尝试。历史 400 缺乏正文，仍不宣称其具体原因已证实。官方文档搜索工具当前网络失败，先依据本机实际客户端与现有代码建立回归，不凭空扩展参数规则。

# Implementation Plan

1. 核对现有协议实现、测试和官方必要契约；记录隔离边界。
2. 先加失败回归，再最小修复已证实问题及安全诊断。
3. 运行定向测试和 task verify，确认外部配置与生产进程未受影响。
4. 审查 diff，补充实际结果和剩余限制。

# Progress

- 隔离捕获实际 Codex CLI 0.163.0 的 WebSocket 预热与 HTTP 请求字段；确认仅 WebSocket 预热带 generate=false。
- 新增回归先运行失败，复现预热被转发、错误统一降级和 generate 非布尔值未校验。
- 已实现本地预热、共享状态分类和真实错误帧，定向 Core／Gateway 测试及隔离实机 CLI 契约通过；待完整质量门与最终审查。

# Implementation Result

## Changes

- 真实本机 Codex CLI 0.163.0 的首帧 generate=false 预热改为网关本地响应，生成连接内随机 ID 并保留输入；不调用上游、不产生模型请求或用量记录。正式请求删除 generate／previous_response_id 等传输字段，合并连接上下文后才转发。
- generate 必须为布尔值；previous_response_id 类型、未知引用、模型与路由继续校验，鉴权及 WebSocket 开关仍在每帧检查。
- HTTP 与 WebSocket 共用非 200 上游失败分类；WebSocket 最终错误帧保留稳定错误码、状态和经过校验的 Retry-After，不再全部变为 502 或 upstream_stream_failed。请求与尝试级 400／429／冷却状态保持一致。
- 独立提取 WebSocket 响应消费，校验 SSE 类型／空正文，收到 response.completed 即结束，不等 EOF；不完整流仍失败，失败后的请求不会被重放。嵌套 response.error 也不能把原始错误信息透传给客户端。
- 没有读取或记录上游原始错误正文来猜测具体原因；安全诊断保留状态及固定分类，不包含真实提示词、邮箱、账号名称或凭据。
- 新增可选离线 CLI 契约：独立 CODEX_HOME、临时工作目录、只读命令沙盒、假 Key、白名单继承环境及随机 loopback 端口；模型完全模拟，子进程不继承当前 Agent 的配置或认证环境。

## Files Changed

- internal/core/proxy/websocket.go、websocket_response.go、websocket_warmup.go：传输与预热处理。
- internal/core/proxy/response.go、upstream_failure.go：共享失败分类与嵌套错误保护。
- internal/core/proxy/websocket_regression_test.go、websocket_stream_regression_test.go、upstream_failure_test.go、codex_cli_test.go：协议边界及实际本机客户端回归。
- internal/application/gateway/codex_socket_error_test.go：真实应用服务／临时数据库的尝试、请求和冷却记录验证。
- docs/product/LOCAL_AGENTS.md、docs/plans/CURRENT.md 与本 Task Record。

## Remaining Risks

历史两次 400 的具体上游原因没有安全错误正文证据，不能宣称完全还原历史根因。已证实并修复的兼容问题是预热被错误转成真实 HTTP 模型请求。本任务不调用真实账号，也不替换生产 Beta.2、切换用户 Codex 接入或重启生产进程；真实上游验收与部署仍待后续单独安排。可选 CLI 契约已验证本机 0.163.0，未对其他版本作泛化保证。


# Verification

- PASS：先添加回归运行失败，复现预热误转发、非布尔 generate 被接受及错误状态丢失，再实施修复。
- PASS：go test ./internal/core/proxy ./internal/application/gateway -count=1，多轮定向测试。
- PASS：显式设置 OMNIPROXY_CODEX_TEST_EXECUTABLE 后运行实际 CLI 契约；最终 -count=3 连续成功，每次只发出一次模拟上游正式请求，且仅有一条成功 WebSocket 请求记录。
- PASS：400／401／402／403／404／429／503／302 状态矩阵、非法 Retry-After、预热上下文、generate 类型、未知 previous ID／模型、空正文／错误类型／不完整流／嵌套错误及完成后不等待 EOF。
- PASS：应用层临时 SQLite 验证 400 不冷却、429 后不重复调用上游，最终请求与尝试状态一致。
- PASS：已检查外部 .codex/config.toml SHA256 与排查前完全一致；生产 PID 8124 及启动时间不变。最终交付前再次核对。
- PASS：新增／修改 Go 文件均 UTF-8 无 BOM、少于 200 行，git diff --check 通过；既有未跟踪任务记录未修改。
- PASS：两轮 task --force verify（末轮在代码冻结后重跑）；格式、go vet／staticcheck／ESLint、类型检查、全包 Go 测试、104 文件／575 项前端测试及 Windows production 构建均成功。最终日志在忽略目录 tmp/codex-forwarding-20261018/verify.log。
- 非阻塞既有告警：Vite 主包超过 500 KiB；jsdom 的 Window.scrollTo 未实现提示，前端测试全部通过。未为本修复修改前端或隐藏告警。
- PASS：最终再次核对外部 Codex 配置哈希一致、生产 PID 8124／20:03:52 启动时间未变；3890 健康接口仍为 production / 2.0.0-beta.2 / ok。

# Documentation

已补充 LOCAL_AGENTS 的 Codex 预热、错误状态与可选离线实机客户端契约，并更新 CURRENT 看板。此为现有协议兼容和安全错误处理修复，不引入技术栈、依赖、数据库迁移或新的架构决策。


# Git

实现提交：`ad79d6ef69497a3b54b1fe3a995c24394a8b8d9b` — fix(gateway): 修复 Codex 预热转发与错误状态丢失。

仅本地保存，不推送远端。

# Result

已完成可重复验证的隔离修复及最终完整质量门，当前 Agent 配置与生产进程保持不变。修复仅保存在仓库并生成本地候选构建，未部署生产；历史上游 400 的唯一根因和真实账号调用仍不能由模拟测试替代确认。

# Completed

2026-10-18 20:32
