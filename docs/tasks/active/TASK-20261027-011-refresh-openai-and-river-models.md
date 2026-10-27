# TASK-20261027-011: 更新 OpenAI 预设与 River 新增模型

## Status

Completed

## Created

2026-10-27 12:32

## Request

用户提供两张截图：

1. 提供商页 OpenAI 卡片的「预设模型」浮层仍显示 `GPT-5.4`、`GPT-5.4 Mini`、`GPT-4.1`、`o3`，
   用户要求「要和最新的一致」。
2. River 官方控制台 `console.river.ai/inference` 的模型页出现新模型
   （`Qwen/Qwen3.8-27B-FP8`、`moonshotai/Kimi-K3`），要求更新接入。

用户随后要求：继续完成任务，完成后自己提交代码并在任务记录中记录任务情况。

## Goal

- 提供商页 OpenAI 预设模型与 OpenAI 官方「Featured models」一致，不再显示已过时型号。
- River 已确认可计费的新模型进入预设、路由、客户端目录和推理档位映射，未确认的模型仍不猜测。

## Background

预设目录 `internal/application/provider/catalog.go` 是提供商卡片上「预设模型」的唯一来源，
只用于展示（`docs/tasks/completed/2026-09/TASK-20260928-003-provider-catalog.md`：模型列表是预览，
不代表账号实时可用列表）。River 的预设、逻辑 ID、路由白名单和能力元数据集中在
`internal/provider/river/{catalog,identity}.go`。

## Scope

- `internal/application/provider/catalog.go`：OpenAI 预设模型改为官方当前推荐的四款。
- `internal/provider/river/catalog.go`：新增 `Qwen/Qwen3.8-27B-FP8`、`moonshotai/Kimi-K3` 的白名单、
  上下文／输出上限、推理档位与输入模态。
- `internal/provider/river/identity.go`：新增两款的公共逻辑 ID 与短显示名。
- 受影响测试：River 目录／路由身份测试（原先假设 River 所有模型都是 1M，需要按新窗口修正），
  前端提供商页 mock 中的过时型号。
- 文档：`docs/product/RIVER.md`、`docs/product/ROUTING.md`、`docs/product/GATEWAY.md`、
  `docs/product/PRD.md`、`docs/architecture/OVERVIEW.md`、`docs/plans/CURRENT.md`。

## Out of Scope

- OpenAI 其余非旗舰型号（`gpt-5.4-nano`、`gpt-5.5-pro`、图像／音频／Realtime／嵌入型号）不进入预设：
  预设只列当前主推的对话模型。
- River 的美元参考价（`internal/core/tokenprice`）不新增 River 型号：River 与直连服务商同名模型
  共用逻辑 ID 但单价不同，现有表按模型 ID 计价，无法表达按服务商分流，需要单独设计。
- 官方文档另列出「GLM-5.3（纯文本）」型号，但公开计费接口 `console.river.ai/api/public/pricing`
  的 `inference` 列表里没有它，模型 ID 与计费口径均未确认，本任务不纳入。
- 已发布版本的 Release Notes、历史任务记录不回溯修改。

## Existing Behavior

- OpenAI 预设：`GPT-5.4`、`GPT-5.4 Mini`、`GPT-4.1`、`o3`。
- River 预设四款：`deepseek-ai/DeepSeek-V4.1-Flash`、`deepseek-ai/DeepSeek-V4.1-Flash-1017`、
  `nvidia/NVIDIA-Nemotron-3.5-Super-VL-120B-A12B-NVFP4`、`zai-org/GLM-5.3-Flash`，
  统一 1048576 上下文、文本＋图片、函数工具。
- River 路由测试断言「River 所有模型都是 1048576 上下文」，并且所有型号都有 `[1m]` 变体。

## Expected Behavior

- OpenAI 卡片显示 `GPT-6 Astra · GPT-6.1 Sol +2`，展开为官方 Featured models 四款。
- River 预设六款，其中新增两款：`Qwen3.8-27B-FP8`（262144 上下文／262143 输出／
  none·low·medium·xhigh）、`Kimi-K3`（1048576 上下文／32768 输出／none·max，默认 max）；
  实际请求仍发送完整官方 ID。
- 已有四款 River 模型的能力、档位和显示名不变；Qwen 因窗口不足 1M 不生成 `[1m]` 客户端选项。
- 未同步过的 River 账号立即看到六款预设；已同步过的账号仍以最近成功的官方目录为准。

---

# Analysis

## Related Modules

