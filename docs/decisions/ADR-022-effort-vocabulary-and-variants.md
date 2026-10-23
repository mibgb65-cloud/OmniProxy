# ADR 022: 推理档位按客户端词汇发布，变体型号由网关折叠

Status: Accepted

Date: 2026-10-24

## 背景

Codex 目录（`omniproxy-models.json`）只接受 official catalog 的推理档位词汇；上游用
`none`／`max` 表达两端，Codex 不认识这两个值，选择器只显示唯一一个已认识的值
（实测 `DeepSeek-V4.1-Flash-1017` 只能停在「高」，无法切换）。

Google 系上游用**独立型号**表达 thinking level（`…-low`、`…-high`，显示名 `Gemini 3.8 Flash (Low)`）。
目录照搬后客户端看到的是多个"模型"：同一型号出现多行、档位变成模型名，
而且 `reasoning_effort` 会被订阅适配器的生成参数白名单拒绝。

## 决策

- 目录向客户端发布的档位必须是客户端自己的词汇：Codex 用 `minimal…xhigh`，
  上游 `none → minimal`、`max → xhigh` 就近映射，其余档位原样；未确认的档位不发布。
- 网关接收请求时按同一套映射还原上游取值（`minimal → none`、`xhigh → max`），
  只接受该模型已确认的枚举，未确认时返回 `unsupported_request_effort` 类安全错误码。
- 上游用独立型号表达档位时，目录把它折叠回基础模型：只发布一行，
  `Metadata.EffortModels` 记录档位 → 变体 ID 的映射（不发给客户端）。
  请求期把客户端档位换成实际存在的变体 ID，并移除已表达的档位字段，
  避免适配器收到不认识的参数。
- 折叠只在同名基础模型同时存在时发生；只有变体的孤立型号保持原样。
  历史保存的变体 ID 通过反向映射还原成「基础模型 + 档位」，不会变成未知模型。
- `Metadata.EffortModels` 属于目录内部数据，`/v1/models` 与 Codex 目录都不输出。

## 后果

- Codex 与 Claude 系客户端选择档位时看到的是自己认识的取值，不再出现"只能选一个"。
- Google 系型号在客户端只占一行；档位由客户端 effort 控件选择，网关负责换型号。
- 上游新增档位变体时，只有同时发布基础模型才会折叠；命名不符合 `(High)`／`(Low)`／
  `(Medium)`／`(Minimal)` 约定的变体保持独立条目。
- 折叠在下次同步模型时生效；已保存的旧变体 ID 仍可解析，但目录刷新后菜单会少掉那些行。
- 档位映射是就近语义（`none` → `minimal`、`max` → `xhigh`），不等于上游关闭思考／最高强度
  在客户端有同名选项；描述文案沿用上游，不夸大能力。
