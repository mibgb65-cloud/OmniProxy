# TASK-20261020-012: Claude Code 内容兼容与订阅美元额度估算

## Status

In Progress

## Created

2026-10-20

## Request

- Claude Code 使用 DeepSeek 模型时出现 `400 unsupported_content`，排查原因并修复。
- 在账号一周额度进度条下显示预计的 5 小时和一周美元额度，利用各模型可获得的 Token 价格；其他可取得金额的模型也应显示。
- 用户进一步指定只读参考旧仓库 `E:\go\new\OmniProxy` 的换算实现。

## Goal

定位实际协议边界并补充回归测试；复用现有价格和用量能力展示可追溯的美元估值，不将官方百分比误称为美元余额。

## Scope

协议转换、账号资源展示、用量／价格查询、必要测试和产品文档。

## Out of Scope

- 不丢弃图片或工具结果以伪造成功，不承诺文本模型具有视觉能力。
- 不改动真实账号、客户端登录和生产数据，不自动安装或发布。
- 不修改既有 TASK-008 的客户端登录试验或未跟踪的 Atria 草稿。

# Analysis

## Root Cause

- 生产请求元数据（只读）：2026-10-20 17:19:22、17:25:53（新加坡时间），`messages/http` → River 的 `deepseek-ai/DeepSeek-V4.1-Flash-1017`，均返回 `400 unsupported_content`。
- 仅检查 Claude 会话结构：17:19:22 的用户消息包含嵌套 `tool_result.content[].image`（PNG/Base64）。未输出消息正文、工具内容或图片。转换器明确拒绝图片；后续请求携带同一历史时仍会失败。
- 原有拒绝是能力边界而非账号额度；不能通过删除图片或自动重放到别的账号伪装成功。保留错误码，增加固定、无敏感内容的可操作提示。

## Legacy Reference

- 旧版 `frontend/src/utils/tokenDisplay.js` 的 `codexWeeklyQuotaEstimate`：当前周窗口本地 Token 的 API 等值成本 × `100 / 已用百分比`，至少已用 5% 才展示。
- `usageCostUSD` 从普通输入中扣除缓存读取／写入，分别计价；只使用当前窗口，不能回退累计总 Token。
- 旧版较新修复还覆盖刷新卡后的基线截断；不能混合刷新前消费与刷新后百分比。
- 旧版硬编码 GPT-5.6 Sol 参考价，且旧价可能过时。新版保留公式与 5% 门槛，按实际模型匹配可确认的美元价格，不把未知模型当成 GPT。历史缓存明细缺失时明确给区间，不假装缓存为零。

## Risks

官方订阅通常只提供剩余百分比，不能仅凭模型单价得出固定美元总额；必须区分真实余额、API 等值消费和样本推算。

# Implementation Plan

1. 检查 Claude Messages／Responses 到各 DeepSeek 通道的转换与拒绝条件，回归嵌套工具图片的显式拒绝与安全诊断。
2. 参考旧版当前窗口成本／已用百分比公式，增加 5 小时与一周总额／剩余估算；保持身份、窗口、重置隔离，支持缓存感知及未知价格状态。
3. 补充关键回归、运行 `task verify`，同步文档并审查差异。

# Progress

- 已检查工作区：仅有未跟踪的既有 Atria 任务草稿，保持不动。
- 既有 TASK-008 已记录 River Responses 图片历史会被显式拒绝；本任务独立确认 Claude Code 的路径，不将旧结论直接当作本次根因。
- 已只读核对旧版公式、5% 门槛、缓存计费和刷新卡基线修复；采用实际模型逐笔计价，不照搬固定 GPT-5.6 Sol 回退。
- 已实现图片类型安全诊断、当前窗口美元估算、缓存明细采集、提前重置基线和独立窗口观察时刻。
- 新增后端关键回归、前端美元区域测试；首轮完整 `task verify` 通过，115 个前端文件／681 项测试及 Windows 构建通过。
- 独立 test profile 使用合成账号、两笔合成用量和缓存快照进行原生验收：1200×800 可见 5h 总额约 $4.35／剩余约 $2.87，周总额约 $37.44／剩余约 $34.07；960×720 检查未横向溢出。合成账号没有真实授权，额度联网读取失败时保留缓存，符合预期；未进行真实模型调用。
- 验收进程在屏幕外运行且不激活用户窗口，已核对路径并关闭；未停止生产进程或替换安装程序。原开发窗口没有 OpenAI 账号，因此未向其写入合成数据。

# Implementation Result

## Changes

