# TASK-20261019-011: 生成额度续期修复的本地 Beta.8 生产安装版

## Status

Completed

## Created

2026-10-19（Asia/Singapore）

## Request

完成后给我生成一个本地可安装的生产 beta 版本。

## Goal / Scope

将当前已验证的额度续期／诊断修复及既有更新页面改动，生成 2.0.0-beta.8 Windows amd64 生产构建和 NSIS 安装包，校验版本资源与 SHA256，提供本地路径。不自动安装、不发布 GitHub、不推送或创建版本标签，不改真实账号、客户端配置和生产数据。

## Analysis

现有版本为 Beta.7，Beta.6／Beta.7 本地安装资产必须保留。额度修复提交为 2616efe，已有完整质量门／610 前端测试通过。仓库 release 脚本强制执行 verify 和 package，并要求完整 SemVer、人工发布说明和独立空资产目录。

为避免覆盖运行中的开发程序，将本任务版本与说明暂存为确定 Git tree，通过 git archive 将该树展开至忽略的独立构建目录，在该不可变源码快照中执行原有 build-release.ps1，验证通过后再提交同一组源文件，输出到主工作区 release-assets/v2.0.0-beta.8。不修改构建脚本，不打包用户未跟踪文件或凭据。

## Implementation Plan

1. 同步 Go／前端 Beta.8 版本，使用现有脚本映射 Windows 数字资源；补充本地发行说明和随包日志回归。
2. 仅暂存本任务文件；从确定 Git tree 导出隔离源码，执行原有生产质量门与 NSIS 打包，通过后再提交。
3. 校验应用／安装器原生版本资源、完整版本、两份发布资产和 SHA256；核对真实生产实例未被替换。
4. 记录实际构建与验证结果、产物路径、摘要和未执行的安装验收；归档任务并更新看板。

## Implementation Result

- Go、前端完整版本和应用 Windows ProductVersion 同步为 2.0.0-beta.8；Windows 数字资源继续为 2.0.0.0，NSIS 数字版本为 2.0.0，未修改安装身份或安装范围。
- 新增 Beta.8 本地发行说明，包含额度查询前官方续期、30 秒预算、安全错误分类及更新页面／可选下载改进；增加三个随运行版本读取 Beta.8 发行说明的回归。
- 从暂存 Git tree e8124a854e602c1b8e17238cf56b2bd3f099ff09 导出独立源码至忽略的 tmp/local-beta8-20261019/clean-source，执行原有 build-release.ps1。质量门和打包均使用 production 标签，不包含 updateqa。
- 主工作区运行中的开发二进制未被发布脚本覆盖；未跟踪用户文件、账号、凭据和本地数据未进入归档源码。Beta.6／Beta.7 资产保留，Beta.8 输出到独立目录。
- 版本／说明提交后的 Git tree 与已构建树完全相同，构建来源可核对；最终任务归档和看板仅为文档更新，不改变该生产二进制。

## Verification

- PASS：主工作区 current-release.test.ts 定向测试，17 项通过，包含 Beta.8 普通版本／v 前缀／构建元数据三种入口。
- PASS：隔离源码 scripts/build-release.ps1 -Version v2.0.0-beta.8 -OutputDirectory <主工作区>/release-assets/v2.0.0-beta.8，exit 0。
- PASS：上述脚本内完整 task --force verify：格式、Go vet／staticcheck、ESLint、TypeScript、全部 Go 测试、107 文件／613 项前端测试、绑定生成、前端生产资源和 Windows amd64 构建。
- PASS：上述脚本内 task --force package，NSIS UTF-8 machine 安装器生成成功；现有机器级安装需要管理员确认。
- PASS：原生 Windows 版本资源脚本两次核验应用／安装器产品身份与版本；资产校验脚本两次确认输出目录仅安装器／同名 SHA256 两个文件且摘要匹配。
- PASS：独立 Get-FileHash 与 SHA256 文件一致；Get-AuthenticodeSignature 为 NotSigned，本地安装包未签名，可能出现未知发布者提示。
- PASS：提交 427bc3344b5657585318b268c311d84452c22a95 的 tree 为 e8124a854e602c1b8e17238cf56b2bd3f099ff09，与导出源码树一致。git diff --check 通过，无业务依赖或数据库迁移变更。
- PASS：构建结束后真实 /healthz 仍为 production／2.0.0-beta.7／ok，生产进程仍为 PID 30124，未安装或替换。

未执行：真实安装／升级、管理员提权及真实账号过期续期。准备隔离 GUI 冒烟脚本的工具调用被策略阻止，脚本未执行，也未改用其他方式绕过；因此不宣称完成新二进制的 GUI 运行验收。已有 jsdom scrollTo 与 Vite chunk／插件耗时警告不影响门禁通过。

## Local Artifact

- 安装器：release-assets/v2.0.0-beta.8/OmniProxy-Setup-v2.0.0-beta.8-windows-amd64.exe
- SHA256 文件：release-assets/v2.0.0-beta.8/OmniProxy-Setup-v2.0.0-beta.8-windows-amd64.exe.sha256
- 大小：19,351,559 bytes。
- SHA256：730f7fce13674312953fca7386b1af4e002bc066bc229041a20a8dfae175a765。
- 完整构建日志与源码归档只保存在忽略的 tmp/local-beta8-20261019/。

## Git

427bc3344b5657585318b268c311d84452c22a95 chore(release): 准备 Beta.8 本地生产安装版

包含额度修复 2616efee603681ee19351731cd8034db8c4a7f79 及此前更新页面改动；仅本地提交，未创建／推送标签，未发布 GitHub。

## Result

Beta.8 本地 Windows amd64 生产安装包及校验文件已生成并核验，可以交付用户自行安装。生产仍为 Beta.7，未自动安装、重启或改动既有授权；实际安装和故障复现不在本轮已通过的验证范围内。