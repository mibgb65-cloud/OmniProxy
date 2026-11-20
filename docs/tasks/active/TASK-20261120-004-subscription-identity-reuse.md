# TASK-20261120-004: 同一订阅账号重新登录复用原账号记录

## Status

Planned

## Created

2026-11-20 10:20

## Request

用户给出两张账号卡截图：`mihbnjdjdhhdhy@gmail.com`（Google / Antigravity）在账号页同时出现两份——
一份「额度分组未知／已退出登录」（创建 2026-11-18 10:23，更新 2026-11-19 15:32），
另一份带完整额度（创建 2026-11-19 21:03，更新 2026-11-20 08:09）。原始需求：

> 还有一个 bug：这两个怎么会同时存在，重新登录了相同账号应该更新掉原有账号的信息吧。

补充确认（用户答复）：

1. 已有的重复卡片由用户手动删除，本任务不改动任何历史数据（不加迁移、不自动删除账号）。
2. ChatGPT／Codex 订阅登录路径上的同类问题一并修复。

## Goal

同一订阅身份（提供商 + 规范化邮箱 + 官方 account_id）在账号表中只保留一条记录：
退出登录后用「添加账号」重新授权同一账号时更新原记录，不再产生第二张卡片；
Google / Antigravity 与 ChatGPT／Codex 两条订阅登录路径行为一致。

## Background

- 订阅账号身份规则在 `docs/product/ACTIVITY.md` 已写明：统计身份为「提供商类型 + 规范化邮箱 + account_id」；
  Team 工作区同一 account_id 的不同成员邮箱仍是不同账号。
- 本机生产库（`D:\OmniPProxy-data\omniproxy.db`，只读核查）确认存在两条 `builtin_google` 记录：
  `acct_4433b0d6…`（`auth_state=logged_out`、`credential_ref=''`、创建 2026-11-18T02:23:54Z）
  与 `acct_e98b09fe…`（`authenticated`、创建 2026-11-19T13:03:48Z），
  两者 `auth_email` 与 `auth_account_id`（`107219188182686367023`）完全相同。
- 账号页「添加账号」弹窗不携带 `accountId`（只有卡片上的「重新登录」携带）。

## Scope

- `internal/application/account`：新增订阅身份查询，`SaveGoogleSubscription` 在新建分支前按身份复用原记录；
  允许旧记录缺失 `auth_account_id` 时由登录补全。
- `internal/application/subscription`：Codex 登录在未指定 `accountId` 时先按身份复用原记录，走既有的替换授权与清理路径。
- 回归测试：Google 身份复用／不同身份新增／旧记录补全身份；Codex 二次登录只保留一条账号记录。
- 文档：`docs/product/GOOGLE_SUBSCRIPTION.md`、`docs/product/SUBSCRIPTION.md`、`docs/product/ACTIVITY.md`（如涉及）、
  ADR-029、`docs/plans/CURRENT.md`。

## Out of Scope

- 不新增数据库迁移，不做重复账号自动合并或删除（历史数据由用户处理）。
- 不改变 WorkBuddy 登录路径。
- 不改变「重新登录只能替换同一 Google 身份」的既有约束。
- 不做真实账号授权复测（登录需要用户本人浏览器授权）。

## Existing Behavior

- Google：`SaveGoogleSubscription` 在 `id == ""` 时无条件 `INSERT INTO accounts`；
  同一账号退出登录后再次「添加账号」会生成第二条记录。
- Codex：`subscription.lifecycle` 在 `job.input.AccountID == ""` 时调用 `CreateSubscription`，
  同样无条件插入，重复添加相同 ChatGPT 账号会产生第二条记录。
- 两次授权会得到不同 `credential_ref`，于是 `account_catalog_versions` 视图版本不同：
  重复卡片各自持有目录与额度快照，网关把它们当成两个候选账号。

## Expected Behavior

- 退出登录后重新登录同一账号（无论走「添加账号」还是卡片「重新登录」）只有一条账号记录，
  `id`、`name`、`enabled`、`created_at` 保持，`auth_state`／`auth_plan`／`auth_email`／凭据与新授权一致；
  旧授权版本的目录与额度快照不重绑定（与既有「重新登录不恢复旧快照」一致）。
- 换了身份（不同 account_id，或同时有 account_id 且不同的邮箱）仍然新增账号。
- 同邮箱但旧记录缺少 `auth_account_id` 时视为同一身份并补全，不再新增重复记录。
- 身份不完整（邮箱或 account_id 缺失）时不做合并，保持既有保存行为。

---

# Analysis

## Related Modules

- `internal/application/account/google_subscription.go`（Google 授权保存）
- `internal/application/account/subscription.go`（Codex 订阅创建）
- `internal/application/subscription/lifecycle.go`（Codex 登录完成）
- `internal/application/account/metadata.go`、`service.go`（账号读取）
- `internal/application/googlesubscription/login.go`（调用方，无需改动）

## Root Cause

授权成功后保存账号的两条路径都只按「有没有传 accountId」决定新增还是更新：
前端「添加账号」不传 `accountId`，因此同一账号再次登录必然走新增分支，
身份（邮箱 + account_id）从未参与判重。

