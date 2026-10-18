# TASK-20261018-014: 推送并发布 v2.0.0-beta.3

## Status

In Progress

## Created

2026-10-18 20:36

## Request

用户明确要求推送代码并发布 Beta.3，承接 Codex 隔离转发修复 TASK-20261018-013。

## Goal

统一 v2.0.0-beta.3 版本，通过现有质量门及发布工作流推送 main／新标签，公开 Windows amd64 预发布安装器与 SHA256，核对公开资产和更新发现。

## Scope

- 版本提升、发布说明、必要验证及项目状态文档。
- 沿用 GitHub CI／Release 流程，不覆盖 Beta.1／Beta.2 或正式 Latest。
- 发布后核对标签、公开 Release、资产摘要与更新客户端行为。

## Out of Scope

- 不修改当前 Agent 配置、真实账号、凭据或生产运行状态。
- 不自动安装／升级本机生产应用，不重写远端历史或现有标签。
- 不把隔离模拟测试等同于真实账号调用验收，不承诺历史 400 已完全定位。

# Analysis

远端 main 停在 971c3ef，本地额外包含已验证的修复与归档两次提交；普通快进推送可行。仓库为公开仓库，当前凭据有 push 权限，默认 master 与正式 Latest v1.3.1 保持不变。Beta.3 标签尚不存在。沿用 release.yml 标签触发、完整质量门、NSIS、原生资源、资产 SHA256 与 draft 完整发布流程；不复用 Beta.1 特殊恢复步骤。完整应用／前端版本提升为 2.0.0-beta.3，Windows 数字资源保持 2.0.0.0。安装器实现未改动，仍进行独立用户级安装／运行中覆盖／卸载验收；不把此验收冒充机器级 UAC 验收。

# Implementation Plan

1. 核对发布契约与远端，提升完整 SemVer 并补充准确发布说明。
2. 完成最终版本验证与安装器／版本资源／摘要校验。
3. 审查提交，推送 main 与不可变新标签，跟踪 CI／Release。
4. 验证公开下载和 Beta 更新发现，保留当前 Agent／生产状态，归档实际结果。

# Progress

- 已确认远端权限／main 快进关系和新标签不存在，保留 Beta.1／Beta.2 资产基线。首次读取 Beta.1 元数据遇到网络 EOF，重试只读请求成功。
- 前两次最终版本构建先后被格式门拦截：版本文件需 gofmt，前端 package.json 需 Prettier（Windows 文本写入带入 CRLF）；均仅格式化本任务的两个版本文件，未跳过检查。最终完整发布构建使用 Git Unix 工具路径与 NSIS，避免既有构建变量查找 uname／tail 的环境告警。

# Implementation Result

已同步应用／前端／Windows 产品完整版本为 2.0.0-beta.3，数字资源保持 2.0.0.0；新增准确的 Codex 修复发布说明，保留真实账号验收边界。现有 Release 工作流和安装器实现没有修改。最终本地完整发布构建已通过，独立用户级安装生命周期验收与远端发布进行中。

# Verification

- PASS：最终 scripts/build-release.ps1 -Version v2.0.0-beta.3，完整 task --force verify（104 文件／575 项前端测试及 Go／格式／静态／类型／Windows production 构建）、NSIS 打包、Windows 原生资源及两项发布资产 SHA256 校验成功。日志 tmp/beta3-release-20261018/build-release-verified.log。
- PASS：go test ./internal/version ./internal/infrastructure/update，版本及更新逻辑定向验证。
- IN PROGRESS：独立用户级安装／覆盖升级／重启／卸载，不接触真实安装或客户端配置。
- IN PROGRESS：GitHub CI／Release、公开下载及实际更新客户端校验。

# Git

待填写；用户已授权普通推送和新的 Beta.3 发布。

# Result

待填写。