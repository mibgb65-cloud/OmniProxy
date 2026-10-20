# TASK-20261020-004: 构建 Beta.11 本地生产安装包

## Status

Completed

## Created

2026-10-20

## Request

修复完成后，构建新的 Beta 生产版本，供用户自行进行本地更新。

## Goal / Scope

- 同步版本为 `2.0.0-beta.11`，编写随包发行说明。
- 从冻结的本地提交隔离执行现有发布脚本，完成完整质量门、Windows amd64 production 构建及 NSIS 打包。
- 校验原生版本、资产与 SHA256，交付 `release-assets/v2.0.0-beta.11/` 中的安装包。

不自动安装、停止生产、修改真实账号／凭据、推送、打远端标签或公开发布；不纳入其他未跟踪任务记录。

## Analysis

当前源码及最近本地安装包为 Beta.10。新版本包含 Responses 长会话重连修复 `a467ffe`，以及其后已提交文档；沿用既有机器级安装身份和路径。

相对 Beta.10 构建源 `21b5e2a`，还包含已提交的授权进度／账号删除修复 `4d0e76c` 和账号轮换设置 `58521d1`（迁移 035）。发行说明需覆盖这些实际变更，不宣称尚未实现的额度快照持久化已交付。

隔离构建只导出冻结提交的受跟踪文件，排除真实配置、凭据及现有未跟踪任务。已保存旧资产和已安装生产程序共 11 个文件的摘要及生产进程身份基线。

## Implementation Plan

1. 核对版本映射、发行说明和现有发布脚本，更新版本及相关回归预期。
2. 提交构建源并导出隔离源码，使用既有 `scripts/build-release.ps1` 完整构建、打包与验证。
3. 独立复核生产标签、原生版本、资产 SHA256 与来源一致性，更新 CURRENT 和任务记录并交付。

## Implementation Result

- Go／前端／Windows ProductVersion 同步为 Beta.11，数字资源及 NSIS 保持 `2.0.0.0`／`2.0.0`。
- 新增 Beta.11 随包发行说明及三种版本读取回归；涵盖实际已提交功能，明确额度快照持久化未实现与未签名／未安装验收边界。
- 已冻结构建源 `6fd860cf9551a043389a84980fe10606489f825b`（tree `7647ba603ed72bed889be77d92c7435ac48788f1`），导出到忽略的 `tmp/local-beta11-20261020/source/`，沿用既有发布脚本完成完整质量门及 NSIS 打包。
- 安装包与同名 SHA256 文件已导出到独立 Beta.11 目录，旧资产保留原样；没有创建标签、推送、公开发布、运行安装器或主动停止生产程序。
- 构建日志、冻结源码、24 项来源核对和独立版本／签名／基线验证摘要保存在忽略的 `tmp/local-beta11-20261020/`，不提交产物、配置或凭据。

## Verification

- PASS：`scripts/set-release-version.ps1 -Version v2.0.0-beta.11`。
- PASS：版本文件／发行说明读取测试的定向 Prettier 检查。
- PASS：`pnpm exec vitest run src/features/about/current-release.test.ts`，26 项通过。
- PASS：从冻结源码运行 `scripts/build-release.ps1 -Version v2.0.0-beta.11 -OutputDirectory <主工作区>/release-assets/v2.0.0-beta.11`，退出码 0。
- PASS：脚本内 `task --force verify`，格式、Go vet／staticcheck、ESLint、TypeScript、全部 Go 测试、109 文件／632 项前端测试、Wails 绑定生成、前端生产构建及 Windows amd64 production 构建通过。
- PASS：脚本内 `task --force package`，NSIS 使用 UTF-8 和既有机器级产品身份生成安装器，不包含 updateqa 等测试入口。
- PASS：原生 Windows 版本 API 在发布脚本及独立核对中验证应用完整版本、数字资源、产品名、公司名和安装器身份。
- PASS：发布资产验证及独立 SHA256 核对一致，目录仅含安装器和同名校验文件；签名状态为 NotSigned。
- PASS：`go version -m` 确认 Go 1.26.2、`-tags=production`、GOOS=windows、GOARCH=amd64、CGO_ENABLED=0，不含 updateqa。
- PASS：构建前后 24 个关键源码／测试／依赖／版本／发布脚本文件与冻结提交 Git blob 一致；确认包含 `a467ffe`、`4d0e76c`、`58521d1`。
- PASS：11 项已安装程序及旧资产文件大小／摘要保持一致，生产 PID 31376、路径及 UTC 启动时刻保持原样。首次进程核对因 PowerShell 自动将 JSON 时间转换为 DateTime，导致字符串比较误报；改用 UTC ticks 后核实未重启，不将该误报当作真实进程变化。
- PASS：`git diff --check`、本任务文件 UTF-8 无 BOM 与本次提交文件范围检查。主工作区业务代码、已有未跟踪记录及真实配置未纳入本次打包提交。
- 非阻塞既有告警：Windows Task 环境缺少部分 Unix 命令、jsdom scrollTo 未实现、Vite 大 chunk 和 Tailwind 构建性能提示；未隐藏或跳过检查。

未执行：真实覆盖安装、管理员确认、安装后生产启动、真实 OAuth／账号轮换／长会话网络及原生 GUI 验收。构建与隔离回归通过不等同于这些场景已通过。

## Artifact

- 安装器：`E:/go/new/wails3-v/OmniProxy/release-assets/v2.0.0-beta.11/OmniProxy-Setup-v2.0.0-beta.11-windows-amd64.exe`。
- 校验文件：同目录 `OmniProxy-Setup-v2.0.0-beta.11-windows-amd64.exe.sha256`。
- 大小：19,403,318 bytes（约 18.5 MiB）。
- SHA256：`600c476645798deb415f47e0cd6ee01b192f7a61b74ff8c1d4400afb14cc37a8`。
- 签名状态：NotSigned。

## Documentation

已新增 Beta.11 发行说明并更新 CURRENT 最新本地安装包信息；不改动历史发行说明或发布规范。

## Git

`6fd860cf9551a043389a84980fe10606489f825b` — `chore(release): 准备 Beta.11 本地生产安装包`。

仅本地提交，不推送；构建结果与任务归档由后续文档提交保存。其他已有未跟踪文件保持不动。

## Result

Beta.11 Windows amd64 production 本地安装包及 SHA256 已生成并验证，可供用户自行覆盖更新。包含长会话恢复及 Beta.10 后已提交优化，未包含尚未实现的额度快照持久化；安装包未签名、未发布、未自动安装，现有生产程序保持运行。

## Completed

2026-10-20（Asia/Singapore）
