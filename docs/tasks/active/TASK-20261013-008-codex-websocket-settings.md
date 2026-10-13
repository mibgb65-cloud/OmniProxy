# TASK-20261013-008: OpenAI 账号设置中的 Codex WebSocket 开关

## Status

In Progress

## Created

2026-10-13（Asia/Singapore）

## Request

在账号页面的 OpenAI 账号设置中新增开关，控制 Codex CLI 与桌面端的 WebSocket，用于处理 `unexpected status 403 Forbidden: websocket proxy disabled, url: ws://127.0.0.1:3000/codex/v1/responses`。

## Goal

定位报错根因，实现可持久化、对 Codex CLI／Desktop 实际生效的 WebSocket 设置，并验证兼容性与错误反馈。

## Scope

OpenAI 账号设置、网关 WebSocket 接入开关、必要的后端接口与验证、相关文档。

## Analysis

本机 Codex 0.155.1 将 responses_websockets／v2 标记为 removed，且拒绝覆盖保留的内置 openai；当前客户端使用 openai 身份与旧应用的 3000/codex/v1 地址。官方 supports_websockets 只适用于自定义提供商，不能通过旧 feature 标志实现可靠客户端开关。

用户明确选择「控制 OmniProxy 网关是否接受 WebSocket，保留原会话身份」。实现应用级开关，放在 OpenAI 账号设置，SQLite 持久化，默认开启以保留现有网关行为。每次握手读取当前设置，关闭时返回明确拒绝，普通 HTTP/SSE 不受影响，无需重启。只控制当前项目的网关，不改写 Codex 身份／认证／配置或另一个旧版本进程。

参考：[官方配置参考](https://learn.chatgpt.com/docs/config-file/config-reference)、[官方开发者设置](https://learn.chatgpt.com/docs/developer-settings)、ADR-005；实机 CLI 的 features list 与只读配置加载验证补充版本事实。

## Implementation Plan

1. 定位 403 来源与现有 WebSocket 配置和接入边界。
2. 核对 Codex 官方配置与当前本机实现，确定最小闭环方案。
3. 实现设置持久化、网关动态接入控制与页面开关。
4. 验证开启／关闭、配置保留与异常路径，运行 `task verify`。
5. 更新相关文档，审查并提交本任务，完成归档。

## Implementation Result

- 按用户选择实现网关级开关，放在 OpenAI 账号设置，所有账号共用；复用既有开关组件和主题样式，按需读取、键盘可用，保存失败保留原值，读取失败禁用并提供重试。
- 迁移 016 创建唯一非敏感设置记录，默认开启；应用层通过参数化 SQL 原子保存，缺失／损坏／读写失败返回稳定错误，不静默恢复默认。
- Core 接受只读策略回调，鉴权后检查新 WebSocket 握手；关闭时返回 403 websocket_disabled。已建立连接在下一次创建响应时也检查，进行中的响应可正常结束；HTTP/SSE 不受影响，无需重启网关。
- Wails 新增 GetCodexSettings 与 SetCodexWebSocketEnabled，并在网关初始化注入持久化策略；不新增依赖，不修改客户端配置、身份、认证、会话或旧进程。
- 同步中英文文案、生成绑定、PRD／网关／订阅／本地接入／架构文档与当前看板。

## Verification

- PASS：本机 codex --version 与 features list 确认 0.155.1 和旧标志 removed；只读覆盖加载确认内置 openai 禁止重定义。官方配置参考与开发者设置已读取核对。
- PASS：代理核心测试验证动态开启／关闭／重新开启无需重启，拒绝新握手和旧连接后续响应、关闭时 HTTP 仍可调用、鉴权先于策略、读取失败拒绝访问且不泄露原始错误。
- PASS：应用层与 Wails 测试验证迁移默认开启、保存／重建服务后持久化、记录缺失／数据库失败明确报错、绑定不需要初始化 Agent 或访问客户端凭据。
- PASS：OpenAI 账号设置与账号详情 13 项针对性前端测试，覆盖按需读取、关闭与焦点恢复、键盘切换、保存及重新打开、失败保留原状态、私有错误隐藏和读取重试。
- PASS：`task verify BIN_DIR=E:/go/new/wails3-v/OmniProxy/tmp/codex-websocket-settings/quality-bin` 退出码 0；格式、Go vet／staticcheck／ESLint、TypeScript、全部 Go 测试、67 个前端文件／354 项测试及 Windows 生产构建通过。
- PASS：隔离 test profile 原生窗口经真实 Wails 绑定关闭／开启，重新打开状态保留，SQLite 与界面状态一致，深浅主题截图人工核对正常；QA 窗口正常退出。
- PASS：按 EXE 路径识别当前仓库开发窗口（PID 23240），验证真实读取、深浅主题及最小 960×800 布局，未修改真实开关或 Codex 配置；原窗口尺寸与主题恢复，最终停留 OpenAI 账号设置。
- PASS：git diff --check、全部本次文件 UTF-8 无 BOM 核对，生成绑定只增加预期方法和设置类型。既有工具探测、jsdom scrollTo、Vite 大包与受管目录刷新提示未影响质量门。

原生脚本、隔离数据、截图和日志保存在 `tmp/codex-websocket-settings/`，不纳入提交。

## Documentation

已同步 PRD、GATEWAY、SUBSCRIPTION、LOCAL_AGENTS、OVERVIEW 与 CURRENT；ADR-005 的身份保护保持兼容，无新架构决策或技术栈调整。

## Deviations

原计划核对客户端配置开关；实机确认旧标志移除且内置 openai 不可覆盖后，向用户说明两种方式。用户明确选择网关接入控制，实际实现不修改客户端配置。

## Remaining Risks

用户报错地址属于旧应用端口 3000，监听进程 PID 31320（另一目录的 OmniProxy）。本开关只影响当前新版网关，不能联动旧进程；Codex 需通过新版一键接入切换地址后才能受控。没有自动修改真实客户端配置或发起真实模型调用，也不保证客户端在禁用 WebSocket 后自动回退 HTTP。

## Git

本次实现、迁移、生成绑定与文档差异已审查，待写入实现提交哈希。

## Result

用户选择的网关级 Codex WebSocket 开关已实现，默认开启、持久保存、动态生效，HTTP/SSE 保持可用，客户端身份和会话保留。质量门与原生验收完成，待提交归档。
