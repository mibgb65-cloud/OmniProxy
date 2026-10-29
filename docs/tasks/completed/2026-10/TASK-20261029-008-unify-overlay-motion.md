# TASK-20261029-008: 统一弹窗与浮层的进入／退出过渡

## Status

Completed

## Created

2026-10-29 13:50（Asia/Singapore）

## Request

用户在 TASK-20261029-006 之后要求：「要一起扫一遍，有问题的地方都修改掉」——把其余弹窗、确认框与浮层里
「关闭时闪一下就消失」或干脆没有任何过渡的地方一并修掉。

## Goal

所有模态弹窗、确认框与浮层的出现和消失都有一致过渡（150ms 淡入淡出 + 轻微缩放，遵循减少动画偏好），
且不再因为条件挂载导致 Radix 拿不到关闭态而瞬间消失。

## Background

TASK-20261029-006 只修了设置窗口、升级浮窗和「更新完成」浮窗。仓库里还有 10 个弹窗类组件与 5 个浮层组件：
其中部分用 `{open ? <Portal/> : null}` 条件挂载（关闭直接卸载），部分虽然常驻 Portal 但只写了进入动画、
没有 `data-[state=closed]` 的退出动画，还有几个完全没有过渡类。项目里已有一份正确写法可参照：
`components/ui/select-field.tsx` 的 `Select.Content` 同时写了进入与退出类并带 `motion-safe:duration-150`。

## Scope

- 弹窗：账号新增／编辑、订阅登录、删除账号确认、请求详情、Google 项目、提供商代理设置、
  路由编辑器、路由删除确认、优先级编辑器、设置窗口、升级浮窗、更新完成浮窗。
- 浮层：账号设置、MiMo 访问详情、模型预览、Google 订阅模型、重置卡。
- 把仍是条件挂载的弹窗／浮层改为常驻，由 Radix Presence 决定卸载时机。

## Out of Scope

- 改变动画时长体系、缓动曲线或引入动画依赖。
- Tooltip、下拉菜单、Tabs 等已有过渡的组件。
- 业务逻辑、数据获取时机、表单状态语义。

## Existing Behavior

- `select-field.tsx`：进入 + 退出类齐全，`motion-safe:duration-150`。
- 设置窗口、升级浮窗、更新完成浮窗：TASK-20261029-006 已修好。
- 其余：有的只有进入动画（关闭瞬间消失），有的完全没有过渡（打开也是硬切）。

---

# Implementation Plan

1. 逐个弹窗／浮层补齐与 `Select.Content` 一致的进入 + 退出过渡类。
2. 条件挂载的四处（账号新增、账号编辑、订阅登录、重置卡浮层）改为常驻。
3. 关闭后仍被保留的组件补上「最后一次数据」，避免退出动画期间内容塌陷或 `open` 恒为真。
4. 复核既有测试，`task verify`、核对生产 CSS、更新文档与任务记录。

---

# Implementation Result

## 过渡类

统一为 `select-field.tsx` 已有的写法，不新增抽象：

```text
遮罩：motion-safe:data-[state=open]:animate-in   …open]:fade-in-0
      motion-safe:data-[state=closed]:animate-out …closed]:fade-out-0  motion-safe:duration-150
面板：遮罩的全部 + …open]:zoom-in-95 / …closed]:zoom-out-95
```

覆盖 12 个弹窗与 5 个浮层，另给 `StatusToast` 补上 `fade-out-0` + `slide-out-to-bottom-2`，
让自动消失的提示也有退出过渡（此前只有进入动画）。

## 条件挂载改为常驻

| 组件 | 处理 |
| --- | --- |
| `add-account-dialog.tsx` | `<AccountForm key={generation}>` 常驻，每次打开换 key |
| `edit-account-dialog.tsx` | 保留最后一次目标 + `key={generation}` |
| `subscription-login-dialog.tsx` | `<LoginContent key={generation} enabled={open}>` 常驻 |
| `reset-credits-popover.tsx` | Portal 常驻，内容仍由 Presence 释放 |
| `providers-view.tsx` | `ProxySettingsDialog` 改为 `open` 属性 + 常驻 |
| `routing-view.tsx` | `RoutingEditor` 去掉 `open &&` 条件（Root 在父组件里） |
| `default-model-routes.tsx` | `ProviderPriorityEditor` 增加 `open` 属性 + 保留最后一次目标 |

## 新增两个小工具

- `lib/use-last-value.ts`：记住最后一个非空值。弹窗关闭时父组件会清空数据，退出动画期间仍需渲染，
  因此保留上一次的值（用 React 的渲染期 state 调整写法，`eslint` 与 `react-hooks` 规则通过）。
- `lib/use-open-generation.ts`：每次打开返回新 key。内容改为常驻后，重新打开必须换实例才能保持空白草稿。