- `internal/application/provider`（内置提供商目录，卡片预设）
- `internal/provider/river`（模型白名单、能力元数据、逻辑 ID）
- `internal/application/routing`（预设注入、公共目录、逻辑 ID 路由）
- `internal/application/modelcatalog`、`internal/application/agentconfig`（账号预设目录、客户端菜单）
- `internal/core/modelcatalog`（`[1m]` 变体、effort 词汇映射、Desktop 摘要）
- `frontend/src/features/providers`（卡片与预设浮层；只有测试 mock 含旧型号）

## Dependencies

- River 官方：`https://docs.river.ai/guides/inference-models/`（能力与上下限）、
  `https://docs.river.ai/guides/inference-controls/`（推理档位）、
  `https://console.river.ai/api/public/pricing`（官方计费模型 ID 与单价）。
- OpenAI 官方：`https://developers.openai.com/api/docs/models`（Featured models）。

## Risks

- 新模型的档位枚举会进入客户端菜单：只发布官方明确列出的取值，不猜测默认档位。
- 路由测试原先假设「River 所有模型都是 1M」，加入 262k 的 Qwen 后必须按真实元数据断言，
  否则会把「无 1M 权益」误测成缺陷。
- 公共逻辑 ID 可能与直连服务商同名模型混淆：`Kimi-K3` 与直连 `kimi-k3` 是不同字符串、不同计费，
  本任务不合并优先级（合并需要新的跨服务商同款模型规则）。

---

# Implementation Plan

1. 建立任务记录，核对 River／OpenAI 官方模型清单与上限。
2. 更新 OpenAI 预设为官方 Featured models。
3. 扩展 River 目录、能力元数据和逻辑 ID，补充推理档位。
4. 修正受影响的 Go 测试（按真实窗口断言）与前端 mock。
5. 同步产品文档与状态看板，运行 `task verify`，提交并归档。

---

# Progress

- 2026-10-27 12:3x 已核对 River 官方能力表、控制参数表和公开计费接口：新增两款确认为
  `Qwen/Qwen3.8-27B-FP8`（262,144／262,143，档位 none·low·medium·xhigh）与
  `moonshotai/Kimi-K3`（1,048,576／32,768，标准档位 none·max，分层需 `chat_template_kwargs`）。
- OpenAI 官方 Models 页 Featured models 为 GPT-6 Astra、GPT-6.1 Sol、GPT-6.1 Terra、GPT-6 Luna。

---

# Implementation Result

## Changes

- OpenAI 内置预设改为官方 Featured models：`GPT-6 Astra`、`GPT-6.1 Sol`、`GPT-6.1 Terra`、`GPT-6 Luna`；
  卡片摘要随之变为 `GPT-6 Astra · GPT-6.1 Sol +2`，不再出现 `GPT-5.4`／`GPT-5.4 Mini`／`GPT-4.1`／`o3`。
- River 官方预设由四款扩到六款，新增 `Qwen/Qwen3.8-27B-FP8`、`moonshotai/Kimi-K3`：
  - 白名单、`Models()`、`SupportsModel` 统一由同一列表驱动，模型发现过滤、路由资格、客户端目录、
    账号预设和提供商卡片预览同步生效，没有第二份清单。
  - `ModelMetadata` 按官方能力表逐款给出窗口／输出上限：Qwen 262144／262143，Kimi-K3 1048576／32768，
    其余三款沿用原值；六款仍是文本＋图片、函数工具。
  - 推理档位按官方控制参数表发布：Qwen `none/low/medium/xhigh`（官方用 `xhigh`，无 high／max）、
    Kimi-K3 `none/max` 且默认 `max`；Kimi-K3 的 `low/high/max` 分层需要 `chat_template_kwargs`，
    适配器不发送，故不公布；Nemotron 保持未确认（不新增档位）。
  - 公共逻辑 ID：`Qwen3.8-27B-FP8`、`Kimi-K3`；实际请求仍发送官方完整 ID。
    `Kimi-K3` 不与直连 `kimi-k3` 合并（不同计费入口）。
- 测试：River 目录测试改为六款并新增档位枚举回归测试；路由测试不再假设
  「River 所有模型都是 1M」，改为按目录元数据逐项断言，且只为 ≥1M 的型号校验 `[1m]` 变体；
  新增内置预设与官方清单一致性测试；前端 mock 中的过时型号同步更新。

## Files Changed

