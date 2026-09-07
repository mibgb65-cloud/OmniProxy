# Privacy Policy / 隐私政策

Effective date / 生效日期：2026-09-07

OmniProxy is a local-first desktop gateway. This policy describes the data handled by the application itself. Services selected by the user process data under their own terms and privacy policies.

OmniProxy 是一个本地优先的桌面网关。本政策说明应用本身处理的数据。用户选择的外部服务会依据其各自的条款和隐私政策处理数据。

## Local data / 本地数据

OmniProxy may store the following data on the user's device:

- provider API keys and OAuth credentials, including access and refresh tokens;
- account labels, provider settings, routing rules and client-configuration backups;
- request metadata such as time, provider, model, status, duration, token counts and sanitized error messages;
- quota, balance and local billing summaries;
- OpenRouter chat conversations created in the built-in chat page;
- application preferences, update state and diagnostic logs.

OmniProxy 可能在用户设备上保存以下数据：

- 服务商 API Key 和 OAuth 凭据，包括访问令牌与刷新令牌；
- 账号标签、服务商设置、路由规则和客户端配置备份；
- 请求时间、服务商、模型、状态、耗时、Token 数量和经过脱敏的错误信息等请求元数据；
- 额度、余额和本地账单汇总；
- 内置 OpenRouter 对话页面创建的聊天记录；
- 应用偏好、更新状态和诊断日志。

On Windows, stored account credentials are protected with the current user's DPAPI context. On macOS, OmniProxy stores a master key in Keychain and uses it to encrypt account credentials. Explicit exports and backups can contain readable credentials and must be protected by the user.

Windows 上保存的账号凭据受当前用户的 DPAPI 上下文保护；macOS 上由 Keychain 保存主密钥并用于加密账号凭据。用户主动导出的账号文件和备份可能包含可读取的凭据，需要由用户自行妥善保管。

The gateway processes request and response bodies in memory so it can proxy and, where required, translate protocols. General request history does not intentionally persist prompt or response bodies. The built-in OpenRouter chat stores its conversation content locally in the desktop WebView until the user clears it or removes application data.

网关会在内存中处理请求和响应正文，以完成转发及必要的协议转换。常规请求历史不会主动持久化提示词或响应正文；内置 OpenRouter 对话会把会话内容保存在桌面 WebView 的本地存储中，直到用户清除会话或删除应用数据。

## Network communications / 网络通信

OmniProxy transfers information only to destinations needed for a feature the user has configured, enabled or invoked:

- API requests, request content and the required authentication data are sent to the provider or custom gateway selected by the user;
- OAuth login and token refresh communicate with the relevant identity provider;
- quota, balance, model-catalog and credential-validation actions communicate with the relevant provider endpoints;
- release builds periodically query GitHub Releases for updates and may download the installer and checksum from GitHub when an update is available;
- optional task automation can open a user-selected application or website. Browser-profile discovery reads local profile metadata needed to list and launch a profile; OmniProxy does not copy browser cookies or passwords;
- a user-configured outbound proxy can observe traffic routed through it.

OmniProxy 只会把信息发送到用户已经配置、启用或主动调用某项功能所需的目标：

- API 请求、请求内容及必要的鉴权数据会发送到用户选择的服务商或自定义网关；
- OAuth 登录和令牌刷新会与相应的身份服务通信；
- 额度、余额、模型列表和凭据验证会访问相应服务商的接口；
- 正式版会定期查询 GitHub Releases，并可能在发现新版本后从 GitHub 下载安装包和校验文件；
- 可选的任务自动化功能可以打开用户选择的应用或网站。浏览器资料检测只读取列出和启动 Profile 所需的本地元数据；OmniProxy 不复制浏览器 Cookie 或密码；
- 用户配置的出站代理能够观察经由该代理发送的流量。

Custom gateways and provider endpoints are controlled by their operators, not by the OmniProxy maintainers. Users should only configure services and proxies they trust.

