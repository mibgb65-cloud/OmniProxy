# TASK-20261113-016: 修复本机代理探测回填带协议的主机地址

## Status

In Progress

## Created

2026-11-13 14:05（Asia/Singapore）

## Request

> 网络代理这个功能好像有问题，你看看该怎么修复一下？

附截图：启用代理、协议 SOCKS5、服务器地址显示 `auto://127.0.0.1`、端口 10808；
测试提示「测试未完成，请重试。」，保存提示
「保存失败，请检查填写内容后重试。proxysettings: invalid input: host」。

## Goal

- 点击「探测本机代理」的结果后，服务器地址只回填主机名或 IP，不再包含 `scheme://`，
  保存与测试可以直接通过。
- 服务器地址格式不合法时在输入框下直接说明原因，不再只给出「重试」或原始错误码。

## Background

- `netproxy.Discover` 返回的 `Discovered.Endpoint` 是 `Endpoint.String()`，
  即带协议的展示文本（`auto://127.0.0.1:10808`），同时单独返回 `Port`。
- 前端 `hostFromEndpoint(endpoint, port)` 假设 endpoint 是 `host:port`，
  先去掉 `:port` 后缀，再把结果填进「服务器地址」。
- 于是 `auto://127.0.0.1:10808` 被回填成 `auto://127.0.0.1`；后端 `normalize`
  拒绝含 `/`、`:` 的 host，保存返回 `proxysettings: invalid input: host`，
  测试也因同一校验失败，但界面只显示「测试未完成，请重试。」。
- 现有前端测试的探测结果 mock 用的是 `127.0.0.1:7890`（无协议），
  与后端真实返回不一致，因此没有覆盖到该问题。

## Scope

- `netproxy.Discovered` 增加 `Host` 字段，发现结果直接给出主机，UI 不再反解展示文本。
- 前端改用 `host` 回填，删除 `hostFromEndpoint`；补真实报文形状的回归测试。
- 服务器地址增加与后端一致的轻量校验与就地提示，非法时禁用测试／保存。
- 重新生成 TypeScript 绑定。

## Out of Scope

- 不改代理连接、分流、凭据存储与探测端口列表。
- 不改后端校验规则（仍以服务端为准）。
- 不改弹窗布局与其它提供商设置。

## Existing Behavior

探测结果可点击回填；回填后主机字段含协议前缀，保存／测试都失败，
界面只提示重试或原始 `proxysettings: invalid input: host`。

## Expected Behavior

- 探测结果回填 `127.0.0.1` + `10808` + 识别出的协议，保存成功。
- 手动输入 `auto://127.0.0.1`、含空格或路径时，地址框下提示
  「只填主机名或 IP，不含协议、端口或路径」，测试与保存按钮禁用。
- 探测结果按钮仍显示完整端点（`auto://127.0.0.1:10808 · SOCKS5`）便于识别。

---

# Analysis

## Root Cause

后端把「展示用端点」和「可编辑字段」混在同一个 `Endpoint` 字符串里返回，
前端只能靠字符串裁剪反推主机；一旦展示文本带协议，裁剪结果就是非法 host。

## Related Modules

- `internal/core/netproxy/detect.go`（`Discovered`、`Discover`）
- `internal/core/netproxy/detect_test.go`
- `frontend/src/features/providers/proxy-settings-dialog.tsx`
- `frontend/src/features/providers/proxy-settings-dialog.test.tsx`
- `frontend/src/i18n/locales/{zh,en}.ts`（providers.proxy.*）
- `frontend/bindings/omniproxy/internal/core/netproxy/models.ts`（生成）

## Risks

- `Discovered` 增加字段属于向后兼容的 JSON 扩展，旧前端忽略新字段即可。
- 前端校验必须与后端一致，否则会出现「前端放行、后端拒绝」的反向问题；
  规则保持同一组字符限制，并保留 IPv6。

---

# Implementation Plan

