# TASK-20261020-013: 构建 Beta.14 本地生产安装包

## Status

In Progress

## Created

2026-10-20

## Request

用户要求构建新的 Beta 生产版本，供其自行本地更新。

## Goal

基于当前已提交功能构建 `v2.0.0-beta.14` Windows amd64 production 全量安装器与 SHA256 文件，交付独立资产，不自动安装或发布。

## Scope

- 同步应用、前端及 Windows 版本元数据，编写离线发布说明。
- 冻结本地提交并导出受跟踪源码，在独立目录执行原有发布脚本、完整质量门和 NSIS 打包。
- 校验安装器／应用原生版本、production 标签、摘要及构建来源。
- 保持旧发布资产、已安装程序、生产账号数据和无关草稿不变。

## Out of Scope

- 自动安装、运行安装器、停止用户应用、推送代码／标签或发布 GitHub Release。
- 将 DeepSeek 文本通道包装为支持图片，或将美元参考估值表述为官方余额。
- 修改未跟踪的 Atria 任务草稿或推进 TASK-008 的客户端试验验收。

# Analysis

当前最新本地安装器和应用／前端版本为 Beta.13，目标 Beta.14 尚不存在。
本次包含已提交的美元额度估算、图片历史安全诊断，以及整理到 main 的启动网关偏好；
旧版 Codex 登录同步试验工具不由安装包自动执行。

沿用 `scripts/build-release.ps1`：完整 `task --force verify`、production 构建、
NSIS 打包、原生版本和发布资产验证。完整版本改为 `2.0.0-beta.14`，
Windows 数字版本仍为 `2.0.0.0`，产品安装身份不变。

NSIS 已安装。只导出受跟踪源码到新的 `tmp/local-beta14-20261020/source/`；
不复制真实数据、不创建长期工作树，不包含未跟踪 Atria 草稿或前一任务的合成验收目录。

# Implementation Plan

1. 核对版本与发布脚本，记录原有资产／已安装程序保护基线。
2. 更新 Beta.14 版本及发布说明，通过版本映射检查并提交冻结源码。
3. 导出受跟踪源码至独立目录，运行原发布脚本完成完整验证和生产打包。
4. 核对版本、来源、SHA256、签名状态及保护基线，更新任务记录并交付安装器。

# Implementation Result

待完成。

# Progress

- 已同步应用、前端和 Windows 完整版本为 Beta.14，数字资源与安装身份保持不变。
- 已新增随包发布说明，以及普通版本、带 v 前缀和本地构建元数据三种运行版本的说明读取回归。
- 已在忽略目录保存 19 个旧资产／已安装程序文件、1 个生产进程的保护基线。
- `scripts/set-release-version.ps1 -Version v2.0.0-beta.14` 与定向 Prettier 检查通过。

# Verification

待验证。

# Documentation

更新发布说明、CURRENT 及本任务记录；不改变架构或技术栈。

# Git

待记录实际提交。

# Result

待完成。
