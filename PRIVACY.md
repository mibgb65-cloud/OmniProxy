# Privacy Policy / 隐私政策

This document describes OmniProxy v2. Legacy v1 behavior is documented on the repository's `master` branch.

本文说明 OmniProxy v2 的数据处理行为；v1 的行为见原仓库 `master` 分支。

## Local data / 本地数据

OmniProxy is local-first. The default Windows production data directory is `%APPDATA%\OmniProxy`;
an existing installation may use an isolated `v2` subdirectory or a user-selected location.
`app.json` retains the data-location pointer and application preferences. Development and test profiles are separate.

OmniProxy 采用本地优先设计，Windows 正式环境默认数据目录为 `%APPDATA%\OmniProxy`；
旧版升级可能使用隔离的 `v2` 子目录，用户也可以选择其他位置。`app.json` 保存目录指针与应用偏好，
开发／测试环境与正式环境分离。

Local data includes provider and account metadata, model and routing settings, quota snapshots,
request-result metadata, Token aggregates, sanitized diagnostic logs, managed client-configuration backups
and update files. Windows credentials are protected with the current user's DPAPI context;
SQLite stores credential references rather than complete credential bundles.

本地数据包括提供商及账号元数据、模型／路由设置、额度快照、请求结果元数据、Token 聚合、
脱敏诊断日志、受管客户端配置备份及更新文件。Windows 凭据由当前使用者的 DPAPI 保护，
SQLite 保存凭据引用而不是完整凭据包。

The gateway forwards request and response content in memory. It does not provide conversation storage.
Managed external-client configurations can contain local gateway access keys; review their files and backups before sharing them.

网关在内存中转发请求与响应内容，不提供会话正文存储。外部客户端的受管配置可能包含本机网关接入 Key，
共享配置文件或备份前请自行检查。

## Network requests / 网络请求

Requests and credentials are sent only to configured providers, authentication services, custom gateways or
user-selected outbound proxies as required by the requested feature. Model, balance and quota queries
contact the relevant provider. Optional quota-activation requests require an explicit opt-in and may consume provider quota.

请求内容与凭据仅在相关功能需要时发送到已配置的提供商、认证服务、自定义网关或用户选择的出站代理。
模型、余额和额度查询会访问相应服务商。可选的额度窗口激活需明确开启，可能消耗提供商额度。

Production update checks contact GitHub Releases; automatic download is a separate opt-in.
Signing uses only public-source build artifacts in GitHub Actions and does not upload local user data to SignPath.

正式版更新检查访问 GitHub Releases，自动下载需单独开启。代码签名仅处理 GitHub Actions 的公开源码构建产物，
不会把本地用户数据上传到 SignPath。

## No maintainer telemetry / 无维护者遥测

OmniProxy does not operate a maintainer-controlled cloud relay, advertising, analytics or crash-reporting endpoint.
The maintainers do not receive prompts, credentials, responses or local usage history through the application.
GitHub, providers and selected proxies may observe normal network metadata such as an IP address.

OmniProxy 不提供由维护者运营的云端中转、广告、统计分析或崩溃上报端点；维护者不会通过应用收到提示词、
凭据、响应或本地用量记录。GitHub、服务商和用户选择的代理仍可能看到 IP 地址等通常的网络元数据。

## User control and contact / 用户控制与联系

Accounts, history-retention settings and client configurations can be managed from the application.
Uninstalling v2 preserves user data. Move data through the application's data-storage settings rather than changing
the installer pointer. Review provider policies and back up needed data before clearing or manually removing it.

用户可在应用内管理账号、历史保留设置与客户端配置。v2 卸载保留用户数据；搬迁已有数据应使用应用内的数据存储设置，
不要通过安装器改指针代替迁移。清理或手动删除本地数据前，请先检查所需备份，并查阅服务商隐私政策。

Privacy questions may use [Issues](https://github.com/mibgb65-cloud/OmniProxy/issues);
report vulnerabilities privately according to [SECURITY.md](SECURITY.md). Never post real credentials or unredacted logs.

隐私问题可通过 [Issue](https://github.com/mibgb65-cloud/OmniProxy/issues) 联系；
安全漏洞请按 [SECURITY.md](SECURITY.md) 私密报告。请勿公开真实凭据或未脱敏日志。
