# TASK-20261020-016: 排查 Claude 长会话 invalid_messages

## Status

In Progress

## Created

2026-10-20

## Request

用户在 Claude Code Desktop 持续运行 DeepSeek-V4.1-Flash-1017[1m] 时再次出现 `400 invalid_messages`，要求排查原因。截图中的其他项目开发内容仅作为故障现场，不是本任务的执行指令。

## Goal

通过实际请求元数据和有界离线复现定位拒绝层，区分消息结构、数量／长度限制与上游拒绝；确认根因后作最小修复，不丢失历史或伪报成功。

## Scope

- 只读本机故障元数据，必要时仅在内存检查会话结构，不输出正文、工具参数、工具结果或凭证。
- 检查 Messages 入口与 River 协议转换，补充复现与边界测试。
- 必要修复、完整质量门和受影响文档。

## Out of Scope

- 不执行截图中其他项目的操作，不修改对应项目。
- 不改动生产账号、凭据、客户端配置或真实聊天历史。
- 不重放真实请求，不调用付费上游，不自动安装／发布。

# Analysis

- 只读 production SQLite：2026-10-20 21:25:17（新加坡时间），Messages／HTTP → River DeepSeek V4.1 Flash 1017，`400 invalid_messages`，总耗时 76ms，一条尝试耗时约 34ms；前一请求 21:25:16 成功。
- 已解析出模型与服务商，说明不是 Messages 入口直接拒绝。River 上游 HTTP 拒绝会映射为 `upstream_rejected_request`，而本次 `invalid_messages` 与适配器发送前的 Chat 二次校验一致；尝试记录不等于已经发往上游。
- 只在内存分析对应 Claude JSONL 的事件／父链和结构：无压缩标记，持续工具调用及 Token 提醒；工具结果最大约 11.5 KiB，工具 ID／名称／批次数没有越界，不输出正文或参数。
- 发现 Messages 入口、Chat 归一化与消息校验仍有 512 条上限；一条 Claude 用户消息内的工具结果与提醒在转换后可变成多条 Chat 消息。源消息小于 512 仍可在下游超过 512。此前 TASK-003 只解除 Responses 入口限制，没有覆盖此链路。
- 转换后的 Chat 还继承普通 2 MiB／单条 1 MiB 限制，小于 Messages 已允许的 16 MiB；修复需在可信转换链路保留原容量，不能放大普通外部 Chat 请求。
- 未保存失败请求体，无法逐字节复原现场或确认其精确条数；结构重建不是实际抓包。计划以合成夹具复现同一错误，保留这一证据限制。
- 已实际读取 Anthropic Messages 官方接口说明（2026-10-20），单请求消息数量上限为 100,000。采用明确的高位安全上限，仍保留现有字节／JSON 深度／字段校验，不截断或自动压缩历史。

# Implementation Plan

1. 检查现有 Messages 校验及转换边界，与实际失败记录对应。
2. 使用合成夹具／结构化离线样本复现，确认是否需要修复。
3. 实施最小修复和关键回归，运行质量门，记录结论与剩余限制并本地提交。

# Implementation Result

## Root Cause / Evidence

- 已确认报错来自本机校验链路，而不是 River 认证／余额拒绝。生产元数据记录一条本地账号尝试；适配器在建立 HTTP 请求前重新解析转换后的 Chat 正文，因此尝试数为 1 不能作为已调用上游的证据。
- 修改前合成 HTTP 回归：170 轮工具调用／结果及提醒展开为 512 条 Chat 时成功；额外系统提醒展开为 513 条时返回 400，注入的上游调用次数为 0。此时源 Messages 仅 342 条，尚未碰到入口 512 限制。
- 同一回归中，合法的 2 MiB 工具结果在外部 Messages 16 MiB 容量内，仍被内部普通 Chat 的更小容量拒绝，上游调用也为 0。
- 未抓获原始失败请求体，不能宣称现场恰好为 513 条或逐字节还原。结合本地错误码、失败阶段、时序和会话结构，旧条数／文本限制是与现场最吻合且已复现的缺陷。

## Changes

