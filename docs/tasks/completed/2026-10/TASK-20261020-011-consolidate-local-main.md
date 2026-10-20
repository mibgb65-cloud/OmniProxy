# TASK-20261020-011: 提交当前改动并清理已覆盖的本地分支

## Status

Completed

## Created

2026-10-20

## Request

用户要求：已经在 main 中的内容不重复合并，删除已覆盖内容的分支；提交当前未提交代码并纳入 main，不更新远端。

## Scope

- 核对所有本地分支的祖先关系、等价补丁及实际功能覆盖。
- 在本地保存分支恢复信息，再删除确认已覆盖的本地分支。
- 审查并按功能提交当前工作区改动，保留已有任务记录，不提交敏感文件。
- 当前就在 main，直接提交到 main，不新建无意义的开发分支或重复 merge。
- 不 fetch／push、不改远端分支、不重写提交历史、不操作真实客户端授权。

## Analysis

- 起始分支 main，HEAD 为 `62adc33bd1f64e948ec64ee30cd6c9f825e676d1`。
- 起始 origin/main 为 `432e6812e055ac38d8a4f51814abf6d57b6b8884`，本地领先 40 个提交。
- 工作区 22 个已跟踪文件修改，31 个未跟踪文件；其中包括本轮产品代码、试验工具和既有任务记录。
- 19 个本地 codex 分支中，15 个已是 main 祖先；另四个需要逐项核对，不以 git cherry 的精确补丁差异直接推断功能缺失或覆盖。
- 只有当前一个工作树，未发现其他分支被工作树占用。

## Implementation Plan

1. 记录并核对分支独有补丁、文档与当前实现，保存本地可恢复 bundle。
2. 复查当前改动的验证结果与敏感数据，按功能提交到 main。
3. 删除已确认覆盖的本地分支，保留任何无法证明已覆盖的分支。
4. 写回提交及分支清理结果，检查工作区、main 和远端跟踪引用。

## Implementation Result

### 分支覆盖核对

- 15 个 codex 分支已经是 main 的祖先，可正常删除本地分支引用。
- `codex/deepseek-claude-context` 的两次代码修复与 main 的 `6285c0c`／`fa69dc6`
  精确补丁等价；其正常桌面启动、真实共享路径、恢复基准等文档也已在 main 的
  BUILD、LOCAL_AGENTS 和 TASK-20261014-008 中保留。
- `codex/gateway-console`、`codex/deepseek-gateway`、`codex/model-routing` 的早期
  功能已回填进 main，并由 `44fdbc49b8958d5a8785ba1ab27ef6a0eafd021c` 实现快照
  纳入历史。当前 main 保留网关／路由界面、DeepSeek 客户端、代理生命周期／流式、
  精确路由和相关测试；永久任务记录还保留原实现哈希。后续架构演进不是遗漏，
  不重新合并旧实现或恢复已被替代的界面与文件。

### 本地恢复备份

已创建并验证完整历史 bundle：

`.git/local-backups/TASK-20261020-011-before-branch-cleanup.bundle`

该文件不在提交范围中，不上传远端。包含清理前所有本地分支引用；没有删除提交对象
或重写 main。四个非祖先分支清理前的头：

| 分支 | 原 HEAD |
| --- | --- |
| codex/deepseek-claude-context | 5785b5413bf34710155f384617104b5c570f1cc7 |
| codex/deepseek-gateway | a8d8dd7538f1e2c489581f8b3f4137595c013e97 |
| codex/gateway-console | ddd38b1fc8e10381baf1a7370e1ce7ebfbc358ad |
| codex/model-routing | 5da06f5182fde8c501a299d617d86ba797d454e8 |

### 提交与清理

已按启动设置、Codex 旧方式试验、River 兼容边界三项分别提交源码，直接位于 main。
已删除 19 个确认覆盖的 codex 本地分支，剩余本地分支只有 main；15 个使用普通删除，
四个非祖先但已覆盖的分支在逐项核对和 bundle 备份后删除，没有重复合并。

十份既有 Completed 运行／排查记录审阅后原样补入版本控制。另一份未跟踪的
`docs/tasks/active/TASK-20261013-017-atria-builtin-provider.md` 是旧 Planned 草稿，
main 已有同任务的 Completed 正式记录；按“不重复合并”保留原草稿在工作区，
不重复提交、不删除或改写用户内容。此文件不是未提交产品代码，最终状态需明确披露。

当前 TASK-20261020-008 仍等待 Desktop 验收，回填实际源码提交哈希后继续保持
In Progress，不因 Git 整理宣称三个产品问题全部修复。

## Verification

- PASS：分支祖先关系、git cherry、main 实现快照、任务来源哈希及当前文件／行为交叉核对。
- PASS：`git bundle verify` 确认完整历史可恢复，所有待删除分支均有本地备份。
- PASS：54 个当前变更文本文件的授权文件路径／字面密钥／JWT／私钥特征扫描与 UTF-8
  无 BOM 检查，未发现需要排除的真实秘密；原授权和试验恢复文件均在仓库外。
- 复用当前代码已通过的完整质量门与 671 项前端测试；本轮仅 Git 整理和记录，不改产品逻辑。
- PASS：提交前九个相关 Go 包回归通过，覆盖网关、路由、代理、账号配置、Codex 基础设施、
  profile、Wails 和试验命令；本轮没有重跑不必要的前端全量测试或改动产品代码。
- PASS：每批提交均执行暂存差异检查，明确列出路径，没有使用 git add -A 纳入未知文件。
- PASS：代码提交均在 main，无额外开发分支／重复 merge，没有改写原提交或删除提交对象。
- PASS：19 个删除分支在操作前逐一核对 bundle 中相同 HEAD，另以工作树和分支集合保护
  避免清理新出现或被占用的分支。
- PASS：origin/main 在清理后仍为 `432e6812e055ac38d8a4f51814abf6d57b6b8884`；
  本轮没有运行任何 fetch、push 或远端写入命令。归档提交后再复核最终状态。

## Git

用户已授权本轮本地提交与清理已覆盖分支；未授权远端更新。

源码实现提交：

- `b6f33987f1abd3d3910c9dddd9b90adbccf7b0f8 feat(settings): 支持启动软件时自动开启网关`
- `3b7e7392030fa98b394d4ce8ae5d65ce4af4aecb feat(codex): 增加带加密备份和恢复的旧版登录试验`
- `dced0551e016ca814f9f28cc0bf33513f7d25252 test(river): 明确图片历史兼容边界并补充回归验证`

本记录、CURRENT、实现哈希回填和既有 Completed 诊断记录随后随 docs(tasks) 归档
提交保存；本记录的归档提交可按路径查 Git 历史。删除分支的恢复点位于前述本地 bundle。

## Result

当前产品代码已提交到本地 main；已经覆盖的历史分支已清理，没有重复合并旧实现。
远端保持不变。仅保留一份未提交的重复 Atria 旧规划草稿，未强行提交或删除用户草稿。

## Completed

2026-10-20（Asia/Singapore）
