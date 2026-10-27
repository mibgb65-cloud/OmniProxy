# TASK-20261027-006: 将 GitHub 默认分支设置为 main

## Status

Completed

## Created

2026-10-27（Asia/Singapore）

## Request

将 main 分支设置为仓库显示的主分支。

## Goal

将 `mibgb65-cloud/OmniProxy` 的 GitHub 默认分支从 master 改为 main，使仓库首页展示 v2 README，保留 master 和所有既有历史。

## Scope

- 使用现有管理员权限修改默认分支，核对 GitHub API 与远端 HEAD。
- 同步本地远端 HEAD 指向和受影响的发布说明文字。
- 记录并提交本次设置变更，不修改分支保护或 Release。

## Analysis

GitHub 当前默认分支为 master，当前身份具有 ADMIN 权限，main 已存在且指向 `5db04c0c0c23a4bb59822a0b6dfdef6617f03f19`。master 为 `4bcee95bf1831e2f70de29281ef23a207de6e146`。

## Implementation Plan

修改默认分支 → 独立读取 API／远端引用确认 → 更新本地 origin/HEAD 与过时发布说明 → 记录并正常提交推送文档。

## Implementation Result

- GitHub 默认分支已由 master 改为 main，仓库首页默认读取 v2 README。
- 本地 `origin/HEAD` 已同步到 `refs/remotes/origin/main`，未切换工作分支或修改业务代码。
- 更新发布指南中的默认分支说明，保留 master 的 v1 历史和明确选择 main 发布工作流的规则。
- 未删除、重置、变基或强推任何分支，未修改分支保护、Release 或既有未提交文档。

## Verification

- 修改前 `gh repo view` 返回默认 master、当前权限 ADMIN；main 已存在。
- 首次修改调用返回 EOF，随后独立读取 GitHub 与远端 HEAD 确认为 master，未把未知结果当作成功或直接盲重试。
- 保持证书校验，仅在该次 gh 调用临时禁用 HTTP/2 后使用仓库 PATCH 重试成功；原有进程环境已恢复，无全局网络配置变更。
- GitHub PATCH 返回 `default_branch=main`；独立 `gh repo view` 同样返回 main。
- `git ls-remote --symref` 确认 `HEAD → refs/heads/main`；设置操作本身未改变 main 的 `5db04c0c...` 或 master 的 `4bcee95b...` 提交。
- `git symbolic-ref refs/remotes/origin/HEAD` 返回 `refs/remotes/origin/main`。
- 发布指南仅修改分支说明，既有发布命令未更改；纯设置与 Markdown 操作不运行 `task verify`。
- 默认分支说明、UTF-8 无 BOM 和 `git diff --check` 检查通过。

## Git

GitHub 默认分支属于仓库设置，不通过源码提交修改。对应发布文档同步提交：

```text
18f498524858989ff71553314fafe7242bc9a12b docs(release): 同步 main 默认分支与 v1 保留规则
```

文档随本记录正常提交推送至 main；保留前序任务未提交文档，不删除、重置或强推任何分支。

## Result

GitHub 仓库默认分支与本地 origin/HEAD 已切换为 main，仓库首页默认展示 v2 的中英文 README。
master 保留，既有提交历史、保护配置及 Release 未改写，未提交的排查文档未纳入此次文档提交。

## Completed

2026-10-27（Asia/Singapore）
