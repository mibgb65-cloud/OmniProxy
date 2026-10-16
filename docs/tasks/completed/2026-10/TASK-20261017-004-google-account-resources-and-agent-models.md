# TASK-20261017-004: Google 账号资源归位与 Agent 模型接入

## Status

Completed

## Created

2026-10-17（Asia/Singapore）

## Request

用户要求将独立的 Gemini 项目资源区整合至对应 API Key 账号卡片，减少页面混乱；
Google / Antigravity 订阅账号左侧显示的模型也应能够接入 AI Agent 使用。

## Goal

账号与资源信息就近展示；订阅可选文本模型通过真实协议适配、路由及 Agent 目录形成可调用闭环。

## Scope

- Google API Key 账号卡片／列表资源展示、项目绑定与刷新入口。
- Antigravity 官方选择器中的文本／工具模型发现、网关协议适配与 Agent 目录。
- 对应回归测试、质量门及产品文档。

## Out of Scope

购买额度、多模态功能、强制覆盖非受管客户端配置及无关模块重构。

# Analysis

现有页面在账号卡片外重复显示项目资源；订阅目录仅开放 Gemini Chat，
Claude／GPT 型号仅展示。需核对既有中立协议转换器并复用，避免只放开目录而无法推理。

选择复用已有有界文本／工具转换器：Chat 保留原生增量 SSE，Responses／Messages
缓冲完整上游结果再编码，明确不宣称多模态能力。仅按官方选择器开放已确认的
Claude Opus／Sonnet 4.6 Thinking 和 GPT-OSS 120B Medium；旧 Gemini ID 保持兼容。
Agent 使用同步按钮写入已有账号目录，并由既有受管目录刷新更新菜单。
跨协议工具签名仅保存在有界、有时限、按授权／项目／模型隔离的内存缓存，不保存消息正文。

# Implementation Plan

1. 核对账号展示、项目资源及订阅模型到 Agent 的完整链路。
2. 以最小组件复用整合 API Key 资源，保留共享项目的准确语义。
3. 扩展已确认文本型号及必要协议转换，保持凭证和账号版本隔离。
4. 补齐关键路径回归测试，运行 task verify，核对原生开发窗口。
5. 同步产品文档、检查差异并提交归档。

# Implementation Result

- API Key 资源整合至对应账号卡片／列表行，保留项目绑定、官方未知余额、错误观察、
  官方用量／账单及本机用量入口；创建／更新时间归位到卡片底部。配额页继续按项目归组。
- 抽出共享资源查询和内容组件，同项目多个 Key 只发起一次查询；凭证版本变化使用新查询键。
- 官方当前选择器中的 Gemini High／Low／Medium、Opus、Sonnet 和 GPT-OSS 全部支持
  订阅 Chat／Responses／Messages 文本和工具请求，普通 API Key 协议边界不变。
- 增加“同步到 AI Agent”，复用账号目录同步和受管客户端菜单刷新；失败保留额度显示，
  成功提示运行中客户端重新加载菜单。原生开发窗口已实际点击同步并确认成功。
- 跨协议复用已有转换器；增加指定函数选择和有界签名恢复，保持授权／项目／模型／工具
  内容摘要隔离，随机生成缺失的工具 ID；没有新依赖、接口或迁移。
- 真实探测发现 Sonnet ID 为 `claude-sonnet-4-6`，而非显示名推测的 thinking 后缀；
  Claude 工具终态为 `OTHER`，仅完整工具块接受该终态。两项均加入回归。

主要涉及 `features/accounts`、`features/google`、`provider/antigravity`、
`application/gateway/google_agent.go`、路由协议目录及相关测试。

# Verification

- 定向 Go（Antigravity、gateway、routing、modelcatalog、agentconfig）通过；新增
  六型号 × 三协议 × JSON／SSE × 两轮工具的本机网关契约回归（72 个请求），并检查
  Codex／Claude 菜单、未支持图片拦截、官方选择器、签名隔离／过期／上限及 OTHER 终态。
- 定向前端 5 文件／28 测试通过；全量前端 88 文件／476 测试通过，覆盖资源归位、共享
  查询、卡片／列表切换、同步重试／脱敏和既有项目绑定错误保护。
- 最终 `task verify BIN_DIR=tmp/google-agent-20261017/verify-bin`：PASS，退出码 0；
  格式、go vet／staticcheck／ESLint、类型检查、全部 Go 测试、476 项前端测试及 Windows
  生产构建通过。已有 jsdom scrollTo、环境缺少 uname／tail 提示不影响成功退出。
- 真实开发账号：官方六型号均标记适配，账号目录同步保存 17 项（含 Gemini 历史兼容项）。
  Opus／Sonnet／GPT-OSS 经真实网关 Responses 各返回 HTTP 200；Gemini Flash Low、Sonnet
  经 Messages 完成工具调用／结果续接，两轮均 HTTP 200。实际用量进入本机记录，未启用付费
  积分或跨通道重试。额外调试只保存协议字段、长度和稳定状态，不保存推理正文或授权信息。
- 按进程路径确认当前原生开发窗口，核对新布局、六型号适配数量、同步成功提示、深浅主题、
  列表模式和 960×600；界面设置和窗口尺寸已恢复。证据在忽略目录 `tmp/google-agent-20261017/`。
- `git diff --check` 通过；其他五份未跟踪任务记录未改动。

# Documentation

同步 Google 订阅、Agent 接入、配额、PRD、架构与 CURRENT。

# Deviations

真实上游确认 Sonnet ID 与 Claude 工具终态后补齐兼容处理。采用现有缓冲协议转换器，
Responses／Messages 暂不承诺逐 Token 增量输出；无需引入新适配框架或数据库迁移。

协议依据：[参考实现请求转换](https://github.com/router-for-me/CLIProxyAPI/blob/e7003d2a3e2cee2d45f9950c067bf8e02a2ac0f8/internal/translator/antigravity/claude/antigravity_claude_request.go)，并以真实上游只读目录及最小推理结果核对。

# Remaining Risks

仅支持文本与函数工具；高级参数、多模态、长上下文及所有型号的真实复杂 Agent 工作负载
仍未穷尽验证。签名仅内存有界保存，淘汰／重启后可能需重新开始工具轮次。官方权限与内部
协议可能变动；运行中的客户端可能缓存模型菜单，需要重新加载。

# Git

实现提交：`dc11eca5fd1093d8afa5c8298693fcbe078a3ede feat(google): 整合账号资源并接入 Agent 订阅模型`。

分支 `codex/river-builtin-provider`；归档记录及 CURRENT 由后续文档提交保存。

# Result

资源已按账号归位；截图中的六款订阅型号已进入 Agent 协议目录，实际同步和推理／工具续接
通过。完整质量门通过，实现已提交并归档。

# Completed

2026-10-17（Asia/Singapore）
