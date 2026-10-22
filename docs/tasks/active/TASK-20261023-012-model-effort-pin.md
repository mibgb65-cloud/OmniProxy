# TASK-20261023-012: 应用内按模型固定思考档位（网关覆盖 Messages 请求）

## Status

In Progress

## Created

2026-10-23（Asia/Singapore）

## Request

用户确认采用「在 OmniProxy 自己这边做档位入口」的方案：选模型时就能选到最高档（相当于选带 `max` 后缀的那个），不要求客户端菜单出现滑杆。

## Goal

在 Claude Code／Claude Desktop 的模型配置卡片上提供按模型的思考强度选择；选定后由网关在该模型的 Messages 请求上强制使用该档位，且只允许上游已确认的取值。

## Out of Scope

- 不改客户端 effort 控件的显示逻辑，也不把第三方模型改名为 Claude 官方 ID。
- 不伪造未确认档位；不改变 Chat／Responses 协议的档位字段语义。
- 不新增 Wails 方法：档位随既有模型配置预览／应用流程提交。

---

# Analysis

## Root Cause

TASK-20261023-011 已确证：Claude Desktop 只为客户端认识的模型 ID 渲染 effort 控件，`claude-omniproxy-*` 路由名不会有入口，配置侧无法注入档位能力。因此调整入口必须落在应用侧，并由网关保证实际发送值。

## Design

- 新增 `model_effort_pins(model_id, effort, updated_at)`（迁移 041），按模型保存一个固定档位。
- `modeleffort` 包只负责读写；能力校验在写入前用目录元数据完成：取值必须属于该模型的 `ReasoningEfforts` 枚举。
- 档位随 `Input.Efforts` 进入既有预览／应用流程：`Plan.Efforts` 回显当前生效值，应用时在同一事务里提交固定档位。
- 网关在 `RouteRequest` 的 Messages 路径末尾读取固定档位并覆盖 `output_config.effort`，同时保证 `thinking` 与档位一致；读取失败只记录日志并沿用客户端选择。

## Related Modules

- `internal/application/modeleffort`（新增）
- `internal/application/routing/effort.go`（读接口）
- `internal/application/agentconfig/effort.go`、`types.go`、`plan.go`、`apply.go`
- `internal/application/gateway/effort_pin.go`、`routing.go`
- `frontend/src/features/gateway/agent-model-card.tsx`、`agent-catalog-preview.tsx`、`agent-config-editor.tsx`、`use-agent-config-form.ts`

---

# Implementation Result

## Changes

- 迁移 041 新增 `model_effort_pins`。
- `modeleffort.Pins/Save`：只读写，不推断能力。
- `routing.EffortPins`：网关读取固定档位的唯一入口。
- `agentconfig`：`Input.Efforts`、`Plan.Efforts`、`validateEfforts`、`effortPlan`；应用时在 `commitFileVersions` 的同一事务里写固定档位。
- 网关：`RouteRequest` 在 Messages 请求上按固定档位覆盖 effort（同时按 `none` 切换 `thinking`）；Chat／Responses 不受影响。
- 前端：模型卡片新增「思考强度」选择（只在模型有已确认档位时出现），随预览／应用提交；中英文文案与提示。

## Technical Notes

`Plan.Efforts` 只回显本次所选模型；显式选择优先于已保存值，空值表示清除。固定档位与配置文件版本同事务提交，避免「配置回滚了但档位已改」。

---

# Verification

## Tests

- [x] Unit Tests
- [x] Integration Tests（网关 Messages 档位覆盖）
- [ ] E2E Tests（需真实客户端会话验收）

## Verification Result

```text
go test ./internal/application/... ./internal/infrastructure/database/... ./internal/transport/... -count=1
  → 全部通过（agentconfig、gateway、routing、database、transport/wails 等）

go test（定向新增）
  → PASS TestClaudeEffortSelectionPersistsAndValidates（写入／回显／清除／未确认档位拒绝）
  → PASS TestEffortSelectionRejectsUnknownAndUnsupportedModels
  → PASS TestPinnedEffortOverridesClientSelection（无档位保留客户端值；固定 max 覆盖；thinking=adaptive；chat 不改写）
  → PASS TestPinnedEffortIgnoresUnavailableConfiguration

vitest（新增）
  → PASS 已确认档位的模型才有强度选择，并显示已保存的档位
  → PASS 选择档位后随应用一起提交
  → PASS 改回已保存值时不提交额外字段
  → 既有 gateway 前端测试 28 文件 / 190 项通过

task --force verify → exit 0
  format（gofmt／prettier）通过
  lint／typecheck 通过
  Go 全部包通过（新增 modeleffort 包无独立测试文件，行为由 agentconfig／gateway 用例覆盖）
  前端 124 文件 / 734 项测试通过
  Windows production 构建通过（bin/omniproxy.exe）
日志：tmp/verify-20261023-012.log
```

- 说明：门禁首轮失败于 `prettier --check`（两个前端文件），`prettier --write` 后复跑 exit 0；未跳过任何检查。
- 未执行：真实 Claude 会话验收（在卡片上选 max 后，上游是否确实按最高档推理）。

---

# Git

未提交：工作区同时存在多个在办任务的改动（`internal/core/agentprotocol`、`docs/product/RIVER.md` 等），按 Git 安全规则不混提交。本任务改动：
`migrations/041_model_effort_pins.sql`、`internal/application/modeleffort/`、`internal/application/routing/effort.go`、
`internal/application/agentconfig/{effort.go,types.go,plan.go,apply.go}`、`internal/application/gateway/{effort_pin.go,routing.go}`、
`frontend/src/features/gateway/{agent-model-card.tsx,agent-catalog-preview.tsx,agent-config-editor.tsx,use-agent-config-form.ts}`、
`frontend/src/i18n/locales/agents-{zh,en}.ts`、`frontend/bindings/omniproxy/internal/application/agentconfig/models.ts`、本记录与文档。

---

# Remaining Risks
- 客户端界面里的档位显示仍由客户端自己判断，可能与网关实际发送值不同；这是客户端限制，不是数据不一致。
- 固定档位只作用于 Messages 协议；Codex 等 Responses／Chat 客户端不参与。
- 未用真实 Claude 会话验收「选 max 后上游确实按最高档推理」。
