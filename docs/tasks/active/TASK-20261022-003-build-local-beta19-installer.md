# TASK-20261022-003: 构建 Beta.19 本地生产安装包

## Status

In Progress

## Created

2026-10-22（Asia/Singapore）

## Request

用户要求继续构建新的 Beta 生产版本，用于自行本地更新。
上一轮只读核对后曾中断，尚未修改版本或启动构建。

## Goal

交付包含已提交 Fast 能力修复及关于页正式入口的 Windows amd64
`v2.0.0-beta.19` production 安装包和匹配 SHA256。

## Scope

- 核对既有发布脚本、版本映射和 Beta.18 产物。
- 同步完整版本、Windows 资源和离线更新日志。
- 在隔离产物目录执行完整门禁、生产打包和资产校验。
- 核对安装器／应用版本、来源、校验和与真实环境保护基线。

## Out of Scope

- 自动安装、重启或替换正在运行的程序。
- 修改真实账号、凭据、数据库和客户端配置。
- 创建／推送公开发布标签或发布 GitHub Release。
- 混入 Gemini 任务状态与 Atria 草稿等无关未提交改动。

# Analysis

当前源码与最新本地产物为 `2.0.0-beta.18`，新增功能已在 main。
标准发布脚本已核对：完整 SemVer 为 Beta.19，Windows 数字资源仍为
`2.0.0.0`，安装器数字字符串为 `2.0.0`，安装身份保持不变。
沿用 Beta.18 的冻结提交／`git archive` 隔离源码构建，执行标准脚本内的
`task --force verify` 和 `task --force package`，不将无关工作树改动混入来源。

本版新增内容是官方服务档位保留（Fast 入口）和关于页正式项目／协议入口；
继承 Beta.18 的内存与周额度修复，不宣称解决 Google 429 或完成真实模型调用验收。
新版本目录尚不存在，全部工具已安装；保留旧发布资产，不覆盖非空目录。

# Implementation Plan

1. 确认影响范围、保护基线和标准发布流程。
2. 升级到 Beta.19，补充真实变更日志并仅提交相关文件。
3. 基于精确源码完成完整门禁和 NSIS 生产安装器构建。
4. 验证原生版本、SHA256、源码来源及保护基线。
5. 归档记录并交付安装包路径和本地更新说明。

# Implementation Result

待完成。

# Verification

待完成。

# Git

待本地提交后记录；不推送。

# Result

待完成。
