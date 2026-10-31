# TASK-20261031-003: 账号 Token 用量改为终身累计

## Status

In Progress

## Created

2026-10-31 16:50

## Request

> 这个 token 额度的记录是不是有什么最大值？未实名我记得之前都 18b 了现在只有 704.69M 了
> 我之前的要求就是这个数值要是账号所有的 token，不分时间的，你看看该怎么优化
> 不能因为记录没有了就减少

## Goal

账号详情／配额页的「Token 用量」（输入、输出、缓存）与请求数必须是该账号身份的**终身累计**：请求明细被保留上限淘汰、账号或提供商被删除、应用重启都不能让数值变小。上游未返回的明细仍保持未知，不伪造零。

## Background

排查确认（本机 `%APPDATA%\OmniProxy\v2\omniproxy.db` 实测）：

- 账号卡片此前 `SUM(request_records ...)`，而 `request_records` 受 `request_retention.max_records`（默认 10000）约束，每次 `Begin`／`Finish` 都会淘汰最老已结束明细。
- 同一账号身份在「用量」页（`usage_buckets`／`usage_totals`，按日／提供商聚合、不受淘汰影响）与账号卡片上的数字相差 1.5–2.4 倍，例如 `rv_1jfif`：用量页 1,793,527,338，卡片 749,404,763；`mibgb65@gmail.com`：1,023,898,320 对 703,496,436。
- 因此卡片数值会随时间下降（用户看到的「18 亿 → 704.69M」），与「账号所有 token、不分时间」的产品预期相反。

## Scope

- 新增迁移 045：按记账身份索引（`subject`）保存的终身累计表 `usage_accounts`，并从 `usage_buckets` 回填。
- `activity.Update` 在记账事务内按差值更新 `usage_accounts`（输入、输出、缓存、已知计数、请求数、完整用量数），与 `usage_totals`／`usage_buckets`／`usage_provider_models` 同源、同事务。
- `AccountTokenUsage` 改从 `usage_accounts` 读取；`pendingRequests` 仍从留存明细取（只用于「正在使用」圆环）。
- 缓存明细无法从桶回填的部分只回填保留期内可确认的值，界面继续显示下界（`≥`）。

## Out of Scope

- 不改前端：`AccountTokenUsage` 的 JSON 结构与三列显示、`≥`／`—` 语义保持不变。
- 不改变「用量」页、仪表盘、请求明细的成本估算口径。
- 不回填已被淘汰明细的具体 Token（只保证此后的累计不再丢失）。

## Existing Behavior

- 账号卡片 Token 用量 = 当前留存明细的合计，明细淘汰后变小，可低于真实用量；账号无明细时显示已知零。
- 同一账号在用量页与账号卡片上数值不一致。

## Expected Behavior

- 账号卡片 Token 用量 = 该身份自记账以来的累计，明细淘汰、账号删除、重启都不减少；只有新用量会增加。
- 升级后首次打开即显示历史累计（从 `usage_buckets` 回填，含 10-18 以来的全部天）。
- 上游未返回的明细仍显示 `—`，只有部分缓存明细时显示 `≥`。

---

# Analysis

## Root Cause

`AccountTokenUsage` 的数据源（留存的 `request_records`）本身是有界的展示数据，被当成累计数据使用。

## Related Modules

- `internal/application/activity`（`observer.go`、`account_token_usage.go`、`quota_estimate.go`、`usage.go`、`retention.go`）
- `internal/transport/wails/activity.go`（绑定不变）
- `migrations/`

## Dependencies

N/A

## Risks

- 迁移需回填历史；只读保留期内的缓存明细，回填值为下界（界面已用 `≥` 区分，不会假的满额）。
- 聚合在记账事务内新增一条 upsert，写入路径多一次语句；读路径由 `GROUP BY` 扫描改为主键查询，账号页多卡渲染更快。

---

# Implementation Plan

1. 迁移 045：建 `usage_accounts`（subject 主键），从 `usage_buckets` 回填 Token／请求／完整用量，并按 `request_records.subject` 回填缓存已知计数。
2. `observer.go`：`Update` 读取旧 `input_known`／`output_known`／`cached_input_tokens`，计算差值（只增不减），在同一事务内 upsert `usage_accounts`。
3. `account_token_usage.go`：改读 `usage_accounts`（主键查询），`pendingRequests` 仍取留存明细；保留未知／部分已知／已知零语义。
4. 测试：更新受影响的用例，新增「明细淘汰后账号累计不变」回归测试。
5. 文档：`ACTIVITY.md`、`PRD.md`、`RIVER.md`、`QUOTA.md`、`SUBSCRIPTION.md`。

---

# Progress

- 定位根因：卡片读 `request_records`（受保留上限淘汰），用量页读 `usage_buckets`（不淘汰），两者相差 1.5–2.4 倍；库内 `request_records` 恒等于 10000 行，历史已写入 20124 条，说明淘汰一直在删。
- 决策：新增按身份索引（`subject`）的累计表 `usage_accounts`，与 `usage_totals`／`usage_buckets`／`usage_provider_models` 在同一记账事务里按差值更新；读路径改主键查询。不在 `request_records` 上做「补回」，因为明细淘汰是刻意的容量策略。
- 用真实库副本验证回填：`rv_1fjif` 回填 1,798,841,626 输入／11,653 请求，`mibgb65@gmail.com` 1,029,907,068，与「用量」页一致。

---

# Implementation Result

## Changes

实际完成：

