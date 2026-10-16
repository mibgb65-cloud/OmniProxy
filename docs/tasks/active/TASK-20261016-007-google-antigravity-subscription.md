# TASK-20261016-007: Google Antigravity 订阅接入验证

## Status

In Progress

## Created

2026-10-16 22:07

## Request

用户询问 Google AI Plus／Pro／Ultra 登录订阅能否接入，要求参考
https://github.com/router-for-me/CLIProxyAPI，并明确同意尝试实现。

## Goal

完成独立于普通 Gemini API Key 的 Google Antigravity 订阅最小链路：受保护授权、
账号模型发现、文本／流式网关转发，并验证多轮工具协议与真实账号可用性的边界。

## Scope

- 检查本机官方 Antigravity 与参考项目的认证协议，选择不硬编码秘密的实现。
- 独立订阅凭证、身份校验、刷新／退出／重新登录及 Google 账号入口。
- 按账号发现模型并接入现有路由、网关与用量记录。
- 关键路径契约测试、前端交互验证、`task verify` 与文档。

## Out of Scope

- 引入 CLIProxyAPI 整套服务或重写现有网关。
- 自动切换到按量付费、购买积分或推断未返回的套餐额度。
- 绕过 Google 登录、地区或账号权限限制。

## Existing Behavior

Google 只允许 API Key；订阅管理及路由目前限定 OpenAI。凭据复用 DPAPI 存储，
数据库仅保存引用。CLIProxyAPI 参考提交为 `e7003d2a3e2cee2d45f9950c067bf8e02a2ac0f8`。
本机存在 `agy 1.0.0`。工作区已有其他任务记录，保持其内容且不纳入本次提交。

## Expected Behavior

Google 账号提供独立订阅入口；授权与模型能力必须来自实际上游，凭据不进入前端、
日志、SQLite 或明文配置。无有效订阅权限时明确失败。

# Analysis

## Related Modules

`application/account`、`application/modelcatalog`、`application/routing`、
`application/gateway`、Google provider、Wails 绑定及账号前端。

## Dependencies

沿用 Go 标准库与现有凭证／UI 组件。Google OAuth 客户端来源及本机官方授权能力待核实。

## Risks

Antigravity 使用客户端内部协议；OAuth 客户端适用性、令牌刷新、工具思考签名及
真实账号权限必须验证。不得复制参考项目硬编码的 OAuth secret 或明文 token 存储。

# Implementation Plan

1. 核实认证与隔离方式，形成最小方案及 ADR，状态改为 In Progress。
2. 实现受保护凭据、授权生命周期与账号入口。
3. 接入实际模型目录、路由、流式请求和使用量。
4. 覆盖安全边界、刷新并发、取消及协议契约，完成可行的真实验证。
5. 运行 `task verify`，同步文档、检查差异并提交本任务变更。

# Progress

- 已建立任务记录并切换到 `codex/google-antigravity-subscription`。
- 本机 `agy 1.0.0` 包含参考实现使用的公开桌面 OAuth 配置；选择运行时有界读取，
  不复制 secret 到源码，不读取用户原有授权。首次只开放 Gemini 文本／工具 Chat，
  保留真实思考签名，其他协议待完整验证再声明支持。

# Implementation Result

## Changes

- 新增 `provider/antigravity`，实现本机公开客户端配置识别、PKCE OAuth、Google 身份／
  项目查询、真实账号模型目录，以及文本／函数工具 Chat JSON／增量 SSE。
- 独立 `application/googlesubscription` 管理一次性 loopback 回调、后台恢复／取消、
  同身份重新登录、刷新互斥、退出及删除。保留 Codex 原有绑定和会话行为。
- Google token bundle 走既有 DPAPI，SQLite 仅保存引用与公开元数据；轮换对目录
  版本同事务重新绑定，新授权使旧目录失效。清理失败位置保留于新密文，后续操作恢复清理。
- 现有模型同步、路由、网关与用量接入订阅账号；固定内部端点，不跨 API Key 重试。
- 前端 Google 浏览器登录入口、状态／重新登录／退出、独立存储说明和未知额度，
  不展示 Codex 重置卡或查询其额度。原 API Key 与其他提供商行为保持兼容。
- 新增 ADR-013、Google 产品说明及 PRD／架构更新，无新增依赖或数据库迁移。

## Technical Notes

本机 agy 内含多个 Google 客户端配置，不能仅按字符串外观选 secret。实现使用公开
客户端 ID 与已知 SHA-256 指纹进行匹配，仅保存指纹到源码。实际配置及 token 不进入仓库。
内部协议兼容 User-Agent 保留 OmniProxy 身份，版本下限来源于固定参考实现。

# Verification

- provider 契约测试：OAuth／PKCE、身份变化、目录过滤、错误脱敏、文本／工具 JSON、
  思考签名两轮、增量 SSE／终态缺失和消费取消通过。
- account 测试：DPAPI 接口隔离、轮换保留目录、新授权目录失效、旧版本冲突及退出重试通过。
- 授权服务测试：本机回调／错误 state、恢复／取消、并发刷新去重与账号隔离通过。
- 新订阅网关 HTTP 测试：真实路由分支、用量归属、429、退出后不可路由通过。
- 相关前端 4 文件／32 项测试通过；本机实际 agy 配置识别通过。
- 最终 `task verify` 完整通过：格式、Go vet／staticcheck、ESLint、TypeScript、全部 Go 测试、
  85 个前端文件／452 项测试及 Windows 生产构建。初次发现的 HTTP 状态常量检查和旧 Google
  单认证方式测试预期已修正，并在最后完整质量门中通过；既有环境的 uname／tail 提示和
  Vite bundle 大小提示不影响成功退出。
- 用户在 Google 页面完成真实独立授权，账号保存到既有开发 profile；同步到 14 个本适配器
  支持的 Gemini 模型。固定该账号经临时本机网关请求 `gemini-2.5-flash`，返回 HTTP 200，
  输入 6／输出 1 Token，数据库记录一致。只发送这一次真实推理，未推断其他模型权限。
- 真实授权探针与原生界面证据位于忽略目录 `tmp/google-subscription-live/`；不会提交
  个人身份、token、授权地址或本机数据库。真实高级模型、长窗口、流式及工具推理未额外消耗额度，
  相应协议已由契约测试覆盖。
- 已人工核对实际开发窗口 `native-current.png`：Google 新账号与订阅类型可见，
  授权弹窗为正确 Google 品牌／浏览器方式／受保护存储说明，布局无溢出。
  UIA 脚本未形成稳定闭环，未将其失败宣称为通过；弹窗交互由前端组件测试覆盖。

# Documentation

已更新 PRD、架构、Google 订阅说明、ChatGPT 说明链接、ADR 与 CURRENT。

# Deviations

首版只开放 Gemini 文本／函数工具的 Chat，而非一次实现三种客户端协议。现有通用
Responses／Messages 转换无法无损保留 Google 工具签名，继续复用会造成多轮失败；
因此明确拒绝其他协议，真实模型能力按账号发现，不引入参考项目的大型转换器依赖。

# Git

待提交。

# Result

已完成独立 Google 订阅最小链路，通过真实登录／14 模型发现／一次 HTTP 200 网关请求、
用量归属、桌面视觉检查及最终完整质量门。真实高级模型、工具／流式及长窗口的权限仍需单独验证。
