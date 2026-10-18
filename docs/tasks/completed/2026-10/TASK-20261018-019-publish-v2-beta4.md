# TASK-20261018-019: 发布 v2.0.0-beta.4

## Status

Completed

## Created

2026-10-18

## Request

用户明确要求完成修复后发布新的 Beta 版本。

## Goal

把已完成真实上游验收的订阅 SSE 修复发布为 v2.0.0-beta.4，提供可校验的 Windows 安装器并核对实际更新发现。

## Scope

- 同步应用／前端／Windows 资源版本，整理发布说明。
- 本地完整质量门、安装器与原生版本／SHA256 校验，按既有脚本进行隔离安装验收。
- 普通快进推送 main、新建并推送 Beta.4 标签，跟踪 CI／Release 和公开资产。
- 验证 Beta.3 可发现 Beta.4，保留旧 Beta 和 stable Latest。

## Out of Scope

- 不改写历史标签，不覆盖已发布 Beta.3。
- 不自动安装或重启当前生产应用，不修改共享 Codex 配置、登录或当前 Agent。
- 不提交既有用户未跟踪文件、临时诊断、凭据及真实配置。

# Analysis

TASK-018 修复提交 ad105c1 已完成两模型／两种入口及双 Codex 内核共六次真实请求，全部成功；完整质量门／575 前端测试通过。当前 main 还包含未推送的 TASK-016 契约测试和任务归档。沿用既有 release.yml、build-release.ps1、NSIS 和 draft 完整发布流程；发布必须在实际远端构建成功后报告。

# Implementation Plan

1. 核对远端发布基线，创建新版本说明并同步完整 SemVer，Windows 数字版本继续 2.0.0.0。
2. 完成最终版本构建、摘要、独立产品身份的安装验收。
3. 审查提交，普通推送 main，CI 通过后推送全新版本标签。
4. 核对公开 Release、安装器摘要、实际更新客户端及旧发布保留情况，归档结果。

# Progress

远端 main 为 ff006db；Beta.4 标签／Release 尚不存在，当前账号有推送权限。已保存 Beta.1–3 与 stable 的发布／资产基线及用户配置哈希；默认 master 保持不动。

# Implementation Result

## Candidate

- 应用／前端／Windows 产品字符串同步 2.0.0-beta.4；Windows 数字资源保持 2.0.0.0。
- 新增发布说明，明确真实 SSE 缺失媒体类型的修复边界、双内核及六次真实请求验收、更新后需重启才生效、未签名和不自动升级。
- 发布及安装器实现不变，不增加依赖或数据库迁移。新版本相对 TASK-018 仅改版本与发布文档，真实上游验收沿用已验证的同一修复代码，不重复产生模型调用。
- 本地安装器 19,253,592 bytes，SHA256 976b52f7ac93aaa774dcd75e668f94f57feb23ce153ca2754f751ac1a0bfc9d7；此为本地候选摘要，不能冒充公开 CI 资产摘要。

## Publication

- 发布提交 6c0bd725caa328c0b65e00ea9de0798db0e45768 已普通快进推送至 main，包含 TASK-016 契约测试及 TASK-018 真实 SSE 修复。
- 首次普通推送遇到 Windows schannel TLS 握手失败；只读复核远端仍为旧提交后，使用单次命令的 http.sslBackend=openssl 重试成功，没有关闭 TLS 验证或更改全局配置。
- 本提交 CI 39139227077 完整成功（verify 9m3s），只读复核 headSha 与发布提交一致。首次 watch 遇到 API EOF，仅恢复同一运行的状态查询，没有重跑构建或把网络错误当作 CI 失败。
- 新建并普通推送 v2.0.0-beta.4：注释标签对象 47a3a2d90bc15dbacf02e0bd2f0280ea49b83882，源码 6c0bd725caa328c0b65e00ea9de0798db0e45768。没有改写已有标签。
- 标签触发 Release 39139988913，build 与 publish 均成功，发布源仍为 6c0bd725caa328c0b65e00ea9de0798db0e45768。
- 公开 Release ID 416532150，draft=false / prerelease=true，发布于 2026-10-18 23:35:47 UTC+08:00。
- 公开安装器 19,271,304 bytes，SHA256 56df132691e7e9b561b183a6e3eb421ce1c9940a24494beeb7127518f62935ff；同名 sha256 文件 114 bytes。实际公开下载、校验文件与 GitHub digest 一致，原生 Windows 版本资源通过。
- Beta.1–3 及 stable 发布 ID／正文摘要／资产 ID、大小和 digest 未变；旧注释标签对象未变，正式 Latest 仍为 v1.3.1。
- 当前 Codex 配置哈希未变，生产 PID 42784 仍为 Beta.3；没有自动安装或重启。



# Verification

- PASS：scripts/build-release.ps1 -Version v2.0.0-beta.4 -OutputDirectory tmp/beta4-release-20261018/local-assets；完整 task --force verify、104 文件／575 前端测试及全部 Go／格式／静态／类型／Windows 构建，随后 NSIS、原生版本资源及两项资产摘要校验通过。
- PASS：独立用户级安装／快捷方式图标／旧统计库哈希保留／v2 迁移和重启／运行中覆盖／已删除桌面入口不重建／运行中卸载／未知文件与两库保留。使用独立产品名、注册项、工作区安装目录、配置及端口 14104／14894；生产仍健康，不冒充机器级 UAC 验收。
- PASS：远端 CI 39139227077，发布源完整质量门成功。
- PASS：Release 39139988913 构建／公开发布全部成功。
- PASS：真实 update.Checker 的 stable 渠道仍发现 v1.3.1，Beta 渠道发现 v2.0.0-beta.4，比较确认 Beta.1／2／3 均可发现新版本；真实 SelectPackage／Download／VerifyFile 下载公开包并验证匹配的 SHA256。
- 网络说明：GitHub 状态 watch 与更新检查曾收到 EOF，均为只读网络请求；恢复同一个远端运行状态、有限重试下载后通过，没有重跑或替换 Release。临时关闭 HTTP/2 的子进程验收通过后，另以默认环境／默认传输再次完整发现、下载和校验成功，不把兼容参数当成应用必需配置。
- PASS：公开安装器原生资源、资产脚本、GitHub digest 及校验文件匹配；安装包 NotSigned，发布说明已注明。
- PASS：旧发布及标签、stable Latest、用户配置哈希和当前生产进程保持不变。



# Git

6c0bd725caa328c0b65e00ea9de0798db0e45768 chore(release): 准备 v2.0.0-beta.4

main 与全新 Beta.4 标签已普通推送；公开发布和下载验收通过。


# Documentation

新增 docs/releases/v2.0.0-beta.4.md 并更新 CURRENT。无需修改安装器、工作流、架构或数据库。

# Result

v2.0.0-beta.4 已公开发布，真实安装包／更新渠道验收通过，旧版本和正式 Latest 保留。发布地址：https://github.com/mibgb65-cloud/OmniProxy/releases/tag/v2.0.0-beta.4 。当前本机应用未自动更新，需用户升级并重启后生效。

# Completed

2026-10-18 23:41
