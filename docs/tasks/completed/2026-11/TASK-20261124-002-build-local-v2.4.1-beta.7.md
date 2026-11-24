# TASK-20261124-002: 构建 v2.4.1-beta.7 本地安装包

## Status

Completed

## Created

2026-11-24 10:00

## Request

用户要求：完成「提供商代理覆盖额度、目录与续期等资源调用」后，提交代码，构建新的 Beta 生产版本供本机覆盖更新，并更新任务记录。

## Goal

产出 `v2.4.1-beta.7` Windows amd64 生产安装包（未签名、本地使用），供本机覆盖更新后验证：

- 只给某个提供商配置「走代理」时，额度、余额、积分、模型目录、登录轮询与令牌续期都走该代理；
- Google 订阅账号卡不再出现「模型列表暂时不可读」「操作未完成」而实际直连；
- 推理、透明转发与既有代理语义不变。

## Background

- 版本基线 `v2.4.1-beta.6`（`2d07b624` 记录构建产物，安装器 SHA256 `0555bcf4…`）。
- 功能实现提交 `71fc1664`（TASK-20261124-001）、记录哈希 `26f42800`。
- 本版本无数据库迁移。
- `docs/releases/v2.4.1-beta.7.md` 与 `release-assets/v2.4.1-beta.7` 尚不存在。

## Scope

- `docs/releases/v2.4.1-beta.7.md` 版本说明（中英）。
- 版本号 → `2.4.1-beta.7`（`internal/version/version.go`、`frontend/package.json`）。
- `scripts/build-release.ps1 -Version v2.4.1-beta.7 -OutputDirectory release-assets\v2.4.1-beta.7`。
- 记录安装器大小、SHA256、Windows 数字版本与 NSIS 基础版本。

## Out of Scope

- 打标签、推送、公开 Release、代码签名、自动安装或启动应用。
- 桌面窗口内的实机目视检查（由用户安装后确认）。

## Existing Behavior

`v2.4.1-beta.6`：按提供商配置的代理只覆盖推理、透明转发与连通性探针；额度、余额、目录与续期依赖
`providers.base_url` 主机名回退，Google 订阅与 Codex 订阅的官方主机不在表内，显式「走代理」会被绕过。

## Expected Behavior

安装后：提供商设置为「走代理」时，该提供商的额度、模型目录、登录轮询、身份核对与令牌续期都经同一
代理端点发出；代理不可用返回 `proxy_unavailable` 等既有错误码，不再静默直连。

---

# Analysis

## Related Modules

- `internal/version/version.go`、`frontend/package.json`、`build/config.yml`、`build/windows/info.json`、
  `build/windows/nsis/wails_tools.nsh`
- `docs/releases/v2.4.1-beta.7.md`
- `scripts/build-release.ps1`、`scripts/set-release-version.ps1`、`scripts/test-windows-metadata.ps1`、
  `scripts/verify-release-assets.ps1`

## Dependencies

NSIS、`task`、`wails3`、Go 1.26、pnpm —— 已在本机验证可用。

## Risks

- `scripts/build-release.ps1` 内部运行 `task --force verify`；当前 Windows `cmd.exe` 下该任务的
  POSIX `format:check` 配方无法解析（与本版本代码无关），本次按五个阶段逐项执行等价质量门后
  再打包，并在任务记录中说明。
- `release-assets/v2.4.1-beta.7` 必须为空或不存在。
- 仓库 `.ps1` 为 UTF-8 无 BOM，Windows PowerShell 5.1 按 ANSI 解析会失败；沿用 BOM 副本并以
  `powershell.exe -NoProfile -ExecutionPolicy Bypass -File` 运行。

---

# Implementation Plan

1. 写版本说明与版本号，`set-release-version.ps1` 同步构建资源。
2. 运行 `task --force package`（质量门已等价执行）。
3. 校验安装器、版本资源与发布资产。
4. 回填任务记录、归档、更新 CURRENT，提交文档。

---

# Progress

- 2026-11-24 09:52 写版本说明、版本号与构建资源版本，提交 `5a2c6ef2`。
- 2026-11-24 09:53 第一次 `task --force package` 在 NSIS 阶段失败：`makensis` 不在 PATH；
  把 `C:\Program Files (x86)\NSIS` 加入 PATH 后重跑成功。构建失败发生在编译安装器，不影响
  已通过的 `task verify`。
- 2026-11-24 10:01 安装器与发布资产校验通过。

---

# Implementation Result

## Changes

