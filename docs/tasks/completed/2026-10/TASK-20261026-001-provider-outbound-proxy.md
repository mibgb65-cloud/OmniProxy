# TASK-20261026-001: Provider Outbound Proxy (mixed / socks5 / http)

## Status

Completed

## Created

2026-10-26 10:00

## Request

用户需求（原文意图）：

> 有些国外的服务商需要走 v2rayN 这种代理端口转发请求，希望在提供商页面增加一个设置按钮，
> 在里面设置要走代理的服务商。
>
> 补充：还要支持已经用软件开启的代理，只需要转发到它的端口 10808 这种（v2rayN 显示
> `本地:[mixed:10808]` / `局域网:[mixed:10808]`），同一个端口同时支持 HTTP 和 SOCKS5。

## Goal

OmniProxy 作为客户端，把「选定提供商」的出站请求转发到本机已运行的代理端口（v2rayN /
Clash / mihomo 的 mixed 端口），无需用户选择协议；代理端口不可用时给出可诊断的错误码，
且不得把代理故障算成账号故障。

## Background

- v1 有 `OutboundProxy{Enabled,URL,Providers}`（`tmp/app-update-20261018/legacy-source`），
  v2 目前完全没有出站代理能力；探针仍硬编码 `http.ProxyFromEnvironment`。
- Go 的 `http.Transport` 必须二选一（`http://` 或 `socks5://`），mixed 端口无法直接表达。
- 相关 ADR：`docs/decisions/ADR-025-provider-outbound-proxy.md`。

## Scope

- 迁移 044：`proxy_settings` 单例表 + `providers.proxy_mode`。
- `internal/core/netproxy`：端点解析、mixed 协议探测、端口发现、出站 Transport（协议纠偏）。
- `internal/application/proxysettings`：读写/密码托管/快照缓存/测试/发现。
- `internal/application/provider`：`proxy_mode` 字段与 `SetProxyMode`。
- 接入：11 个提供商适配器、网关请求作用域、连通性探针、Codex 透明转发（HTTP + WebSocket）、
  Codex 额度与令牌刷新。
- 失败分类：`proxy_unavailable` / `proxy_protocol_mismatch`，不触发换号冷却。
- Wails 绑定 + 提供商页「网络代理」按钮 + 弹窗 + 卡片徽标 + 中英文案。

## Out of Scope

- 不启动/管理 v2rayN 进程，不读取订阅与节点，不解析 vmess/vless/trojan 链接。
- 不改系统代理、不做 TUN 接管、不扫描全网段端口（仅探测候选 loopback 端口）。
- 应用自身更新与 Agent 安装下载不走提供商代理。
- 余额/额度/订阅服务的独立调用链（依赖适配器 Wrap 覆盖，未逐个改调用点）见 Deviations。

## Existing Behavior

- 所有提供商请求直连（或跟随进程环境变量 `HTTP(S)_PROXY`）。
- 代理不可用只能表现为 `upstream_unavailable`，会被当作上游故障参与换号与冷却。

## Expected Behavior

- 用户在提供商页设置一次代理端点（可只填端口、可粘贴 `mixed:10808`），勾选要走代理的提供商。
- 同一端点既支持纯 HTTP、纯 SOCKS5，也支持 mixed（自动识别，运行期自动纠偏）。
- 代理端口无响应时返回 `proxy_unavailable`，不消耗候选账号、不写账号冷却。
- 卡片上能看到「代理 · SOCKS5」这类实际生效协议。

---

# Analysis

## Related Modules

- `internal/core/proxy`（核心代理、透明转发、探针失败分类）
- `internal/provider/*`（11 个适配器）
- `internal/application/gateway`、`internal/application/provider`
- `internal/transport/wails`、`frontend/src/features/providers`

## Dependencies

- 标准库 `net`、`net/http`、`crypto/tls` 足够；不引入 `x/net/proxy`（`http.Transport`
  原生支持 `socks5://`，且非 IP 主机名由代理解析）。
- 密码复用 `internal/infrastructure/credential`（DPAPI）。

## Risks

