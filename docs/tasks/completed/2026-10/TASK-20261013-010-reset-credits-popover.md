# TASK-20261013-010: 重置卡改为账号旁局部面板

## Status

Completed

## Created

2026-10-13

## Request

用户要求重置卡界面直接显示在截图红框标记的账号卡片按钮附近，不再使用全屏悬浮窗。

## Goal

点击重置卡按钮后显示紧凑的局部面板，保持刷新、明细、使用确认和结果核对能力。

## Scope

- 重置卡面板定位、尺寸、键盘及关闭交互。
- 相关交互回归验证、订阅文档及当前状态看板。

## Out of Scope

后端接口、重置权益规则、真实重置卡消耗和其他弹窗。

## Analysis

现有 `reset-credits-dialog.tsx` 使用 Radix Dialog 的全屏遮罩与 720 × 620 居中容器。仓库已有 `radix-ui`，可以使用非模态 Popover 定位于按钮下方，无需新增依赖或手工维护坐标。复用现有内容和业务 hook，保留使用中关闭保护。主要风险是小窗口边缘定位、紧凑内容溢出和非模态焦点管理。

## Implementation Plan

1. 将重置卡入口改为非模态 Popover，按钮下方左对齐，约 360 × 440，启用边界避让。
2. 复用内容与确认流程，补充局部面板关闭、焦点恢复及使用中保护验证。
3. 运行相关测试、`task verify`，在项目原生开发窗口核对布局，更新文档并提交。

## Implementation Result

- 将全屏 Radix Dialog 改为非模态 Popover，移除整页遮罩／模糊与居中定位，组件同步重命名为 `ResetCreditsPopover`。
- 默认从入口下方左侧展开 360 × 440 面板，距入口 8px、窗口边界保留 12px，启用边缘避让并根据可用高度缩小。
- 收紧间距与卡片内边距，标题可换行，数量和底部操作不被压缩，仅明细区域内部滚动。
- 继续复用原有数据缓存、刷新、使用确认、结果恢复／核对和幂等业务逻辑；没有后端或依赖变更。
- 保留关闭按钮、入口切换、Escape／点击外部关闭、焦点恢复、可访问标题／描述及使用中关闭保护。

## Verification

- PASS：两个重置卡测试文件共 9 项测试，保留数量／确认／单次提交／未知结果恢复等回归，新增非模态语义、三种关闭入口、焦点恢复、外部操作可响应、真实异步使用中关闭保护与完成后解锁。
- PASS：`VITEST_MAX_WORKERS=4 task verify BIN_DIR=E:/go/new/wails3-v/OmniProxy/tmp/reset-credits-popover/quality-bin`，退出码 0；格式、Go vet／staticcheck、ESLint、TypeScript、全部 Go 测试、69 个前端测试文件／363 项测试和 Windows 生产构建通过。仅限制本次验证 worker 数，没有修改项目配置。
- PASS：按进程路径识别本仓库开发窗口 PID 42068，核对面板位于入口下方；深浅主题及最小 960 × 800 窗口正常，卡片和列表两种模式均可打开局部面板。
- PASS：原生确认区完整显示，取消可回到卡片列表；分别聚焦首尾卡片可滚动到可见区域，截图已核对。仅打开确认与取消，未兑换真实重置卡。
- PASS：`git diff --check`，本次文件 UTF-8 无 BOM 核对。

原生脚本、截图和质量日志位于忽略目录 `tmp/reset-credits-popover/`。质量门中的既有工具探测、jsdom scrollTo 和打包体积提示未影响通过。

## Documentation

已更新 `docs/product/SUBSCRIPTION.md` 的面板打开／关闭规则和 `docs/plans/CURRENT.md`；没有产品范围或架构变更。

## Deviations

None.

## Git

实现提交待记录。

## Result

重置卡在账号按钮旁显示紧凑面板，整页保持清晰可交互，质量门和原生窗口验收通过。

## Remaining Risks

未在原生验收中消耗真实重置卡；兑换接口及持久化逻辑沿用已有实现，由既有回归测试验证。

## Completed

2026-10-13（Asia/Singapore）
