# ADR 023: Codex 入口恢复 v1 形态的透明转发

Status: Accepted

Date: 2026-10-24

Supersedes: ADR-021 的「不引入通用 URL 转发」与 ADR-005 中「网关复刻 Codex 请求」一条

## 背景

v2 的 Codex 入口是协议复刻：网关只接受固定端点，重建请求头（`Authorization`、
`ChatGPT-Account-ID`、`OpenAI-Beta`、`Originator`），按订阅端点要求改写正文
（强制 `store=false`、`stream=true`、删除 `max_output_tokens`／`temperature`／
`top_p`），WebSocket 在本地终结后转成 HTTP 调用，令牌续期交给官方组件。

v1（旧项目 `E:\go\new\OmniProxy`）的做法相反：入口是透明反向代理——路径透传
（`/backend-api/codex`、`/codex`、`/v1` 逐个剥离后拼上游基址）、客户端头部整体
透传（`session_id`、`conversation_id`、`originator`、`User-Agent`、压缩与
`OpenAI-Beta` 协商都保留）、正文不改写、WebSocket 真正拨到上游双向中继、访问
令牌由网关自己用 `refresh_token` 续期。

用户要求把 Codex 的处理流程换回 v1 的做法，同时保留客户端 Base URL
`http://127.0.0.1:PORT/codex/v1`。

## 决策

- `/codex/...` 与 `/agent/codex/<本地接入Key>/...` 前缀之后的**任意路径**进入
  `internal/core/proxy` 的透明转发：不再有端点白名单，只做 v1 的路径归一。
- 请求头整体透传，仅替换鉴权头：删除客户端自带的 `Authorization`、`X-Api-Key`、
  `Api-Key`、`ChatGPT-Account-Id`，写入网关借出的 `Authorization: Bearer …` 与
  `ChatGPT-Account-ID`。网关自有标记（`X-OmniProxy-*`）与跳转级头部不转发。
- 请求正文不再改写：按 32 MiB 有界读取后原样转发，保留客户端压缩编码，不解析
  字段。
- WebSocket 升级时真正拨到上游，按帧双向中继；不再把 WS 帧翻译成 HTTP 请求。
- 上游基址可配置（`codex_settings.upstream_base_url`，默认
  `https://chatgpt.com/backend-api/codex`），与 v1 的设置项对齐。
- 令牌续期回到 v1 形态：官方保存的访问令牌临近过期时，网关用同一份
  `refresh_token` 走 `auth.openai.com/oauth/token` 自行换新，结果只在本进程内存
  缓存，不回写官方存储。
- 例外保留在 `chat/completions` 与 `messages`：订阅端点只接受 Responses，
  v1 对 `chat/completions` 也做转换，因此这两条入口仍走协议转换；`/v1/codex-models`
  仍是本地目录。
- 鉴权、loopback / Host 校验、并发槽、请求期限、接入 Key 归属、故障切换冷却
  等 v2 边界全部保留；透传只放宽路径、头部与正文。

## 后果

- 上游看到的是真实客户端指纹（会话、压缩协商、UA），不再被网关统一改写；
  订阅端的压缩项、compact、以及未来新增的端点不需要逐个建模即可使用。
- 令牌会进入 `http.Header` 参与网络请求（v1 语义），因此 Core 在请求结束后清零
  该头部，且不写日志、不落库、不返回前端。
- 正文不再被网关校验：畸形负载由上游拒绝，错误正文原样回传给客户端。这一取舍
  换来与官方客户端一致的行为。
- 客户端保持原生 WebSocket 默认选择，不再新增已移除的
  `features.responses_websockets`／v2 标记（2026-10-24 握手修复补充）。
  受管刷新仅在修订与备份归属可证明时清理旧值；是否接收由网关开关控制，
  不保证客户端对拒绝立即回退 HTTP。
- 路径只允许在固定上游主机内变化（拒绝 `..` 与控制字符）；上游基址仍是唯一决定
  目标主机的配置项。
