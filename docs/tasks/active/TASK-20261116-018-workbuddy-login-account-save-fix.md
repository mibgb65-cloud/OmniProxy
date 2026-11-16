# TASK-20261116-018: 修复 WorkBuddy AI 浏览器登录后无法保存账号

## Status

Completed

## Created

2026-11-16 20:40（Asia/Singapore）

## Request

用户安装 `v2.3.1-beta.15` 后在账号页点击「登录 WorkBuddy AI」→「开始授权」，浏览器完成
WorkBuddy AI 登录（页面提示返回官方客户端），返回 OmniProxy 后登录弹窗显示红色错误
「操作未完成，请检查网络和登录状态后重试。」，账号未创建。

## Goal

让浏览器登录成功后的账号能真正落库：修复新建账号写入失败，补上登录失败的稳定错误码与日志，
并在界面上说明「浏览器提示返回客户端属正常」，使失败时可诊断、可操作。

## Background

- 上游流程本身是通的：`POST /v2/plugin/auth/state` → 浏览器登录 → 轮询
  `GET /v2/plugin/auth/token` 拿到令牌 → `GET /v2/plugin/login/account` 拿到身份。
  失败发生在最后一步「保存账号」。
- 数据库证据（`%APPDATA%\OmniProxy\v2\omniproxy.db` 只读副本）：
  `providers` 已有 `builtin_workbuddy_ai`（迁移 050 已执行、`workbuddy_quota_snapshots` 表已建），
  但 `accounts` 中 **没有任何 `auth_method='workbuddy_login'` 的行**。
- 日志只有 `WorkBuddy AI model auto sync incomplete; retry is available`，没有任何登录失败记录
  （登录失败路径没有日志，属于「静默失败」）。
- 界面文案来自 `workbuddy.error.operation_failed`，说明后端返回的错误码为空
  （`workbuddyai.ErrorCode(err)` 取不到 `*workbuddyai.Error`／`*proxy.Error`）。

## Analysis

### 根因：新建账号的 INSERT 列数与值数不匹配

`internal/application/account/workbuddy.go` 的 `createWorkBuddyAccount`：

```sql
INSERT INTO accounts(id,provider_id,name,credential_ref,enabled,created_at,updated_at,
                     auth_method,auth_plan,auth_state,auth_account_id)
VALUES(?,?,?,?,1,?,?,?,?,?,'authenticated',?)
```

- 列 11 个；`VALUES` 里有 4 + 1(`1`) + 5 + 1(`'authenticated'`) + 1 = **12 个值**。
- 占位符 10 个，但只传了 9 个参数。

该语句照 `google_subscription.go` 抄写时去掉了 `auth_email` 列，却没有同步去掉对应的值。
SQLite 在 prepare 阶段直接失败：

```text
Error: in prepare, 12 values for 11 columns
```

（用 sqlite3 在数据库副本上按同一列顺序复现，同样的 11 列 + 12 值即报此错。）

`createWorkBuddyAccount` 把该错误包成 `errors.Join(ErrInvalidInput, store.Delete(ref))`，
而 `login.go` 的 `errorCode` 只认 `*workbuddyai.Error` 与 `*proxy.Error`，因此返回空码，
最终落到 `operation_failed`，界面只能显示通用文案。**只有「新建账号」会走到这条语句**，
重新登录/续期走 `writeWorkBuddyCredentials`，所以这是首装用户的必现故障。

### 次生问题

1. **登录失败无日志**：`workbuddy.Service.complete` 失败时只把码写进会话状态，
   `transport/wails` 也没有 `SetWarn` 之类的钩子，现场只剩自动同步的 WARN。
2. **错误码未映射**：账号层的 `ErrInvalidInput`／`ErrConflict`（以及真实的 DB 失败）都会
   退化成 `operation_failed`，用户无法判断该重试、重新登录还是报告 bug。
3. **界面说明缺失**：登录页把带 `loginSessionId` 的授权视为桌面端来源，登录完成后会提示
   返回 WorkBuddy 客户端（并可能拉起已安装的客户端）。这是上游页面的既有行为，
   但弹窗没有任何说明，用户会以为登录跑到了别处。

