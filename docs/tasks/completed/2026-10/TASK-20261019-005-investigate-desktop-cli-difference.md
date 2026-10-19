# TASK-20261019-005: 定位 Codex Desktop 与 CLI 转发差异

## Status

Completed

## Created

2026-10-19 11:50

## Request

用户明确：成功请求来自本会话 Codex CLI；通过 Codex Desktop（ChatGPT.exe）的请求失败，要求查明原因和处理方法。

## Goal / Scope

区分真实 Desktop 外壳、内核、配置及请求形态与 CLI 的差异，尽量在隔离环境复现；对证实的原因提出或实施最小修复。

保护本会话及生产：不重启、不改共享配置，不输出凭据、账号身份、提示词及回复正文，不重放真实用户请求。读取日志仅提取有界安全诊断字段。

## Analysis

TASK-004 已定位 12 次失败均为上游 200 后首事件类型推断未通过，6 次 WebSocket 后 6 次 HTTP；用户本轮补充的客户端归属排除了“Desktop 已恢复”的推断。历史 Desktop 内核最小隔离请求成功不能替代外壳真实形态验收。

## Implementation Plan

1. 检查真实 Desktop 运行内核、官方日志入口及安全配置摘要。
2. 对照双内核／代理的本地协议与请求结构，区分模型、工具、续接及外壳覆盖。
3. 在独立目录和随机端口进行不改真实配置的最小验证；根据证据处理或给出具体后续步骤。
4. 完成必要检查，记录结论、验证与限制。

## Implementation Result

### 已确认事实

- Desktop 实际程序为 MSIX ChatGPT.exe，内核 0.163.0-alpha.10.2；本会话 CLI 内核为 0.163.0。
- 用户明确成功记录属于 CLI；本地只读线程元数据也显示 Desktop 与 codex-tui 为不同来源。
- 当前共享配置为 openai / gpt-6.1-sol / xhigh，压缩与两项 WebSocket 特性配置均关闭，本地代理入口有效；不输出含接入 Key 的原始 URL。
- 官方排查说明明确两端可携带不同 Codex 版本。历史隔离测试只覆盖内核最小请求，不能覆盖实际 Desktop 外壳附加工具／会话参数。
- 已定位实际桌面日志目录，不复制原始正文或对话。

### 本轮进一步排查与实现

- 用户确认只测试了全新 Desktop 会话，旧会话续接不作为主要原因。
- 只读解析原生运行日志的固定设置标签，并关联线程来源：失败 Desktop 的 incremental_tools=true；成功 CLI 的 incremental_tools=false。其他特性也有差异，单凭开关不同不能证明因果。
- 在独立 HOME／CODEX_HOME、随机 loopback 端口、模拟凭据与空工作目录，以稳定版 0.163.0 和实际 Desktop 0.163.0-alpha.10.2 生成请求，四组协议均正常；追加两端实际特性布尔快照后四组仍完成。模拟请求与假凭据仅存放在忽略的 tmp 目录，没有复制真实会话。
- 合成请求表明预热帧与生成帧的工具声明位置确实不同；不把该结构差异直接判定为错误。固定订阅端点可以处理一部分合成 additional_tools 请求。
- 使用当前启用的应用隔离订阅账号完成两次小型合成 Desktop 请求和一次最小基线，均获得合法 response.created / response.completed；缺失 Content-Type 可正常推断。仅输出分类、字段大小与完成状态，不保存正文或凭据。原历史失败账号已无法关联当前账号表，不能将当前账号的探测当成原故障复现。
- 两次使用完整特性形态的合成请求在收到 HTTP 响应前遇到 EOF / upstream_unavailable，与原先 200 后 invalid_upstream_response 是不同失败阶段，未将两者混为一谈。
- 确认 response.created 会回显工具定义，当前 512 KiB 首事件预读上限可能对工具较多的客户端产生影响；本地边界测试已覆盖完整事件恰好达到上限和超过上限，但尚无历史 Desktop 首事件大小，不能声称生产已证实为超限。
- 增加固定首事件诊断与有界预读大小，原始拒绝／接受规则不变；Core 再做枚举白名单保护，并允许安全字段进入桌面日志详情。可在下一次真实 Desktop 失败时区分前缀超限、读取／截断、非 SSE JSON／HTML及具体校验阶段。
- 增加诊断不改变字节流、大事件边界、未知诊断不泄露、桌面详情可读取等测试。

## Verification

- PASS：go test ./internal/provider/chatgpt ./internal/core/proxy ./internal/infrastructure/logger -count=1。
- PASS：安全诊断先行 task --force verify；合并额度切换与兼容修正后，由 build-release.ps1 再执行最终完整质量门，576 项前端测试、Go、格式／静态／类型检查及 Windows 构建均通过。
- PASS：实际 CLI 0.163.0 与 Desktop 0.163.0-alpha.10.2 在独立模拟上游中完成额度切换、600 KiB 首事件及 turn.completed 的原生契约；不等同于真实 Desktop 外壳验收。
- 未重启生产、未更改真实客户端配置、账号或数据库；未重放真实用户请求。


### 合并交付

- 用户随后要求与 5h／周额度耗尽切换一起修复，生成本地 Beta.6 安装包自行测试，明确不发布 GitHub；相关实现与产物见 TASK-20261019-006。
- 修复已复现的合法大首事件兼容边界：提供商推断和 Responses HTTP／WebSocket 单事件统一为 2 MiB；其他协议上限不变。补充严格执行前额度拒绝分类，普通 200 错误及开始生成后错误不重放。
- 安装包及 SHA256 已生成；本轮未替换已安装生产程序，未改共享 Codex 配置。

## Git

f31918dbf55ddcce1a6f9215f15aaf741a964f23 fix(gateway): 修复订阅额度轮换并兼容 Desktop 流响应

仅本地提交，未推送、打远端标签或发布。

## Result

已完成双端差异核对、可复现兼容修正与安全诊断，并合入本地 Beta.6 供用户安装测试。历史 Desktop 响应正文未留存，不能将某个特性开关或单一分支断言为全部生产失败的唯一根因；真实 ChatGPT.exe 新会话的恢复情况待用户复测。

## Completed

2026-10-19 12:56
