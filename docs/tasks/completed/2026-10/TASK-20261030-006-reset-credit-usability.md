# TASK-20261030-006: 重置卡「使用这张卡」按钮不可用

## Status

Completed

## Created

2026-10-30 22:05（Asia/Singapore）

## Request

账号的「重置卡」面板里，卡片上的「使用这张卡」按钮是灰的，点不动；
用户怀疑后端接口没接上。

## Goal

让「可用」徽标与「使用这张卡」按钮使用同一判定；可用卡一定能点、不可用卡一定能说明原因。

## Background

面板数据来自官方 `/backend-api/wham/rate-limit-reset-credits`（TASK-20261027-015 接入）。
后端 `SubscriptionResetStatus` / `SubscriptionConsumeReset` 两个绑定都已存在并注册，
不是缺接口。

## Scope

- 单张重置卡的「可用」判定（前端展示与后端兑换前置检查）。
- 不可用状态的文案与测试。

## Out of Scope

官方接口字段解析、兑换的幂等与回执语义、重置卡数量读取路径、其他订阅功能。

## Existing Behavior

- 徽标只看 `card.status`，`available` 时显示「可使用」。
- 按钮要求 `status === "available" && resetType === "codexRateLimits" && 未过期`。
- 后端 `checkResetCredit` 同样要求 `resetType == "codexRateLimits"`。

---

# Analysis

## 现象定量

截图（377×438，即 360×440 的弹窗）灰度采样：

| 区域 | 峰值亮度 |
| --- | --- |
| 卡片标题 | 250（正常白字） |
| 「使用这张卡」 | 138 |
| 「使用下一张可用卡」 | 250 |
| 刷新图标 | 138 |

「使用这张卡」与刷新图标是半透明禁用态，底部「使用下一张可用卡」是正常启用态。
弹窗内没有 pending 警示块、也没有错误提示，说明状态查询已成功返回、没有待核对操作，
`locked` 为 false —— 这也解释了为什么底部按钮是亮的。

## 根因

底部按钮只用 `locked || !summary || availableCount <= 0`，与卡片类型无关，所以它是亮的；
单张卡按钮还要求 `resetType === "codexRateLimits"`，而徽标只看 `status`。两者口径不一致：

- 徽标（status）：「可使用」
- 按钮（status + resetType + 有效期）：禁用

即官方这张卡的 `reset_type` 不是字面量 `codexRateLimits`（大小写或下划线写法不同），
界面就出现「写着可使用、按钮是灰的」。同一字面量比较也写在后端 `checkResetCredit` 里，
即使前端放开也会被后端以 `reset_not_usable` 拒绝。

---

# Implementation Plan

1. 后端：把「可用」判定抽成 `usableResetCredit`，`reset_type` 只做大小写／下划线归一
   （`codexRateLimits` / `codex_rate_limits` / `rateLimits` 均可用），状态与有效期仍是权威信号。
2. 前端：抽出同一份归一判定，「可用」徽标与按钮共用，避免再次出现口径分叉；
   状态为 available 但类型不被接受时徽标显示「当前不可兑换」。
3. 补 Go 与前端测试，跑 `task verify`，然后出 `v2.2.1-beta.3` 供本地更新。

---

# Implementation Result

## 代码

| 位置 | 改动 |
| --- | --- |
| `internal/application/subscription/reset.go` | 新增 `usableResetCredit` / `usableResetType`；`checkResetCredit` 改用归一判定（大小写与 `_`/`-`/空格忽略），过期、已使用、其他权益类型仍拒绝 |
| `frontend/src/features/accounts/reset-credit-list.tsx` | 新增 `usableResetType` / `resetCreditUsable`；徽标与按钮共用；状态 available 但类型不被接受时徽标显示「当前不可兑换」 |
| `frontend/src/features/accounts/reset-credits-content.tsx` | 刷新按钮只在兑换进行中禁用，读取中图标 `motion-safe:animate-spin`；核对记录期间显示「正在核对最近一次使用记录…」 |
| `frontend/src/i18n/locales/subscription-{zh,en}.ts` | 新增 `subscription.reset.status.unusable`、`subscription.reset.checking` |

## 交付构建（v2.2.1-beta.3）

`scripts/build-release.ps1 -Version v2.2.1-beta.3 -OutputDirectory release-assets/v2.2.1-beta.3`
（日志 `tmp/release-v2.2.1-beta.3b.log`，exit 0）：

| 项目 | 结果 |
| --- | --- |
| 脚本内 `task --force verify` | exit 0（145 文件／887 项前端测试） |
| NSIS 打包 | `Total size: 20356443 / 81203528 bytes (25.0%)` |
| 安装器 | `release-assets/v2.2.1-beta.3/OmniProxy-Setup-v2.2.1-beta.3-windows-amd64.exe`，20,356,443 bytes |
| SHA256 | `5ba8696e9490ab5c57fab45fe79e3bb6a461081e9ea3bf4e7fc80a4f4ff95553` |
| `bin/omniproxy.exe` | 23,301,632 bytes，SHA256 `e704b9b063b82b058b3cd36341c23a1cac723fe1f05e53e6cae90f8e2d9f59c6` |

首次构建在 `format:check` 失败（`reset-credits-content.tsx` 未按 Prettier 格式化），
格式化后重建通过；`build/windows/info.json` 被脚本改写为 CRLF，提交前已还原为 LF。

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
| `go test ./internal/application/subscription -run TestReset` | 通过（新增 4 种写法各一项；原有 5 项不可用分支仍拒绝） |
| `reset-credits.test.tsx` | 6 项通过（新增 2 项：写法归一并可兑换、其他权益类型标为不可兑换） |
| `reset-credits-popover.test.tsx` | 5 项通过 |
| `tsc --noEmit` | 通过 |
| 发布脚本内 `task --force verify` | exit 0（145 文件／887 项前端测试、Go 全量包、Windows production 构建） |
| `test-windows-metadata.ps1` / `verify-release-assets.ps1` | 通过／通过 |

未覆盖（需用户实机确认）：原生 WebView2 下点击「使用这张卡」是否真的完成兑换（取决于该账号的官方接口返回）。

---

# Documentation

- 新增 `docs/releases/v2.2.1-beta.3.md`（中英双语）。
- 更新 `docs/plans/CURRENT.md`（顶部摘要、`Current Version`、`Recently Completed`）。

---

# Deviations

1. 修复不依赖「官方 reset_type 到底是什么」：判定改为归一并保留状态与有效期作为权威信号，
   这样无论官方用哪种写法都能用；仍拒绝明显属于其他权益、已使用或已过期的卡。
2. 没有改动官方兑换接口、幂等键与回执语义；后端的前置检查只是与界面口径对齐。

---

# Git

## Commit

```text
557e3cd0 fix(subscription): 重置卡可用性判定统一并放宽类型写法
```

归档与 CURRENT.md 回填为后续文档提交。未推送、未打标签、未签名。

---

# Result

「使用这张卡」按钮不再因为官方 `reset_type` 写法差异而无故变灰：徽标与按钮共用同一归一判定，
后端兑换前置检查同步放宽；刷新按钮在读取期间保持可用并显示旋转，核对记录时有说明文案。
`task verify` exit 0（145 文件／887 项前端测试），`v2.2.1-beta.3` 安装包 20,356,443 bytes，
SHA256 `5ba8696e9490ab5c57fab45fe79e3bb6a461081e9ea3bf4e7fc80a4f4ff95553`，未签名、未自动安装、未推送、未公开发布。

---

# Completed

2026-10-30 22:20
