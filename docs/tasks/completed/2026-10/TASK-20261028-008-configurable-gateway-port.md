# TASK-20261028-008: Configurable Gateway Port

## Status

Completed

## Created

2026-10-28 21:52

## Request

> 目前的这个项目的端口是固定了3000，但是有用户反馈，这个端口可能会和某些服务的端口冲突，他希望能够提供更改这个端口的功能，你看看该怎么实现呢，如果有冲突需要避让这个冲突端口吧

用户希望：

1. 本地网关端口不再固定为 3000，可以自行更改。
2. 端口与其它服务冲突时自动避让，而不是只报错。

## Goal

- 本地网关监听端口可由用户持久化配置（应用内设置），环境变量仍然可以覆盖。
- 端口被占用时自动挑选同区间的可用端口并继续工作，不再只能手动重试。
- 已受管的本地 Agent 配置随端口变化更新地址，未受管／被外部改动的配置保持不动并明确告知需要重新配置。

## Background

- 端口此前来自 `internal/platform/profile` 的 profile 常量，只允许
  `OMNIPROXY_PROXY_PORT` 覆盖，普通用户没有界面入口。
- 端口没有落盘偏好，冲突时 `proxy.Server.Start` 返回 `port_unavailable`，
  `startGatewayOnStartup` 只记录日志，用户必须自己处理占用后手动重试。
- `docs/product/HARNESS_OPENCODE.md`、`docs/product/KILO_QWEN_MIMO_CODE.md`
  已经承诺「端口改变时仅刷新已受管配置」，但此前没有任何端口变更入口，
  该承诺没有实际触发点。

## Scope

本任务包含：

- `profile.Config` 新增持久化字段 `proxyPort`（0 = 跟随 profile 默认），
  `Detect` 按 `OMNIPROXY_PROXY_PORT` > `app.json` > profile 默认 的顺序解析。
- 新增 `internal/core/netport`：loopback 端口可用性探测与同区间可用端口挑选。
- `proxy.Server` / `gateway.Service` 支持运行期切换端口（停止 → 换端口 → 按原状态重启）。
- `agentconfig.Service` 支持更新端口，并新增 `RebasePort`：只改写
  `agent_backups.applied_hash` 仍与磁盘一致的受管配置里的 `127.0.0.1:<旧端口>`。
- 桌面绑定：`ProxyPortSettings()` 查询 + `SetProxyPort(port)` 应用
  （校验、占用时自动避让、落盘、重启监听器、刷新受管客户端）。
- 启动自动启动网关时端口占用自动避让，取代原来只报错的行为。
- 设置页新增「本地网关端口」卡片（当前端口、默认端口、可用性、保存、需要重新配置的客户端）。
- 文档：PRD / GATEWAY / BUILD / CURRENT。

## Out of Scope

- 不改变控制接口端口（3890/3891/3892）的配置方式。
- 不引入端口区间选择、SOCKS 监听、非 loopback 监听。
- 不改动客户端接入 Key、模型目录与协议行为。
- 不为「端口被其它进程占用」提供进程识别或自动结束进程。

## Existing Behavior

- 生效端口在启动时确定，运行期不可变；冲突时网关启动失败，错误码 `port_unavailable`。
- 启动自动启动失败只写日志，用户需手动处理后再点「启动」。
- 已写入客户端的地址不会随端口变化更新。

## Expected Behavior

- 设置页可以读出当前端口、默认端口、是否可用、是否被环境变量锁定。
- 保存新端口；若目标端口被占用，自动改用同区间内第一个可用端口并在界面说明。
- 保存后：监听器在新端口重启（原运行状态保持）、`app.json` 持久化、受管客户端配置改写到新端口。
- 启动自动启动时若端口被占用，同样自动避让并持久化，日志记录原端口。
- `OMNIPROXY_PROXY_PORT` 生效时设置页只读，避免环境变量与界面互相覆盖。

---

# Analysis

## Related Modules

- `internal/platform/profile`（端口解析与持久化）
- `internal/core/netport`（新增）
- `internal/core/proxy`（`Server.SetPort`）
- `internal/application/gateway`（`Service.SetPort`）
- `internal/application/agentconfig`（`SetPort`、`RebasePort`）
- `internal/transport/wails`（`ProxyPortSettings` / `SetProxyPort`、启动避让）
- `frontend/src/features/settings`、`frontend/src/services/omniproxy.ts`、i18n

