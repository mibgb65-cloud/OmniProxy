# TASK-20260924-006: Settings Page - Data Storage Management

## Status

Completed

## Created

2026-09-24 15:22

## Request

“我们首先来做设置页面，显示数据存储位置，可以更改数据存储的位置，显示当前占用的存储空间，提供清理按钮。”

## Goal

设置页提供完整的数据存储管理能力：

1. 显示当前数据存储位置（数据目录、日志目录、数据库文件）
2. 允许更改数据存储位置（原生文件夹选择 + 手动输入 + 校验，重启生效，关机时自动迁移现有数据）
3. 显示当前占用的存储空间（数据库 / 日志 / 总计）
4. 提供清理按钮（清理日志、压缩数据库），并反馈释放的空间

## Background

- 设置页此前是占位页；数据目录由 `internal/platform/profile` 在启动时解析（env > 默认），运行期不可见、不可改。
- 技术栈文档将 Application Settings 列为 React Hook Form + Zod 的目标场景；本任务是第一个设置表单，按规范在此功能首次需要时引入依赖。
- 架构约束：UI 不直接访问 SQLite/文件系统；一切经 Wails 绑定 → Application Service。

## Scope

- Go：profile 持久化配置（固定位置 app.json，含 previousDataDir 崩溃保护）、storage 包（目录大小/迁移）、database Vacuum、logger 文件句柄（Close/Clean）、AppService 新绑定（StorageInfo / SetDataDir / SelectDirectory / CleanStorage）、关机时数据迁移
- 前端：features/settings（查询 + 变更 + 清理）、RHF+Zod 表单、formatBytes 工具、i18n 中英案文、测试
- 文档：TECH_STACK（RHF/Zod 接入状态）

## Out of Scope

- 请求历史/用量数据的清理（后端功能尚未存在）
- 数据目录的多配置切换
- 云同步/备份

## Existing Behavior

- Settings 页为占位符；`profile.Detect()` 每次启动按 env > 默认解析数据目录；无存储信息展示；无清理能力。

## Expected Behavior

- Settings 页展示：数据目录、日志目录、数据库文件及各自大小、总占用。
- 用户可通过原生选择器或输入路径更改位置；后端校验（可创建、可写）；保存后提示重启生效；应用退出时自动将数据迁移到新位置（迁移失败则回滚设置并记录日志）。
- 清理按钮：清理日志（删除轮转备份并截断当前日志）、压缩数据库（VACUUM），按钮反馈释放的空间。
- 若上次保存了新位置但迁移未完成（崩溃），下次启动自动检测并回退到仍有数据的旧位置。

---

# Analysis

## Related Modules

- `internal/platform/profile`（配置持久化 + Detect 逻辑）
- `internal/infrastructure/storage`（新增：目录大小、迁移）
- `internal/infrastructure/database`（Vacuum）
- `internal/infrastructure/logger`（Files 句柄）
- `internal/transport/wails`（AppService 绑定 + 关机迁移 + main 装配）
- `frontend/src/features/settings/**`、`frontend/src/app/routes/settings.tsx`、`frontend/src/services/omniproxy.ts`、`frontend/src/i18n/locales/*`

## Dependencies

- 新增前端依赖：`react-hook-form@7.88.0`、`zod@4.6.5`、`@hookform/resolvers@5.9.1`（技术栈既定的表单方案，本功能首次使用）
- Wails v3 `application.Get().Dialog.OpenFile()`（原生文件夹选择）

## Risks

- 关机迁移涉及打开中的 DB/日志句柄：按“先关句柄再迁移”排序，失败时回滚 app.json
- Windows 跨卷移动需 copy+delete 回退（os.Rename 跨卷失败）
- Windows 上 O_APPEND 句柄不支持 SetEndOfFile，截断需“关闭→按路径截断→重开”（已实现并测试）

---

# Implementation Plan

1. profile：`config.go`（app.json 固定路径 Load/Save/SetDataDir，原子写入，previousDataDir）+ `profile.go` Detect 应用存储配置（env > stored > default）与启动回退守卫
2. `internal/infrastructure/storage`：DirSize / FileSize / MoveDir（rename，跨卷回退 copy）
3. database：`Vacuum(ctx)`；logger：Setup 返回 `*Files`（Close + Clean）
4. AppService：StorageInfo / SetDataDir / SelectDirectory / CleanStorage；ServiceShutdown 迁移；main.go 装配 db 与 Files
5. Go 测试（config/storage/database/logger/AppService）
6. 重新生成 bindings；前端：services 封装、features/settings 组件与 hooks、settings 路由、formatBytes、i18n
7. 前端测试；文档同步；task verify；提交；完成本记录

---

# Progress

- 确认 Wails v3 提供 `application.Get().Dialog.OpenFile().CanChooseDirectories(true)`，可在 transport/wails 层实现原生选择器（不违反 Core 不依赖 Wails 的边界）
- 确认原生对话框取消时返回 error 且无路径：SelectDirectory 记录 debug 日志并返回空字符串（取消），避免把取消当错误弹给用户
- Windows 截断踩坑：O_APPEND 句柄 SetEndOfFile 返回 Access denied，改为关闭→os.Truncate→重开
- TanStack Query v5 会给 mutationFn 传第二个 context 参数：hooks 中包装 `(x) => fn(x)` 归一化签名（测试断言与生产调用都只收到一个参数）
- i18n 插值踩坑：新增 settings 文案误用单括号 `{size}`，i18next 需 `{{size}}`；已全局修正