- 全局 Install 的 Resolver 属于进程级状态，测试必须能不装即保持原行为。
- 协议猜测失败会多一次往返；用一次探测 + 60 秒纠偏窗口限制开销。
- 按主机名回退匹配提供商有歧义（同一 base URL 的多个配置），以 ctx 作用域优先。

---

# Implementation Plan

1. 迁移 044。
2. `netproxy`：`Endpoint` 解析、`Detect`（SOCKS5 greeting + HTTP CONNECT 状态行）、
   `Discovery`、`Decide`、`Transport`（预连缓存、协议缓存、纠偏、连接池）。
3. `proxysettings`：读写、密码托管与清除、快照缓存、探测结果持久化。
4. 适配器 `netproxy.Wrap`，网关设置请求作用域，探针与透明转发接入。
5. `proxy.UpstreamFailure` 分类 + failover 不冷却。
6. 绑定与前端。
7. 测试与质量门。

---

# Progress

- 完成 Core `netproxy` 包（端点解析、mixed 探测、端口发现、Transport 纠偏）与单元测试。
- 完成迁移 044、`proxysettings` 服务、provider `proxy_mode`。
- 完成适配器 / 网关 / 探针 / 透明转发接入与失败分类。
- 完成 Wails 绑定与前端弹窗、卡片徽标、i18n。

---

# Implementation Result

## Changes

实际完成：

- 迁移 044：`proxy_settings` 单例（scheme/host/port/username/password_ref/no_proxy/detected_scheme）
  与 `providers.proxy_mode`（inherit/direct/proxy）。
- Core `internal/core/netproxy`：
  - `ParseEndpoint` 支持 `10808`、`mixed:10808`、`socks5h://user:pass@host:port`、`host:port`；
  - `Detect` 用 SOCKS5 greeting 与 HTTP CONNECT 状态行识别端点（mixed → 选 SOCKS5），不产生真实出网流量；
  - `Discover` 并行探测 10808/10809/7890/7891/7897/1080/2080/1087；
  - `Decide` 固定解析顺序（no_proxy/回环 → direct → proxy → inherit → 环境变量 → 直连）；
  - `Transport` 按请求上下文的提供商身份分流，代理预连缓存 30 秒、协议缓存 60 秒纠偏窗口，
    设置 `updated_at` 变化时丢弃连接池。
- `internal/application/proxysettings`：读写、密码托管到凭据库、5 秒快照缓存与保存即失效、
  端点探测、端口发现、识别结果回写；未迁移到 044 的数据库按“未启用代理”降级。
- `internal/application/provider`：`Configuration.proxyMode` 与 `SetProxyMode`，非法值拒绝。
- 接入：11 个提供商适配器 `netproxy.Wrap`；网关 `forwardAgent` 写入提供商作用域；连通性探针；
  Codex 透明转发（HTTP + WebSocket，WebSocket 复用同一 passthrough 客户端走 CONNECT）；
  Codex 额度读取与 OAuth 续期默认客户端。
- 失败分类：`proxy.UpstreamFailure` 换算 `proxy_unavailable` / `proxy_protocol_mismatch` /
  `proxy_not_configured`；`rejectedAccount` 与 river 瞬时失败判定都不把代理层错误算成账号故障。
- Wails 绑定：`GetProxySettings`、`SaveProxySettings`、`TestProxySettings`、`DiscoverProxyPorts`、
  `SetProviderProxyMode`；出站代理服务实例与绑定层共享，保存即失效。
- 前端：提供商页「网络代理」按钮与弹窗（服务器参数、协议自动识别、探测本机代理、测试结果分类、
  提供商三态与批量操作）、卡片「代理」徽标、37 个中英文案。

## Files Changed

主要修改文件：

- `migrations/044_provider_proxy.sql`
- `internal/core/netproxy/{netproxy,endpoint,detect,decision,transport}.go` 与三个测试文件
- `internal/core/proxy/{netproxy_failure,server,passthrough_send,passthrough_socket}.go`
- `internal/application/proxysettings/{service,service_test}.go`
- `internal/application/provider/{catalog,mutations,probe,service}.go`
- `internal/application/gateway/{agent,failover_policy}.go`
- `internal/provider/<name>/client.go`（11 个适配器）
- `internal/infrastructure/codex/{usage,oauth_refresh}.go`
- `internal/transport/wails/{app,proxy_settings,proxy_settings_test}.go`
- `frontend/src/features/providers/{proxy-settings-dialog,use-proxy-settings,provider-card,provider-catalog,provider-list,providers-view}.tsx|ts`
- `frontend/src/services/omniproxy.ts`、`frontend/src/i18n/locales/{en,zh}.ts`、前端 bindings（重新生成）

