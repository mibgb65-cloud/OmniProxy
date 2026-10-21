# TASK-20261021-001: Google 订阅错误分类排查与修复

## Status

Completed

## Created

2026-10-21（Asia/Singapore）

## Request

用户在客户端选择 Google 模型后收到 “The configured API key was rejected by the inference provider”，询问原因和修复方式。

## Goal

依据本地只读请求元数据解释实际失败；修复已确认的 Google 订阅 HTTP 400 被误报为认证失败的问题，不将错误分类修复宣称为真实账号恢复。

## Scope

- Google / Antigravity 上游错误到安全错误码、网关 HTTP 状态的映射。
- OAuth 令牌端点与普通推理／控制端点的错误边界。
- 适配器与网关回归测试、相关产品说明。

## Out of Scope

- 不读取或输出真实凭据，不修改生产数据库、客户端配置或已安装应用。
- 不重放真实消息，不发送付费推理，不自动重新授权、安装或发布。
- 不在缺少上游拒绝证据时放宽 Google 429 自动换号规则。

# Analysis

## Evidence

- 只读生产请求元数据：2026-10-21 09:26:17（新加坡时间），`gemini-3.8-flash-high` / Messages 成功；09:26:19、09:26:20 为 `401 / reauth_required`；09:26:22 至 09:26:43 连续 `429 / rate_limited`。
- 生产 Google 账号使用 `subscription_login`，不是 API Key。
- `internal/provider/antigravity/client.go` 将所有上游 HTTP 400、401 都映射为 `reauth_required`，随后 `ProxyError` 输出 HTTP 401。非令牌端点的无效请求因此会被误报为需要重新认证。
- 历史未保存原始上游状态／正文，无法确认这两次 401 是否源自 HTTP 400，也无法仅凭 429 区分短时限流与额度耗尽。
- 只读、按当前授权版本和身份摘要核对的额度快照最后更新于 2026-10-21 00:42（新加坡时间）；Gemini 分组当时显示 100%，不能据此推断约九小时后失败请求的实时资源状态。

## Related Modules

- `internal/provider/antigravity`
- `internal/application/gateway`
- `docs/product/GOOGLE_SUBSCRIPTION.md`

## Risks

- 必须保留 OAuth 令牌端点 HTTP 400 的重新授权语义，避免破坏过期／撤销令牌的恢复提示。
- 不暴露上游错误正文、令牌或请求内容。
- 不更改真实 401／403／429 的分类，不引入额外重试。

# Implementation Plan

1. 添加回归测试，验证普通端点 400 不再输出 401，令牌端点 400 和真实 401 保持原行为。
2. 最小修改固定端点的状态映射，沿用稳定错误码与脱敏。
3. 验证 Messages 网关错误及请求记账，执行定向测试和 `task verify`。
4. 更新实际结果和产品排查说明，审查差异，仅提交本任务文件。

# Implementation Result

## Changes

- 非 OAuth 令牌端点的上游 HTTP 400 归为 `400 / invalid_request`；令牌端点 400 及真实 401 仍为 `401 / reauth_required`，403／429 不变。
- 所有错误继续只返回固定安全码，不解析或暴露上游正文，不新增重试。
- 新增 12 组固定端点／状态回归，扩展真实回环网关测试，覆盖 Chat／Responses／Messages 的 200、400、401、403、429、记账归因和不重试。
- 补充产品排查说明，区分 API Key、订阅 OAuth、无效请求与限流。

## Remaining Limitations

- 已确认并修复的是错误映射缺陷，不证明历史两次 401 一定是误分类。
- 未保存原始失败请求及上游拒绝原因，不推断实际参数错误；未验证真实 Google 推理恢复。
- 未修改正在运行的 Beta.15，用户需使用含本修复的新构建后才能生效。

# Verification

- 修复前定向回归：普通端点五组 400 被误报为重新授权，网关 400 返回 401，测试按预期失败，证实缺陷。
- 修复后定向回归：
  `go test ./internal/provider/antigravity ./internal/application/gateway -run 'TestRequestErrorClassification|TestGoogleSubscriptionRoutesWithoutAPIKeyAndRecordsUsage' -count=1` 通过。
- `task verify`：格式、lint、typecheck、Go 测试及 118 个文件／703 项前端测试通过；构建阶段进入 sources 扫描后长时间无子进程或输出，核对进程身份后仅终止本次 Task 进程。不是代码测试失败。
- `task --force verify`：PASS（退出码 0）。格式、lint、typecheck、Go 测试、118 个文件／703 项前端测试及 Windows amd64 production 构建全部通过。只跳过增量缓存判定，不跳过质量检查，不修改 Taskfile。
- 最终测试错误处理小幅完善后补充 `go vet`、`staticcheck`（上述两个 Go 包）及 `git diff --check`，均通过。
- 本任务代码、测试、产品说明与任务记录通过 UTF-8 无 BOM 检查。
- 产物：`bin/omniproxy.exe`，22,277,632 bytes；未执行此产物、安装、替换现有应用或发送真实推理请求。
- 非失败警告：既有 jsdom `scrollTo` 未实现提示、前端产物分块大小提示；未为消除警告引入无关修改。

# Documentation

- 已更新 `docs/product/GOOGLE_SUBSCRIPTION.md`、`docs/plans/CURRENT.md`。
- 无依赖、迁移或架构边界变化，不新增 ADR。

# Git

## Commit

`f214c797513be48b241da1066191410974f978a5 fix(google): 避免将无效请求误报为认证失败`

# Result

已完成真实请求元数据排查、错误分类修复、三协议回归、完整质量门和 Windows production 构建。保留真实认证错误和限流的语义，不读取或输出凭据，不修改生产账号、客户端配置或已安装应用。

修复仅解决已证实的误分类问题；历史两次 401 的真实上游原因、后续 429 的具体限制来源，以及真实账号推理恢复仍未确认，需运行含修复的版本后进一步核对。

# Completed

2026-10-21（Asia/Singapore）
