# TASK-20261030-002: 修复账号页提供商切换栏的底部滚动抖动

## Status

In Progress

## Created

2026-10-30（Asia/Singapore）

## Request

用户反馈截图中的“切换提供商”区域在滚动到页面最底部时上下移动，影响体验，希望优化，必要时固定到底部。

## Goal

提供商切换栏位置稳定，不随页面底部滚动上下抖动，同时保留内容可访问性、横向滚动、键盘操作和减少动画偏好。

## Scope

- 定位账号页切换栏与滚动容器之间的布局关系。
- 采用最小范围的布局修复，避免遮挡账号内容。
- 补充针对性回归测试，执行适用质量检查。

## Out of Scope

- 提供商业务逻辑、账号数据及后端接口变更。
- 无关页面重构或视觉改版。

## Analysis

- 根因：Dock 是长内容网格的最后一行，使用 `position: sticky; bottom: 0`；
  到滚动末尾时，其父网格底边与账号页／主内容区的底部留白限制 sticky 定位，
  切换栏会从视口底部回到文档流位置。隔离浏览器恢复旧布局后复现：
  到底前后 Dock 顶部从 681px 移至 633px，上移 48px，正好对应两层底部留白。
- 方案取舍：只微调 sticky 偏移仍依赖内容高度和父容器边界；全局 fixed 则需要
  额外同步侧栏宽度、内容留白和遮挡占位。采用主视口内的有界工作区：
  展开后仅详情区域独立滚动，Dock 作为网格的固定底行，不覆盖账号内容。
- 复用现有 ScrollRegion、主题、横向切换及 FLIP，不新增依赖；
  概览卡片墙和独立详情页继续沿用原来的主滚动行为。

## Implementation Plan

1. 阅读相关规范、账号页布局和既有测试，明确滚动抖动根因。
2. 修复切换栏定位与内容滚动的边界，补充回归测试。
3. 执行针对性测试与 `task verify`，检查差异并补充实际验证结果。

## Implementation Result

### Changes

- 主滚动视口提供定位边界和共用页面留白变量；展开后的账号工作区按相同留白填满视口，
  不依赖侧栏宽度、标题栏高度或 JS 尺寸观察。
- 详情复用 ScrollRegion 独立纵向滚动，并限制边界滚动向外传递；
  Dock 作为不参与纵向滚动的底行，保留横向滚动、键盘、Tooltip 和现有动画。
- 收起后恢复概览卡片墙的主滚动行为；仅一个提供商时不显示空 Dock，详情仍可滚动。
- 布局验证发现 cloneNode 不复制滚动偏移：已滚动 1963px 的详情在切换动画副本中回到 0。
  补充快照偏移捕获及接入布局后的恢复，复验动画副本保持 1963px。
- 新增两个工作区结构／单提供商回归测试和一个动画快照滚动偏移测试。

### Files Changed

- `frontend/src/app/routes/__root.tsx`
- `frontend/src/features/accounts/accounts-view.tsx` 及对应测试
- `frontend/src/features/accounts/bento-accounts.css`
- `frontend/src/features/accounts/flip-snapshot.ts` 及对应测试
- `docs/product/APPEARANCE.md`、`docs/plans/CURRENT.md` 与本记录

### Technical Notes

无新增依赖、后端接口、数据迁移或凭据变更。独立详情路由不改变。
测试夹具只使用合成账号，位于 Codex 可视化工作目录，不写入项目源码或真实用户数据。

## Verification

- [x] 5 个相关测试文件共 32 项通过，包括账号展开／切换／收起、独立滚动边界、
  单提供商、焦点与路由、Dock 键盘／横向滚动、快照及 FLIP。
- [x] 隔离 Chromium 11 项检查通过：1280×800、900×560、760×560、140% 字号下
  到底后连续滚轮不移动 Dock；侧栏收起适配；Home／End 仅横向移动；
  长内容切换到零账号；从已滚动概览展开；滚动快照恢复；连续切换；深浅主题。
  主视口无纵向残留／横向溢出，最后一个账号可见且不被 Dock 遮挡，页面无异常。
- 首次 `task verify`：格式、lint、类型、Go 测试和 145 文件／882 项前端测试通过；
  构建增量依赖检查阶段长时间无输出，主动中断。随后补充快照回归，
  针对性 32 项通过。
- 首次 `task --force verify`：格式、lint、类型、Go 测试通过；前端 882／883 项通过，
  未修改的 `stepfun.test.tsx` 批量添加测试触发既有 5000ms 超时，门禁 exit 201。
  单独重跑该文件 5／5 通过；不修改测试或超时，使用进程级 `VITEST_MAX_WORKERS=4`
  降低并发后重新执行完整质量门。
- [x] 最终 `VITEST_MAX_WORKERS=4 task --force verify`：PASS，exit 0。
  格式、go vet／staticcheck／ESLint、TypeScript、Go 全量测试、
  145 文件／883 项前端测试、Wails 绑定生成、前端生产构建及 Windows amd64
  production 构建全部通过。保留既有 jsdom `Window.scrollTo` 未实现提示与
  Vite 大 chunk 提示，不掩盖告警、不更改测试配置。
- [x] 生产 CSS 包含展开工作区定位、共用留白、详情滚动边界和 Dock 布局规则。
- [x] `git diff --check` 通过；全部 9 个变更文件均验证为 UTF-8 无 BOM。
- [ ] 原生 WebView2 实机视觉验收未执行，隔离 Chromium 结果不替代原生验收。

## Documentation

- 已同步 `docs/product/APPEARANCE.md` 与 `docs/plans/CURRENT.md`。
- PRD、架构、技术栈、开发规范和 ADR 无需变更。

## Deviations

为保留独立滚动引入后的视觉连续性，增加动画快照的滚动位置恢复；
这是浏览器验证中实测发现并修复的直接关联问题，未更换动画实现。
质量门的最终复验限制为 4 个 Vitest worker，只影响本次验证进程，不修改仓库测试配置。

## Git

实现与验证完成，待记录实现提交；保留工作区中与本任务无关的未跟踪文件。

## Result

已将提供商切换栏固定在账号工作区底部，详情独立滚动，不再因父容器边界产生 48px
上下移动；动画副本同步保留滚动位置。完整质量门与隔离布局验收通过，
原生 WebView2 实机视觉验收保留为明确限制。