## Risks

- 误合并：若判重条件太宽，会把 Team 中同邮箱不同工作区的成员或不同账号合并。
  处理：身份必须同时满足规范化邮箱相同，且 `account_id` 相同或其中一方为空。
- 覆盖有效授权：合并会替换原记录的凭据；同一身份下这就是用户期望的结果（重新登录），
  且目录／额度快照会按新授权版本失效。
- 并发：登录保存本身在 `s.ops` 全局互斥内，查询与更新之间不会被另一个登录插入打断。

---

# Implementation Plan

1. 在 `internal/application/account` 增加订阅身份判重查询（按提供商 + 规范化邮箱 + account_id）。
2. `SaveGoogleSubscription`：`id == ""` 时先判重，命中则走既有替换授权路径；放宽「旧记录缺 account_id」的校验。
3. `subscription.lifecycle`：未指定 `accountId` 时先判重，命中则复用替换授权 + 清理旧隔离配置的分支。
4. 增加 Go 回归测试（account 包 3 个、subscription 包 1 个）。
5. 更新产品文档、ADR-029、CURRENT，运行 `task verify` 并提交。

---

# Progress

- 2026-11-20 10:20 创建记录，完成根因定位（只读核查生产库两条重复记录）。

---

# Implementation Result

## Changes

实际完成：

- 新增 `account.Service.IdentityAccount`：按「提供商 + 规范化邮箱 + 官方 account_id」查已有订阅账号；
  旧记录缺少 account_id 时按同提供商同邮箱认领，双方 account_id 都存在且不同时不匹配。
- `SaveGoogleSubscription`：未传 accountId 时先判重，命中则更新原记录；允许旧记录缺少 `auth_account_id` 时由本次授权补全
  （`writeGoogleCredentials` 的 UPDATE 现在同时写入 `auth_account_id`）。
- `subscription.lifecycle`：Codex 登录未传 accountId 时先判重，命中则走既有替换授权与 `cleanRetired` 清理路径；
  身份查询失败按 `account_unavailable` 失败保存，不静默新增重复账号。
- 回归测试：Google 身份复用／不同身份新增（含同 account_id 不同邮箱）／旧记录补全身份；
  Codex 二次登录复用原记录并清理旧隔离配置；身份元数据读取失败仍保存已完成的授权。

## Files Changed

- `internal/application/account/subscription_identity.go`（新增）
- `internal/application/account/google_subscription.go`
- `internal/application/account/google_subscription_test.go`
- `internal/application/subscription/lifecycle.go`
- `internal/application/subscription/identity_reuse_test.go`（新增）
- `internal/application/subscription/test_helpers_test.go`
- `docs/decisions/ADR-029-subscription-identity-uniqueness.md`（新增）、`docs/decisions/README.md`
- `docs/product/{PRD.md,ACTIVITY.md,GOOGLE_SUBSCRIPTION.md,SUBSCRIPTION.md}`

## Technical Notes

- 判重发生在授权成功、拿到官方身份之后的保存路径，因此对「添加账号」「卡片重新登录」「事件恢复」全部入口生效。
- 复用路径保留 `id`／`name`／`enabled`／`created_at`，只更新凭据与身份元数据；`account_catalog_versions` 随新
  `credential_ref` 变化，旧目录与额度快照按既有「重新登录不恢复旧快照」语义失效。
- 不改动历史数据：已有重复卡片由用户在账号页手动删除；用量累计按 `usage_accounts.subject` 保存，删除重复卡片不减少累计。

---

# Verification

## Tests

- [x] Unit Tests：`internal/application/account`（Google 身份三个用例）、`internal/application/subscription`（复用与元数据失败两个用例）
- [x] Integration Tests：Codex 登录生命周期在真实迁移后的 SQLite 上完成两次授权的账号复用与旧配置清理
- [ ] E2E Tests：N/A（真实 Google／ChatGPT 授权需用户本人浏览器登录）

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
EXIT=0
```

Go 全部包通过；前端 164 文件／1005 项测试通过；Windows production 构建完成。

---

# Documentation

- [x] PRD（订阅身份唯一的账号行为）
- [x] ACTIVITY（账号表身份唯一，链接 ADR-029）
- [x] ADR-029（订阅账号身份唯一，重新登录复用原记录）
- [x] GOOGLE_SUBSCRIPTION、SUBSCRIPTION（添加账号复用行为与已有重复卡片的手动处理）
- [x] CURRENT（构建与任务看板）

---

# Deviations

- 原计划只修 Google；按用户答复一起修 Codex 登录路径。
- 原计划未提及历史重复数据的自动清理；按用户答复不做任何自动删除（只修今后登录）。

---

# Git

## Commit

```text
待回填
```

---

# Result

同一订阅身份在账号表中只保留一条记录：未携带本地账号的重新授权（「添加账号」）更新原记录，
不再产生重复卡片；Google 与 ChatGPT／Codex 两条订阅登录路径一致。已有重复卡片保持原样，由用户手动删除。

---

# Completed

待回填

