# Code signing policy / 代码签名策略

Free code signing provided by [SignPath.io](https://signpath.io/), certificate by [SignPath Foundation](https://signpath.org/).

OmniProxy 的免费代码签名服务由 [SignPath.io](https://signpath.io/) 提供，证书由 [SignPath Foundation](https://signpath.org/) 提供。

## Scope / 适用范围

This policy applies only to OmniProxy artifacts built from the public source repository at <https://github.com/mibgb65-cloud/OmniProxy>.

本策略只适用于由公开源码仓库 <https://github.com/mibgb65-cloud/OmniProxy> 构建的 OmniProxy 产物。

OmniProxy is released under the OSI-approved [MIT License](LICENSE) and is not commercially dual-licensed. Third-party components remain under their respective open-source licenses.

OmniProxy 使用 OSI 批准的 [MIT License](LICENSE) 发布，不采用商业双重许可；第三方组件继续适用其各自的开源许可证。

Eligible Windows artifacts are:

- the project-owned `OmniProxy.exe` application binary when it is submitted by the release workflow;
- the project-owned NSIS installer named `OmniProxy-Setup-<tag>-windows-amd64.exe`.

允许签名的 Windows 产物包括：

- 由发布工作流提交的项目自有应用程序 `OmniProxy.exe`；
- 项目自有的 NSIS 安装包 `OmniProxy-Setup-<tag>-windows-amd64.exe`。

OmniProxy signing credentials must not be used for unrelated projects, locally built files or third-party executables. Third-party runtime installers and standalone libraries must not be signed with the OmniProxy signing policy.

OmniProxy 的签名权限不得用于无关项目、本机构建文件或第三方可执行文件。第三方运行库安装程序和独立库不得使用 OmniProxy 的签名策略进行签名。

## Trusted build and release process / 受信任构建与发布流程

Release artifacts must:

1. originate from this repository and the revision identified by a version tag;
2. be built by `.github/workflows/release.yml` on GitHub-hosted runners;
3. install dependencies from the lock files in the repository and pass the workflow's backend, frontend and source checks;
4. carry `OmniProxy` product metadata and the numeric version derived from the release tag;
5. be uploaded as a GitHub Actions artifact before any SignPath signing request;
6. receive manual approval from an authorized approver for every release-signing request;
7. be published only after the signed artifact is downloaded and its SHA-256 checksum is regenerated.

发布产物必须：

1. 来源于本仓库以及版本标签所标识的修订；
2. 由 `.github/workflows/release.yml` 在 GitHub 托管的 Runner 上构建；
3. 根据仓库锁文件安装依赖，并通过工作流中的后端、前端和源码检查；
4. 包含 `OmniProxy` 产品元数据以及由发布标签派生的数字版本；
5. 在提交 SignPath 签名请求前先上传为 GitHub Actions Artifact；
6. 每个正式签名请求都由授权审批人手工批准；
7. 下载已签名产物并重新生成 SHA-256 校验值后才能公开发布。

The signature embedded in a released artifact is authoritative. An artifact must not be described as SignPath-signed unless its signature has been verified in the release workflow.

公开产物中嵌入的数字签名是判断签名状态的最终依据。只有在发布工作流中验证过签名的产物，才能标注为已由 SignPath 签名。

## Team roles / 团队角色

- Committer and reviewer / 提交者与审查者：[mibgb65-cloud](https://github.com/mibgb65-cloud)
- Signing approver / 签名审批人：[mibgb65-cloud](https://github.com/mibgb65-cloud)

Changes from contributors without direct commit access require review before merge. Changes to build scripts, release workflows, dependency definitions and this policy are security-sensitive and require the same review standard as application code.

没有直接提交权限的贡献者所提交的变更必须经过审查才能合并。构建脚本、发布工作流、依赖定义和本策略的变更属于安全敏感变更，其审查标准与应用代码相同。

Everyone assigned to a signing role must use multi-factor authentication for both GitHub and SignPath. Signing API tokens must be stored only as encrypted GitHub Actions secrets and granted the minimum required permissions.

所有签名角色都必须为 GitHub 和 SignPath 启用多因素认证。签名 API Token 只能保存为 GitHub Actions 加密 Secret，并仅授予所需的最小权限。

## Privacy and user control / 隐私与用户控制

OmniProxy is local-first and does not send credentials, prompts, responses or usage history to infrastructure operated by its maintainers. Network requests are sent to GitHub and to providers, gateways or proxies selected by the user as described in the [privacy policy](PRIVACY.md).

OmniProxy 采用本地优先设计，不会把凭据、提示词、响应或用量历史发送到维护者运营的基础设施。应用会依据[隐私政策](PRIVACY.md)访问 GitHub，以及用户选择的服务商、网关或代理。

Features that modify client configuration, enable autostart, consume a quota reset credit or send a quota-activation request require an explicit user action or opt-in setting. Release builds perform documented scheduled update checks and may download a checksum-protected update from GitHub; matching restore or uninstall paths are provided where applicable.

修改客户端配置、启用开机启动、消耗额度刷新卡或发送额度激活请求等功能，需要用户主动操作或明确启用。正式版会按文档说明定期检查更新，并可能从 GitHub 下载带校验文件保护的更新；适用场景同时提供恢复或卸载路径。

## Verification / 验证

For a signed Windows artifact, users can open **Properties → Digital Signatures** or run:

对于已签名的 Windows 产物，用户可以打开“属性 → 数字签名”，或运行：

```powershell
Get-AuthenticodeSignature -LiteralPath .\OmniProxy-Setup-<tag>-windows-amd64.exe | Format-List
```

The expected signer for artifacts covered by the free OSS program is SignPath Foundation. SHA-256 files provide integrity checks but do not replace signature validation.

免费开源计划所覆盖产物的预期签名者为 SignPath Foundation。SHA-256 文件用于完整性校验，但不能代替数字签名验证。