## Scope

- 修复 `createWorkBuddyAccount` 的 INSERT（列、占位符、参数三处对齐）。
- 账号保存失败改为稳定码 `save_failed`；`login.go` 增加 `invalid_input`／
  `configuration_changed` 映射；补 `workbuddy.Service.SetWarn` 并在登录失败时记录
  阶段、错误码与错误文本（不含令牌）。
- Wails 层接线 `SetWarn`，并对 `SubscriptionBegin`／`openWorkBuddySubscription` 的失败记 WARN。
- 前端：错误码白名单补 `save_failed`／`invalid_input`，中英文案；登录弹窗增加一句
  「浏览器提示返回客户端属正常，登录结果会自动回到这里」的说明。
- 测试：账号创建（回归 INSERT）、续期重绑目录、错误码映射、前端错误码映射。
- `docs/product/WORKBUDDY.md` 补充登录流程与新增错误码。

## Out of Scope

- 不改上游协议、UA 指纹与请求头。
- 不改变「额度只展示、不参与调度」的既定策略。
- 不做真实积分消耗的调用验证（Chat／额度读取）；本次只验证登录与账号落库路径。
- 不改数据库结构（迁移 050／051 已随 beta.15 发布，无新迁移）。

---

# Implementation Result

## Changes

- **修复新建账号 INSERT**（根因）：`internal/application/account/workbuddy.go` 的
  `createWorkBuddyAccount` 改为 11 列对应 11 个值、9 个占位符对应 9 个参数
  （`VALUES(?,?,?,?,1,?,?,?,?,'authenticated',?)`）。原语句照 Google 版抄写时删了
  `auth_email` 列但留下了它的值，SQLite 在 prepare 阶段就报 `12 values for 11 columns`。
- **区分「登录失败」与「保存失败」**：新增 `account.ErrWorkBuddySave`；
  `createWorkBuddyAccount` 的写入失败与 `writeWorkBuddyCredentials` 的数据库/事务失败
  都带它（版本冲突仍保留 `ErrConflict`）。
- **错误码换算**：`workbuddy/login.go` 的 `errorCode` 在适配器码之后追加账号层映射——
  `ErrWorkBuddySave → save_failed`、`ErrConflict → configuration_changed`、
  `ErrInvalidInput → invalid_input`，未知才落 `operation_failed`。
- **登录失败可诊断**：`workbuddy.Service.SetWarn` 新增诊断钩子，`complete` 在失败时带上
  阶段（`token`／`identity`／`save`）与稳定码上报；`transport/wails` 接到 `s.log.Warn`，
  并对 `SubscriptionBegin` 的失败补 `WorkBuddy AI authorization request failed` 日志。
- **界面**：错误码白名单与中英文案补 `save_failed`／`invalid_input`；浏览器授权卡片新增一句
  说明（`workbuddy.loginReturnHint`）——上游授权页会提示返回 WorkBuddy 客户端，属正常现象。
  登录弹窗的启动/操作错误改为按品牌映射（`useSubscriptionLogin` 的 `mapError` 选项），
  否则 WorkBuddy 的启动失败仍会走 Codex 的错误字典。
- **测试**：
  - `internal/application/account/workbuddy_test.go`（新）：新建账号落库（回归本次 INSERT）、
    同身份重新登录不重复建号、令牌不进账号元数据、续期后目录仍绑定新版本、
    受保护存储失败给出 `credential_unavailable`。
  - `internal/application/workbuddy/login_test.go`：账号层错误码换算、诊断钩子的阶段与码。
  - `frontend/src/services/workbuddy.test.ts`（新）：错误码前缀识别与 `save_failed`／
    `invalid_input` 白名单、未知回落。

## Files Changed

- `internal/application/account/workbuddy.go`
- `internal/application/account/workbuddy_test.go`（新）
- `internal/application/workbuddy/login.go`、`service.go`、`login_test.go`
- `internal/transport/wails/subscription.go`、`workbuddy_subscription.go`
- `frontend/src/services/workbuddy.ts`、`frontend/src/services/workbuddy.test.ts`（新）
- `frontend/src/features/accounts/subscription-login-dialog.tsx`、`use-subscription-login.ts`、
  `subscription-browser-login-card.tsx`、`subscription-login-config.ts`、
  `frontend/src/i18n/locales/workbuddy-{zh,en}.ts`
