# TASK-20261120-008: Verify Configured Proxy Scheme Before Use

## Status

Completed

## Created

2026-11-20 19:30

## Request

用户给出一个远端代理地址并询问能否在本项目的出站代理里使用：

```text
curl --proxy "http://cspbklyy:ergdhu0mx08l@9.142.31.120:5278/" https://ipv4.webshare.io/
这个是不是没法在我们的代理使用，你看看呢
```

## Goal

确认该端口实际协议，判断 OmniProxy 出站代理能否使用它；修掉「用户按 URL 前缀选了
HTTP，但端口只说 SOCKS5」时请求挂起、无法自愈也无法诊断的问题。

## Background

实测（curl + 本项目 netproxy）：

```text
curl -x http://cspbklyy:***@9.142.31.120:5278 https://ipv4.webshare.io/     → 25 秒超时，0 字节响应
curl -x socks5h://cspbklyy:***@9.142.31.120:5278 https://ipv4.webshare.io/ → 200，出口 IP 9.142.31.120
netproxy.Detect                                                            → socks5=true http=false scheme="socks5"
```

分步计时（同一代理，3 轮）：TCP dial 1 ms、SOCKS5 greeting 回复 1.4–1.9 秒、用户名密码子协商
+0.5–0.7 秒、CONNECT 回复 +0.5–0.7 秒。即该端口是**纯 SOCKS5**（带认证），对 HTTP `CONNECT`
既不回应也不断开，且握手本身就慢，接近旧探测预算。

## Scope

- `internal/core/netproxy`：显式协议首次使用前做一次有界握手确认；确认失败时探测另一种协议
  并切换，切换结果回写 `detected_scheme`。
- `internal/application/proxysettings`：协议变化时清除旧 `detected_scheme`；测试探测预算适配
  远端代理。
- 测试：静默 SOCKS5 端口的回归用例、显式协议保持用例、协议变化清除识别结果用例。
- 文档：`docs/product/NETWORK_PROXY.md` 与 `docs/plans/CURRENT.md`。

## Out of Scope

- 不新增协议（vmess/vless/trojan），不解析远端订阅，不做代理分组。
- 不改 `no_proxy`、分流顺序与失败分类语义。
- 不为该远端代理提供内置预设或凭据托管。

## Existing Behavior

- `scheme=auto`：首请求探测（SOCKS5 greeting + HTTP CONNECT），按结果建池并回写识别结果。
- `scheme=http|socks5`：直接建池，不做任何验证；协议猜错时只有「对端主动拒绝」
  （EOF / malformed response）才会触发 60 秒一次的重放纠偏。
- 对端接受 TCP 却不回应错误协议时，请求挂到调用方超时，错误码为空。

## Expected Behavior

- 显式协议首次使用前做一次有界握手确认；确认通过则照旧使用。
- 确认失败而另一种协议可用时自动切换并回写识别结果，之后不再重复探测。
- 两种协议都不应答时保留用户配置（与修改前一致），不把瞬时抖动升级成硬错误。
- 用户改变协议后，旧的识别结果失效。

---

# Root Cause

`Transport.schemeFor` 把显式 `http`／`socks5` 当作既定事实：`remember` 后直接建
`http.Transport` 连接池，跳过探测。而运行期纠偏只在 `isProtocolFailure` 认出协议类错误
（EOF、`malformed http response` 等）时触发。该代理对 HTTP `CONNECT` 保持沉默，
`http.Transport` 一直等到调用方 context 超时，产生的是普通超时错误，于是既不自愈，
也没有 `proxy_protocol_mismatch` 这类可诊断错误码——每个请求都挂到超时。

旧探测预算 2 秒也偏紧：该代理 SOCKS5 greeting + 认证实测 1.9–2.6 秒，`auto` 探测因此
时好时坏（同一地址两次运行，`auto` 一次 6.6 秒成功、一次 20 秒超时）。

---

# Implementation Plan

1. `detect.go` 增加单协议确认函数，复用 `probeSOCKS5` / `probeHTTP`。
2. `transport.go` 的 `schemeFor` 改为「显式协议先确认 → 失败探另一种 → 都不行保留配置」。
3. 握手预算由 2 秒提到 4 秒；`Test` 探测总预算 5 秒提到 10 秒。
4. `proxysettings.Save` 在协议变化时清除 `detected_scheme`。
5. 新增回归测试，同步文档并跑 `task verify`。

---

# Progress

- 用 curl 与本项目 netproxy 复现：`http` 挂起、`socks5` 200；分步计时定位到「端口不说 HTTP」。
- 实现确认与自愈后，真实代理 `auto`／`http`／`socks5` 三种配置均 200。
- 第一次 2 秒预算下确认仍偶发超时（1.9 ms 级抖动即失败），改为 4 秒并复测稳定。

---

# Implementation Result

## Changes

- Core `netproxy`：
  - 新增 `Confirm(ctx, endpoint, scheme) bool`：复用 SOCKS5 greeting 与 HTTP CONNECT 状态行探测，
    只做握手、不产生真实出网流量。
  - `Transport.schemeFor`：显式 `http`／`socks5` 也先确认；确认失败时探测另一种协议，成立就切换
    并 `recordDetection` 回写 `detected_scheme`；两种都不应答时保留用户配置。
  - `detectAttemptTimeout` 2 秒 → 4 秒（探测与确认共用单协议预算）。
