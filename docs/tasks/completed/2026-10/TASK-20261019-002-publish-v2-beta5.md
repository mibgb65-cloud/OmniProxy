# TASK-20261019-002: 汇总 main 修复并发布 v2.0.0-beta.5

## Status

Completed

## Created

2026-10-19

## Request

把之前的修改都合并到 main 分支，并且推送发布 beta5 版本。

## Goal

核对已有分支与 main 的包含关系，将已完成的图片输入和 Codex 额度自动激活修复推送至 main，发布可下载且经过验证的 v2.0.0-beta.5。

## Scope

- 审计待发布提交及历史分支包含关系，保留无关未跟踪文件。
- 统一完整 Beta.5 版本、发行说明；执行完整质量门与隔离安装生命周期验证。
- 正常推送 main，等待精确源提交 CI 通过，再创建新标签并运行 Release。
- 验证公开预发布、真实更新器发现与下载、校验和及旧版本不变。

## Out of Scope

- 不自动安装、重启生产软件或当前 Agent；不改共享 Codex 配置、登录或生产数据。
- 不覆盖旧标签或发行资产，不改仓库默认分支与稳定版 Latest。
- 不重复真实额度激活或付费模型请求；未完成草稿不混入发布。

# Analysis

## Existing Behavior

main 本地已有四个未推送提交，包含 b2eb0a1 图片输入修复与 d51455e 额度窗口自动激活修复及各自验证归档；当前版本为 Beta.4。

## Related Modules

internal/version/version.go、frontend/package.json、build/windows/info.json、发布脚本、GitHub CI/Release、docs/releases、docs/plans/CURRENT.md。

## Risks

新迁移 034 必须保留既有额度尝试记录；外部配置保护不能绕过；公开构建的哈希须以实际下载资产为准，不能混用本地候选包。网络暂时失败时先核对已有运行，避免重复发布。

# Implementation Plan

1. 核对本地/远程分支、发行基线、当前生产健康和配置哈希。
2. 更新 Beta.5 版本与发行说明，执行完整发布构建和隔离安装/升级/卸载验收。
3. 审查差异、提交并正常推送 main，等待精确提交 CI 成功。
4. 创建并推送唯一 Beta.5 标签，等待 Release 构建及发布成功。
5. 实测公开更新器与资产，确认旧发行/稳定版/运行软件未变；归档并推送验收文档。

# Progress

- 已核对工作树：当前 main，已有四个本任务待发布提交，无已跟踪文件改动；用户既有未跟踪记录不纳入提交。

- 远程 main 可快进，Beta.5 标签与发行均不存在；已保存旧 Beta.1–4／v1.3.1 发行和标签、共享配置哈希及生产 Beta.4 健康基线。
- 历史分支审计：网关／DeepSeek／精确路由已随 44fdbc4 实现快照及 fdfd29e 文档纳入 main；Claude 配置两修复已分别等价纳入 6285c0c／fa69dc6，任务归档内容相同。其余分支均为 main 祖先，不用制造重复合并或重放旧代码。

# Implementation Result

- 统一应用／前端／Windows ProductVersion 为 2.0.0-beta.5，数字版本保持 2.0.0.0，不修改技术栈或发布工作流。
- 新发行说明明确图片能力、独立 16 MiB、额度滑动观测与未知结果保护、迁移 034 及重新应用配置的边界。
- 本地完整发布构建与独立用户级安装／升级／卸载通过；main 正常快进推送、精确提交 CI、新标签 Release 和公开资产验证均已完成。

# Verification

- PASS：scripts/build-release.ps1 -Version v2.0.0-beta.5；内部 task --force verify 全部通过，Go／静态检查／格式／类型、104 文件／576 前端测试、Windows 生产构建及 NSIS 打包成功。既有 bundle 和 jsdom scrollTo 提示不影响门禁。
- PASS：Windows 原生资源与本地安装器／SHA256 资产验证。
- PASS：独立产品名／登记键、目录及 15105／15895 端口的用户级静默安装、图标、旧统计库保留、真实应用启动与重启、运行中覆盖升级／卸载、尊重已删除桌面入口、保留未知文件与新版数据，共 11 项全部通过。
- PASS：隔离构造 schema 033、禁用合成账号和旧发送登记；实际 Beta.5 启动迁移至 034，原目标／周期／发送时间原样保留，新增观测与确认字段为空，未自动开启激活。
- 机器级 UAC／真实 Desktop 外壳／真实账号自动激活本轮不冒充已验证。历史图片真实验收沿用 TASK-20261018-020，不重复付费请求。
- PASS：CI 39189311494 与 Release 39190351905 的 headSha 均为 1310382cb02c5e3ba8ed014b32e030e812f6bab3，最终全部 success。只推送新标签，无强制推送、无重写历史。
- PASS：Release ID 416854522，2026-10-19T02:47:39Z（新加坡 10:47:39）公开，draft=false／prerelease=true；注解标签对象 5ad0e93364b101500717e0c9d3439a590ab3fa29 指向上述源提交。
- PASS：真实默认更新器稳定渠道仍为 v1.3.1，Beta 渠道为 v2.0.0-beta.5；Beta.1–4 均判定其为更新版本。默认下载客户端实际经合法 GitHub 重定向下载安装器及校验文件，并运行 VerifyFile，无网络替代或模拟公开响应。
- PASS：公开安装器 19,299,192 bytes，SHA256 a12b4a554051a0e60d6bd7ebfbce9184f660c39a0d523eed9186fb913b894adb；114-byte 校验文件 SHA256 0d3eff1285eefb8c023d58ac6352e0845ee954972bffe990f6c134647fffdc8e。两文件大小及实际内容与 GitHub digest 一致，发布资产脚本及 Windows 原生版本校验通过。
- PASS：Beta.1–4／v1.3.1 的发行 ID、正文哈希、资产 ID／大小／digest 及旧标签对象与发布前一致；正式 Latest 保持 v1.3.1。
- PASS：共享 Codex 配置哈希、生产健康及 PID 9440／启动时间与发布前一致，仍运行 Beta.4；未自动安装或重启。
- 安装器 Authenticode 为 NotSigned，已在发行说明披露。

# Documentation

已更新发行说明、UPGRADE 与 CURRENT；已有功能修复任务归档保留历史验收边界，不把当时“仅本地”的历史事实覆盖成伪造验收。

# Deviations

None.

# Git

- b2eb0a187969663552eda944e396c5ef01a54a20：原生图片能力与请求容量修复。
- d51455e66f99331be940741466b0c66d2a5a74cc：额度滑动占位窗口自动激活修复。
- 1310382cb02c5e3ba8ed014b32e030e812f6bab3 chore(release): 准备 v2.0.0-beta.5；包含此前四个待推送提交，已正常推送 main，为唯一 Beta.5 发布源。
- 发布验收归档另作 docs-only 提交推送 main，不移动发行标签。

## Remote Evidence

- https://github.com/mibgb65-cloud/OmniProxy/actions/runs/39189311494
- https://github.com/mibgb65-cloud/OmniProxy/actions/runs/39190351905
- https://github.com/mibgb65-cloud/OmniProxy/releases/tag/v2.0.0-beta.5

# Result

已完成的修复全部在 main 并已推送，v2.0.0-beta.5 已公开且实际下载校验成功。历史版本和当前软件／Agent 保持原状；需要用户自行升级，图片配置须预览确认后重新应用，真实自动激活效果需升级后核验。

# Completed

2026-10-19 10:50