## Dependencies

`net`、`encoding/json` 标准库；无新增第三方依赖。

## Risks

- 端口变化会改写用户客户端配置文件：只改写 `applied_hash` 与磁盘一致的受管文件，
  其它情况保留原文件并报告「需要重新配置」，避免覆盖外部改动。
- 自动避让会持久化新端口：与客户端地址改写必须成对执行，否则会出现
  客户端指向旧端口、网关监听新端口的错配。
- 运行期切换端口涉及停止/启动监听器：复用现有 `Stop` 语义（先取消在途请求）。

---

# Implementation Plan

1. `profile`：`Config.ProxyPort`、`DefaultProxyPort`、`ProxyPortEnvOverride`、
   `SetProxyPort`、`Detect` 解析顺序 + 测试。
2. `internal/core/netport`：`Available` / `NextAvailable` + 测试。
3. `proxy.Server.SetPort`、`gateway.Service.SetPort` + 测试。
4. `agentconfig`：`SetPort`、`RebasePort`（含 harness／bundle 客户端内部字节） + 测试。
5. `transport/wails`：`proxy_port.go` 绑定与启动避让 + 测试（含更新原端口冲突测试）。
6. 前端：服务函数、hook、设置页卡片、i18n、组件测试；重新生成绑定。
7. 文档与 Task Record 回填，运行 `task verify`。

---

# Progress

- 已定位端口的三处真相：profile 常量／`OMNIPROXY_PROXY_PORT`；`proxy.Options.Port`
  在 `gateway.New` 时固化；`agentconfig.Service.port` 决定写入客户端的地址。
- 已确认 `docs/product` 中「端口改变时仅刷新已受管配置」的承诺缺少触发点，
  本任务补齐。

---

# Implementation Result

## Changes

实际完成：

- `profile`：新增持久化字段 `proxyPort`（0 = 跟随 profile 默认）、`DefaultProxyPort`、
  `ProxyPortEnvOverride`、`SetProxyPort`；`Detect` 端口解析顺序改为
  `OMNIPROXY_PROXY_PORT` > `app.json` > profile 默认，非法存量值回落默认而不是阻断启动。
- 新增 `internal/core/netport`：`Available`（能否在 127.0.0.1 监听）、
  `NextAvailable`（同区间首个可用端口）、`ErrNoAvailablePort`。
- `proxy.Server.SetPort`：运行中先 `Stop`（取消在途请求）再按新端口启动，未运行只改端口；
  非法端口返回 `invalid_configuration`，占用仍由 `Start` 报 `port_unavailable`。
- `gateway.Service.SetPort`、`agentconfig.Service.SetPort`：把新端口交给监听器与客户端配置构建。
- `agentconfig.Service.RebasePort`：只改写 `agent_backups.applied_hash` 与磁盘一致的受管配置中的
  `127.0.0.1:<旧端口>` / `localhost:<旧端口>`；harness 与 pi/goose 的封装备份先解码内部字节再改写；
  外部改动过的配置保留原文件并列入 `Manual`，备份表不可用时整体中止。
- 桌面绑定 `ProxyPort()` / `SetProxyPort(port)`：查询生效端口、默认端口、可用性、运行状态与
  环境变量锁定；保存时先挑端口（占用则同区间避让 20 个端口）→ 停止监听 → 落盘 → 换端口 →
  改写受管客户端 → 恢复监听；落盘失败保持原端口继续服务，环境变量锁定时拒绝修改。
- 启动自动启动遇到 `port_unavailable` 时改为自动避让并持久化，随后重试启动；环境变量锁定时
  仍按原样上报冲突。
- 设置页新增「本地网关端口」卡片：当前／默认端口、可用性、运行状态、保存／恢复默认、
  避让说明、需要重新配置的客户端列表、环境变量锁定时只读；保存后刷新应用信息、网关与 Agent 缓存。
- 文档：PRD、GATEWAY、BUILD、OVERVIEW、CURRENT。

## Files Changed

主要修改文件：

- `internal/platform/profile/profile.go`、`config.go`、`desktop.go`、`profile_test.go`
- `internal/core/netport/netport.go`（新增）、`netport_test.go`
- `internal/core/proxy/server.go`、`server_port_test.go`
- `internal/application/gateway/service.go`
- `internal/application/agentconfig/port_rebase.go`（新增）、`port_rebase_test.go`
- `internal/transport/wails/proxy_port.go`（新增）、`proxy_port_test.go`
- `internal/transport/wails/gateway_startup.go`、`gateway_startup_test.go`
- `frontend/src/features/settings/proxy-port-setting.tsx`（新增）、`use-proxy-port.ts`（新增）、
  `proxy-port-setting.test.tsx`（新增）、`settings-content.tsx`
