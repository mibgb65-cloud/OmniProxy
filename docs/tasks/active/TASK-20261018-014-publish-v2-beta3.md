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
- 不自动安装／升级本机生产应用，不改写 main 历史；仅按用户后续明确授权，使用预期旧标签锁定移动尚未发布的 Beta.3 标签，其他标签不变。
- 不把隔离模拟测试等同于真实账号调用验收，不承诺历史 400 已完全定位。

# Analysis

远端 main 停在 971c3ef，本地额外包含已验证的修复与归档两次提交；普通快进推送可行。仓库为公开仓库，当前凭据有 push 权限，默认 master 与正式 Latest v1.3.1 保持不变。Beta.3 标签尚不存在。沿用 release.yml 标签触发、完整质量门、NSIS、原生资源、资产 SHA256 与 draft 完整发布流程；不复用 Beta.1 特殊恢复步骤。完整应用／前端版本提升为 2.0.0-beta.3，Windows 数字资源保持 2.0.0.0。安装器实现未改动，仍进行独立用户级安装／运行中覆盖／卸载验收；不把此验收冒充机器级 UAC 验收。

# Implementation Plan

1. 核对发布契约与远端，提升完整 SemVer 并补充准确发布说明。
2. 完成最终版本验证与安装器／版本资源／摘要校验。
3. 审查提交，推送 main 与不可变新标签，跟踪 CI／Release。
4. 验证公开下载和 Beta 更新发现，保留当前 Agent／生产状态，归档实际结果。

# Progress

- 用户明确授权修复后移动尚未发布的 Beta.3 标签并继续发布；根因修复由 TASK-20261018-015 跟踪，标签替换必须先确认无公开 Release 并使用预期旧对象锁定。

- 已确认远端权限／main 快进关系和新标签不存在，保留 Beta.1／Beta.2 资产基线。首次读取 Beta.1 元数据遇到网络 EOF，重试只读请求成功。
- 前两次最终版本构建先后被格式门拦截：版本文件需 gofmt，前端 package.json 需 Prettier（Windows 文本写入带入 CRLF）；均仅格式化本任务的两个版本文件，未跳过检查。最终完整发布构建使用 Git Unix 工具路径与 NSIS，避免既有构建变量查找 uname／tail 的环境告警。

- 独立 QA 脚本首次执行因 PowerShell 保留变量 HOME 命名冲突在安装前停止；仅修正忽略目录中的夹具变量为 qaHome，未修改产品代码或生产数据，随后重跑。

- 独立 QA 首次安装实际成功，随后开始菜单断言使用了隔离用户目录切换前的路径而失败。只读核对确认桌面／隔离开始菜单的目标和显式图标均正确；通过新进程查询确认父 PowerShell 缓存了切换前的 Known Folder；改由继承隔离环境的新进程读取期望位置，从已成功安装状态重跑全部快捷方式断言并继续启动、运行中覆盖和卸载验收，未删除或放宽断言。

# Implementation Result

已同步应用／前端／Windows 产品完整版本为 2.0.0-beta.3，数字资源保持 2.0.0.0；新增准确的 Codex 修复发布说明，保留真实账号验收边界。现有 Release 工作流和安装器实现没有修改。最终本地完整发布构建已通过，独立用户级安装生命周期验收最终通过。main 与 Beta.3 标签已经推送，但远端 CI 暴露既有账号版本一致性缺陷；Release 已主动取消且未公开，等待修复和对既有标签处理的明确授权。

# Verification

- PASS：TASK-015 根因修复后重新完整构建发布候选，质量门／575 项前端测试、NSIS、原生资源和摘要成功，独立 Codex 契约连续三次通过；新候选 tmp/beta3-release-20261018/fixed-local-assets，未复用修复前产物。

- PASS：最终 scripts/build-release.ps1 -Version v2.0.0-beta.3，完整 task --force verify（104 文件／575 项前端测试及 Go／格式／静态／类型／Windows production 构建）、NSIS 打包、Windows 原生资源及两项发布资产 SHA256 校验成功。日志 tmp/beta3-release-20261018/build-release-verified.log。
- PASS：go test ./internal/version ./internal/infrastructure/update，版本及更新逻辑定向验证。
- PASS：独立用户级首次安装、显式图标、历史库哈希保留、v2 迁移／重启、运行中覆盖／卸载、已移除桌面入口不重建、未知文件与两库保留全部通过。夹具目录修正后重跑全部对应断言，结果 tmp/beta3-release-20261018/isolated/result.json。正式应用仍健康。
- FAIL：GitHub CI 39128657083，源 fc7a00845adef914ea8a703efdfa82c7cfac9940。Go 测试 TestStepOptionsLifecycleAndVersionIsolation 报 stale target read key；TestQueryCannotPublishResourcesAfterRegionOrKeyTypeChanges/billing 报 stale resource published。Codex 新增测试通过，不代表整体质量门成功。
- 根因分析：account.Update 直接使用 now()，而 now() 仅为 time.Now().UTC().Format(time.RFC3339Nano)，未确保新 updated_at 与旧版本不同；UseAPIKeyAt 与 MiniMax 结果发布依赖该值判断旧快照。快速操作在低分辨率／重复时钟下可能共用同一版本。需确定性相同时钟回归与根因修复，不能靠睡眠或反复重跑隐藏问题。
- CANCELLED：Release 39129204984 已取消，在 Install build tools 阶段结束，发布作业未执行；gh release view v2.0.0-beta.3 返回 release not found。没有公开 Beta.3 安装包，未执行公开下载／更新发现验收。
- PASS：main 已普通快进推送；远端新注释标签的 peeled SHA 与构建源码相同。旧标签未改写，正式 Latest 仍为 v1.3.1。
- PASS：Codex 配置 SHA256 与基线一致，生产 PID 8124 未重启，生产健康接口仍为 2.0.0-beta.2。

# Git

用户已授权普通推送和新的 Beta.3 发布。

- 源提交／已推送 main：fc7a00845adef914ea8a703efdfa82c7cfac9940。
- 已推送注释标签对象：266fe44dca4698a8fca67933dfdce06f32210e82，v2.0.0-beta.3，peeled 至上述源提交。
- 标签已经共享，不擅自移动／删除。修复后若继续使用 Beta.3，需要用户明确授权重建尚未发布的该标签；否则另行确认新版本号。

# Result

未完成公开发布。已完成版本、构建、原生安装验收及 main／标签推送；远端 CI 检出账号版本隔离缺陷，已阻止公开 Release。用户已明确授权先修复该缺陷，再移动已经推送但尚未发布的 Beta.3 标签；当前由 TASK-015 实施修复与验证，尚未重新发布。当前 Agent 配置及本机生产安装保持不变。