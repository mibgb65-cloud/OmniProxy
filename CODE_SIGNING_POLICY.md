# Code signing policy / 代码签名策略

## Current status / 当前状态

SignPath approval was not obtained and OmniProxy does not use that service. v2 Windows release installers
use a maintainer-controlled **self-signed code-signing certificate**. This is not a publicly trusted CA certificate,
does not establish a verified legal publisher and does not guarantee removal of SmartScreen warnings.
Historical artifacts must be checked individually; the application executable is not separately signed.

SignPath 申请未获通过，项目不使用该服务。v2 Windows 发布安装包使用维护者控制的**自签代码签名证书**，
不是公开受信任 CA 证书，不代表发布者经过法律身份认证，也不保证消除 SmartScreen 警告。
历史产物须逐个核验；应用程序本体暂不单独签名。

## Certificate and keys / 证书与私钥

The public certificate is [build/windows/codesign.cer](https://github.com/mibgb65-cloud/OmniProxy/blob/main/build/windows/codesign.cer).
Release verification pins the exact certificate's SHA256, not just its subject name.
Only the public certificate is committed. Private keys and passwords must never enter Git, logs, release assets or chat.

公开证书见 [build/windows/codesign.cer](https://github.com/mibgb65-cloud/OmniProxy/blob/main/build/windows/codesign.cer)。
发布校验固定匹配证书的 SHA256，不只比较名称。Git 只保存公开证书，私钥及密码不得进入源码、日志、
Release 资产或聊天。

The project key is retained in the maintainer's current-user Windows certificate store. Actions uses an encrypted PFX
and its password through this repository's encrypted Secrets, scoped to the signing step. Do not use the key for other projects.

项目私钥保存在维护者的 Windows 当前用户证书库；Actions 通过本仓库 encrypted Secrets 使用加密 PFX
及其密码，凭据只传给签名步骤，不用于其他项目。

## Release process / 发布流程

Public assets must originate from the tagged public source and GitHub-hosted Release workflow, pass the quality gate,
version and asset checks, then receive an Authenticode signature using the pinned project certificate.
SHA256 must be regenerated after signing. Signing failure, certificate mismatch or file tampering prevents publication.
Do not replace this process by uploading locally built installers.

公开产物必须来自版本标签对应的公开源码及 GitHub 托管 Release 流程，通过质量门、版本和资产检查后，
使用固定的项目证书进行 Authenticode 签名。签名后重新生成 SHA256；签名失败、证书不匹配或文件篡改
都阻止发布，不得以手工上传本机构建安装包替代。

For integrity verification only, the isolated Actions Runner temporarily trusts the exact public certificate and removes
that trust afterward. This does not make the certificate publicly trusted. The installer only ships the public certificate
and **never imports it into a user's Root or TrustedPublisher store**, disables SmartScreen or changes security settings.

仅为校验完整性，隔离 Actions Runner 临时信任精确匹配的公开证书，验证后清理；这不会赋予证书公开信任。
安装器只分发公开证书，**不导入用户的 Root／TrustedPublisher 证书库**，不关闭 SmartScreen 或修改安全设置。

## Verification and limitations / 验证与边界

Check the embedded certificate and compare it with the versioned public certificate. On a normal machine that has not
explicitly trusted it, `Get-AuthenticodeSignature` may report an untrusted root. That is expected for self-signing.
A SHA256 file detects byte changes but is not proof of publisher identity.

请核对内嵌证书与对应版本的公开证书。在未主动信任该证书的普通电脑上，
`Get-AuthenticodeSignature` 可能报告根不受信任，这是自签名的预期边界。
SHA256 用于检查文件字节变化，不代表发布者身份认证。

If publicly trusted signing is needed later, use a legitimately issued code-signing certificate or trusted signing service.
Self-signing is not a substitute. On key compromise, rotate the project certificate and Secrets and notify users;
any user who manually trusted the old certificate must remove that trust themselves.

未来如需公开受信任签名，应使用正规 CA 代码签名证书或受信任签名服务，自签不能替代。
私钥泄露时应轮换项目证书与 Secrets 并通知用户；曾手工信任旧证书的用户应自行撤销该信任。

OmniProxy remains under the [MIT License](LICENSE). See also [PRIVACY.md](PRIVACY.md) and [SECURITY.md](SECURITY.md).

OmniProxy 继续采用 [MIT 许可证](LICENSE)，另见[隐私政策](PRIVACY.md)及[安全政策](SECURITY.md)。
