# TASK-20261117-009: 受管客户端目录刷新失败可见并能一键恢复

## Status

Completed

## Created

2026-11-17 17:05（Asia/Singapore）

## Request

用户要在 DeepSeek Harness 里调用 OmniProxy 的模型，发现国内版同步的模型没出现在 DSH 的
OmniProxy 提供商里。排查结论：受管客户端配置被客户端自己改写过（`agent_backups.applied_hash`
与当前文件哈希全部不一致），后台目录刷新被安全策略挡住，只在日志里记
`managed client catalog refresh failed / refresh_failed`；界面上「一键配置」按钮对
`config_changed` 是**禁用**的，用户只能自己发现「重新读取配置」这个刷新图标。

## Goal

让「受管客户端目录为什么没有自动更新」在界面上一眼可见、并且一步可恢复：

- 后台目录刷新失败时，把**每个客户端**的稳定失败码保留并在客户端卡片上显示（不再只有日志）；
- `config_changed` 时主按钮不再是死路：提供「重新读取并应用」，一次点击完成读取＋预览＋应用；
- 客户端接入页给出汇总提示，说明目录已停止自动更新与恢复方式。

## Background

- 刷新逻辑：`internal/application/agentconfig/refresh.go` 的 `RefreshCatalog` 只改写
  「文件仍等于上次写入版本」的客户端；外部改动返回 `config_changed`（`writeAtomic` /
  `refreshClient` 的版本校验）。这是有意的保护，不是 bug。
- 现有 UI：`agent-client-settings.tsx` 在 `item.errorCode` 非空时禁用「一键配置」与「恢复」，
  只显示 `agents.error.config_changed` 文案；`agents.reread`（刷新图标）才是正确路径。
- `Item.ErrorCode` 已由 `Inspect` 从文件哈希比对得出；但**其它刷新失败码**
  （`write_failed`、`config_locked`、`invalid_config`、`models_unavailable`、
  `backup_unavailable`…）完全没有暴露，只在日志里聚合一条 `refresh_failed`。
- 2026-11-17 实测：6 个受管客户端（codex、claude、claude-desktop、harness-web、
  harness-desktop、pi）全部 `config_changed`，DSH 的模型清单停留在旧版本。

## Scope

- `agentconfig.Service`：记录每个客户端最近一次自动刷新的稳定失败码（内存），
  `Inspect` 在 `ErrorCode` 为空时用它补上；`Apply`／`Restore` 成功后清除该客户端的记录。
- 前端：`config_changed` 时主按钮可用并改文案「重新读取并应用」；客户端接入页汇总提示
  「N 个客户端需要重新应用」；错误文案写清后果与动作。
- 测试：Go 覆盖「刷新失败码进入 Item」「重新应用后清除」；前端覆盖按钮文案与可点击状态。
- 文档：`docs/product/LOCAL_AGENTS.md`。

## Out of Scope

- 不改变「外部改动不自动覆盖」的安全策略；不做批量静默覆盖。
- 不改 DSH 配置内容本身（由用户点「应用配置」写入）。
- 不动 WorkBuddy 通道逻辑。

---

# Implementation Result

## Changes

- `internal/application/agentconfig`：
  - `RefreshCatalog` 逐客户端记录稳定失败码（`config_changed`／`write_failed`／
    `config_locked`／`invalid_config`／`models_unavailable`／`backup_unavailable`…）到内存，
    每次刷新整体替换（成功的客户端自动消失）。
  - 新增 `failureCode`（从 `agent: <code>` 文本与 `errors.Join` 聚合错误里取第一个可识别码）、
    `setIssues`／`issue`／`clearIssue`；`Service` 增加 `issueMu`／`issues`。
  - `Inspect` 在没有更具体的 `config_changed` 时用刷新失败码补 `ErrorCode`；
    `Apply`、`Restore` 成功后清除该客户端记录。
- 前端：
  - `agent-client-settings.tsx`：`config_changed` 时主按钮可用，文案变「重新读取并应用」，
    点击直接走「重新读取→开预览」路径（编辑器带 `reread`，应用时带 `acceptCurrentConfig`）；
    其它错误码仍禁用配置与恢复。
  - `local-agents.tsx`：列表页汇总提示「N 个客户端的配置文件已被其他工具修改，
    模型目录暂停自动更新」并列出客户端。
  - i18n：新增 `agents.rereadAndApply`、`agents.reapplyNotice`，改写
    `agents.error.config_changed`（点出「重新读取配置」或「重新读取并应用」两种入口与不会丢改动）。
