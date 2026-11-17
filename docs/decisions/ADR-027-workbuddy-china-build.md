# ADR 027: WorkBuddy 国内版作为独立内置订阅服务商

## Status

Accepted

## Date

2026-11-17

## Context

WorkBuddy 是腾讯 CodeBuddy 桌面 Agent 的两个构建：国际版 `www.workbuddy.ai`
（已由 ADR-026 接入，kind `workbuddy_ai`）与国内版 `copilot.tencent.com`。两者共用同一套
插件 API（登录、续期、`/v3/config` 目录、`/v2/chat/completions`、`/billing/meter` 额度），
但存在必须区分的真实差异（社区插件 0.2.36 只读调研，未在真实账号验证）：

- 站点与登录平台标识不同：`platform=workbuddy-ai` 与 `platform=workbuddy`，登录页版本参数
  影响企业微信入口；
- 客户端指纹不同：UA 第二段分别是 `workbuddy-ai` 与 `WorkBuddy`；
- 国内版为 Claude 模型额外提供原生 Anthropic `POST /v1/messages`，而 Chat 端点上这些模型
  忽略推理档位；Messages 上还额外接受 `max` 档位；
- 国内版 `glm-*` 实测能读图（国际版会把图片丢掉）；
- 国内版另有签到接口、云托管企业域名（`<ent>.sso.copilot.tencent.com` 等）与
  `www.workbuddy.cn` 的 API Key 入口。

两个构建是**两套独立账号体系与套餐**，同一用户可能同时拥有两边账号。

## Decision

1. **独立内置服务商**：新增 `builtin_workbuddy`（kind `workbuddy`、展示名
   「WorkBuddy（国内版）」、迁移 052），与国际版 `builtin_workbuddy_ai` 并列。认证方式复用
   `workbuddy_login`，凭据结构与 DPAPI 存储不变；账号按 kind 与提供商隔离，令牌不能互挂。
2. **适配器按 build 参数化，不复制包**：`internal/provider/workbuddyai` 引入 `Build`
   （站点、登录平台、UA、Messages 能力、`glm-*` 图片行为），`New` 仍构造国际版，
   `NewForBuild` 构造任意构建。国际版常量与行为保持不变。
3. **协议路由按模型选择**：国内版 Claude 模型在调用方使用 Messages 协议时原样转发到
   `/v1/messages`（不 gzip、不改写系统提示词、不做 11128 重试）；其它情况仍走 Chat，
   由网关聚合后按客户端协议回包。目录对 Claude 模型追加 `max` 档位。
4. **额度只读展示**：Credits 三窗口沿用国际版口径与快照表；签到（国内版独有）与云托管
   企业域名不在本期范围，遇到时给明确提示、不猜主机、不静默降级。

## Alternatives Considered

- **账号级「国际版／国内版」选项**（像 Kimi／MiniMax 的地区选项）：登录前就必须选定构建
  （`state?platform=` 不同），账号卡名称对其中一侧必然错，且要在网关、目录版本、额度快照
  三条链路按账号分叉；比新增一个 kind 改动更大、回归风险更高。
- **只改 Base URL 复用同一 provider**：无法同时存在两版账号，登录平台标识与 UA 也不对。
- **国内版单独复制一份适配器包**：两套 20k 行级测试与常量会立刻漂移；差异只有站点与指纹。

## Consequences

- 用户可在账号页分别登录国际版与国内版，各自同步目录、建路由、看额度；两版模型互不干扰。
- Claude 模型必须用 Messages 协议才带得动推理档位；Chat 客户端仍可用但档位无效，文档写明。
- 上游指纹与业务码仍非公开契约，发版后可能整体失效；两个构建的版本常量需要同步更新。
- 国内版签到、云托管企业域名与 API Key 认证留待后续任务，当前不宣称支持。

## References

- ADR-026（国际版接入与请求形态对齐）
- `docs/product/WORKBUDDY.md`、任务记录 `docs/tasks/completed/2026-11/TASK-20261116-015`、
  `TASK-20261117-006`（规划）、`TASK-20261117-007`（实施）
