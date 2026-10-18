# TASK-20261018-019: 发布 v2.0.0-beta.4

## Status

In Progress

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

待远端 CI 与发布验收。


# Verification

- PASS：scripts/build-release.ps1 -Version v2.0.0-beta.4 -OutputDirectory tmp/beta4-release-20261018/local-assets；完整 task --force verify、104 文件／575 前端测试及全部 Go／格式／静态／类型／Windows 构建，随后 NSIS、原生版本资源及两项资产摘要校验通过。
- PASS：独立用户级安装／快捷方式图标／旧统计库哈希保留／v2 迁移和重启／运行中覆盖／已删除桌面入口不重建／运行中卸载／未知文件与两库保留。使用独立产品名、注册项、工作区安装目录、配置及端口 14104／14894；生产仍健康，不冒充机器级 UAC 验收。
- 待核对：远端 CI／Release、公开资产、实际更新发现／下载及旧发布保留。


# Git

待提交，仅授权普通快进推送及新建 Beta.4 标签。
