# TASK-20261120-006: 用量统计按模型家族合并推理档位变体

## Status

Completed

## Created

2026-11-20 10:25

## Request

用户给出用量页「按提供商与模型」表的两行截图（`gemini-4-neon-xhigh` 与 `gemini-4-neon-high`），要求：

> 我觉得这个 Gemini 的模型记录需要优化一下：你看这个，他们应该是合并记录用量，它只是思考程度不同，并不是不一样的模型，明白吧，这种仅仅思考程度不同的，不应该分开。

## Goal

用量统计里「仅推理档位（思考程度）不同的同一模型」合并成一行累计，不再按上游变体 ID 拆成多行；
请求明细与记录页仍保留每次请求实际使用的型号 ID（那是单次请求的事实）。

## Background

- 上游用独立模型 ID 表达推理档位（Google 订阅：`gemini-4-neon-high`／`-low`／`-medium`／`-xhigh`，
  `gemini-4-hydrogen-…`、`gemini-4-argon-…`、`gemini-3.8-flash-…` 同理），OmniProxy 的目录已按
  ADR-022 把它们识别为同一家族的不同档位变体，网关也会在请求时解析到具体变体。
- 但统计侧按「实际模型 ID」分组：`usage_provider_models(provider_id, model)` 与
  `usage_buckets.model` 都保存变体 ID，因此同一个模型在用量页被拆成多行。
- 本机生产库只读核查（`D:\OmniPProxy-data`）：`builtin_google` 的模型行包含
  `gemini-4-neon-high`（612 请求）、`gemini-4-neon-xhigh`（672）、`gemini-4-hydrogen-{high,medium,xhigh}`、
  `gemini-4-argon-{high,low,medium}`、`gemini-3.8-flash-{high,low,medium}`；
  目录表 `account_models` 同时保存了这些家族的完整变体清单（`gemini-4-neon-{high,low,medium,xhigh}` 等）。
- 相关展示位置：用量页「按提供商与模型」表、用量页模型占比卡、用量页模型每日趋势（前六 + 其他）、
  仪表盘「模型 Token 占比」（前六 + 其他）。

## Scope

- `internal/application/activity`：新增模型家族判定与合并工具，并在四处模型维度输出上应用：
  `ProviderModels`（表）、`Models`（占比卡）、`ModelDays`（每日趋势）、仪表盘 `Models`（占比卡）。
- 家族判定使用证据规则，不按名字猜测：同一基础名出现 ≥2 个档位后缀，或基础名本身也是已知模型。
- 数据来源：已记录的模型（`usage_provider_models`）与已同步的目录模型（`account_models`）取并集。
- 合并行显示基础模型名；「按提供商与模型」行额外返回被合并的变体列表，前端在悬浮提示里展示。
- 前端文案说明档位变体已合并；更新 `docs/product/ACTIVITY.md`、`docs/product/PRD.md` 与 ADR-030。

## Out of Scope

- 不修改数据库结构、不新增迁移：分组在读取时完成，历史数据原样保留。
- 不改变请求明细、记录页、日志与导出中的实际模型 ID。
- 不改变路由、目录、额度与计费口径（Token 与请求数仍按实际发生的请求累计，只是展示时归并）。
- 不改动 `usage_buckets`／`usage_provider_models` 的写入路径。

## Existing Behavior

同一个 Gemini 模型的 `-high`／`-xhigh` 等档位变体在用量统计中各自一行，请求数与 Token 分开累计，
「模型占比」「模型每日趋势」「仪表盘模型占比」也把变体当作不同模型。

## Expected Behavior

- 用量页「按提供商与模型」中，`gemini-4-neon-high` 与 `gemini-4-neon-xhigh` 合并为一行 `gemini-4-neon`，
  请求数与输入／输出 Token 为两者之和；悬浮提示列出被合并的具体变体。
- 用量页模型占比、模型每日趋势、仪表盘模型占比同样按家族归并；前六名与「其他」都按归并后的家族计算。
- 证据不足的型号（只有一个档位后缀且没有基础模型）保持原样，不猜测合并；
  `-thinking`、`-tiered`、`-lite`、`-free` 等非档位后缀不合并；`-tiered-high` 与 `-tiered-xhigh`
  归并为各自的 `-tiered` 家族。
- 请求明细、记录页与日志仍显示实际使用的变体 ID，导出不变。

---

# Analysis

## Related Modules

- `internal/application/activity/provider_models.go`、`usage.go`、`dashboard.go`、`types.go`
- `internal/core/modelcatalog/effort_variants.go`（复用 `EffortVariantBaseName` 档位后缀词汇）
- `frontend/src/features/activity/{provider-model-usage,usage-share,usage-model-trend}.tsx`、`frontend/src/features/dashboard/dashboard-model-share.tsx`
- 文案 `frontend/src/i18n/locales/{activity,dashboard}-{zh,en}.ts`

## Root Cause

统计以「实际模型 ID」为分组键。上游把推理档位编码进模型 ID，目录层已经知道这些 ID 属于同一家族，
但统计层没有使用这层信息，于是同一个模型被拆成多条。

## Risks

- 误合并会把两个真实模型的用量加在一起（例如名字里恰好带 `-high` 的独立型号）。
  处理：只接受有证据的家族（≥2 个档位后缀或基础名本身存在），并复用目录已有的档位词汇。
