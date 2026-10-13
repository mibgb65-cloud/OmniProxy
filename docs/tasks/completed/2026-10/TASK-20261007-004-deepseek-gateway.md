# TASK-20261007-004: DeepSeek 网关最小调用闭环

## Status

Completed

## Created

2026-10-07 11:03

## Request

实现网关后端最小闭环，先尝试 DeepSeek；保留此前不影响其他任务的要求。

## Goal

本机网关能启动和停止，使用已保存的 DeepSeek 账号安全转发普通／流式 Chat Completions；网关页面展示真实状态并支持短请求测试。

## Analysis

- 当前代理核心和提供商请求适配器尚未实现。账号编辑已在最新提交完成，网关准备页的工作区改动需要保留。
- DeepSeek 官方当前接口：`POST /chat/completions`，模型 ID `deepseek-flash`／`deepseek-v4-pro`；流式使用 SSE。首版提供本地 `/v1/chat/completions` 和 `/v1/models`。
- Core 使用 Go 标准库，不能依赖 Wails、React 或数据库。Application 复用已有提供商／账号服务，凭证只在 UseAPIKey 回调内使用。
- 首版选择一个已启用 DeepSeek 账号，不实现多账号重试、健康调度或路由规则。
- 在独立 worktree 开发和验证；共享文件和生成绑定回填时按当前版本核对，不覆盖其他任务改动。

## Implementation Plan

1. 基于最新账号提交建立独立分支，整合已有网关页面，任务切换为 In Progress。
2. 实现代理生命周期、输入边界、本机请求保护、取消／超时和并发限制。
3. 实现 DeepSeek 适配器与账号选择，保护鉴权头、不跟随重定向、不返回含敏感信息的上游错误正文。
4. 通过 Wails 绑定和事件接入真实状态、启停和固定短请求测试，复用中英字典。
5. 覆盖正常、流式、失败、端口占用、取消、启停竞争和敏感信息场景；条件允许时使用已有账号验证短真实请求。
6. 运行 task verify，更新能力说明、架构与任务记录，审查并在独立分支提交，然后最小回填。

## Risks

凭证与请求内容不得写入日志、返回错误或持久化。仅监听 loopback；拒绝浏览器跨源请求和不匹配的 Host，客户端鉴权头不向上游透传。响应、输入、请求时间和并发均需有限边界。真实验证只使用现有账号，不要求用户在聊天中发送密钥，不修改用户账号或数据目录。

## Sources

- https://api-docs.deepseek.com/api/create-chat-completion/
- https://api-docs.deepseek.com/ （当前入门说明）

## Implementation Result

- 新增 `internal/core/proxy`：本机监听、幂等启停、停止取消请求、Host／Origin 保护、请求／响应校验、120 秒期限与并发／大小限制。
- 新增 `internal/provider/deepseek`：官方模型 ID、HTTPS／本机 HTTP 端点、受控 Bearer 鉴权、普通与逐事件 SSE 转发、禁止重定向。错误正文不透传，HTTP 200 错误对象也安全分类；缺失或不完整的 SSE 终止事件会中断响应。
- 新增 `internal/application/gateway`：复用已有账号与提供商服务，每次选择最早创建的已启用 DeepSeek 账号，通过 UseAPIKey 使用凭证；无密钥缓存、请求内容持久化或隐式重试。
- 接入 Wails GatewayInfo／GatewayStart／GatewayStop／GatewayTest 和类型化 gateway:status 事件；退出应用停止代理，HTTP 服务日志接入中央 slog／Redactor。
- 页面展示真实状态、DeepSeek 资源、带 /v1 的 Base URL；可选择模型与流式验证，验证最多输出 16 个 Token，不展示或缓存回复内容。验证中可以停止网关。
- 更新能力说明、PRD、架构与 CURRENT；不引入依赖、不迁移数据库、不修改账号业务、凭证存储或全局 UI 状态。

## Verification

- 独立 worktree 的最终 `task verify`：格式、Go vet、staticcheck、ESLint、TypeScript、全部 Go 测试、前端 23 个文件／130 项测试、Wails 绑定生成及 Windows 生产构建通过。
- 网关前端专项：4 个文件／28 项用例通过，覆盖启停、失败重试、事件刷新与清理、SSE 选项、停止取消入口、状态未知和安全反馈。
- Go 契约／集成覆盖普通与流式请求、首事件实时刷新、完整终止事件、截断流、并发上限、端口不可用、启停竞争、取消／超时、输入边界、跨源／Host 防护、重定向、错误正文和日志隐私，以及受保护密钥回调后清理。
- 显式真实验证使用现有开发数据库（只读）与 DPAPI 凭证，在隔离随机端口上到达 DeepSeek，但所选账号返回 HTTP 401，未验证真实模型回复。用户明确选择先完成实现、真实调用稍后验证，随后未继续发起外部模型请求。
- 视觉验证使用独立 Vite 9258 端口及隔离 Edge 配置／测试元数据，检查中文暗色运行状态、英文亮色 960px 停止状态与请求验证布局。
- PowerShell 文档示例通过语法解析；Vite 既有的大于 500 kB 提示保留，构建成功。
- 回填当前工作区后，Go 代理／适配器／应用网关／Wails／账号测试通过；TypeScript 和网关／账号前端 8 个文件／56 项测试通过。
- 回填清单 45 个网关文件。78 个现有账号、状态存储、凭证／基础设施和共享服务文件 SHA-256 保持不变；用户分支、HEAD、暂存区未改变。

## Documentation

已更新 `docs/product/GATEWAY.md`、PRD、架构与 CURRENT；Task Record 是完整验证与认证待办的依据。功能沿用既定分层和技术栈，无需新增 ADR。

## Deviations

按用户要求将首个适配器从计划中的 OpenAI 改为 DeepSeek；真实请求认证失败后，用户明确将有效凭证的真实调用验证延后。端口编辑、其他协议／提供商、自动调度、重试、用量和请求历史仍按后续任务处理。

## Git

独立分支 `codex/deepseek-gateway`：

```text
16b384e87d50c8e0bceb33691f1001858d166f6f feat(gateway): 打通 DeepSeek 本地代理与流式请求
```

基于最新账号提交整合已完成的网关准备页，在独立分支提交并回填当前工作区。用户原分支、HEAD 和暂存区未修改，其他任务文件未覆盖或纳入本任务提交。

## Result

DeepSeek 网关代码闭环和界面已完成并回填，自动化验证通过。真实调用已到达提供商但认证失败，更新有效凭证后的普通／流式模型回复留待用户后续验证。

## Completed

2026-10-07 12:03
