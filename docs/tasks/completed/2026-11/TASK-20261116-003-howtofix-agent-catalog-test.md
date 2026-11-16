# TASK-20261116-003: HowToFix 模型进入受管客户端目录的回归测试

## Status

Completed

## Created

2026-11-16 08:45（Asia/Singapore）

## Request

用户询问 HowToFix 现在能否正常接入 Agent 工具（Claude Code／Codex／DSH 等），
需要可核对的结论而不是口头保证。

## Goal

用一条回归测试固定「HowToFix 账号同步到的模型会出现在受管客户端目录」这一行为，
覆盖 Chat 客户端、Claude 与 Codex 三类投影路径。

## Background

客户端目录从「已配置路由」投影生成（`agentconfig.clientEntries` → `catalog.ForProtocol`），
不按服务商类型过滤；HowToFix 同步出的模型默认携带
`chat_completions`／`responses`／`messages` 三种协议，因此无需为任何客户端单独适配。

## Scope

- 新增 `internal/application/agentconfig/howtofix_catalog_test.go`：
  断言 `glm-5.3-flash` 出现在 `claude`、`codex`、`harness-web`、`harness-desktop`、
  `opencode`、`qwen-code` 的受管目录中，并出现在 Codex 目录 JSON 的 `slug` 列表里。
- 无产品代码改动。

## Out of Scope

- 不改变任何客户端配置写法、不新增能力声明（能力未知仍保持未知）。
- 不改网关、不改路由、不改目录同步。

---

# Implementation Result

## Changes

- 新增回归测试 `TestHowToFixDiscoveredModelReachesManagedClients`。
- 无产品代码改动。

## Files Changed

- `internal/application/agentconfig/howtofix_catalog_test.go`（新）

## Technical Notes

- 测试同时验证 Codex 目录文件（`CodexJSON`）与各客户端 `clientEntries` 投影，
  覆盖「messages／responses／chat_completions」三条协议分支。
- 能力未知只影响声明：Claude 的 `behavesAs` 与 effort 设置、Codex 的图片输入仍不发布，
  但模型本身可以出现在目录里被选中。

---

# Verification

## Tests

- [x] `go test ./internal/application/agentconfig/... -run TestHowToFixDiscoveredModelReachesManagedClients` 通过
- [x] 全量 Go 测试与前端测试随 `task verify` 通过
- [ ] E2E：真实桌面客户端菜单未验证（需用户在本机重启客户端后确认）

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

---

# Documentation

No documentation changes required.（结论已写入 `docs/product/HOWTOFIX.md` 的客户端说明。）

---

# Git

Not committed by Agent.

---

# Result

已用回归测试固定 HowToFix 模型进入受管客户端目录的行为：Claude Code／Codex／Harness／
OpenCode／Qwen Code 的目录投影都包含同步到的 HowToFix 模型；能力未知模型只影响
effort 与图片声明，不影响可选中与可调用。

---

# Completed

2026-11-16 08:47（Asia/Singapore）