- 前六名与「其他」必须在归并之后计算，否则热门家族的变体会挤掉其它模型；实现上先归并再取前六。
- 历史模型名可能已不在当前目录中；已记录模型也参与证据计算，不依赖目录同步状态。

---

# Implementation Plan

1. `internal/application/activity/model_family.go`：家族映射（纯函数）+ 从数据库汇总模型名。
2. 四处模型维度输出改为先归并再排序／取前六；`ProviderModel` 增加 `variants`。
3. 前端类型、悬浮提示与四处文案说明。
4. Go 与前端回归测试；`task verify`；更新文档与 ADR-030；提交并构建新的本地 Beta。

---

# Progress

- 2026-11-20 10:25 创建记录，完成生产库只读核查与展示位置梳理。

---

# Implementation Result

## Changes

实际完成：

- 新增 `internal/application/activity/model_family.go`：按「已记录模型 + 已同步目录模型」计算家族映射
  （`gemini-4-neon-high` → `gemini-4-neon`），只合并有证据的家族；目录表缺失时按已记录模型判断，
  不让整页用量读取失败。
- `providerModels`：改为读取后按 (提供商, 家族) 归并再排序；`ProviderModel` 增加 `variants`，
  与显示名不同的原始型号（含单个档位变体）会在悬浮提示里列出。
- `Usage.Models`：新增 `modelsBreakdown`，归并后再排序并截断 1000 条。
- `Usage.ModelDays`：读取近 30 天全部 (日, 模型) 汇总，先归并再取前六家族，最后按日输出。
- 仪表盘 `readDashboardModels`：同一家族映射归并后再交给前端取前六／其他。
- 前端：模型名悬浮提示显示被合并的原始型号；「按提供商与模型」「模型用量占比」「模型每日用量」
  「仪表盘模型占比」四处提示文案说明档位变体已合并；Wails 绑定重新生成（`ProviderModel.variants`）。
- 文档：ADR-030、`docs/product/ACTIVITY.md`、`docs/product/DASHBOARD.md`、`docs/product/PRD.md`、
  ADR 索引。

## Files Changed

- `internal/application/activity/model_family.go`（新增）、`provider_models.go`、`usage.go`、`dashboard.go`
- `internal/application/activity/model_family_test.go`（新增）、`provider_models_test.go`
- `frontend/src/features/activity/provider-model-usage.tsx` 与其测试
- `frontend/src/i18n/locales/{activity,dashboard}-{zh,en}.ts`
- `frontend/bindings/omniproxy/internal/application/activity/models.ts`（生成）
- `docs/decisions/ADR-030-merge-reasoning-level-variants-in-usage.md`（新增）、`docs/decisions/README.md`
- `docs/product/{ACTIVITY,DASHBOARD,PRD}.md`

## Technical Notes

- 档位后缀判定复用 `internal/core/modelcatalog.EffortVariantBaseName`，与目录／网关的档位词汇一致，
  避免统计层另立一套后缀表。
- 归并发生在读取时：`usage_provider_models` 与 `usage_buckets` 的写入路径、历史数据、请求明细都不变。
- 前六名与「其他」在归并之后计算，否则同一家族的多个变体会占满名额。
- 本机生产库只读核对（`D:\OmniPProxy-data`）合并效果：`builtin_google` 的模型行由 11 行降到 8 行，
  `gemini-4-neon-high`（672 请求）与 `-xhigh`（612）合并为 `gemini-4-neon`（1,284 请求），
  `gemini-4-hydrogen-{high,medium,xhigh}` 合并为 `gemini-4-hydrogen`。

---

# Verification

## Tests

- [x] Unit Tests：家族判定（含单档位不合并、基础模型确认、tiered 家族、非档位后缀）、
  提供商表归并、模型占比归并、每日趋势前六在先归并后选取、仪表盘占比归并
- [x] 回归：`internal/application/gateway` 的历史 schema 记账测试（目录表缺失时用量仍可读）
- [x] Frontend：合并行只显示基础模型名、模型数按合并计算
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
EXIT=0
```

Go 全部包通过（含历史 schema 的网关记账测试）；前端 164 文件／1006 项测试通过；Windows production 构建完成。

---

# Documentation

- [x] PRD（模型维度统计按家族归并）
- [x] ACTIVITY（统计规则与模型维度说明）
- [x] DASHBOARD（仪表盘模型占比口径）
- [x] ADR-030 与 ADR 索引
- [x] CURRENT（构建与任务看板）

---

# Deviations

- 原计划只合并「按提供商与模型」表；实现时同步覆盖模型占比、模型每日用量与仪表盘模型占比，
  否则同一页面两套口径、且前六名仍会被变体挤占。
- 目录（`account_models`）只作为补充证据：历史 schema 的网关记账测试没有目录表，
  用量读取必须继续可用；已记录模型仍可独立确认家族。
- 只有单个档位被使用、但基础名已存在时，合并行也会在悬浮提示里列出原始型号（哪怕只有一个），
  避免「行名变了却查不到原因」。

---

# Git

## Commit

```text
待回填
```

---

# Result

用量统计的模型维度按模型家族归并推理档位变体：同一模型的 `-high`／`-low`／`-medium`／`-xhigh`
在「按提供商与模型」、模型占比、模型每日用量与仪表盘模型占比中都算同一个模型，请求数与 Token 相加；
请求明细与导出仍保留每次实际使用的型号 ID。

---

# Completed

待回填

