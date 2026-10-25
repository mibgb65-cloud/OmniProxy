# ADR-025: 提供商出站代理（mixed / SOCKS5 / HTTP）

## Status

Accepted

## Date

2026-10-26

## Context

部分海外提供商的接口在本机网络下不可达，用户已经运行 v2rayN / Clash 这类客户端，并希望
OmniProxy 只把「选定提供商」的出站请求转发到它的本地监听端口。v2rayN 默认监听形如
`本地:[mixed:10808]` 的端口：**同一个端口同时接受 HTTP CONNECT 和 SOCKS5**。

约束与事实：

- Go 的 `http.Transport.Proxy` 必须给定一个确定的协议（`http://` 或 `socks5://`），
  无法表达「两者皆可」。
- 协议猜错的失败表现与「上游挂了」几乎一致（`upstream_unavailable`），会把本地代理配置问题
  伪装成提供商故障。
- 代理端口没开是最常见的误配置，而账号故障切换会对失败账号写入冷却。

## Decision

1. **一份连接参数 + 每个提供商一个三态开关**。`proxy_settings`（单例）存协议、主机、端口、
   账号与密码引用、绕行列表；`providers.proxy_mode ∈ {inherit, direct, proxy}` 决定单个提供商
   是否走代理。`inherit` 跟随全局 `enabled`。
2. **协议由探测决定，用户不需要选**。`scheme` 允许 `auto`：对端点分别做 SOCKS5 greeting
   (`05 01 00`) 与 HTTP `CONNECT` 状态行探测；两者皆通（mixed）时选 SOCKS5，因为它是纯 TCP
   隧道，`ws://` 与 `wss://` 都不依赖代理侧的 HTTP 解析。探测结果写入 `detected_scheme`
   作为缓存，端点变化即失效。
3. **运行期纠偏**。Transport 按端点缓存实际协议；当连接建立但协议不成立（`malformed HTTP
   response`、SOCKS 版本字节错误、`0xFF` 认证拒绝、立即 RST）时切换另一种协议重试一次
   （同一端点 60 秒内最多一次），成功后回写缓存。这样 v2rayN 把 mixed 改成纯 socks 也不会
   让用户重配。
4. **代理层故障必须与上游故障分开**。请求前对代理地址做带 30 秒缓存的 TCP 预连，失败直接
   返回 `netproxy.ErrUnavailable`；核心边界换算成 `proxy_unavailable`，协议都不成立换算成
   `proxy_protocol_mismatch`。这两类错误不参与换号、不写账号冷却。
5. **解析顺序**：`no_proxy` / loopback → 提供商 `direct` → `proxy`（全局未启用则报错，不静默
   直连）→ `inherit` 跟随全局 → 环境变量兜底 → 直连。作用域优先取请求 ctx 里的提供商身份，
   其次按目标主机名匹配，最后用全局默认。
6. **只做进程内出站**。不接管系统代理、不做 TUN、不管理代理客户端生命周期、不读取订阅。
   应用更新与 Agent 安装下载不走提供商代理（流量与可用性考虑）。

## Consequences

- 用户只需「打开 v2rayN，填 10808」，不关心协议；错误信息能指出是代理端口问题。
- 新增一层 Transport 包装：适配器仍然保持自己的超时与 TLS 配置，只把底层 RoundTripper
  交给我们分流。
- 进程级 `Install(Resolver)` 是可变全局状态；未安装时 `Wrap` 原样返回，测试与旧行为不变。
- 按主机名回退匹配对「同一 base URL 的多个配置」有歧义，以请求作用域优先来消歧。

## Alternatives considered

- **让用户手选协议**：mixed 端口上两个都"应该"能用，选错就是无解的沉默失败。
- **每提供商单独存一份代理地址**：同一代理改端口要改十处，且与"全局唯一出口"的现实不符。
- **只支持 `HTTP_PROXY` 环境变量**：会影响应用更新下载，且需要重启，无法按提供商区分。
- **把 BaseURL 改指向代理端口**：丢失 TLS SNI 与目标主机语义，CONNECT 无法工作。
