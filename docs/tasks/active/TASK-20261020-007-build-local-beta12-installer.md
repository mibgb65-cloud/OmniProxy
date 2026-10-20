# TASK-20261020-007: 构建 Beta.12 本地生产安装包

## Status

In Progress

## Created

2026-10-20

## Request

用户要求修复完成后构建新的 Beta 生产版本，供其自行本地更新。

## Goal / Scope

- 将版本同步为 `2.0.0-beta.12`，编写准确的随包发行说明。
- 包含 Beta.11 之后已提交的额度快照持久化、River 模型／分页优化、
  订阅状态刷新误报修复。
- 冻结本地构建源，以既有发布脚本隔离执行完整质量门、production 构建和 NSIS 打包。
- 校验 Windows 原生版本、来源、安装器与 SHA256，交付独立 Beta.12 资产目录。

## Out of Scope

不自动安装、停止或替换生产程序，不修改真实账号／凭据／客户端配置；
不创建发布标签、推送或公开发布；不纳入无关未跟踪文件。

---

# Analysis

当前完整版本及最近本地安装包为 Beta.11。构建必须使用生产标签和既有安装身份，
不能将开发运行包作为更新安装器交付。Beta.11 之后的实际变更需逐项写入发行说明。

已确认包含额度留存 `f5de7ca`、River／分页 `3dd7426`、订阅刷新 `ac9d5a7`，
沿用原有 Windows 数字版本 `2.0.0.0`、NSIS 版本 `2.0.0` 与机器级产品身份。
NSIS 可用，目标 Beta.12 资产目录不存在或为空；已有资产及已安装程序共 15 个文件、
1 个生产进程的摘要／启动时刻基线已保存在本任务忽略目录。

---

# Implementation Plan

1. 核对发布规范、脚本、版本读取测试和旧资产／生产程序基线。
2. 更新版本与发行说明，运行定向检查，提交并冻结构建源。
3. 只导出冻结提交的受跟踪文件，隔离运行发布脚本的完整质量门及打包。
4. 独立核对生产标记、Windows 版本、SHA256 和来源一致性，确认旧资产不变。
5. 更新 CURRENT 和实际验证记录，归档任务并交付安装包。

---

# Progress

- 已创建任务记录，尚未修改版本、配置或发布文件。
- 已同步 Go／前端／Windows 完整版本为 Beta.12，新增随包发行说明和三种版本读取测试。
- 旧资产与生产程序只读保存基线，不读取真实配置或凭据。
- 版本映射脚本与发行说明定向 Prettier／29 项回归已通过，准备冻结构建源。

---

# Implementation Result

待实施。

---

# Verification

- PASS：`scripts/set-release-version.ps1 -Version v2.0.0-beta.12`。
- PASS：前端包与发行说明测试文件的定向 Prettier 检查。
- PASS：`pnpm exec vitest run src/features/about/current-release.test.ts`，29 项测试通过。
- 完整发布脚本、NSIS、原生版本及独立资产校验待执行。

---

# Artifact

待构建。

---

# Git

待本地提交；不推送。

---

# Deviations

暂无。
