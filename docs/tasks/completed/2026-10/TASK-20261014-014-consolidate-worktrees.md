# TASK-20261014-014: 将 OmniProxy 工作区统一到原目录

## Status

Completed

## Created

2026-10-14（Asia/Singapore）

## Request

解释 `E:/go/new/wails3-v` 下多个 OmniProxy 目录的原因，将最新 OmniProxy 代码同步到 `OmniProxy`，删除其余 OmniProxy 目录，保留 `OmniMail-win`。

## Goal

仅保留 `E:/go/new/wails3-v/OmniProxy` 一个项目工作区，使用最新 main，保留 Git 提交、分支和未提交的独有内容，并从统一目录恢复开发环境。

## Analysis

Git 当前登记七个工作树，共享原目录的 `.git`；最新代码在 `OmniProxy-main`，HEAD 为 `7743bf1`。原目录使用旧功能分支，有两份未跟踪任务记录；main 另有一份未跟踪运行记录。必须先保存独有内容和检查其余工作树，正常停止 main 的开发进程，再解除 main 工作树并在原目录切换 main。

其余五个功能工作树均无已跟踪或未跟踪的源码变更。Atria 和 MiMo 分支已是 main 的祖先；Claude 两个实现提交在 main 中有等价补丁，剩余两个提交属于文档。三个旧网关／路由分支保留独立历史，main 已有相应实现与后续更新；本任务统一目录，不把旧代码重新合并进最新 main，所有分支引用保留。

删除仅针对登记的六个附加工作树。删除前验证绝对路径的父目录为 `E:/go/new/wails3-v`、名称在明确名单内且不是符号链接／重解析点。独有任务记录及临时验收资料保存到原目录内的忽略目录，重建所需依赖和构建缓存可随工作树清理。

## Implementation Plan

1. 检查全部工作树的变更、忽略文件及分支提交关系，保存未提交／独有资料。
2. 正常退出精确匹配 main 路径的开发程序，保留独立生产版。
3. 验证删除路径范围，使用 Git 工作树管理命令移除附加目录；原目录切换到最新 main。
4. 从正常桌面环境在原目录启动开发版，核对分支、窗口、热更新与健康接口。
5. 核对只剩 OmniProxy 和 OmniMail-win，归档任务结果。

## Implementation Result

- 正常执行 main 开发应用 41916 的托盘 Quit，完成开发服务退出；独立生产版 31320 保持运行。
- main 未跟踪运行记录已复制到原目录，原目录原有两份未跟踪记录保持原样。六个工作树的独有临时资料及日志保存到 `tmp/worktree-consolidation/preserved/`，移动前后逐文件验证相对路径、大小和 SHA-256。
- 使用 Git 的 `worktree remove` 删除六个登记的附加工作树，无强制删除、分支删除或历史重写；每次删除前核对绝对路径、父目录、明确名单、重解析点、共同仓库和当前工作树状态。
- 原目录通过 `git switch main` 切换至 `7743bf1`，与整理前最新 main 完全一致；全部 18 个分支引用逐项对比未变化。
- Git 开发规范补充本机单一工作区及临时工作树清理规则，避免再次堆积长期目录。
- 通过资源管理器打开隐藏启动器，在统一原目录执行普通 `task dev`；启动器核对分支为 main、共享目录实体路径无 Codex 重定向。开发程序 PID 33204，实际路径为 `E:/go/new/wails3-v/OmniProxy/bin/omniproxy.exe`，前端 HMR 与开发版健康接口正常。
- 上级目录最终只剩 `OmniProxy`、`OmniMail-win`；邮件项目路径及创建时间值与操作前一致，所有移除和移动路径都明确排除该项目。

## Verification

- PASS：原目录 main HEAD 与整理前最新 main `7743bf1b2e14ab0e48bd495f9a07e1df16a3d21e` 完全一致；18 个分支引用逐项比对不变，Git 共同目录仍为原 `OmniProxy/.git`。
- PASS：`git worktree list --porcelain` 仅登记原目录一项，上级目录仅保留 OmniProxy 与 OmniMail-win。
- PASS：原目录两份既有未跟踪记录 SHA-256 不变，main 原有运行记录复制前后哈希一致；10 组独有临时资料／日志保存并逐文件验证。
- PASS：正常桌面 Wails 开发构建完成，含依赖检查、绑定生成、TypeScript／Vite 打包与 Go 编译；实际代码来源是统一原目录的 main。
- PASS：启动器核对真实 Claude 共享目录，`Connected to frontend dev server!` 日志、原生窗口可见且响应正常，UI Automation 确认提供商页面加载。
- PASS：`http://127.0.0.1:9245/@vite/client` 返回 HTTP 200；3891 `/healthz` 返回 `status=ok`、`profile=development`。
- PASS：`git diff --check`，没有新增功能代码、配置或依赖变更，仅更新工作区规范和本任务文档。
- 本任务未重复完整 `task verify`：仅移除附加工作树并检出相同代码，不修改功能实现；该 main 在上一个任务刚通过完整质量门／392 项前端测试。本次重新构建和原生启动验证覆盖目录迁移及依赖路径风险。
- 验证清单、SHA-256 清单、正常桌面启动日志和运行报告位于忽略目录 `tmp/worktree-consolidation/`。

## Documentation

Git 工作流增加统一开发目录和临时工作树清理说明，CURRENT 同步任务概览；历史任务的旧路径保持原样，保留当时验收背景。

## Remaining Risks

旧网关／路由分支仍有独立提交历史，全部保留在 Git 中；本任务没有再次合并旧代码。独有临时资料含历史验收快照，存放在 Git 忽略目录中，可按需手动清理。

## Git

`7e80c35` — `docs(workspace): 将 OmniProxy 统一到原目录并记录清理验收`。

原有三份未跟踪记录及本地资料不纳入提交。分支引用一致性检查在本任务文档提交之前完成；本任务仅为 main 增加整理与归档文档提交，其余分支引用不变。

## Result

最新 main 已统一到原 OmniProxy 目录，六个附加工作树已清理，Git 历史、分支和独有资料保留；OmniMail-win 保留，统一目录的正常桌面开发版已运行并验收。

## Completed

2026-10-14（Asia/Singapore）
