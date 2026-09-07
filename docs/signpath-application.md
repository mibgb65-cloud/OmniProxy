# SignPath Foundation application / SignPath Foundation 申请准备

This document is a maintainer checklist and an English application draft. It does not contain credentials and does not enable signing by itself.

本文用于维护者自查并提供英文申请草稿，不包含任何凭据，也不会自行启用签名。

## Before submitting / 提交前

- [x] Public source repository: <https://github.com/mibgb65-cloud/OmniProxy>
- [x] OSI-approved project license: [MIT](../LICENSE)
- [x] Public releases in the format to be signed: <https://github.com/mibgb65-cloud/OmniProxy/releases>
- [x] Documented functionality and installation instructions: [README](../README_EN.md)
- [x] Published [privacy policy](../PRIVACY.md)
- [x] Published [security policy](../SECURITY.md) and private vulnerability-reporting path
- [x] Published [code signing policy](../CODE_SIGNING_POLICY.md)
- [x] GitHub Actions release build from public source
- [x] Product name and numeric release version configured for Windows metadata
- [ ] Push these documents and build changes to the public default branch
- [ ] Enable multi-factor authentication for every GitHub and SignPath account with a signing role
- [x] Audit Go runtime dependencies with `go-licenses` and npm packages from `package-lock.json`; repeat after dependency changes
- [ ] Add a `Code signing policy` link to the current public download/release description, or publish a new release that includes the standard policy footer
- [ ] Submit the [SignPath Foundation application](https://signpath.org/apply.html)

## Suggested application text / 建议申请文本

### Project description

> OmniProxy is a local-first desktop gateway for AI API clients. It lets users configure their own provider credentials, routes requests through a loopback-only local proxy, injects authentication locally, performs account scheduling and retry handling, displays quota and usage information, and can configure supported local clients to use the gateway. The application is built with Go, Wails and Vue and is released for Windows and macOS.

### Open-source and distribution statement

> OmniProxy is distributed under the OSI-approved MIT License without commercial dual licensing. Its source code, build scripts and release workflow are public. Windows NSIS installers and checksums are already published through GitHub Releases.

### Build provenance

> Release artifacts are built from version tags by GitHub Actions on GitHub-hosted runners. The workflow installs frontend dependencies from `package-lock.json`, builds and tests the frontend and Go backend, and packages the Windows NSIS installer. After approval, the unsigned Windows artifact will be uploaded as a GitHub Actions artifact, submitted through SignPath's official GitHub Action, manually approved, downloaded, verified and published with a checksum generated after signing.

### Privacy and security statement

> OmniProxy has no maintainer-operated cloud relay, telemetry, advertising or crash-reporting service. Credentials and operational history stay on the user's device. API request content is sent only to the provider, custom gateway or outbound proxy selected by the user. The proxy and control API bind to loopback by default. OAuth, quota checks, update checks and optional automation are documented in the project's privacy policy and require the relevant user configuration or action.

### Requested artifacts

> We request Authenticode signing for the project-owned Windows application executable and/or the NSIS installer produced by the public release workflow. No unrelated or third-party executable will be signed with the OmniProxy policy.

### Project links

- Repository: <https://github.com/mibgb65-cloud/OmniProxy>
- Releases: <https://github.com/mibgb65-cloud/OmniProxy/releases>
- License: <https://github.com/mibgb65-cloud/OmniProxy/blob/master/LICENSE>
- Privacy policy: <https://github.com/mibgb65-cloud/OmniProxy/blob/master/PRIVACY.md>
- Code signing policy: <https://github.com/mibgb65-cloud/OmniProxy/blob/master/CODE_SIGNING_POLICY.md>
- Release workflow: <https://github.com/mibgb65-cloud/OmniProxy/blob/master/.github/workflows/release.yml>

The application form must be submitted with the maintainer's real contact name and email. Do not commit those details here unless the maintainer intentionally wants them to be public.

申请表仍需由维护者填写真实姓名和联系邮箱。除非维护者明确希望公开这些信息，否则不要把它们提交到仓库。

## After approval / 审核通过后

1. Install the SignPath GitHub App for this repository.
2. Create the SignPath project, artifact configuration and release-signing policy provided during onboarding.
3. Store only the API token in `SIGNPATH_API_TOKEN` as an encrypted GitHub Actions secret.
4. Store the organization ID and approved slugs as GitHub Actions variables.
5. Integrate `signpath/github-action-submit-signing-request@v2` using the exact identifiers assigned by SignPath.
6. Keep manual approval enabled and make the release fail closed if signing or signature verification fails.
7. Generate and publish checksums only after signing changes the artifact bytes.

Do not add guessed organization IDs, policy slugs or artifact-configuration XML before onboarding. SignPath must approve the exact artifact structure and metadata restrictions first.

在审核和接入完成前，不要填写猜测的组织 ID、策略标识或产物配置 XML；应先由 SignPath 确认具体产物结构和元数据限制。