- `frontend/src/services/omniproxy.ts`、`frontend/src/i18n/locales/{zh,en}.ts`
- `frontend/bindings/omniproxy/internal/transport/wails/*`（`wails3 generate bindings` 重新生成）

## Technical Notes

- 端口的三处真相（profile 常量／持久化、`proxy.Options.Port`、`agentconfig.port`）现在由
  `applyProxyPort` 一处按固定顺序同步，避免出现「网关在新端口、客户端仍指向旧端口」。
- 受管配置改写用「整段文本替换 + 版本摘要校验」，不做 JSON/TOML 重排，保留注释、其它提供商与
  密钥引用；端口按完整数字匹配，3000 不会误伤 30000。
- 自动避让只扩大 20 个候选端口；找不到可用端口时返回错误并保持原状态。
- 端口未变化且网关正在运行时直接返回，不打断在途请求。

---

# Verification

## Tests

- [x] Unit Tests
- [x] Integration Tests（wails 绑定层 + 真实 SQLite／临时 profile 配置目录）
- [ ] E2E Tests：不适用（无 GUI E2E 框架）

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
exit 0

format:check  PASS（gofmt + prettier）
lint          PASS（go vet、staticcheck、eslint）
typecheck     PASS（tsc --noEmit）
test          PASS（Go 全部包；前端 139 文件／849 项测试）
build         PASS（wails3 build，Windows amd64 production，bin/omniproxy.exe 23,212,032 bytes）
```

补充定向验证：

- `go test ./internal/core/netport/`、`./internal/core/proxy/`、
  `./internal/application/agentconfig/`、`./internal/transport/wails/`、
  `./internal/platform/profile/` 全部通过（新增端口探测、切换、改写与绑定用例）。
- 真实回环端口占用场景由测试覆盖：占用端口后 `SetProxyPort` 自动避让并持久化；
  启动自动启动冲突时避让到可用端口；`OMNIPROXY_PROXY_PORT` 锁定时既不避让也不落盘。
- 未做原生窗口视觉／真实客户端联调：设置卡片与端口改写后的客户端地址待人工验收。

---

# Documentation

- [x] PRD
- [x] Architecture（`OVERVIEW.md` 运行 profile 端口解析）
- [ ] Tech Stack：不需要
- [ ] Coding Rules：不需要
- [ ] Testing：不需要
- [x] Build（`BUILD.md` 端口列改为「默认值」并补充解析顺序）
- [ ] ADR：不需要（没有新的长期架构选择，端口解析沿用 profile 现有规则）
- [x] CURRENT

---

# Deviations

原计划：在设置页保存端口时只重启监听器。

实际方案：保存流程固定为「挑端口 → 停止监听 → 落盘 → 换端口 → 改写受管客户端 → 恢复监听」，
并新增 `agentconfig.RebasePort`。原因：`docs/product/HARNESS_OPENCODE.md`、
`docs/product/KILO_QWEN_MIMO_CODE.md` 已承诺「端口改变时仅刷新已受管配置」，
若只换监听端口会让已接入的本地客户端静默失效。

原计划：启动避让同样处理 env 锁定场景。实际：`OMNIPROXY_PROXY_PORT` 生效时既不避让也不落盘，
冲突照常上报，避免环境变量与持久化偏好互相覆盖。

---

# Git

## Commit

```text
4f09490a feat(gateway): make the local proxy port configurable and avoid conflicts
```

未推送、未打标签、未重打安装包。commit 字段由后续 docs 提交回填。

---

# Result

本地网关端口不再固定：设置页「通用设置 → 本地网关端口」可更改端口并立即生效
（`OMNIPROXY_PROXY_PORT` 仍优先且此时界面只读），端口被占用时自动避让到同区间可用端口，
启动自动启动同样避让；端口变化后已受管客户端地址同步改写，外部改动过的配置保持原样并提示
重新配置。`task verify` exit 0（前端 139 文件／849 项测试、Windows production 构建通过），
真实窗口与客户端联调待人工验收。

---

# Completed

2026-10-28 22:35

