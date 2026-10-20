# TASK-20261020-001: 持久保留订阅额度与重置时间

## Status

In Progress

## Created

2026-10-20

## Request

已经成功获取过的额度及重置信息应持续留存，软件退出重进或页面切换都不清除；后续正常获取新数据时再更新。

## Goal

恢复最近成功的额度／重置时间，后台刷新不闪回占位，刷新失败保留旧数据；持久数据必须按账号身份及运行环境隔离。

## Scope

- 检查订阅额度读取、前端查询缓存和账号身份语义。
- 最近成功快照的持久保存／恢复、后续更新与错误处理。
- 覆盖页面切换、重启、过期快照、失败及账号身份隔离回归。

## Out of Scope

- 不修改真实账号或 Token，不额外发送模型／额度激活请求。
- 不把旧额度当作实时数据或推算新的重置时间。
- 不发布、安装或替换生产程序。

# Analysis

根因：前端只保留 10 分钟的内存缓存，进程退出没有恢复来源。
采用当前 profile SQLite 中的有界公开额度快照，随账号删除级联清理；按提供商、成员邮箱、工作区隔离，身份字段补全时以同一隔离授权配置作为辅助校验。
账号详情和额度总览共用查询选项，先恢复本地快照，再串行查询最新数据。首页保留被动订阅共享缓存、不在挂载时批量调用官方接口的既有行为。
显示合并保留暂缺窗口／时间；调度和自动激活继续只读取真正的实时查询结果，不能把留存数据当成新鲜观测。

# Implementation Plan

1. 阅读相关规范与现有额度缓存实现，确认持久化层及身份隔离边界。
2. 实施最小闭环，成功写入、缓存先显示、刷新失败不清空，补齐回归。
3. 执行针对性验证与 `task verify`，更新受影响文档及 CURRENT。
4. 审查差异、本地提交并归档任务；不纳入其他已有未跟踪文件。

# Progress

- 已建立任务记录，尚未修改业务代码。
- 已完成迁移 036、事务保存／本地恢复、显示与实时数据分离，以及前端恢复后刷新。
- 定向四个后端包通过；数据库重开、错误保留、部分窗口、身份隔离、级联清理等关键回归连续三轮通过。
- 六个前端文件共 35 项通过；补充留存状态概况回归后两个文件共 13 项通过，正在运行完整质量门。
- 最终完整质量门通过：110 个前端文件／639 项测试、全部 Go 测试、格式／lint／类型检查与 Windows 构建完成。

# Implementation Result

## Changes

- 成功获取的套餐、额度百分比、绝对重置时间与获取时间写入 profile 自有 SQLite，每账号仅一条，页面切换、内存 GC 和进程退出都不淘汰。
- 本地恢复先于官方查询队列，多个组件共享一次恢复和一次实时读取；正常刷新原位更新，失败保留旧快照和安全错误。
- 官方暂缺窗口或重置目标时只对界面合并上次信息，带留存标记；过期保留原值并显示待刷新，不推算满额或新目标。
- 调度／自动激活继续只使用实时原始数据和内存观察，显示历史不会成为新鲜权重、可用额度或激活依据。
- 缓存按本地账号、提供商、成员和工作区隔离；工作区暂缺／补全时辅以同一隔离授权配置摘要。相同已确认身份重新授权保留，换身份不复用；退出登录隐藏但留存，删除账号级联清理。
- 存储内容限定为公开显示信息与身份摘要，不包含凭据、明文身份、重置卡或上游正文。保存／校验失败不清空旧记录，返回稳定错误。

## Files Changed

- `migrations/036_subscription_quota_snapshots.sql`。
- `internal/application/subscription/quota_cache*`、`quota_cached.go`、`quota_snapshot.go`、额度读取与诊断。
- Wails 订阅初始化、额度绑定与工具生成的绑定。
- 共用额度查询、额度卡片／概况状态、双语言文案与回归。
- ADR-019、PRD、架构概览、订阅／配额说明、CURRENT 和本记录。

## Technical Notes

无新依赖、Core API 或凭据格式变化。倒计时仍按原绝对目标和当前时间计算，应用退出期间不会冻结成旧剩余时长。现有 60 秒 stale／10 分钟内存 GC 策略不变，本地数据库独立持久留存。

# Verification

## Tests

- PASS：`go test ./internal/application/subscription ./internal/application/quotaactivation ./internal/application/gateway ./internal/transport/wails -count=1`。
- PASS：`go test ./internal/application/subscription -run 'TestQuotaSnapshot|TestQuotaCache|TestPartialQuota' -count=3`。
- PASS：前端类型检查；持久化／刷新／额度展示／总览／首页共六个文件，35 项测试。
- PASS：留存状态概况及持久化两个文件，13 项测试（与上一组存在重叠，不按独立数量相加）。
- 数据库新连接及新服务实例恢复，不打开官方组件；实时部分缺失与旧显示分离；存储损坏、非法值、旧版本／旧时间写入、成员／工作区、退出／删除边界均有覆盖。
- 前端覆盖换页和新 QueryClient 恢复、慢查询、失败、后续成功、过期不满额、多个组件共享、同工作区换成员，以及本地恢复失败后的正常查询。

## Quality Checks

- PASS：`task verify BIN_DIR=tmp/quota-persistence-20261020/verify-bin`，exit 0。格式检查、Go vet／staticcheck、ESLint、TypeScript、全部 Go 测试、110 个文件／639 项前端测试、Wails 绑定生成、实际前端生产构建与 Windows amd64 production 构建通过。
- PASS：`git diff --check`。
- 现有 jsdom scrollTo 未实现、Vite 大 chunk／插件耗时提示为非阻断信息，未跳过或删除测试。
- 验证期间仓库有其他聊天独立提交的长会话修复和 Beta.11 版本准备，已核对差异；本任务不修改其版本、资产或任务记录，不将其未跟踪文件纳入提交。

## Limitations

只执行隔离数据库、模拟官方组件和组件测试，未主动读取真实账号或执行真实额度查询／激活；原生 GUI 与安装版重启的人工验收未执行。
更新前已经丢失且从未落盘的数据无法凭空恢复；使用本功能后成功获取的数据会持久保留。本次不发布或替换生产程序。

# Documentation

已同步 PRD、Architecture Overview、SUBSCRIPTION、QUOTA、CURRENT，新增 ADR-019；技术栈和构建规范不变。

# Deviations

选择本地数据库而非浏览器 localStorage 或单纯延长内存缓存，复用 profile／账号生命周期和身份隔离。
留存只用于显示，不放宽实时额度、调度或自动激活判定。

# Git

待审查与本地提交。

# Result

待完成后填写。
