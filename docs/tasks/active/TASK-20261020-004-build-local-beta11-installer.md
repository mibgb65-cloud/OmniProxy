# TASK-20261020-004: 构建 Beta.11 本地生产安装包

## Status

In Progress

## Created

2026-10-20

## Request

修复完成后，构建新的 Beta 生产版本，供用户自行进行本地更新。

## Goal / Scope

- 同步版本为 `2.0.0-beta.11`，编写随包发行说明。
- 从冻结的本地提交隔离执行现有发布脚本，完成完整质量门、Windows amd64 production 构建及 NSIS 打包。
- 校验原生版本、资产与 SHA256，交付 `release-assets/v2.0.0-beta.11/` 中的安装包。

不自动安装、停止生产、修改真实账号／凭据、推送、打远端标签或公开发布；不纳入其他未跟踪任务记录。

## Analysis

当前源码及最近本地安装包为 Beta.10。新版本包含 Responses 长会话重连修复 `a467ffe`，以及其后已提交文档；沿用既有机器级安装身份和路径。

相对 Beta.10 构建源 `21b5e2a`，还包含已提交的授权进度／账号删除修复 `4d0e76c` 和账号轮换设置 `58521d1`（迁移 035）。发行说明需覆盖这些实际变更，不宣称尚未实现的额度快照持久化已交付。

隔离构建只导出冻结提交的受跟踪文件，排除真实配置、凭据及现有未跟踪任务。已保存旧资产和已安装生产程序共 11 个文件的摘要及生产进程身份基线。

## Implementation Plan

1. 核对版本映射、发行说明和现有发布脚本，更新版本及相关回归预期。
2. 提交构建源并导出隔离源码，使用既有 `scripts/build-release.ps1` 完整构建、打包与验证。
3. 独立复核生产标签、原生版本、资产 SHA256 与来源一致性，更新 CURRENT 和任务记录并交付。

## Implementation Result

- Go／前端／Windows ProductVersion 同步为 Beta.11，数字资源及 NSIS 保持 `2.0.0.0`／`2.0.0`。
- 新增 Beta.11 随包发行说明及三种版本读取回归；涵盖实际已提交功能，明确额度快照持久化未实现与未签名／未安装验收边界。
- 下一步冻结构建源并运行既有发布脚本。

## Verification

- PASS：`scripts/set-release-version.ps1 -Version v2.0.0-beta.11`。
- PASS：版本文件／发行说明读取测试的定向 Prettier 检查。
- PASS：`pnpm exec vitest run src/features/about/current-release.test.ts`，26 项通过。
- 待执行：隔离完整质量门、打包及独立资产／版本核验。

## Documentation

新增 Beta.11 发行说明；完成后更新 CURRENT。

## Git

待本地提交；不推送。

## Result

待完成。
