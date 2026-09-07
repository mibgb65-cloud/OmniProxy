# OmniProxy Release Checklist / 发布检查清单

Use this checklist before promoting a beta build to a stable release.
正式版发布前，用这份清单复核 beta 构建、安装更新、卸载和发布资产。

## 1. Prepare / 准备

- Confirm the worktree is clean: `git status --short`.
- Confirm curated release notes exist: `docs/releases/<tag>.md`.
- Confirm the version tag follows `vMAJOR.MINOR.PATCH` or `vMAJOR.MINOR.PATCH-beta.N`.
- Confirm the README, Release description, [privacy policy](../PRIVACY.md), [code signing policy](../CODE_SIGNING_POLICY.md), and [MIT license](../LICENSE) remain mutually consistent.
- Confirm every maintainer with a signing role has multi-factor authentication enabled for GitHub and SignPath.
- Run local validation:
  - `cd OmniProxyBackend && go test ./...`
  - `cd frontend && npm test`
  - `cd frontend && npm run build`
  - `node scripts/check-lines.mjs`

## 2. Beta Validation / Beta 验证

- Install the latest beta over the previous installed version.
- Check that config, account pool, request history, billing summary, and theme state are preserved.
- Open About -> Update Diagnostics and confirm:
  - current update state is readable;
  - `status.json` and `update.log` paths are shown;
  - copy diagnostics produces useful text.
- Check manual update detection from About.
- Check titlebar update prompt behavior:
  - "稍后" hides the automatic prompt for 24 hours;
  - "跳过此版本" hides only the current version;
  - manual check still shows the current update state.

## 3. Windows Install And Update / Windows 安装与更新

- Fresh install from `OmniProxy-Setup-<tag>-windows-amd64.exe`.
- Upgrade from the previous stable version.
- Upgrade from the latest beta version.
- Verify the installer:
  - has `ProductName` and `CompanyName` set to `OmniProxy`;
  - has `ProductVersion` and `FileVersion` matching the numeric part of the release tag;
  - keeps existing data under `%USERPROFILE%\.omniproxy`;
  - preserves user-deleted desktop shortcut during upgrade;
  - removes old autostart entries on uninstall;
  - starts the app after install when expected.
- Verify in-app update:
  - downloads the `.exe` asset;
  - verifies the matching `.sha256`;
  - persists `status.json`;
  - starts the silent auto-update installer;
  - app exits cleanly and relaunches after install.
- Verify uninstall:
  - default uninstall keeps user data;
  - explicit data removal mode removes local app data only when requested.

## 4. macOS Unsigned DMG / macOS 未签名 DMG

- Download `OmniProxy-<tag>-darwin-universal-unsigned.dmg`.
- Verify SHA256 against the release `.sha256` file.
- Open the DMG and confirm it contains:
  - `OmniProxy.app`;
  - `Applications` shortcut;
  - `README.txt`.
- Confirm README explains:
  - the build is unsigned and not notarized;
  - Finder right click -> Open may be required;
  - update requires quitting OmniProxy and replacing the app in Applications.
- Verify app launch on Apple Silicon.
- Verify replacement install over an older app bundle.

## 5. Release Assets / 发布附件

- Confirm the Release workflow completed all jobs:
  - Windows installer;
  - macOS universal unsigned DMG;
  - Publish release.
- Confirm the release is marked prerelease only for beta tags.
- Confirm exactly these assets are attached:
  - `OmniProxy-Setup-<tag>-windows-amd64.exe`
  - `OmniProxy-Setup-<tag>-windows-amd64.exe.sha256`
  - `OmniProxy-<tag>-darwin-universal-unsigned.dmg`
  - `OmniProxy-<tag>-darwin-universal-unsigned.dmg.sha256`
- Confirm the Release description contains a `Code signing policy` link.
- If SignPath signing is enabled for this release:
  - confirm the signing request originated from the same GitHub Actions run and required manual approval;
  - confirm only the signed Windows artifact is selected for publication;
  - run `Get-AuthenticodeSignature` and verify a valid SignPath Foundation signature;
  - confirm the Windows SHA-256 file was generated after signing.
- Run the asset verifier when assets are available locally:

```powershell
powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\verify-release-assets.ps1 -Version <tag> -AssetDirectory <asset-dir>
```

## 6. Stable Promotion / 正式版提升

- Only promote after the latest beta has completed install, update, and uninstall validation.
- Create the stable tag from the validated commit.
- Confirm the stable release notes summarize the beta line and list any user-visible upgrade guidance.
- Confirm publishing the stable release removes GitHub Release entries and assets for the matching `<stable-tag>-beta.*` line only.
- Confirm beta Git tags and `docs/releases/<beta-tag>.md` remain available for history and changelog links.
- After publish, install the stable Windows asset over the latest beta once.
- Confirm beta users can detect the stable tag as a newer available update.
