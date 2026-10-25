# TASK-20261025-012: 自动激活在后台持续检测并自动刷新

## Status

Completed

## Created

2026-10-25 15:21

## Request

用户附「OpenAI 账号设置」截图（自动激活 Codex 额度窗口已开启）并反馈：

> 这个自动激活功能目前是必须要我点开软件才会自动去激活吧，我想要的是能够即使软件在后台，它也会自动的去检测是否激活了时间，不必要我打开软件还要点进去设置，这个应该要自动刷新。

原始意图：

1. 软件在后台（窗口没打开／最小化）时，自动激活仍然按周期检测额度窗口，不需要用户打开界面。
2. 不需要点进「OpenAI 账号设置」才能看到检测结果，界面应自动刷新。

## Goal

- 开启自动激活后，进程真正保持在后台运行：关闭主窗口不再结束应用，后台检测循环继续。
- 检测结果由后端主动推送，前端在任意页面即可自动刷新状态并弹出结果提示，不必打开设置面板。
- 应用启动／唤醒后不必干等下一个 5 分钟周期才能完成滑动目标的第二次观测。

## Background

现有实现（ADR-009）：

- `quotaactivation.Service.Start` 在 `ServiceStartup` 启动单循环，检测周期固定 5 分钟，额度读取信号限频唤醒。
- 检测开关由设置表读取，默认关闭；关闭时循环空转，不查询账号。
- 前端状态查询和结果提示都挂在 `CodexQuotaActivationSettings`／`OpenAIAccountSettings` 内部，而 Radix Popover 关闭时不挂载内容。

## Scope

本任务包含：

- 后端自适应复检：出现「接近满周期、需要第二次独立观测」的候
  选窗口时，用短间隔补一次复检，而不是等 5 分钟；连续补检有上限，避免长期高频查询。
- 后端检测完成事件：每轮检测结束推送一次脱敏状态（只有计数与稳定错误码）。
- 桌面壳：开启自动激活时把窗口关闭行为落到托盘，保证后台进程存活；启动时对已开启的用户同样校正。
- 前端：抽出共享的激活状态查询，新增应用级监视器，在任何页面自动刷新状态并给出结果提示；设置面板只保留开关与文案。
- 文案与文档：说明后台运行语义。

## Out of Scope

本任务不处理：

- 自动开机启动（仍由用户显式开启，不因自动激活代劳）。
- 修改额度激活判定（窗口条件、滑动观测容差、持久去重、至多一次发送保持不变）。
- 自动兑换重置卡、修改客户端配置或会话。
- 新增数据库迁移。
- 构建／发布安装包。

## Existing Behavior

- 关闭主窗口按 `CloseBehavior` 执行，默认 `exit`，应用退出后后台检测随即停止。
- 检测循环只在应用进程存活时运行；启动时立即检测一次，之后每 5 分钟一次。
- 启动后若旧观测已超过 15 分钟，滑动候选必须等下一轮 5 分钟检测才能凑够两次观测。
- 前端状态轮询与结果提示只在设置面板打开（Popover 内容挂载）时存在。

## Expected Behavior

- 开启自动激活后关闭主窗口，应用留在托盘，后台检测继续按周期运行；用户仍可从托盘退出。
- 每轮检测结束时后端推送状态，任意页面都能自动刷新检测状态；有实际结果（发送／待复查／失败）时弹出一次提示，安静轮次不重复打扰。
- 出现滑动候选时，启动／唤醒后约 35 秒完成第二次独立观测，可立即给出结论；连续补检有上限，之后回到 5 分钟节奏。

---

# Analysis

## Related Modules

- `internal/application/quotaactivation`：后台循环、观测与状态。
- `internal/transport/wails`：设置保存、事件绑定、窗口关闭行为、启动顺序。
- `internal/platform/profile`：窗口关闭偏好。
- `frontend/src/features/accounts`：激活设置面板、提示条、状态查询。
- `frontend/src/app/routes/__root.tsx`：应用级挂载点。
- `frontend/src/i18n/locales`：中英文文案。