自定义网关和服务商接口由其运营者控制，不受 OmniProxy 维护者控制。用户应只配置自己信任的服务和代理。

## No maintainer telemetry / 不收集维护者遥测

OmniProxy does not operate a maintainer-controlled cloud relay, analytics service, advertising system or crash-reporting endpoint. The maintainers do not receive credentials, prompts, responses, local history or chat conversations through the application.

OmniProxy 不运行由维护者控制的云端中转、统计分析、广告或崩溃上报服务。维护者不会通过应用收到用户凭据、提示词、响应、本地请求历史或聊天记录。

Standard network metadata such as an IP address may still be visible to GitHub, an AI provider, a custom gateway or an outbound proxy when the application contacts that service.

应用访问 GitHub、AI 服务商、自定义网关或出站代理时，这些服务仍可能看到 IP 地址等标准网络元数据。

## User controls and retention / 用户控制与保留期限

- Accounts and credentials can be edited or removed in Account Management.
- Request history and billing data can be cleared in the application.
- OpenRouter chat conversations can be cleared from the chat page.
- Client configuration changes have matching restore actions where supported.
- The Windows uninstaller asks whether local OmniProxy data should be retained or removed.
- Local files may also be removed manually after OmniProxy has exited. The default production data directory is `~/.omniproxy`.

- 用户可以在账号管理中编辑或删除账号与凭据。
- 请求历史和账单数据可以在应用内清除。
- OpenRouter 聊天记录可以在对话页面清除。
- 支持一键配置的客户端同时提供相应的恢复操作。
- Windows 卸载程序会询问保留还是删除 OmniProxy 本地数据。
- 退出 OmniProxy 后也可以手动删除本地文件；正式版默认数据目录为 `~/.omniproxy`。

## Third-party services / 第三方服务

Depending on the user's configuration, data may be processed by OpenAI, Anthropic, Google, DeepSeek, Kimi, Xiaomi MiMo, Zhipu, MiniMax, OpenRouter, TokenRouter, sub2api, new-api, AnyRouter, Zo Computer, Prem or a custom gateway. The current provider list is documented in the [support matrix](README_EN.md#support-matrix). Important external policies include:

根据用户配置，数据可能由 OpenAI、Anthropic、Google、DeepSeek、Kimi、Xiaomi MiMo、智谱、MiniMax、OpenRouter、TokenRouter、sub2api、new-api、AnyRouter、Zo Computer、Prem 或自定义网关处理。当前服务商列表见[支持矩阵](README.md#支持矩阵)。部分重要外部政策包括：

- [GitHub Privacy Statement](https://docs.github.com/en/site-policy/privacy-policies/github-general-privacy-statement)
- [OpenAI Privacy Policy](https://openai.com/policies/privacy-policy/)
- [Anthropic Privacy Policy](https://www.anthropic.com/legal/privacy)
- [Google Privacy Policy](https://policies.google.com/privacy)
- [OpenRouter Privacy Policy](https://openrouter.ai/privacy)

Users are responsible for reviewing the current terms and privacy policy of every provider or custom endpoint they choose.

用户有责任查阅并遵守所选择服务商或自定义接口的最新条款和隐私政策。

## Contact / 联系方式

Privacy questions can be submitted through the [OmniProxy issue tracker](https://github.com/mibgb65-cloud/OmniProxy/issues). Suspected vulnerabilities must be reported privately according to the [security policy](SECURITY.md). Issues are public by default; do not include API keys, OAuth tokens, full authorization headers, exported account files or unredacted logs.

隐私问题可以通过 [OmniProxy Issue](https://github.com/mibgb65-cloud/OmniProxy/issues) 提交；疑似安全漏洞必须按照[安全政策](SECURITY.md)私密报告。Issue 默认公开，请勿附上 API Key、OAuth Token、完整鉴权请求头、导出的账号文件或未经脱敏的日志。
