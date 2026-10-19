# TASK-20261019-013: 构建 Codex 兼容修复的本地 Beta.9 生产安装包

## Status

In Progress

## Created

2026-10-19（Asia/Singapore）

## Request

用户要求完成后在本地构建生产环境的 Beta 安装包，由用户自行安装测试。

## Goal / Scope

将已完成的 Codex Desktop 文件变更与上下文压缩修复生成 Windows amd64 本地生产 Beta 安装包，核对完整版本、Windows 元数据与 SHA256，交付安装路径。不自动安装、不发布 GitHub、不创建／推送标签，不修改真实账号、数据、客户端配置或生产实例。

## Analysis

当前最新本地资产为 Beta.8，Codex 兼容修复提交 697942b 已完成质量门与双核心离线验收。本次递增为 2.0.0-beta.9，保留 Beta.6／7／8 资产和安装身份。现有 build-release.ps1 校验源码／前端版本并映射 Windows 数字版本，强制运行 verify、package、原生元数据和 SHA256 资产校验。

为避免覆盖主工作区 bin/，从仅暂存本任务版本、随包说明及回归的确定 Git tree 导出忽略目录中的隔离源码；在该源码中执行原有发布脚本。构建完成后提交相同 tree，再单独更新结果与归档。不引入构建捷径、业务依赖、数据库迁移或 updateqa 标签。

## Implementation Plan

1. 同步 Go／前端 Beta.9 版本及 Windows 元数据，补充文件变更／压缩修复的本地发布说明和随包说明回归。
2. 保存生产／开发二进制与真实 Codex 配置只读基线，仅暂存本任务文件并导出确定 Git tree。
3. 在隔离源码执行完整生产质量门与 NSIS 打包，输出到独立空 Beta.9 资产目录；校验产品版本、SHA256 和签名状态。
4. 核对构建来源与现有生产实例，记录未执行的安装／真实账号／GUI 验收，提交版本与任务归档。

## Implementation Result

待完成。

## Verification

待完成。

## Git

待记录。

## Result

待完成。