## Dependencies

无新增依赖。

## Risks

- 短间隔补检提高额度读取频率：用「候选窗口存在 + 连续上限 3 次」双重约束，之后回到 5 分钟。
- 事件推送频率：只在每轮检测结束时推送一次，事件不含身份、令牌或正文。
- 关闭行为被改成托盘属于用户可见变更：只在开启自动激活时校正，并在面板文案与文档中说明；托盘菜单仍可退出。
- 前端应用级监视器在缺少绑定（浏览器预览／测试）时不得抛出可见错误。

---

# Implementation Plan

1. `quotaactivation`：新增可配置的常规／补检间隔、候选标记与状态回调；`run` 改为可重置定时器，按需短间隔补检且有上限；检测结束时回调状态。
2. `transport/wails`：注册 `subscription:activation-updated` 事件并绑定状态推送；`SetCodexQuotaAutoActivation(true)` 与启动时校正窗口关闭行为到托盘。
3. 前端：新增 `use-codex-quota-activation` 共享查询（事件驱动 + 低频兜底）、`codex-quota-activation-watcher` 应用级监视器并在根路由挂载；设置面板改用共享查询并移除内联提示条。
4. 文案：中英文补充后台运行语义。
5. 测试：Go 侧覆盖短间隔补检、事件回调、关闭行为校正；前端覆盖应用级提示去重与安静轮次不提示、面板仍直接保存。
6. 文档：ADR-009 追加说明、SUBSCRIPTION、QUOTA、CURRENT。

---

# Progress

- 已定位三个根因：默认关闭窗口即退出应用、滑动候选要等下一个 5 分钟周期、状态与提示只在设置面板挂载时才存在。
- 后端自适应补检与状态回调完成，`task verify` 前先跑通包内定向测试。
- 前端改为应用级监视器后发现根布局测试缺少 QueryClient，确认根布局测试与本组件无关后按模块替换。
- 完整质量门 `task verify` exit 0（Go 全部包、130 文件／787 项前端测试、Windows production 构建）。

---

# Implementation Result

## Changes

实际完成：

- 后台循环改为可重置定时器，并新增短间隔补检：一轮检测中出现「接近满周期但证据不足」的候选时，
  约 35 秒后再检测一次；连续补检上限 3 次，之后回到 5 分钟周期。被限频的唤醒只推迟到原定最早
  时刻，不会把下一次检测越推越远。
- 每轮检测结束回调一次脱敏状态（只有计数与稳定错误码），桌面壳转成 `subscription:activation-updated` 事件。
- 开启自动激活时把窗口关闭行为落到托盘；启动时若开关仍为开启也校正一次。托盘「退出」仍是真正退出，
  用户之后显式改回「关闭窗口退出」不会被后台改写。
- 前端新增共享查询 `useCodexQuotaActivation`（事件驱动 + 30 秒兜底），设置面板改用它；
  新增应用级 `CodexQuotaActivationWatcher` 并在根布局挂载，结果提示不再依赖设置面板是否打开，
  同一结果（忽略完成时间）不重复弹出。
- 中英文文案说明后台运行语义。

## Files Changed

- `internal/application/quotaactivation/{types,lifecycle,check,account_check}.go`
- `internal/transport/wails/{app,gateway,desktop_settings,quota_activation}.go`
- `frontend/src/features/accounts/{use-codex-quota-activation.ts,codex-quota-activation-watcher.tsx,codex-quota-activation-settings.tsx,openai-account-settings.tsx}`
- `frontend/src/app/routes/__root.tsx`、`frontend/src/i18n/locales/subscription-{zh,en}.ts`
- 测试：`internal/application/quotaactivation/background_test.go`、`internal/transport/wails/quota_activation_test.go`、
  `frontend/src/features/accounts/codex-quota-activation-watcher.test.tsx` 及既有两个测试文件的适配

## Technical Notes

