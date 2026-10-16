# TASK-20261016-008: 修正 Google 订阅标签的品牌误用

## Status

In Progress

## Created

2026-10-16 22:55

## Request

用户截图指出 Google 提供商卡片错误显示“ChatGPT / Codex”。要求修正此次接入遗漏。

## Goal

提供商卡片与相关账号展示使用所属提供商的订阅名称，Google 不再显示 OpenAI 品牌。

# Analysis

## Root Cause

`ProviderCard` 将通用认证类型 `subscription_login` 直接翻译为仅适用于 OpenAI 的
`subscription.type`；配额账号卡片也存在同样遗漏。账号详情／总览已有 Google 专用标签。

## Scope

仅修正订阅品牌标签与最小共享选择逻辑，复用现有中英文翻译，保持认证／转发行为。
检查相关通用订阅文案，保留原有其他任务记录且不纳入提交。

# Implementation Plan

1. 按提供商品牌选择订阅标签，并统一已存在的重复判断。
2. 补充 Google／OpenAI 品牌回归检查，核对原生提供商卡片。
3. 运行相关检查与质量门，检查差异、提交并归档。

# Implementation Result

提供商卡片、配额卡片和账号详情／总览共享 `subscriptionTypeKey`，根据提供商 kind
或内置 ID 返回 Google／OpenAI 标签；未知提供商显示中性“订阅登录”，不冒用品牌。
复用 Google / Antigravity 的中英文标签，为中性回退补齐中英文翻译。
通用编辑／删除提示经核对没有 OpenAI 品牌误用，无需修改。

# Verification

- 中英文提供商卡片、OpenAI 保留标签、未知提供商回退、Google 配额页无 Codex 查询等回归通过。
- 相关 3 文件／42 项前端测试通过。
- `task verify` 全部通过：格式、lint、类型检查、Go 测试、85 个前端文件／456 项测试与 Windows 构建。
- 实际开发窗口的提供商页可见 Google / Antigravity 和 OpenAI 的 ChatGPT / Codex 各自标签。
  无需重新登录或发送推理请求。既有 uname／tail 与 bundle 体积提示不影响质量门成功退出。

# Documentation

更新 CURRENT 和任务结果；无新的产品能力或架构变更。

# Git

待提交。

# Result

已修复截图中的品牌误标，并同步消除配额页同类问题；账号与提供商标签共用相同选择逻辑。