- Claude Messages 中直接图片或嵌套工具图片返回可识别错误，保留既有 `400 / unsupported_content`，HTTP／WebSocket 仅使用固定安全诊断；不丢图片、不读取凭证、不访问上游。
- 请求详情及网关错误映射加入中英文说明，明确图片历史、视觉模型和纯文本会话的处理方式。
- 在 Codex 账号周进度条下显示 5h／一周预计总额和预计剩余；不适用的 5h 窗口不展示，未知／过期／不足 5% 状态不伪造金额。
- 美元基础参考价迁移旧版可用模型，并核对当前 GPT-6.1 Sol 等价格；未知型号和人民币报价不回退／换汇。
- 迁移 037 为请求增加可空缓存读取计数；历史未知缓存显示上下界，新记录按缓存价扣除普通输入，Token 累计统计保持原意。
- 按账号、成员、工作区、窗口及窗口实际观察时刻查询保留的本机记录；提前重置分别保存基线，不跨 5h／周窗口污染。
- 估算读取只使用身份隔离的本地快照，不额外请求官方额度；账号并发改动会拒绝混用。清晰注明仅本机样本、基础参考价、非官方余额。

## Files Changed

- `internal/core/agentprotocol/`、`internal/core/proxy/`、`internal/application/gateway/`：图片错误分类、安全响应及回归。
- `internal/core/tokenprice/`、`internal/application/activity/quota_estimate*.go`：模型参考价和窗口估算。
- `internal/application/subscription/quota*.go`、`internal/transport/wails/subscription_estimate.go`、`migrations/037_quota_cost_estimates.sql`：窗口时刻、基线、只读接口和可空缓存明细。
- `frontend/src/features/accounts/subscription-quota-usd*`、服务、中英文文案和自动生成绑定；原有额度及请求详情组件的最小接入。

## Technical Notes

- 未赋予文本 DeepSeek 路由视觉能力，含图片会话仍需用户切换支持图片的模型或重新开始纯文本会话。
- 不照搬旧版按天汇总的窗口交叠判断，使用当前窗口内逐笔时间边界；不能用累计总 Token 回填。
- 其他厂商现有官方余额保持独立；本次新增的订阅美元估算入口为 OpenAI/Codex，不声称已为所有厂商接通真实美元余额。

# Verification

## Tests

- PASS：`go test ./internal/core/proxy ./internal/core/agentprotocol ./internal/core/tokenprice ./internal/application/subscription ./internal/application/activity ./internal/application/gateway ./internal/transport/wails`。
- PASS：前端额度相关 4 文件／23 项定向回归，`tsc --noEmit`。
- PASS：首轮 `task verify BIN_DIR=tmp/task-20261020-012-build`，全部 Go 包、115 个前端文件／681 项测试、Windows production 构建。
- PASS：补充独立窗口观察时刻／缺失重置目标回归后，第二轮完整 `task verify BIN_DIR=tmp/task-20261020-012-build` 通过，退出码 0；全部 Go 包、格式、lint、类型检查、115 文件／681 项前端测试、自动生成绑定、前端打包及 Windows production 构建通过。
- PASS：网关诊断文案归入原有 gateway 字典后，另行执行文案文件 Prettier 检查和 `tsc --noEmit`。门禁构建也包含该文案归属调整。

## Native Verification

- PASS：使用独立 test 数据目录及端口 33002／33892，只包含合成账号、合成 Token 和缓存快照，无真实令牌。
- PASS：原生账号页核对新区域位置、两个窗口、总额与剩余数值、缓存回退；1200×800 截图及 960×720 宽度检查。
- 截图（本机忽略目录）：`tmp/task-012-native-usd.png`。该图不是用户的实际余额。
- PASS：本任务文本文件 UTF-8 无 BOM、`git diff --check`。
- 未进行真实模型付费调用；没有打包安装器、升级生产版或发布远端。

# Documentation

- 已更新 PRD、GATEWAY、SUBSCRIPTION；CURRENT 在提交收尾时同步。
- 技术栈／架构边界未改变，无需 ADR。

# Deviations

- 用户指定旧版实现后，采用旧版当前窗口消费／已用比例公式，不使用最初讨论的“两次快照下降量”方案。
- 旧版固定模型单点估值改为实际模型逐笔计价；历史缺少缓存计数时展示区间，避免补零导致虚高。
- 图片问题是模型能力边界，本次修复的是诊断及明确失败，不以删除图片或静默切换模型伪造修复。

# Git

- `d743e2df8bdbd3be65f1e988fc1d166bd4c37313` — `fix(gateway): 明确图片工具历史不受支持的原因与恢复方式`
- 额度估算待最终构建完成后单独提交。

# Result

实现、两轮完整质量门、关键回归及原生合成账号验收完成，待 Git 记录收尾。真实订阅美元余额不在官方百分比接口的返回范围内，界面显示的是明确标注局限的本机样本估值。