---

# Implementation Result

## Changes

- `internal/platform/profile/config.go`：固定位置 app.json（`OMNIPROXY_CONFIG_DIR` 可覆盖目录）的 Load/Save（原子写入）/SetDataDir（校验：非空、绝对路径、不相同、不互相嵌套、可创建可写）/RevertDataDir；`effectiveDataDir()` 启动守卫（新位置无 db 而旧位置有 → 回退）
- `profile.Detect()`：数据目录解析顺序 env > app.json > 默认；回退时持久化修正
- `internal/infrastructure/storage`：DirSize / FileSize / MoveDir（rename 优先，跨卷 copy 回退，覆盖语义）
- `internal/infrastructure/database/vacuum.go`：VACUUM
- `internal/infrastructure/logger`：Setup 返回 `*Files`（Close / Clean：删编号备份 + 截断当前日志并返回释放字节）；rotatingFile.Truncate 采用关闭→按路径截断→重开
- `internal/transport/wails/storage.go`：StorageInfo / SetDataDir / SelectDirectory（原生选择器，标题由前端传入以遵循界面语言）/ CleanStorage（logs|database）；relocateIfPending：关机时关句柄→迁移→失败回滚配置
- `main.go`：装配 db 与 Files 到 App/AppService
- 前端：`features/settings/use-storage.ts`（query + 3 个 mutation，invalidates storage-info）、`storage-overview.tsx`（位置/大小/总计 + 两个清理按钮 + 待生效提示）、`data-dir-form.tsx`（RHF+Zod + 浏览按钮 + 保存 + 重启提示）、`routes/settings.tsx` 真实页面、`lib/format.ts`（formatBytes）、services 四个封装、i18n 29 个 settings 键（中英）

## Files Changed

- Go：`internal/platform/profile/{config.go,config_test.go,profile.go,profile_test.go}`、`internal/infrastructure/storage/{storage.go,storage_test.go}`、`internal/infrastructure/database/vacuum.go`、`internal/infrastructure/logger/{files.go,files_test.go,logger.go,logger_test.go,rotating.go}`、`internal/transport/wails/{app.go,storage.go,storage_test.go}`、`main.go`
- 前端：`frontend/package.json`、`frontend/pnpm-lock.yaml`、`frontend/src/features/settings/*`、`frontend/src/app/routes/settings.tsx`、`frontend/src/services/omniproxy.ts`、`frontend/src/lib/{format.ts,format.test.ts}`、`frontend/src/i18n/locales/{zh.ts,en.ts}`、`frontend/bindings/**`（重新生成）
- 文档：`docs/architecture/TECH_STACK.md`、`docs/plans/CURRENT.md`（随任务完成更新）

## Technical Notes

- app.json 位于固定路径（不随数据目录移动），`previousDataDir` + 启动守卫保证“保存了但没迁移成”不会导致数据看似丢失
- 迁移只在退出时执行：运行期数据仍写旧位置，退出时关句柄后整体移动，下次启动用新位置
- 后端错误信息（英文）直接呈现给用户并带上下文；前端 Zod 只做基础非空校验，权威校验在后端

---

# Verification

## Tests

- [x] Unit Tests（Go：config 10 例、storage 5 例、logger Files 3 例、AppService 9 例；Frontend：format 3 例、storage-overview 4 例、data-dir-form 5 例）
- [ ] Integration Tests — N/A
- [ ] E2E Tests — N/A

## Quality Checks

- [x] Format
- [x] Lint
- [x] Type Check
- [x] Test
- [x] Build
- [x] Verify

## Verification Result

```text
task verify
PASS（Go 全部通过；前端 30 个测试通过）
```

---

# Documentation

- [ ] PRD — 无需更新（设置页属于既有核心能力）
- [x] Architecture — 无需更新（前端结构未变，仅新增 features/settings）
- [x] Tech Stack（RHF/Zod 移出“推迟”并加入依赖表）
- [ ] Coding Rules — 无需更新
- [ ] Testing — 无需更新
- [ ] Build — 无需更新
- [ ] ADR — 无需更新（无重大架构决策）
- [x] CURRENT（随任务完成更新）

---

# Deviations

- 原计划：SelectDirectory 取消与错误难以区分时返回错误。实际：记录 debug 日志并返回空字符串（取消语义）。原因：原生对话框取消也返回 error，直接把取消当错误会产生骚扰性错误提示；日志保证不可静默。
- 原计划：截断当前日志用 `File.Truncate(0)`。实际：关闭→`os.Truncate`→重开。原因：Windows 上 O_APPEND 句柄不允许 SetEndOfFile（Access denied）。
- 原计划未包含 `OMNIPROXY_CONFIG_DIR`。实际：新增该环境变量。原因：测试需要隔离真实用户配置目录，顺带为便携部署提供能力。

---

# Git

## Commit

```text
94fecf7 feat(settings): add data storage management to settings page
```

（本任务的 commit hash 由紧随其后的记录提交回填：提交内容无法包含自身 hash。）

---

# Result

设置页具备完整数据存储管理：位置与占用可见、可经原生选择器/输入更改（校验 + 重启生效 + 退出自动迁移 + 崩溃回退）、日志清理与数据库压缩可操作并反馈释放空间。`task verify` 全绿。

---

# Completed

2026-09-24 15:50