- `docs/product/WORKBUDDY.md`

## Technical Notes

- 修复后同一份 SQL 在数据库副本上用 sqlite3 复验：11 列 + 11 值可插入；
  原语句形状稳定复现 `12 values for 11 columns`。
- 保存失败不要求用户重新登录：令牌仍在上游有效，重试保存即可；因此单独给 `save_failed`
  而不是复用 `sign_in_expired`。
- 登录失败的日志只含阶段与稳定码，不含令牌、账号名或上游正文（Redactor 之外再加一层）。
- 无数据库结构变化：迁移 050／051 已随 `v2.3.1-beta.15` 发布，本修复只改 SQL 语句与错误处理。

---

# Verification

## 质量门（`task verify`，2026-11-16 20:37–20:50 Asia/Singapore）

```text
task verify → exit 0
```

| 阶段 | 结果 |
| --- | --- |
| format | `gofmt -l main.go internal` 无输出；prettier 全部通过 |
| lint | `go vet`／`staticcheck` 无告警；`eslint` 0 error（`reset-credit-list.tsx` 两条既有 warning） |
| typecheck | `tsc --noEmit` exit 0 |
| test | `go test . ./internal/...` 全通过；前端 **157 files / 964 tests**（+1 文件、+3 测试） |
| build | 绑定重新生成 + vite（2789 modules）+ Windows production 构建 |

## 分项验证

- `go test ./internal/application/account/ ./internal/application/workbuddy/ -count=1`：
  新增的 4 条 Go 测试通过（新建账号落库、同身份不重复建号、续期保持目录、保存失败可识别）。
- `cd frontend && npx tsc --noEmit` exit 0；`npx vitest run src/services/workbuddy.test.ts src/features/accounts`
  → 44 files / 248 tests 通过。
- **SQL 复验**：在用户数据库副本上用 sqlite3 按原语句的列顺序执行
  `... VALUES('y',…,'u','extra')` → `Error: in prepare, 12 values for 11 columns`；
  修正后的 11 列 11 值语句插入成功。与代码侧根因一致。

## 未验证

- 真实账号重试登录（安装 `v2.3.1-beta.16` 后由用户执行）：本次只能验证到「保存路径不再报错」，
  目录同步与后续 Chat 仍需真实账号。
- 上游 `/v3/config`、真实 Chat、Credits 读数仍未验证（消耗积分）。

---

# Deviations

- 计划只写「修复 INSERT」，实际同时补了失败日志与错误码换算：根因正是因为错误码塌成
  `operation_failed` 且没有日志，才让一次 SQL 拼写错误表现为「网络或登录状态问题」。
- 前端新增 `mapError` 选项（`useSubscriptionLogin`）：不改它的话，WorkBuddy 的 Begin 失败
  仍会被 Codex 的错误字典吞成通用码。
- 无其他偏离；未新增迁移、未改上游协议。

---

# Documentation

- [x] `docs/product/WORKBUDDY.md`（浏览器授权提示返回客户端属正常、`save_failed`／`invalid_input` 错误码、真实验证进度）

---

# Git

## Commit

Not committed by Agent.

---

# Result

`v2.3.1-beta.15` 上「浏览器登录成功却建不出账号」是**本地 SQL 写错**，不是登录协议问题：
上游 `auth/state → auth/token → login/account` 三步在真实账号上均已走通，失败只在最后的
INSERT——11 列的语句配了 12 个值，SQLite 直接拒绝；而错误码又塌成 `operation_failed`、
没有任何日志，于是界面显示成「请检查网络和登录状态」。

修复后：新建账号能正常落库；保存失败给 `save_failed`（可重试，不必重新登录）；
版本冲突给 `configuration_changed`；登录失败在日志里留下阶段与稳定码；浏览器授权卡片
明确说明「提示返回 WorkBuddy 客户端属正常」。真实验证需安装新构建（TASK-20261116-019）。

---

# Completed

2026-11-16 20:52（Asia/Singapore）