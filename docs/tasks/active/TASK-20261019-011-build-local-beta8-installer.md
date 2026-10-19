# TASK-20261019-011: 生成额度续期修复的本地 Beta.8 生产安装版

## Status

In Progress

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

待完成。

## Verification

待完成。

## Local Artifact

待生成。

## Git

待记录。

## Result

待完成。