## 保留数据的四处

`delete-account-dialog`（账号名）、`routing-delete-dialog`（规则别名）、`request-dialog`（请求编号）、
`provider-priority-editor` / `edit-account-dialog` / `priority target` 都用 `useLastValue` 保留最后一次数据，
否则退出动画的 150ms 内会显示空标题或空编号。

## 顺带修正的副作用

- `useProxySettings` 增加 `enabled`，并把 `queryFn` 改成惰性 `() => getProxySettings()`：
  弹窗常驻后不会在未打开时读取代理设置，也不会在缺少该 mock 的测试里被意外访问。
- `useSubscriptionLogin` 增加 `enabled`，`starting` 不再因查询被禁用而恒为真；
  订阅登录内容常驻后不会在账号页加载时就去读订阅会话。
- `ProviderPriorityEditor` 原本 `<Dialog.Root open>` 恒为真（靠父组件卸载来关闭），
  改为受控 `open` 才能常驻；`proxy-settings-dialog` 同理。

---

# Verification

## Quality Checks

- [x] Format
- [x] Lint
- [x] Type Check
- [x] Test
- [x] Build
- [x] Verify

## Verification Result

| 项目 | 结果 |
| --- | --- |
| `task verify` | exit 0 |
| gofmt / prettier `format:check` | 通过 |
| `go vet` / `staticcheck` / `eslint` | 通过 |
| `tsc` typecheck | 通过 |
| Vitest | 145 文件／880 项全部通过 |
| Windows production 构建 | 通过 |
| 生产 CSS | 含 `data-[state=closed]:animate-out / fade-out-0 / zoom-out-95 / slide-out-to-bottom-2` 与 `motion-safe:duration-150` |

审计脚本结果（改动后）：

- 「条件挂载的 Portal / 弹窗内容」：0 处。
- 「遮罩或面板缺退出类」：0 处。
- 「进入／退出不对称」：仅 `tooltip.tsx`，其进入态用的是 Radix 的 `data-[state=delayed-open]`，本身对称，无需改。

过程中修复的失败（未放宽任何断言）：

- `provider-navigation`：`ProxySettingsDialog` 常驻后访问了测试未 mock 的 `getProxySettings`，
  改为惰性 `queryFn` 后恢复。
- `accounts-view`「separate dialogs」：表单常驻导致第二次打开残留上次输入，用 `useOpenGeneration` 换 key 恢复。
- `provider-priority`「取消返回焦点 / 保存后关闭」：`ProviderPriorityEditor` 的 `open` 恒为真导致关不掉，
  改为受控 `open` 后恢复。

## 未覆盖

- 原生 WebView2 实机观感与 140% 字号待用户确认。
- 未新增单测：过渡由 CSS 关键帧驱动，jsdom 不执行动画；行为回归由既有 880 项测试守住。

---

# Documentation

- `docs/product/APPEARANCE.md`：弹窗过渡段落补全覆盖范围（具体弹窗与浮层清单）、
  「关闭后仍释放内容、重新打开换新实例」与取消清单。
- `docs/plans/CURRENT.md`：顶部摘要与 `## Recently Completed`。

---

# Deviations

1. 没有抽出共享的过渡类常量：`select-field.tsx` 已确立「内联过渡类」的先例，抽出常量需要把 20 处
   `className` 改成模板字符串，改动面更大且 Tailwind 扫描依赖字面量。最终沿用既有写法，
   由审计脚本保证「进入／退出成对出现」。
2. `ProviderPriorityEditor` 与 `ProxySettingsDialog` 的 props 从「靠挂载/卸载控制」改为受控 `open`，
   属于组件契约变化（仅内部调用点与测试各一处）。
3. `useSubscriptionLogin`、`useProxySettings` 新增 `enabled` 选项，默认 `true`，既有调用方行为不变。
4. 重新打开换 key 依赖 `useOpenGeneration` 的 effect 时序：首帧仍是旧实例，`generation` 变化后立即重挂，
   视觉上无差异（内容尚未展示）。

---

# Git

## Commit

```text
a226264d fix(ui): 统一弹窗与浮层的退出动画
```

归档与 CURRENT.md 回填为后续文档提交。未推送、未打标签。

---

# Result

12 个弹窗、5 个浮层与全局提示的出现／消失都改为同一套 150ms 淡入淡出 + 轻微缩放（`motion-safe`），
不再有任何「关闭闪一下」的组件；条件挂载全部常驻，关闭后内容仍由 Radix Presence 释放，
重新打开保持空白草稿。`task verify` exit 0（145 文件／880 项前端测试、Windows production 构建），
生产 CSS 与审计脚本确认进入／退出成对存在。

---

# Completed

2026-10-29 14:30
