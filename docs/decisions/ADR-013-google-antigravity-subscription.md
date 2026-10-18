# ADR-013: Google Antigravity 订阅通道

## Status

Accepted

失败重试条款局部由 [ADR-015](ADR-015-account-failover.md) 取代（Partially superseded）；原跨计费／地区及共享额度限制保持。

## Context

Google AI 订阅与 Gemini API Key 独立计费。用户要求参考 CLIProxyAPI 接入 Google
订阅。参考项目通过 Google OAuth 与 Antigravity 客户端内部 HTTP 协议转发请求，
无需每次启动 Agent；其明文 token 文件与硬编码 OAuth secret 不符合本仓库约束。

## Decision

- 新增独立 Antigravity 适配器，Google API Key 行为保持兼容；不引入整套参考 SDK。
- 首次登录从 PATH 中已安装的官方 `agy` 程序有界读取公开桌面 OAuth 客户端配置；
  只接受已知客户端 ID 与配置指纹匹配的固定格式配置，无法识别时明确失败，不猜测。
  不执行该程序、不读取既有登录、不在源码或日志复制其 secret。
- 本机随机 loopback 端口接收浏览器 OAuth，使用随机 state、PKCE、一次性回调、
  超时与显式取消。每个 OmniProxy 账号独立授权，重新登录必须确认相同 Google 身份。
- 客户端配置与 token bundle 一起经 DPAPI 保存，SQLite 仅保存 credential_ref 和
  白名单身份元数据。刷新使用账号级受控回调及版本比较，退出删除本机授权。
  旧密文位置先登记到新 bundle，清理失败保留引用，刷新／退出／删除重试清理。
- 模型来自同一账号的 `fetchAvailableModels`。首版只声明已实现的 Gemini 文本／
  工具 Chat JSON/SSE；工具签名必须保留，不补造签名、不使用跳过校验标记。
- 不自动购买或消耗付费积分，不跨到普通 Gemini API Key 失败重试；未知额度显示未知。

## Alternatives Considered

- 官方 CLI 的无界面 Agent：引入额外工具执行和会话持久化语义；本机版本尚无
  稳定的结构化输出参数，不适合作为最小网关适配器。
- 完整嵌入 CLIProxyAPI：重复现有网关／调度／存储边界，依赖范围过大。
- 硬编码桌面 OAuth secret：违反仓库安全规则。

## Consequences

首次登录需要兼容的官方 agy；程序更新导致客户端配置变化时明确提示不兼容。
内部端点及套餐实际权限仍需真实账号验证；参考项目存在实现不代表 Google 提供
稳定的通用订阅 API 合约。协议能力必须以契约与真实验证逐步扩大。

## Sources

- https://github.com/router-for-me/CLIProxyAPI/tree/e7003d2a3e2cee2d45f9950c067bf8e02a2ac0f8
- https://antigravity.google/docs/plans