- 迁移 045 新建 `usage_accounts`（`subject` 主键，含 requests／completed_usage／input＋output／input_known＋output_known／cached_input_tokens＋cached_known）。建表后从 `usage_buckets` 回填 Token、请求数与完整用量数（`unassigned` 与空身份不入表），再从仍保留的 `request_records` 回填缓存已知计数与缓存 Token。
- `activity.Update` 在记账事务内新增 `updateAccountUsage` upsert：请求数只在首次绑定身份时 +1，Token／完整用量按差值累计，`input_known`／`output_known` 只增，缓存「只增不减」（后续快照缺缓存字段或上游把同一请求的缓存值改小时不回退）。同时把明细的缓存写入改为「已知值只增不减、缺少缓存字段不改写」，避免同一份缓存被重复计入，也避免已知缓存被后续快照降级为未知（未知与真实零继续区分）。
- `tokenUsage`／`AccountUsage`／`accountTokenUsage` 改读 `usage_accounts` 主键；`pendingRequests` 仍从留存明细统计（只用于「正在使用」圆环）。未知（`—`）、部分已知（`≥`）、已知零的判定语义保持不变。
- 身份边界与每日桶／用量页对齐：订阅身份为「提供商类型 + 邮箱 + 工作区」，API Key 身份为「提供商类型 + 账号 ID」；本地账号行或提供商配置 ID 不再参与拆分（这正是删除重加后仍能继续合并历史、且卡片数字不变小的前提）。

## Files Changed

- `migrations/045_account_usage_totals.sql`
- `internal/application/activity/account_token_usage.go`
- `internal/application/activity/observer.go`
- `internal/application/activity/account_token_usage_test.go`
- `internal/application/activity/account_usage_test.go`
- `internal/application/activity/account_usage_migration_test.go`
- `internal/application/gateway/accounting_test.go`
- `docs/product/ACTIVITY.md`、`PRD.md`、`QUOTA.md`、`RIVER.md`、`SUBSCRIPTION.md`

## Technical Notes

- 累计值与「用量」页、仪表盘、提供商／模型汇总同源同事务，因此三者不会互相漂移。
- 明细淘汰后 `usage_accounts` 与 `SUM(request_records)` 不再相等，这是本任务的目的；缓存累计在明细缺失时仍保持下界语义。

---

# Verification

## Tests

- [x] Unit Tests：`account_token_usage_test.go` 新增「淘汰 5 条明细后累计仍是 105000／删除账号行不减少」与「同一请求重复缓存快照不重复计入、不降级为未知」两项回归；`account_usage_test.go` 的删除断言改为终身口径
- [x] Integration Tests：`account_usage_migration_test.go` 覆盖 045 回填（Token／请求／已知计数／缓存／排除 unassigned）；`gateway/accounting_test.go` 补上 045 才能建出的 `usage_accounts`
- [ ] E2E Tests：N/A

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
exit 0
gofmt / frontend format:check 通过
staticcheck . ./internal/... 通过
go test . ./internal/...：51 个包 ok
frontend vitest：146 文件 / 890 项通过
Windows production 构建：bin/omniproxy.exe 23,309,312 bytes
```

补充说明：实现过程中一次 `task verify` 先返回 exit 1，日志里没有任何失败步骤（Go 测试与前端测试均通过），随后重跑即 exit 0，未复现；随后又因缓存明细只增不减的改动重跑一次，同样 exit 0。真实库副本（`%APPDATA%\OmniProxy\v2\omniproxy.db` 的拷贝）应用迁移 045 后回填结果已核对：`rv_1fjif` 1,798,841,626 输入 / 11,653 请求、`mibgb65@gmail.com` 1,029,907,068 / 4,081 请求，与「用量」页一致，`unassigned` 未进表。

---

# Documentation

- [x] PRD：Token 三列改为终身口径，累计表补一句
- [x] ACTIVITY：统计规则加迁移 045，`## 账号本机用量` 重写数据源与身份口径
- [x] RIVER：账号卡片 Token 用量改为终身累计，删除「清理请求历史会缩小统计范围」
- [x] QUOTA：无官方额度接口账号的本机用量改为终身累计
- [x] SUBSCRIPTION：估算下方 Token 三列改为终身累计
- [x] CURRENT

---

# Deviations

1. 原计划「卡片改读 `usage_buckets`」→ 实际新增独立累计表 `usage_accounts`。原因：桶按 `(day,subject,model)` 分组，读一侧要 `GROUP BY` 扫描且没有缓存／已知明细字段，而账号页同时渲染多张卡片；独立表读是主键查询，写入与其它累计同事务。
2. 原计划保持「subject + provider_id + account_id + email + workspace」多重过滤 → 实际只按 `subject`。原因：这两者本来就矛盾——订阅身份索引刻意不含本地账号 ID（删除重加要合并历史），旧过滤又按当前账号 ID 排除，结果同一账号在用量页与卡片上数值不同。统一到 `subject` 后卡片与用量页一致，且删除重加不会让数字归零。代价：同一成员邮箱＋工作区若存在两条本地账号行，会显示同一份累计（与用量页一致）。
3. 计划外修复：明细的缓存字段原来会被「已知输入但未带缓存字段」的后续快照写成 NULL，导致累计要么被降级成未知、要么在再次上报时重复计入。现改为「已知缓存只增不减、缺少缓存字段不改写」，未知与真实零继续区分。

---

# Git

## Commit

（回填）

---

# Result

账号卡片／配额页的「Token 用量」现在是该记账身份的终身累计：请求明细淘汰、账号或提供商删除、应用重启都不会让它变小，只有新的代理请求会增加。实测本机数据中 `rv_1jfif` 的卡片从 749,404,763 回到与「用量」页一致的 1,798,841,626（升级后由迁移 045 回填），`mibgb65@gmail.com` 从 703,496,436 回到 1,029,907,068。

---

# Completed

（回填）