- `proxysettings`：
  - `Save` 在 host／port／username／**scheme** 任一变化时清除 `detected_scheme`，避免历史识别
    压过新的显式选择。
  - `testTimeout` 5 秒 → 10 秒，覆盖两种协议各一次握手。
- 测试：`fakeProxy` 增加 `socks5-silent` 模式（接受 TCP、不回应错误协议），新增
  `TestTransportConfirmsConfiguredSchemeBeforeUse`、`TestTransportKeepsConfirmedConfiguredScheme`、
  `TestSaveClearsDetectionWhenSchemeChanges`。

## Files Changed

- `internal/core/netproxy/detect.go`
- `internal/core/netproxy/transport.go`
- `internal/core/netproxy/transport_test.go`
- `internal/core/netproxy/proxy_test_helper_test.go`
- `internal/application/proxysettings/service.go`
- `internal/application/proxysettings/service_test.go`
- `docs/product/NETWORK_PROXY.md`
- `docs/plans/CURRENT.md`

## Technical Notes

- 确认只在「运行期缓存与 `detected_scheme` 都没有」时发生，即每个配置版本一次；之后走内存缓存
  或持久化识别结果。本地 mixed 端口多一次握手往返（毫秒级），远端错误配置最多多花一次握手预算。
- 不在请求失败后再纠正：沉默端口的 `CONNECT` 会挂到调用方超时（网关侧可达 60 秒以上），
  期间没有任何协议类错误可供纠偏。
- 显式协议确认通过时不回写 `detected_scheme`，避免把用户的显式选择固化成「探测结果」；
  只有真正发生切换才回写。

---

# Verification

## Tests

- [x] Unit Tests
- [x] Integration Tests（`go test . ./internal/...` 全量）
- [ ] E2E Tests（无独立 E2E 套件；本任务用真实远端代理做了手工端到端验证）

## Quality Checks

- [x] Format
- [x] Lint
- [x] Type Check
- [x] Test
- [x] Build
- [x] Verify

## Verification Result

```text
task verify
EXIT=0
format:check PASS（gofmt 无输出 / prettier 全部通过）
lint PASS（go vet + staticcheck；前端 eslint 0 error，2 个既有 react-refresh warning）
typecheck PASS（tsc --noEmit）
test PASS（Go 全部包；前端 164 files / 1006 tests）
build PASS（wails production + bin/omniproxy.exe）
```

第一次 `task verify` 有 2 项前端用例超时（`accounts-view.test.tsx` 5 秒超时、
`agent-detail-workspace.test.tsx` 找不到 `已接入模型` 标题）。两文件单独重跑 19/19 通过，
且本次未改动任何前端代码，判为整机负载下的既有偶发；第二次整门 EXIT=0。

真实代理（`9.142.31.120:5278`，用户名密码认证）端到端实测，修复后：

```text
scheme=auto   elapsed=6232ms status=200 body="9.142.31.120" detections=[socks5]
scheme=http   elapsed=6931ms status=200 body="9.142.31.120" detections=[socks5]
scheme=socks5 elapsed=5535ms status=200 body="9.142.31.120" detections=[]
```

修复前同一代码路径：`scheme=http` 20 秒超时（`context deadline exceeded`，`CodeOf` 为空）、
`scheme=auto` 因 2 秒探测预算偶发 20 秒超时。

---

# Documentation

- [ ] PRD
- [ ] Architecture
- [ ] Tech Stack
- [ ] Coding Rules
- [ ] Testing
- [ ] Build
- [ ] ADR（ADR-025 的细节完善，不需要新 ADR）
- [x] CURRENT

更新 `docs/product/NETWORK_PROXY.md`：显式协议确认规则、握手预算 4 秒、协议变化清除识别结果、
远端带认证代理属可用端点。

---

# Deviations

1. 原计划「两种协议都不确认成功时返回 `proxy_protocol_mismatch`」，实际改为保留用户配置：
   瞬时抖动不应升级成硬错误，且行为与修改前一致，避免引入新的失败面。
2. 计划外把 `detectAttemptTimeout` 从 2 秒提到 4 秒、`Test` 总预算从 5 秒提到 10 秒：
   实测该远端代理握手 1.9–2.6 秒，2 秒预算会让确认与 `auto` 探测都不可靠。
3. 计划外补充「协议变化清除 `detected_scheme`」：否则历史识别结果会压过用户的新选择，
   使显式切换协议永远不生效。

---

# Git

## Commit

```text
ab16e41c fix(netproxy): 显式协议首次使用前握手确认并在必要时自愈
```

---

# Result

该远端代理可以使用：它是纯 SOCKS5（带认证），`http://` 前缀只是拼写形式。OmniProxy 现在
无论协议选「自动识别」「SOCKS5」还是「HTTP」，首次请求都会先握手确认并在必要时自愈到
SOCKS5 后回写识别结果，不再出现每个请求挂到超时且无错误码的情况。等价的 curl 命令是
`curl -x socks5h://<user>:<pass>@9.142.31.120:5278 https://ipv4.webshare.io/`。
用户侧只需把协议设为「自动识别」或「SOCKS5」，地址填 `9.142.31.120`、端口 `5278`、
用户名与密码填代理账号；已安装的旧版本需要更新构建才包含本修复。

---

# Completed

2026-11-20 19:56