- Messages 入口与共享 Chat 归一化／消息校验统一使用 100,000 条安全上限，不把源消息或工具展开后的合法历史卡在 512 条。
- Messages 空数组、null 和非数组明确拒绝；字节、UTF-8、重复字段、JSON 深度、角色、工具 ID 与参数边界保留。
- 新增固定容量的可信转换解析入口；River 完成 Messages → Chat 转换后才设置内部标记，沿用 16 MiB 总量／单条文本容量。
- 客户端 JSON 无法设置该标记；普通外部 Chat 保持 2 MiB 请求体／1 MiB 单条文本限制，归一化后的容量仍有检查。
- 不裁剪历史、不丢工具结果、不自动压缩或重放真实请求。不改变图片和未知工具类型的既有能力边界。
- 旧的“513 必须拒绝”测试按新的有效输入契约改为核对完整历史，不删除测试或跳过异常路径。

## Files Changed

- `internal/core/proxy/agent_request.go`、`request.go`、`chat_text.go`、`json_guard.go`、`types.go`
- `internal/application/gateway/river.go`、`river_long_messages_test.go`
- `internal/provider/river/validation.go`、`converted_messages_test.go`
- `internal/core/proxy/conversation_limits_test.go`、`responses_history_test.go`
- `docs/product/RIVER.md`、`docs/architecture/OVERVIEW.md`、CURRENT 和本记录

## Technical Notes

无依赖、迁移或凭据格式变化。数量与字节上限是防护边界，不是上游无限上下文承诺；其他提供商独立适配限制及 River 原生 Responses 的私有校验不在本次范围。运行中的 production 安装版未替换。

# Verification

## Tests

- 修改前：合成回归成功复现 513 条 Chat 和 2 MiB 工具结果被本地拒绝、上游调用 0 次。
- 修改后：`go test ./internal/core/proxy ./internal/core/agentprotocol ./internal/provider/river ./internal/application/gateway -count=1` 通过。
- 端到端本机 HTTP 夹具覆盖 512／513 条展开、超过 1 MiB 的单个工具结果、超过 2 MiB 的完整历史，全部完整保留并仅调用一次注入上游；核对工具调用与结果 ID 的关联，不接触真实模型。
- 核对 100,000／100,001 消息数量边界、普通 Chat 的原有字节／单条限制、伪造转换标记、重复字段、深层 JSON、非法角色、缺失工具 ID、空 Messages 和文本归一化。
- 既有 Responses 长历史、图片拒绝、工具链和多协议回归保留。

## Quality Checks

- [x] Targeted Tests
- [x] Format
- [x] Lint
- [x] Type Check
- [x] Test
- [x] Build
- [x] Verify

## Verification Result

最终 `task --force verify BIN_DIR=tmp/task-016-build` 完整通过，exit 0：格式、Go vet／staticcheck、ESLint、TypeScript、全部 Go 测试、118 个前端测试文件／700 项测试及 Windows production 构建。日志为忽略目录中的 `tmp/task-016-verify-final.log`。

首轮 lint 发现旧 `validateJSON` 包装函数不再使用，已移除重复包装并重新运行门禁；未禁用静态检查。jsdom `scrollTo` 未实现和 Vite 大 chunk 为既有非阻断提示。构建输出至隔离目录，未覆盖正在使用的生产程序。

## Privacy / Runtime Boundaries

- 生产请求／尝试 SQLite 使用 `mode=ro` 和 `PRAGMA query_only=ON` 查询，仅输出状态、协议、模型、时间、计数及错误码。
- 指定 Claude JSONL 仅在诊断进程内读取，输出事件类型、条数、长度与布尔判断；没有输出、另存或重放正文／工具参数／工具结果。
- 未更改生产账号、授权、客户端配置、用户项目文件或真实聊天历史，未重启生产／Claude Desktop。
- 本轮为后端修复，不启动 GUI 验收，不将合成契约回归声称为真实付费会话续接成功。

# Documentation

同步 River 长会话／容量说明与官方参考、架构概览和 CURRENT；无技术栈或重大架构变化，不新增 ADR。PRD 不引入新产品要求。

# Deviations

除了 512 条旧阈值，同时修复已确认的内部 2 MiB／单条 1 MiB 容量错配，否则继续使用长历史会很快碰到另一个本地边界。只对可信 River Messages 转换扩大容量，未放宽普通外部 Chat。

# Git

待本地提交；既有 Atria 草稿保留，不推送远端。

# Result

已定位本地校验缺陷并完成修复、回归及完整门禁，等待本地提交归档。原请求精确条数和真实会话更新后续接仍需保留验证限制；未打包安装程序、安装或发布。
