# Security Policy / 安全政策

## Supported versions / 支持版本

Security fixes are provided for the latest stable release. Users should upgrade to the newest stable version before reporting an issue that may already have been fixed.

安全修复面向最新稳定版本提供。若问题可能已经修复，请先升级到最新稳定版本再进行报告。

## Reporting a vulnerability / 报告安全漏洞

Please report suspected vulnerabilities privately through [GitHub Security Advisories](https://github.com/mibgb65-cloud/OmniProxy/security/advisories/new).

如发现疑似安全漏洞，请通过 [GitHub Security Advisories](https://github.com/mibgb65-cloud/OmniProxy/security/advisories/new) 私密报告。

Include the affected version, operating system, impact, reproduction steps and a minimal proof of concept when available. Do not include real API keys, OAuth tokens, exported account files, full authorization headers or unredacted logs. Use obvious fake credentials in reproductions.

请尽量提供受影响版本、操作系统、影响范围、复现步骤和最小化验证样例。不要提交真实 API Key、OAuth Token、导出的账号文件、完整鉴权请求头或未经脱敏的日志；复现材料应使用明显的假凭据。

Please do not open a public Issue for an unpatched vulnerability. General hardening suggestions that do not disclose an exploitable issue may use the public issue tracker.

尚未修复的漏洞请勿通过公开 Issue 报告；不包含可利用漏洞细节的一般加固建议可以使用公开 Issue。

## Scope / 范围

Security reports may cover the desktop application, loopback proxy and control API, credential storage, client-configuration writers, update mechanism, release workflow and published artifacts. Vulnerabilities in an upstream AI provider or an unrelated third-party service should be reported to that service instead.

安全报告可以涉及桌面应用、回环代理与控制 API、凭据存储、客户端配置写入、更新机制、发布工作流和公开产物。AI 服务商或无关第三方服务自身的漏洞应直接报告给相应服务方。

## Coordinated disclosure / 协调披露

Please allow reasonable time to investigate and publish a fix before public disclosure. The maintainer may request additional information, provide a remediation build or coordinate a disclosure date according to the severity and affected release scope.

公开披露前，请为调查和发布修复预留合理时间。维护者可能会要求补充信息、提供修复版本，并根据严重程度和受影响版本范围协调披露时间。