- 文档：`docs/product/LOCAL_AGENTS.md` 增加「外部改动→停更→可见原因→一键恢复」的说明。

## Files Changed

- `internal/application/agentconfig/{refresh.go,service.go,apply.go,restore.go,types.go,catalog_failures_test.go}`
- `frontend/src/features/gateway/{agent-client-settings.tsx,local-agents.tsx}`
- `frontend/src/features/gateway/{agent-detail-workspace.test.tsx,local-agents.test.tsx}`
- `frontend/src/i18n/locales/{agents-zh.ts,agents-en.ts}`
- `docs/product/LOCAL_AGENTS.md`

## Technical Notes

- 没有新增前端字段：`Item.ErrorCode` 原本已经由「文件哈希 != applied_hash」给出
  `config_changed`，这次把**其它刷新失败码**也补进同一个字段，避免改 bindings 与类型。
- 刷新失败码只在内存里保存（进程内），重启后由下次刷新重新计算；不落库，避免把诊断数据
  写进用户数据库。
- 「重新读取并应用」不是静默覆盖：仍走预览，且应用时带 `acceptCurrentConfig` 才允许覆盖
  已变化的文件——与既有安全边界一致。

---

# Verification

## 质量门（`task verify`，2026-11-17 19:37–19:59 Asia/Singapore）

```text
task verify → exit 0
```

| 阶段 | 结果 |
| --- | --- |
| format | gofmt 无输出；prettier 通过 |
| lint | `go vet`／`staticcheck` 无新增告警；`eslint` 0 error |
| typecheck | `tsc --noEmit` exit 0 |
| test | Go 全包通过；前端 158 files / 969 tests |
| build | 绑定 + vite + Windows production 构建通过 |

首次验证失败于三条既有前端断言：它们要求 `agents.error.config_changed` 文案包含
「重新读取配置」。文案本身是对的（编辑器里的按钮仍叫「重新读取配置」），
于是把文案改成同时点出「重新读取配置」与「重新读取并应用」两个入口，复跑通过。

## 新增测试

| 行为 | 测试 |
| --- | --- |
| 刷新失败码进入 `Item.ErrorCode`，`config_changed` 优先，重新应用后清除 | `agentconfig/catalog_failures_test.go` |
| `failureCode` 解析单个／聚合／非法码 | 同上 |
| `config_changed` 时主按钮为「重新读取并应用」、恢复按钮禁用、应用带 `acceptCurrentConfig` | `agent-detail-workspace.test.tsx` |
| 列表页汇总提示只列出受影响的客户端 | `local-agents.test.tsx` |

## 未验证

- 未在本机真实客户端（Codex／Claude／Harness）上执行一次「重新读取并应用」；单测用夹具覆盖。
- 本机仍有 6 个客户端处于 `config_changed`，需要用户点一次才能恢复自动刷新。

---

# Documentation

- [x] `docs/product/LOCAL_AGENTS.md`
- [x] `docs/plans/CURRENT.md`、`docs/releases/v2.3.1-beta.22.md`（随构建任务）

---

# Deviations

- 规划里写「客户端卡片显示失败原因」，实现时复用了既有 `ErrorCode` 字段而不是新增
  `RefreshError`：新增字段要重新生成 bindings，且 `config_changed` 本来就在该字段里，
  合并后语义统一（「这个客户端的配置需要处理」）。
- 文案调整导致三条既有断言失败，选择改文案而不是改测试：文案要同时覆盖详情页
  （按钮叫「重新读取配置」）与卡片（按钮叫「重新读取并应用」）两个入口。

---

# Git

## Commit

e2e53b7d fix(agentconfig): 让受管客户端目录停更可见并能一键恢复

---

# Result

受管客户端目录停更不再只在日志里：后台刷新失败会按客户端记录稳定码并在卡片／列表页显示；
`config_changed` 时主按钮改为「重新读取并应用」，一次点击读回外部改动、预览并带
`acceptCurrentConfig` 应用，成功后恢复自动刷新。`task verify` exit 0（前端 158 文件／969 项测试）。

---

# Completed

2026-11-17 20:02（Asia/Singapore）
