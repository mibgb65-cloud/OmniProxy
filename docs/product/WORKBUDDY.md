# WorkBuddy AI 订阅通道

WorkBuddy AI 是腾讯 CodeBuddy 计划的国际版桌面 Agent（<https://www.workbuddy.ai>），
一个账号订阅一套模型（GPT、Gemini、GLM、Kimi、DeepSeek、MiniMax 等）。
OmniProxy 把它当作**内置订阅服务商**接入：用浏览器完成 WorkBuddy AI 登录后，
本机网关以 `workbuddy_ai/<model>` 转发请求，受管客户端（Claude Code、Codex、DSH、
OpenCode 等）按既有路由投影自动可用。

- 服务商 ID：`builtin_workbuddy_ai`
- kind：`workbuddy_ai`
- 认证方式：`workbuddy_login`（只有浏览器登录，没有 API Key、没有设备码）
- 上游基址：`https://www.workbuddy.ai`

## 使用

1. **服务商 → WorkBuddy AI → 添加账号**，选择「登录 WorkBuddy AI」。
2. 浏览器打开腾讯的登录页；完成登录后 OmniProxy 自动轮询取得令牌（最多 5 分钟）。
3. 登录成功后后台自动同步一次 `/v3/config` 模型目录；同步完成前该服务商没有可路由模型。
4. 在**路由**页为需要的模型建规则（或使用同名默认映射），客户端目录随之出现。
5. 账号卡显示 Credits 额度与「刷新 / 重新登录 / 退出」。

> 登录令牌保存在本机受保护凭据存储（DPAPI），SQLite 只存不透明引用与可展示身份；
> 令牌不会返回前端、不写日志。退出会先停用授权再清理密文，清理失败保留引用以便重试。

## 模型目录

- 来源：`GET /v3/config`（按账号的套餐发放），字段映射：
  `maxInputTokens → ContextWindow`、`maxOutputTokens → OutputTokenLimit`、
  `supportsImages → 图片输入`（`glm-*` 一律按文本处理）、`reasoning.supportedEfforts → 推理档位`。
- 缺失字段保持**未知**，不补 0、不猜测；`/v3/config` 失败时保留上次成功快照。
- 目录与授权版本绑定：令牌续期会在同一事务内重绑目录，**不会**因为续期而丢模型。
- 目录同步失败不阻塞旧路由；修复网络或重新登录后自动重试。

## 协议边界

| 项 | 事实 |
| --- | --- |
| Chat Completions | 原生：`POST {base}/v2/chat/completions`，**上游只接受流式** |
| Responses / Messages | 由网关 `agentprotocol` 转换成 Chat 后再转发，并按调用方原协议回包（不是上游原生） |
| 非流式客户端 | 网关以上游流式拉取、聚合成完整回包后再返回，不把半个工具调用交给客户端 |
| 图片 | 仅当目录声明 `supportsImages`；`glm-*` 按文本处理，图片由图片描述模型代读 |
| 推理档位 | 只发布目录声明的档位；上游未声明时不发布 |
| 请求形态 | 按官方桌面客户端形态发送（UA、`X-Product`、`X-IDE-*`、`X-Agent-*`、OpenAI SDK 指纹、gzip 请求体、会话 id 头）。上游对不匹配的 UA 返回 `10085` |
| 认证失败 | 401 先尝试续期一次，仍失败才换号；登录失效要重新登录 |
| 额度不足 | 上游 402/429 按既有错误语义回传，不放大重试 |

## Credits 额度

账号卡与配额页读取 `GET /billing/meter/get-user-resource-summary`（企业账号走
enterprise meter），展示三种窗口或一种：

- **Credits**：套餐每周期授予的基础积分（`aside`，不单独停止账号）；
- **Extra credits**：bonus、充值、赠送等其余积分（`aside`）；
- **Total credits**：两者合计，**只有它决定账号是否可用**；
- **Prepaid credits**：只有余额、没有总量的预付包。

窗口带周期重置时间与最早到期时间（`runsOutAt`），供「先花快过期的」判断参考；
**首版额度只用于展示，不参与调度**（不按额度跳过账号），避免把上游解读当成确定规则。

免费模型另有每日 token 窗口（厂商不返回、需按本机记账重算），**首版未接入**：
此时页面按无额度接口服务商处理，显示本机 Token 用量。

## 错误码

界面与日志只使用稳定错误码，不回显上游原文：

| 码 | 含义与下一步 |
| --- | --- |
| `sign_in_expired` | 登录已失效，重新登录 |
| `unauthorized` | 上游拒绝令牌：先自动续期一次，仍失败则重新登录 |
| `channel_rejected` | 上游按渠道规则拒绝了系统提示（11128）；换模型或稍后重试 |
| `signature_invalid` | 会话里的思考签名与上游不匹配（11159）；新开对话或换模型 |
| `model_not_entitled` | 账号套餐不含该模型（11158）；换模型 |
| `user_agent_refused` | 上游拒绝了客户端标识（10085）；升级 OmniProxy |
| `rate_limited` | 上游限流；按 `Retry-After` 等待 |
| `upstream_unavailable` | 上游不可达或 5xx；稍后重试 |
| `invalid_response` | 上游返回了无法识别的结构；保留旧快照 |
| `refresh_rejected` | 续期令牌被拒；重新登录 |
| `credential_unavailable` | 本机受保护令牌不可用；重新登录 |
| `busy` / `cancelled` / `login_expired` | 登录会话状态，重新发起登录 |
| `identity_changed` | 重新登录的账号与已保存账号不一致；换回原账号 |
| `configuration_changed` | 请求期间账号被修改；重试 |
| `browser_failed` | 系统浏览器打开失败；手动打开登录页 |

## 安全与合规

- 令牌只存受保护凭据存储；日志只记录固定错误码，不记录令牌、账号名或上游正文。
- 上游只允许固定 HTTPS 主机；HTTP 仅允许回环地址（测试用），并禁止重定向。
- 请求形态向官方客户端对齐属于绕开上游客户端识别，可能带来账号风险；
  与 Google 订阅通道（ADR-013）同级，见 ADR-026。OmniProxy 不承诺账号不受限制。

## 未验证与未接入

- **未用真实账号验证**：登录、`/v3/config` 目录、Chat 转发、Credits 三窗口均按社区插件
  `@magpie-community/opencode-workbuddy-auth` 0.2.36 的协议事实实现，尚未真机核对；
  首次使用请以小额度模型验证。
- 未接入：国内版 WorkBuddy（`copilot.tencent.com`）及其 Claude 模型的 Messages 路由、
  每日签到（国际版无该接口）、桌面登录文件导入（5.6+ 加密不可读）、API Key 认证、
  免费模型每日 token 窗口、额度参与调度、队列/平衡轮换。
- 上游版本指纹固定在 WorkBuddy AI 5.9.3 / CLI 2.165.0；官方发版后可能需要更新
  `internal/provider/workbuddyai` 中的版本常量。

## 相关

- 任务记录：`docs/tasks/active/TASK-20261116-015-workbuddy-ai-subscription-provider.md`
- ADR：[ADR-026](../decisions/ADR-026-workbuddy-ai-subscription.md)
- 代码：`internal/provider/workbuddyai/`、`internal/application/workbuddy/`、
  `internal/application/account/workbuddy.go`、`internal/application/gateway/workbuddy.go`