- 短间隔补检只影响「什么时候再读一次额度」，不改变判定条件、发送前复核与持久去重；补检本身也可能
  直接进入发送，与下一轮 5 分钟检测等价，只是更早。
- 状态事件与 `subscription:quota-updated` 分开：前者带计数，后者只是让额度缓存失效，两者都不含身份。
- 前端去重键忽略 `updatedAt`，否则每轮完成都会重复弹出同一条结果。

---

# Verification

## Tests

- [x] Unit Tests
- [x] Integration Tests
- [ ] E2E Tests（本仓库无 E2E 框架；等价覆盖见下）

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
exit=0
```

- PASS：`internal/application/quotaactivation`：新增 3 项测试——滑动候选在启动后由短间隔补检完成第二次观测并发送（发送时读取次数 ≥ 2，且常规周期被设为 1 小时，证明不是等满周期）、已开始窗口不请求补检、每轮结束只回调一次脱敏状态；包内既有 40+ 项测试全部通过。
- PASS：`internal/transport/wails`：新增 2 项测试——开启自动激活把关闭行为落到托盘、用户改回退出后不会被后台改写；包内既有测试全部通过。
- PASS：`go test .` 根包通过；`go vet . ./internal/...`、`staticcheck . ./internal/...`、`gofmt -l` 无输出。
- PASS：前端 130 文件／787 项测试通过，其中新增 `codex-quota-activation-watcher.test.tsx` 2 项（空轮次不提示＋事件推送后立即提示、未开启不查询不提示），并适配 `codex-quota-activation-settings.test.tsx`、`openai-account-rotation.test.tsx`（提示条改由应用级监视器渲染）、`page-scroll-restoration.test.tsx`（根布局新组件与本测试无关，按模块替换）。
- PASS：`pnpm lint`（eslint）、`pnpm typecheck`（tsc --noEmit）、`prettier --check` 无错误。
- PASS：`wails3 generate bindings` 重新生成后新增 `subscription:activation-updated` 类型事件（仅生成的 `eventdata.d.ts` 变化）；Windows production 构建产出 `bin/omniproxy.exe`。
- 未执行：真实账号后台激活、托盘常驻的系统级行为与真实关闭窗口后的长周期检测未在本机验收；本任务只验证代码路径、事件与前端行为。

---

# Documentation

- [x] PRD（第 72 行补充后台检测与自动刷新）
- [x] Architecture（OVERVIEW 新增「后台常驻与结果推送」小节）
- [ ] Tech Stack（无变化）
- [ ] Coding Rules（无变化）
- [ ] Testing（无变化）
- [ ] Build（无变化）
- [x] ADR（ADR-009 追加 2026-10-25 说明）
- [x] CURRENT
- [x] 产品文档：`SUBSCRIPTION.md`、`QUOTA.md`

---

# Deviations

1. 原计划「短间隔补检最多 3 次」在实现中保留，但补检的触发条件是「本轮存在 waiting 且仍满足
   `NeedsObservation`」。若候选长期无法判定，3 次后回到 5 分钟周期，不会持续高频查询。
2. 原计划自动激活开启时直接改写关闭行为；实现改为「只把 `exit` 改成 `tray`，已是托盘则不写、
   失败只记录告警、用户之后显式改回退出不会被后台覆盖」，避免把用户选择反复改回去。
3. 结果提示的去重键改为「计数 + 错误码」，不再使用整份状态 JSON（含完成时间），否则每轮检测
   都会重复弹出同一结果。原计划未写这一点，属于实现中发现的必要修正。

---

# Git

## Commit

```text
4ec3667 feat(quota): 自动激活常驻后台并自动刷新结果
```

---

# Result

自动激活不再依赖界面：开启后关闭主窗口会留在托盘，进程在后台按周期检测；出现滑动候选时启动／唤醒后
约 35 秒补一次独立观测（连续最多 3 次），不必等满 5 分钟；每轮结束推送脱敏状态，任意页面自动刷新并
提示结果，不需要打开设置面板。发送判定、持久去重与至多一次语义未变，`task verify` exit 0。