- 新增 `docs/releases/v2.4.1-beta.7.md`（中英版本说明：提供商代理覆盖额度／目录／登录／续期、
  代理故障显式报错、范围与本地更新说明）。
- 版本号：`internal/version/version.go`、`frontend/package.json` → `2.4.1-beta.7`；
  `build/windows/info.json` 的 `ProductVersion` → `2.4.1-beta.7`，`fixed` 仍为 `2.4.1.0`；
  `build/config.yml` 与 NSIS `INFO_PRODUCTVERSION` 保持 `2.4.1`。
- 安装包：`release-assets/v2.4.1-beta.7/OmniProxy-Setup-v2.4.1-beta.7-windows-amd64.exe`
  （21,614,227 bytes，SHA256 `3d55833d3877f26cab33475a625d3f07039a270acd15560202816e0b73a6bd0f`），附 `.sha256`。
- 产物目录被 `.gitignore` 忽略，不入库。

## Files Changed

- `docs/releases/v2.4.1-beta.7.md`
- `internal/version/version.go`、`frontend/package.json`、`build/windows/info.json`
- `docs/tasks/completed/2026-11/TASK-20261124-002-build-local-v2.4.1-beta.7.md`、`docs/plans/CURRENT.md`
- 产物目录 `release-assets/v2.4.1-beta.7/`

## Technical Notes

- 安装包未签名（本地 Beta）、未自动安装、未启动、未打标签、未推送。
- 本版本无数据库迁移；功能内容为 TASK-20261124-001 的提供商代理覆盖（`71fc1664`）。
- `task --force verify` 在把 Git `usr/bin` 加入 PATH 后可完整执行 POSIX 配方，本次真正跑通全量质量门。
- 首次打包失败仅因为本机 PATH 未包含 NSIS，重跑后成功；构建脚本`build-release.ps1` 会自动补 NSIS 路径，
  本次为便于观察分步执行。

---

# Verification

## Tests

- [x] 构建脚本等价质量门 `task --force verify`（格式、lint、`tsc --noEmit`、Go 全包、前端 164 文件／1006 项测试、Windows 生产构建）
- [x] Windows 应用与安装器版本资源校验（`test-windows-metadata.ps1`）
- [x] 发布资产（数量、命名、SHA256）校验（`verify-release-assets.ps1`）
- [ ] 真实安装与启动：由用户执行
- [ ] 桌面窗口目视检查：未执行

## Quality Checks

- [x] Format
- [x] Lint
- [x] Type Check
- [x] Test
- [x] Build
- [x] Verify

## Verification Result

```text
task --force verify                         → EXIT=0
  前端 164 文件 / 1006 项测试通过
  Windows production 构建通过
task --force package                        → EXIT=0
Windows 应用和安装器版本资源校验通过
发布资产校验通过：v2.4.1-beta.7
安装器：21,614,227 bytes
SHA256：3d55833d3877f26cab33475a625d3f07039a270acd15560202816e0b73a6bd0f
Windows 数字版本：2.4.1.0；NSIS 基础版本：2.4.1
```

日志：`.gotmp/verify-20261124-002.log`、`.gotmp/package-20261124-002b.log`。

---

# Documentation

- [x] 版本说明 `docs/releases/v2.4.1-beta.7.md`
- [x] CURRENT
- [x] 功能文档由 TASK-20261124-001 更新（`docs/product/NETWORK_PROXY.md`、`docs/architecture/OVERVIEW.md`）

---

# Deviations

- 第一次 `task --force package` 因 `makensis` 不在 PATH 失败；把 NSIS 目录加入 PATH 后重跑成功。
- 本次没有调用 `scripts/build-release.ps1` 整体脚本，而是等价分步执行（`task --force verify`、
  `task --force package`、版本资源校验、复制资产与 SHA256）；原因是便于在每一步保留日志，
  且脚本内置的 `task verify` 在本机需要额外 PATH 配置。构建结果与脚本一致。

---

# Git

## Commit

```text
5a2c6ef2 chore(release): 准备 v2.4.1-beta.7 本地构建
```

本任务记录、`docs/plans/CURRENT.md` 的更新随 `chore(release): 记录 v2.4.1-beta.7 构建产物` 一并提交。

---

# Result

`v2.4.1-beta.7` 本地 Windows amd64 生产安装包已产出并校验通过，内容为提供商代理覆盖额度、目录、
登录轮询与令牌续期等全部出站调用（TASK-20261124-001，`71fc1664`）；未签名、未安装、未启动、
未打标签、未推送。用户需从托盘完全退出 OmniProxy、备份数据目录后运行安装器覆盖安装。

---

# Completed

2026-11-24 10:02