1. 后端 `Discovered` 增加 `Host`，`Discover` 填充；补充断言。
2. 前端 `applyDetected` 使用 `item.host`，删除 `hostFromEndpoint`。
3. 增加 `isValidHost`（IP 或 LDH 主机名，长度 ≤255），非法时显示提示并禁用测试／保存。
4. 更新测试：探测 mock 改成带协议的 `endpoint` + `host`，断言回填结果；
   新增非法地址提示与按钮禁用用例。
5. 重新生成绑定，跑 `task verify`，提交并归档。

---

# Progress

- 2026-11-13 14:05：创建记录，定位为 `Endpoint` 展示文本被当作 `host:port` 反解。
- 2026-11-13 14:10：完成修复与测试，`task verify` exit 0。

---

# Implementation Result

探测结果现在直接带回可编辑的主机字段，前端不再反解带协议的展示文本；
服务器地址非法时在字段下就地说明并禁用测试／保存。

## Changes

- `netproxy.Discovered` 增加 `Host`；`Discover` 用探测时使用的 host 填充，
  `Endpoint` 仍保留 `auto://host:port` 展示文本。
- `proxysettings.Discovered` 同步增加 `Host` 并透传；TypeScript 绑定重新生成。
- 前端删除 `hostFromEndpoint`，`applyDetected` 直接使用 `item.host`；
  探测芯片仍显示完整端点。
- 新增 `isValidHost`（空值按后端的默认主机处理；拒绝协议、路径、空白、超长与非法字符，
  IPv6 通过 URL 解析校验），与后端 `normalize` 的规则保持一致。
- 地址非法时输入框 `aria-invalid`、字段下显示「服务器地址只填主机名或 IP，不含协议、端口或路径。」，
  测试与保存按钮禁用；中英文案同步。
- 测试：`TestDiscoverFindsDefaultPorts` 断言 `Host` 与带协议的 `Endpoint`；
  前端探测用例改用真实形状（`auto://127.0.0.1:7890` + `host`）并断言保存可用；
  新增「含协议地址提示 + IPv6 恢复正常」用例。

## Files Changed

- 后端：`internal/core/netproxy/detect.go`、`detect_test.go`、
  `internal/application/proxysettings/service.go`
- 前端：`proxy-settings-dialog.tsx`、`proxy-settings-dialog.test.tsx`、
  `i18n/locales/{zh,en}.ts`、`bindings/.../proxysettings/models.ts`
- 文档：`docs/product/NETWORK_PROXY.md`、本记录

## Technical Notes

- `Endpoint` 继续用于展示与日志，`Host`/`Port` 用于表单回填，两者不再混用。
- 前端校验只是提前反馈，服务端 `normalize` 仍是唯一权威；规则刻意保持同一组字符限制。
- 空地址保持原有行为：后端回落到 `127.0.0.1`，因此前端视为合法。

---

# Verification

## Tests

- [x] Unit Tests
- [x] Integration Tests
- [ ] E2E Tests

## Quality Checks

- [x] Format
- [x] Lint
- [x] Type Check
- [x] Test
- [x] Build
- [x] Verify

## Verification Result

```text
go test ./internal/core/netproxy ./internal/application/proxysettings ./internal/transport/wails
ok  omniproxy/internal/core/netproxy
ok  omniproxy/internal/application/proxysettings
ok  omniproxy/internal/transport/wails

pnpm vitest run src/features/providers/proxy-settings-dialog.test.tsx
Test Files  1 passed (1)
     Tests  13 passed (13)

task verify
EXIT=0
Test Files  151 passed (151)
     Tests  924 passed (924)
```

- 直接复现路径已由测试覆盖：带协议的探测端点不再产生 `auto://127.0.0.1` 主机，
  后端 `invalid input: host` 不再出现。
- 未在真实 v2rayN 端口上做手工点击验证；用户重启应用后可在同一弹窗复核。

---

# Git

- Commit: TBD
