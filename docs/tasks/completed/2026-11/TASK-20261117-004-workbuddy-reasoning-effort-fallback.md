# TASK-20261117-004: 补齐 WorkBuddy AI 目录里未声明档位列表的推理档位

## Status

Completed

## Created

2026-11-17 09:40（Asia/Singapore）

## Request

用户问「他们的模型都没有推理等级的吗？」——需要确认 WorkBuddy AI 的模型是否有推理档位，
以及 OmniProxy 的目录里为什么一部分模型没有档位。

## Goal

让目录里**所有声明支持推理**的模型都带上可选的推理档位：上游给了
`reasoning.supportedEfforts` 就照抄；只给了 `reasoning.effort`（`onlyReasoning: true`）的模型
按社区插件实测的规则补 `low／high／max` 加上它自己的档位，并把它的 `effort` 作为默认档位。

## Background

真实 `/v3/config`（Free 账号，2026-11-17 实测，25 个模型）分三类：

| 类型 | `supportsReasoning` | `reasoning` | 例子 |
| --- | --- | --- | --- |
| 声明档位列表 | `true` | `supportedEfforts: [...]` | `glm-5.3`、`gpt-6.1-sol`、`grok-4.7`、`gemini-3.8-flash`、`minimax-m3.1-flash` |
| 只有 `effort` | `true` + `onlyReasoning: true` | `{"effort":"high","summary":"auto"}` | `max-model`(high)、`gemini-4-carbon`(medium)、`gemini-4-argon`(medium)、`kimi-k3`(medium) |
| 无推理 | `nil` | 无 | `default-model`、`hy-image-3.5`、`kling-video-v3` |

`discovery.go` 的 `normalizeModel` 只用 `supportedEfforts`，所以第二类 4 个模型被存成
「无档位」，`gemini-4-*`／`kimi-k3` 连 `effort=medium` 这个默认档位也丢掉了。

社区插件 0.2.36 的 `effortsOf()` 对这类模型的处理是：`supportedEfforts` 为空但
`supportsReasoning === true` 时给出 `["low","high","max"]` 并并入它自己的 `effort`，
按 `EFFORT_ORDER = low, medium, high, xhigh, max` 排序；`supportsReasoning !== true` 时返回空。
插件在真实账号上按这个规则请求过这些模型，因此这是上游实际接受的档位集合。

## Scope

- `internal/provider/workbuddyai/discovery.go`：`normalizeModel` 增加上述回退；
  档位统一按 `low/medium/high/xhigh/max` 排序（未知档位保持上游顺序排在后面）。
- `internal/provider/workbuddyai/discovery_test.go`：新增「只有 effort」与
  「supportsReasoning=false」两类断言，覆盖回退与不误补。
- `docs/product/WORKBUDDY.md`：模型目录一节说明档位来源与回退规则。

## Out of Scope

- 不把 `none` 当档位暴露（上游 `canDisableThinking` 不可靠，插件也只把客户端的 `none`
  映射成 `low`，不对外列出）。
- 不改网关侧的档位透传、计费与协议转换。
- 不改前端档位展示（已按目录值直接渲染）。

---

# Implementation Result

## Changes

- `internal/provider/workbuddyai/discovery.go`：
  - 新增 `defaultEfforts = ["low","high","max"]`（上游只给 `effort` 时补的通用档位）与
    `effortOrder = ["low","medium","high","xhigh","max"]`（补档位时的排序）。
  - `normalizeModel` 把档位收集拆成 `cleanEfforts`（跳过 `none`、去重、校验拼写、上限 16）；
    当上游未给任何档位、但 `supportsReasoning: true` 时，补 `low/high/max` 加它自己的
    `reasoning.effort`，排序后再选默认档位（`defaultEffort` → `effort`）。
  - `supportsReasoning: false`、无 `reasoning`、或根本没声明推理的模型仍不发布档位。
- `internal/provider/workbuddyai/discovery_test.go`：目录夹具新增 `effort-only`（`effort=high`）
  与 `effort-only-medium`（`effort=medium`）两条，断言补成 `low/high/max`（默认 `high`）与
  `low/medium/high/max`（默认 `medium`）；原有 `kimi-k3`（无 reasoning）仍无档位。
- `docs/product/WORKBUDDY.md`：模型目录一节写清两种上游形态、补全规则与不发布档位的模型；
  协议边界表的「推理档位」行、真实验证进度的 `/v3/config` 一条同步更新。

## Files Changed

- `internal/provider/workbuddyai/discovery.go`、`discovery_test.go`
- `docs/product/WORKBUDDY.md`

## Technical Notes

- 真实 `/v3/config` 实测（2026-11-17，Free 账号，25 个模型）：21 个有 `supportedEfforts`；
  `max-model`（`effort=high`）、`gemini-4-carbon`／`gemini-4-argon`／`kimi-k3`（`effort=medium`）
  是 `supportsReasoning: true` + `onlyReasoning: true` 且只有 `effort`；
  `default-model`、`hy-image-3.5`、`kling-video-v3` 没有 `reasoning`。
- 补全口径来自社区插件 0.2.36 的 `effortsOf()`：`supportedEfforts` 为空但
  `supportsReasoning === true` 时返回 `["low","high","max"]` 并入自身档位，按
  `EFFORT_ORDER` 排序；`supportsReasoning !== true` 返回空。
