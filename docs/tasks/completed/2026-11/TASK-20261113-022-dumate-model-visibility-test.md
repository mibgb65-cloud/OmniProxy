# TASK-20261113-022: 补充 DuMate 预设模型可见性回归

## Status

Completed

## Created

2026-11-13 21:58（Asia/Singapore）

## Request

> 模型在哪看？

## Goal

- 用测试确认并固定 DuMate 两条预设模型在何处可见、何时可见：
  没有已登录账号时不进入路由目录，导入后出现在模型目录、默认映射与 `/v1/models`。

## Background

- TASK-20261113-020 实现 DuMate 时只在适配器与路由层加了预设逻辑，
  没有针对「目录可见性」的专项回归。
- 用户更新到 `v2.2.3-beta.24` 后询问在哪里查看模型，需要给出可核对的位置，
  同时避免后续改动让模型在无账号时误发布或导入后仍不出现。

## Scope

- 新增 `internal/application/routing/dumate_test.go`。

## Out of Scope

- 不修改业务代码、版本号、安装包或前端。
- 不重跑完整 `task verify`（只新增测试文件）。

## Existing Behavior

DuMate 适配器公布 `model-text`、`dm-auto-model/text.L0` 两条别名；
路由层只在存在 `dumate_cookie` 且 `authenticated` 的启用账号时发布它们。

## Expected Behavior

无账号时 `Resolve` 不可路由、目录为空；导入账号后两条模型都可路由、
出现在 `Catalog` 与 `DefaultRoutes`，且不发布未验证的上下文容量。

---

# Implementation Result

- 新增 `TestDuMatePresetModelsAppearOnlyWithAuthenticatedAccount`，覆盖：
  - 无账号时不发布、不可路由；
  - 登录账号存在时两条别名均可路由到 `builtin_dumate`；
  - 目录条目协议为三种、只声明文本、上下文容量保持未知；
  - 默认映射包含两条模型；
  - 账号停用、改成 `api_key`、登录态失效或提供商停用时均不再路由；
  - 恢复登录态后重新可路由。

# Verification

- `go test ./internal/application/routing/ -count=1` 通过（包内全部测试）。
- `staticcheck ./internal/application/routing/` 无输出。
- 未重跑 `task verify`：仅新增测试文件，不改变生产代码或构建产物。

# Git

- Commit: `c570dd53`（test(routing): 覆盖 DuMate 预设模型目录与路由资格）

# Result

已用回归测试固定 DuMate 模型可见性，并据此回答用户「模型在哪看」。
