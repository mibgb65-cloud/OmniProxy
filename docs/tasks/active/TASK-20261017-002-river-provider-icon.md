# TASK-20261017-002: 补齐 River 品牌图标

## Status

In Progress

## Created

2026-10-17（Asia/Singapore）

## Request

用户反馈 River 账号卡片显示通用机器人图标，希望优化为品牌图标。

## Goal

使用官方 River 素材，使提供商、账号和其他共享图标入口保持一致。

# Analysis

共享组件 `provider-brand-icon.tsx` 缺少 River 品牌资源映射，触发通用 Bot 回退。沿用现有本地静态素材、固定尺寸及装饰语义，无需变更后端、模型或账号数据。

官方首页引用的 192 × 192 favicon 自带深蓝底和品牌留白；保留原始 PNG，在既有 36 × 36 圆角容器内完整展示，避免再套白色底块造成双重边框或缩小标志。静态本地资源离线可用，所有共享入口同时修复。

# Implementation Plan

1. 核对官方图标，保存本地素材与来源。
2. 在共享组件增加 River 映射，检查现有品牌回归覆盖。
3. 完成必要检查及原生深浅主题验收，记录结果并提交归档。

# Implementation Result

- 新增 River 官网原始 192 × 192 PNG，本地离线加载，SHA-256 与下载原件一致；来源及商标说明记录于 `frontend/public/provider-icons/SOURCE.md`。
- 共享图标组件增加 `river` 映射，使用完整底块呈现模式，保留官方底色、留白、比例与既有 36 × 36 圆角尺寸；`aria-hidden` 和空 alt 保持装饰语义。
- 提供商、账号总览／详情和配额入口沿用该组件，无需单独修改调用处。既有品牌样式保持原逻辑。
- 现有品牌测试矩阵纳入 River，防止再次回退 Bot。

# Verification

- `task verify BIN_DIR=tmp/river-icon-20261017/verify-bin`：PASS，退出码 0；格式、Go vet／staticcheck／ESLint、类型检查、Go 全套、前端 88 文件／473 测试与 Windows 生产构建全部通过。
- 原生验收：按进程路径定位当前项目 `bin/omniproxy.exe`，确认 HMR 后的 River 账号卡片已显示官方图标；在亮色／暗色主题中截图核对完整比例与固定尺寸，随后恢复原暗色主题与提供商页面。
- 本机证据：`tmp/river-icon-20261017/river-header-{dark,light}.png`、`native-verification.json`、`task-verify.log`（忽略，不提交）。未读取完整密钥、修改账号或调用推理。
- 官方 PNG 签名、192 × 192 尺寸与 SHA-256 已核对；`git diff --check` 通过。改动使用 UTF-8 无 BOM／LF，既有其他任务记录保持原样。
- UI/UX 技能首次检索未命中目标；收窄至 icons 后采用装饰图标隐藏语义建议，继续遵循现有仓库图标规范，未引入新的图标库。

# Documentation

更新素材来源、CURRENT 与本记录；无产品能力或架构变化。

# Remaining Risks

当前修复针对内置 `river` 类型；使用未知种类或自定义中转域名仍采用原有通用图标。未引入远程运行时图片依赖。

# Git

待提交。

# Result

待完成。