- `none` 不作为档位发布：上游 `canDisableThinking` 与实际行为不一致，插件也只把客户端的
  `none` 映射成 `low`，不对外列出。
- 已存目录不会自动变化：需要在「路由 → 模型目录」手动同步（或等下一次自动同步）后，
  网关与模型档位选择器才会看到补齐后的档位。

---

# Verification

## 质量门（`task verify`，2026-11-17 09:25–09:40 Asia/Singapore）

```text
task verify → exit 0
```

| 阶段 | 结果 |
| --- | --- |
| format | gofmt 无输出；prettier 通过 |
| lint | `go vet`／`staticcheck` 无告警；`eslint` 0 error |
| typecheck | `tsc --noEmit` exit 0 |
| test | Go 全包通过；前端 157 files / 965 tests |
| build | 绑定 + vite + Windows production 构建通过 |

首次 `task verify`（09:25 起）退出码为 1，但捕获到的尾部只有已完成的构建日志，Go 与前端测试
都已打印通过，未能定位到具体步骤；随后单独 `task build` 与完整 `task verify` 复跑均 exit 0，
判断为环境抖动（历史上有同类端口占用抖动测试），未改代码。

## 真实目录对照

`/v3/config` 原始字段（真实令牌，诊断用临时测试打印后立即删除）与补全后的预期：

| 模型 | 上游 `reasoning` | 目录档位 | 默认档位 |
| --- | --- | --- | --- |
| `glm-5.3`／`glm-5.3-flash`／`kimi-k3.1-pro` | `supportedEfforts: low/high/max` | 原样 | `high` |
| `gpt-6.1-sol`／`gpt-6-sol`／`gpt-6-luna`／`minimax-m3.1-flash`／`gpt-6-astra` | `supportedEfforts: low/medium/high/xhigh/max` | 原样 | `high`／`max` |
| `grok-4.7`、`gemini-3.8-flash`、`deepseek-v4.1-flash-sg` | `supportedEfforts: low/medium/high(、xhigh)` | 原样 | `high`／`medium` |
| `hy4std`／`hy4flash`／`hy4flash-free` | `supportedEfforts: ["high"]` | 原样（单档） | `high` |
| `max-model` | `{"effort":"high","summary":"auto"}` | `low/high/max`（新增） | `high`（新增） |
| `gemini-4-carbon`／`gemini-4-argon`／`kimi-k3` | `{"effort":"medium","summary":"auto"}` | `low/medium/high/max`（新增） | `medium`（新增） |
| `default-model`／`hy-image-3.5`／`kling-video-v3` | 无 | 无 | 无 |

## 未验证

- 补齐的档位没有逐个模型发真实请求验证（沿用社区插件在真实账号上的实测口径）；
  发一次带 `reasoning_effort` 的对话即可确认，但会消耗调用额度。
- 网关把客户端档位透传到上游的链路沿用既有实现（TASK-20261117-001/002 已跑通真实对话）。

---

# Documentation

- [x] `docs/product/WORKBUDDY.md`（模型目录、协议边界、真实验证进度）
- [x] `docs/plans/CURRENT.md`
- [x] `docs/releases/v2.3.1-beta.20.md`（随构建任务）

---

# Deviations

- 计划只写「按插件口径补 low/high/max」，实际把档位收集抽成 `cleanEfforts` 并新增
  `sortEfforts`：补进来的档位顺序依赖排序（`medium` 需要落在 `low` 与 `high` 之间），
  而原有「上游给什么顺序就照抄」的行为保持不变（只对补全集合排序）。
- 没有把 `none` 加进档位（见 Out of Scope）；因此声明了 `canDisableThinking: true` 的
  `gpt-5.6-*`／`gpt-6-astra` 在目录里也不提供「关闭思考」这一档。

---

# Git

## Commit

d0cde578 fix(provider): 补齐 WorkBuddy 只给 effort 的模型推理档位

---

# Result

WorkBuddy AI 的模型**有**推理档位：真实 `/v3/config` 的 25 个模型里 21 个直接声明
`supportedEfforts`（如 `glm-5.3` 为 `low/high/max`、`gpt-6.1-sol` 为
`low/medium/high/xhigh/max`、`hy4std` 只有 `high`），另 4 个（`max-model`、`gemini-4-carbon`、
`gemini-4-argon`、`kimi-k3`）只给 `reasoning.effort`，`default-model`／`hy-image-3.5`／
`kling-video-v3` 完全没有推理声明。

之前目录里那 4 个模型没有档位，是因为 `normalizeModel` 只读 `supportedEfforts`；现在按社区
插件在真实账号上的实测口径补 `low/high/max` 加自身档位（`max-model` → 默认 `high`；
`gemini-4-*`、`kimi-k3` → `low/medium/high/max`，默认 `medium`）。`task verify` exit 0
（Go 全包、前端 157 文件／965 项、Windows production 构建）。随 `v2.3.1-beta.20` 构建交付；
升级后需在「路由 → 模型目录」同步一次才会写入新档位。

---

# Completed

2026-11-17 09:56（Asia/Singapore）

