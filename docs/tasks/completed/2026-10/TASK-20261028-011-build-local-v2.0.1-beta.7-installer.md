# TASK-20261028-011: 构建本地 v2.0.1-beta.7 生产安装包

## Status

Completed

## Created

2026-10-28 23:12（Asia/Singapore）

## Request

用户要求提交代码，然后构建新的 Beta 生产版本用于本地更新，并完成任务 Markdown 更新。

## Goal

交付包含账号页 Dock 越界与遮挡修复的 Windows amd64 production 安装包，版本递增至
`v2.0.1-beta.7`，完成质量门、安装器元数据和发布资产校验，并本地提交、归档任务记录。

## Scope

- 核对已提交修复及版本源，更新 Beta 版本和发布说明。
- 使用既有发布脚本构建、打包并生成 SHA256 校验文件。
- 同步 CURRENT、关联任务交付状态及本任务记录。

## Out of Scope

自动安装或停止用户运行中的应用、签名、打标签、推送、公开发布、修改业务代码或数据库。

## Existing Behavior

上一份本地安装包为 `v2.0.1-beta.6`。Dock 修复已本地提交：
`970990e0`（实现）与 `efa3a11c`（记录归档），当前工作区干净。

## Analysis

- 已核对 `BUILD.md`、`RELEASE.md`、`UPGRADE.md`、前一版构建记录及现有发布脚本。
- `build-release.ps1` 校验源码/前端完整版本，映射 Windows 数字资源，再依次执行完整质量门、NSIS 打包、原生元数据与发布资产校验。
- Beta.6 已含端口配置；本次相对 Beta.6 的用户可见改动只有 TASK-20261028-010 的 Dock 越界、遮挡修复。
- 不修改安装身份、数据路径或数据库迁移；复用既有未签名本地构建流程，不自动安装、不使用私钥、不公开发布。

## Implementation Plan

1. 核对发布脚本与版本映射，采用下一 Beta `2.0.1-beta.7`，Windows 数字版本保持 `2.0.1.0`。
2. 更新版本源与人工整理的发布说明，运行 `scripts/build-release.ps1`。
3. 检查生产 EXE、安装器、SHA256、版本资源及产物路径。
4. 更新任务结果与 CURRENT，检查差异，本地提交并回填提交哈希、归档。

## Implementation Result

- 版本完整字符串统一为 `2.0.1-beta.7`：`internal/version/version.go`、
  `frontend/package.json`、`build/windows/info.json` 的 `ProductVersion`。
- Windows 数字版本保持 `2.0.1.0`；`build/config.yml` 与 NSIS 产品版本保持 `2.0.1`，
  安装身份和路径语义不变。
- 新增本地发布说明 `docs/releases/v2.0.1-beta.7.md`；相对 Beta.6 的代码差异只有账号页
  Dock 布局/动画修复，不含其他业务模块、数据库迁移或依赖变动。
- `bin/omniproxy.exe`：23,217,664 bytes，production 构建，原生产品版本 `2.0.1-beta.7`。
- 本地安装包：
  `release-assets/v2.0.1-beta.7/OmniProxy-Setup-v2.0.1-beta.7-windows-amd64.exe`
  （20,318,085 bytes）。
- 配套 `.exe.sha256`：
  `515bff6d371f4a84ea8742af32998c51d36c3ebfef42860b9f8f0478a65d4664`。
- 安装器签名状态为 `NotSigned`；未使用项目私钥、未改变用户信任库。

### Files Changed

- `internal/version/version.go`、`frontend/package.json`、`build/windows/info.json`
- `docs/releases/v2.0.1-beta.7.md`
- `docs/plans/CURRENT.md`
- TASK-20261028-010 交付补充及本记录

### Technical Notes

发布脚本重写 `info.json` 后恢复既有排版并统一 LF，无 BOM，最终只留下版本字符串一行差异。
自动生成绑定没有差异；`bin/`、`release-assets/` 和临时日志按既有规则忽略，不纳入 Git。

## Verification

```text
scripts/build-release.ps1 -Version v2.0.1-beta.7 -OutputDirectory release-assets/v2.0.1-beta.7
PASS（exit 0；日志 tmp/release-v2.0.1-beta.7.log）

task --force verify（由发布脚本执行）
PASS：format / go vet / staticcheck / ESLint / TypeScript / Go tests
Frontend：139 files / 851 tests PASS
Vite production / Windows amd64 production build：PASS

task --force package（由发布脚本执行）
PASS：NSIS UTF-8 打包，Total size 20,318,085 / 80,932,663 bytes（25.1%）

scripts/test-windows-metadata.ps1
PASS：应用完整产品版本 2.0.1-beta.7，安装器产品版本 2.0.1
两者固定 FileVersionNumber / ProductVersionNumber 均为 2.0.1.0

scripts/verify-release-assets.ps1
PASS：目录只含本次安装器及匹配的 SHA256 文件

git diff --check
PASS
```

版本与文档文件已检查为 UTF-8 无 BOM；版本源不存在旧 Beta.6 字符串。
保留既有 jsdom `window.scrollTo`、Vite 大 chunk 及 Windows 上 `uname`／`tail`
探测提示，均未导致质量门或打包失败，不做无关修正。

未执行自动安装、真实覆盖安装验收、签名、打标签、推送或公开发布；
TASK-20261028-010 记录的原生视觉验收限制不由本次打包代替。

## Documentation

已更新 `docs/releases/v2.0.1-beta.7.md`、CURRENT、TASK-20261028-010 与本记录。
无产品范围、架构、技术栈、数据库或构建流程变更。

## Deviations

None。既有发布脚本一次执行通过；仅还原脚本生成 JSON 的排版及 LF，避免无关格式差异。

## Git

```text
13dc37e8c23e372ef4821d94e393d59a1bef783e
chore(release): 构建 2.0.1-beta.7 本地生产安装包
```

已本地提交版本源、发布说明与构建校验记录；随后回填本哈希、归档并同步 CURRENT。
未提交构建产物、未推送远端。

## Result

包含 Dock 越界与遮挡修复的 `v2.0.1-beta.7` 本地生产安装包已构建并校验通过。
覆盖更新由用户执行，当前运行中的安装版及真实用户数据未被本任务操作。

## Completed

2026-10-28 23:25（Asia/Singapore）