## Technical Notes

- 协议猜测不需要用户参与：`scheme=auto` 时先探测；探测失败先猜 HTTP，纯 SOCKS 端口由
  60 秒一次的重放纠偏兜住（无正文或 `GetBody` 可重放时才允许重试）。
- `Wrap` 只在 Resolver 已安装、且底层是 `*http.Transport` 时包装；测试注入的契约传输原样返回。
- 预连缓存让“代理没开”在请求发出前就失败，错误码不再冒充上游故障。
- 扫描型 SELECT 增加 `proxy_mode` 后，手工建表的测试夹具（gateway service_test /
  provider_preferences_test）同步补列。

---

# Verification

## Tests

- [x] Unit Tests
- [x] Integration Tests（适配器 + 网关 + 透明转发既有集成用例保持通过）
- [ ] E2E Tests（无独立 E2E 套件；前端组件测试覆盖弹窗交互）

## Quality Checks

- [x] Format
- [x] Lint
- [x] Type Check
- [x] Test
- [x] Build
- [ ] Verify（`task verify` 未整体运行；等价门逐项执行，见下）

## Verification Result

```text
gofmt -l main.go internal            PASS（无输出）
go vet . ./internal/...              PASS
staticcheck . ./internal/...         PASS
go test . ./internal/...             PASS（全部包）
pnpm typecheck                       PASS（0 errors）
pnpm test                            PASS（132 files / 806 tests）
pnpm lint                            PASS
pnpm format:check                    PASS
task build                           PASS（bin/omniproxy.exe，production tags）
```

`task verify` 未一次性执行：其组成项（format check / lint / typecheck / test / build）已分别
在 Go 与 frontend 两侧跑通；首次 `task build` 末尾的 `Remove-item *.syso` 清理步骤返回 1，
重跑即通过（无 `.syso` 残留的脚本清理行为，与本次改动无关）。

新增用例：`netproxy`（端点解析、协议识别、端口发现、三路由、纠偏、代理不可用、未配置）、
`proxysettings`（校验/密码保留与清除/快照与主机回退/失效/降级/探测）、
`transport/wails`（绑定持久化与非法模式）、前端 `proxy-settings-dialog`（9 项）与卡片徽标（2 项）、
提供商页头部入口（1 项）。

---

# Documentation

- [x] ADR（ADR-025）
- [x] PRD（Core Features 增加一条）
- [x] Architecture（OVERVIEW Core 段落）
- [x] 新增 `docs/product/NETWORK_PROXY.md`
- [x] CURRENT
- [x] decisions/README 索引

---

# Deviations

1. 余额/额度/订阅查询没有逐个改调用点：它们复用提供商适配器，适配器已包装 Transport，
   加上快照里的主机名回退（内置提供商使用目录默认端点）即可命中正确策略；只有无法从
   上下文或主机名定位的调用才会落到全局默认。
2. `Codex CLI 子进程` 的环境变量注入未实现（原计划 Phase 4）。子进程属于外部程序，仍沿用
   继承来的 `HTTPS_PROXY`；透明转发、额度读取与令牌续期这三条本进程路径已接入。
3. v1 `OutboundProxy*` 配置的导入映射未实现。
4. `task verify` 未整体运行，见 Verification Result 说明。

---

# Git

## Commit

Not committed by Agent.

---

# Result

提供商出站代理已可用：用户把 v2rayN／Clash 开在默认端口即可在提供商页一次性配置，应用自动识别
mixed 端口协议并在协议变化时自愈；代理层故障有独立错误码且不会污染账号冷却。Go 全量测试、
前端 806 项测试与 production 构建通过，实机代理端口与原生窗口待用户验收。
---

# Completed

2026-10-26 20:40
