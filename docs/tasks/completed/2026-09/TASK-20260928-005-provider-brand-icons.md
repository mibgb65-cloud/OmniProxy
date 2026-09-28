# TASK-20260928-005: 使用真实提供商品牌图标

## Status

Completed

## Created

2026-09-28 15:39

## Request

将提供商页面当前通用的 AI Bot 图标替换为各厂商真实的品牌图标。

## Goal

九家内置提供商展示可辨认的本地品牌 SVG；自定义提供商保持通用图标。图标在亮暗主题下清晰、离线可用，不增加大的运行时依赖。

## Existing Behavior

九家内置提供商卡片都渲染同一个 Lucide Bot 图形，仅背景颜色不同。

## Expected Behavior

OpenAI、Anthropic、Google、DeepSeek、智谱、月之暗面、MiniMax、小米 MiMo、阶跃星辰分别显示对应品牌标识。

---

# Analysis

## Options

1. 引入完整 React 品牌图标包：使用方便，但目前包体约 9.5 MB，且带来 `antd-style` 等额外依赖和对等依赖。
2. 从 Lobe Icons MIT 授权的静态 SVG 包中选取九个所需文件并随应用打包：没有运行时依赖，离线可用，需记录来源与授权。

采用方案 2。品牌标识仅用于识别提供商，不代表厂商与 OmniProxy 的合作或背书。

## Related Modules

- `frontend/src/features/providers/provider-card.tsx`
- `frontend/public/provider-icons/`（新建）
- `docs/plans/CURRENT.md`

## Risks

- 部分品牌有亮暗两版标志，需在两种主题中检查对比度。
- 外部图标包可能更新或更名，使用固定版本并保留素材来源。
- 品牌商标属于各自权利人；只作识别用途，不自行改绘图形。

---

# Implementation Plan

1. 从固定版本的静态 SVG 包核对九家 slug、选择合适的亮暗或彩色图标，并记录 MIT 授权。
2. 页面按 provider kind 引用本地图标；未知/自定义 kind 保留现有通用图标。
3. 做组件测试、亮暗主题视觉检查，运行 `task verify`，复查差异并提交。

---

# Progress

- 已从 `@lobehub/icons-static-svg@1.95.1` 精确抽取九个 SVG（约 14 KB），未引入运行时依赖；附带 MIT 授权和来源说明。
- 初次亮色窗口检查发现 Kimi 彩色标识的主体为白色，在白色底块上不可见；已为该图标使用深色底块，保留上游 SVG 原样。
- 九家图标映射和自定义图标回退测试通过；实际 Wails 窗口的亮暗主题均已检查，随后恢复了原亮色主题。
- `task verify` 全部通过，生产构建产物包含九个 SVG、来源说明及授权文本；开发环境已恢复。
- Wails 与 Vite 分别在 3891、9245 端口正常监听；提交后工作区干净。

---

# Implementation Result

## Changes

- 将九家内置提供商卡片上的通用 Bot 替换为对应的本地 SVG 品牌标识；自定义和未知 kind 继续显示通用 Bot。
- 为单色 OpenAI、Anthropic、小米 MiMo 标志使用深色底块和反相图形；Kimi 彩色标志含白色主体，也使用深色底块；其他彩色标志使用白色底块，使亮暗主题均可读。
- 仅包含所需的九个上游静态 SVG，未新增 npm 依赖；将 Lobe Icons MIT 授权和固定版本来源说明一起打包。

## Files Changed

- `frontend/public/provider-icons/*.svg`
- `frontend/public/provider-icons/{SOURCE.md,LICENSE.lobe-icons.txt}`
- `frontend/src/features/providers/{provider-card.tsx,provider-card.test.tsx}`
- `docs/plans/CURRENT.md`、本任务记录

## Technical Notes

图标使用 `<img>` 读取随应用打包的 SVG，品牌图片在可访问性树中作为装饰元素，名称仍由相邻文字提供。素材保留上游原样，来源版本固定为 `@lobehub/icons-static-svg@1.95.1`。

---

# Verification

## Tests

- [x] 九家内置品牌图标路径和自定义回退图标测试。
- [x] 亮色、暗色 Wails 实际窗口视觉检查。
- [x] 生产 `frontend/dist/provider-icons` 文件与授权文本检查。

## Quality Checks

- [x] Format
- [x] Lint
- [x] Type Check
- [x] Test
- [x] Build
- [x] Verify

## Verification Result

`task verify` PASS：Go 测试通过；前端 15 个测试文件、69 个测试通过；Windows 生产构建成功。构建目录含九个品牌 SVG、MIT 授权和来源说明。亮暗主题窗口检查显示九个标志均清晰，其中 Kimi 的白色主体在深色底块上可见。`git diff --check` PASS。

---

# Documentation

已更新 CURRENT 看板；素材目录包含固定版本、原项目、MIT 授权和商标归属说明。其他产品/架构文档无需修改：提供商行为与接口未变化。

---

# Deviations

视觉检查后将 Kimi 彩色标志改为深色底块。没有修改上游 SVG，也没有增加依赖。

---

# Git

## Commit

`401c2c90258a89c33937d2ca4790de81cbf44ef6 feat(frontend): 为九家提供商使用真实品牌图标`

---

# Result

九家内置提供商显示可辨认的本地品牌标识；自定义项保留通用图标，亮暗主题均可用。

---

# Completed

2026-09-28 15:55
