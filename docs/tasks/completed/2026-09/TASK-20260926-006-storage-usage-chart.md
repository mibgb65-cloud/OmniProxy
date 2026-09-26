# TASK-20260926-006: 存储占用图表

## Status

Completed

## Created

2026-09-26 10:20

## Request

用户希望设置页的数据存储区域不再只有文字，能用图表直观展示缓存、日志、账号数据等占用情况。

## Goal

用可读的图形和数值展示数据目录的真实占用分布，保留位置与清理操作。

## Existing Behavior

设置页仅用文字列出数据库文件、日志目录和两者之和；缓存、SQLite 临时文件和其他目录文件未计入。

## Expected Behavior

- 展示数据库、日志、缓存、其他文件的占用及总计；图形旁保留文字和数值，零数据也有明确状态。
- 数据库明确标注包含账号等结构化数据；当前账号功能未落地，不将数据库大小伪装为账号独占空间。
- 数据目录中的 SQLite WAL 等文件计入数据库，其他文件计入总数。

# Analysis

## Related Modules

- `internal/infrastructure/storage`、`internal/transport/wails/storage.go`
- `frontend/src/features/settings`、`frontend/src/i18n/locales`、Wails 生成绑定

## Dependencies

使用已有 SVG 和图表配色 token，不新增图表依赖。

## Risks

运行中的日志与 SQLite 文件大小会变化，界面显示的是查询时快照。

# Implementation Plan

1. 在存储层单次遍历数据目录，将文件分入数据库、日志、缓存、其他类别。
2. 扩充 StorageInfo 的 items，生成 Wails 绑定；设置页加入占比图、图例和数值，保留路径与清理按钮。
3. 覆盖含 WAL、缓存、其他文件和空目录的测试；运行 `task verify` 并检查差异。

# Progress

已确认账号后端尚未上线；数据库以整体大小显示，并说明未来账号配置属于其中。采用单次目录遍历形成真实占用分类。

# Implementation Result

## Changes

- 数据目录单次遍历，将普通文件归入数据库、日志、缓存、其他文件；SQLite 的 WAL、SHM、journal 计入数据库。
- 设置页新增环形占比图、总量、四类图例与大小/占比、手动刷新；路径和清理操作保留。
- 图表使用现有主题色与 SVG，无新依赖；中英文文案说明账号元数据和系统凭据的统计边界。

## Files Changed

- `internal/infrastructure/storage/usage.go`、`usage_test.go`
- `internal/transport/wails/storage.go`、`storage_test.go`
- `frontend/src/features/settings/storage-usage-chart.tsx`、`storage-overview.tsx`、`storage-overview.test.tsx`
- `frontend/src/i18n/locales/zh.ts`、`en.ts`

## Technical Notes

`StorageInfo` 的字段结构不变，仅 `items` 增加缓存与其他文件两项。Wails 绑定已重新生成，生成文件没有差异。账号功能尚未上线，数据库文件大小不能准确拆成独立的账号占用。

# Verification

## Tests

- [x] Go：存储分类、缺失/非法目录、Wails 存储快照
- [x] 前端：分布、零占用、刷新与原有清理流程；共 43 个测试通过

## Quality Checks

- [x] `task verify`：格式、Go vet、staticcheck、ESLint、TypeScript、测试、Windows 生产构建全部通过
- [x] `git diff --check`

## Verification Result

2026-09-26：`task verify` PASS。当前 PATH 未包含 Go bin，运行时临时加入路径以调用已安装的 `task`、`staticcheck`、`wails3`。Vite 报告已有 500 KB 包体积提醒，不影响构建。

# Documentation

- [x] `docs/plans/CURRENT.md` 完成时更新
- [x] 本 Task Record 记录实现与验证
- PRD、架构、技术栈无需修改：功能沿用现有存储接口和 UI 技术栈。

# Deviations

原计划提到重新生成绑定；已生成并确认结构没有差异，因此没有提交生成文件。开发机上的 `task` 工具未加入 PATH，临时补充 Go bin 后完整验证通过。

# Git

## Commit

```text
9e74acd feat(settings): 增加数据存储占用图
```

（归档记录本身另行提交，无法在同一提交中记录自身哈希。）

# Result

设置页可直观看到真实数据目录的占用分布，含缓存、日志和数据库临时文件；账号数据的统计边界在界面中明确说明。

# Completed

2026-09-26 10:30