- `internal/application/provider/catalog.go`、`internal/application/provider/catalog_test.go`（新增）
- `internal/provider/river/catalog.go`、`internal/provider/river/identity.go`、`internal/provider/river/catalog_test.go`
- `internal/application/routing/river_test.go`、`internal/application/routing/river_identity_test.go`
- `frontend/src/features/providers/providers-view.test.tsx`、`frontend/src/app/provider-navigation.test.tsx`
- `docs/product/RIVER.md`、`docs/product/ROUTING.md`、`docs/product/GATEWAY.md`、`docs/product/PRD.md`、
  `docs/architecture/OVERVIEW.md`、`docs/plans/CURRENT.md`

## Technical Notes

- 只改数据与测试，没有新增适配器分支：River 六款模型共用既有 Chat／Responses 原生转发与
  Messages→Chat 转换，档位校验继续走 `validEffort` → `ModelMetadata` 的唯一来源。
- Qwen 的 262k 窗口使「River 模型一律 1M」的历史测试前提失效；把断言改成按元数据比较，
  比给 Qwen 补一个假的 1M 变体更接近真实能力（客户端不会出现 `Qwen3.8-27B-FP8[1m]`）。
- 已同步过的 River 账号仍以最近成功的官方目录为准，需要再点一次「同步官方模型」才会出现新模型；
  未同步账号立即看到六款预设。这是既有语义，本次未改变。
- River 型号未加入美元参考价表：直连 `deepseek-flash` 与 River Flash 共用逻辑 ID 但单价不同，
  现有 `tokenprice` 按模型 ID 计价，无法表达按服务商分流，属独立设计问题。

---

# Verification

## Tests

- [x] Unit Tests：`go test ./internal/...` 全部通过（新增 river 目录档位用例、provider 预设用例）
- [x] Integration Tests：`internal/application/{routing,modelcatalog,agentconfig,gateway}` 覆盖预设注入、
  逻辑 ID 路由、客户端目录与发现快照语义
- [ ] E2E Tests：N/A（本任务只改预设与元数据，未改请求链路）

## Quality Checks

- [x] Format（gofmt／prettier）
- [x] Lint（go vet／staticcheck／eslint）
- [x] Type Check（tsc）
- [x] Test（Go + Vitest）
- [x] Build（Windows production）
- [x] Verify（`task verify`）

## Verification Result

```text
task verify
EXIT=0
go test ./internal/...            PASS
frontend: 132 files / 809 tests   PASS
windows:build:native              PASS（bin/omniproxy.exe 23,127,040 bytes）
```

未使用真实 River／OpenAI Key 发起推理，也没有启动原生窗口做视觉验收；预设与元数据来自官方文档与
公开计费接口，真实账号可用性与实际窗口仍由上游响应决定。

---

# Documentation

- [x] PRD（`docs/product/PRD.md`：River 六款预设与新逻辑 ID）
- [x] Architecture（`docs/architecture/OVERVIEW.md`：River 模型数量与新增型号）
- [ ] Tech Stack
- [ ] Coding Rules
- [ ] Testing
- [ ] Build
- [ ] ADR（无架构决策变化；`Kimi-K3` 不合并属既有跨服务商规则范围）
- [x] CURRENT（`docs/plans/CURRENT.md` 追加本条完成记录）
- [x] 产品文档（`docs/product/RIVER.md` 能力表与说明、`ROUTING.md`、`GATEWAY.md`）

---

# Deviations

- 原计划只更新「模型清单」，实际同时修正了两处测试前提：路由测试的「River 一律 1M」假设，
  以及 River 目录测试写死的四款／输出上限。原因：加入 262k 的 Qwen 后这些断言不再成立，
  必须按真实元数据断言，否则会把正确实现测成缺陷。
- 原计划未包含 OpenAI 预设以外的型号；确认官方 Featured models 只有四款，未追加
  nano／pro／图像／音频型号，避免把非主推型号塞进卡片预览。
- 官方文档另列的「GLM-5.3（纯文本）」未纳入：公开计费接口的 `inference` 列表没有它，
  模型 ID 与计费口径未确认，按项目规则不猜测。

---

# Git

## Commit

待填写。

---

# Result

- 提供商页 OpenAI 预设与官方 Featured models 一致。
- River 预设增至六款，新增 Qwen3.8-27B-FP8 与 Kimi-K3，能力、档位、公共 ID 均按官方公开资料接入；
  未确认的型号、档位与计费口径保持未知。
- 完整质量门通过；真实 Key 推理与原生窗口视觉仍待用户验收。

---

# Completed

2026-10-27 13:00
