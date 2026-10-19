# TASK-20261019-017: 精简 OpenAI 设置并增加账号轮换方式

## Status

Completed

## Created

2026-10-19

## Request

1. 移除 Codex WebSocket 的共用范围、无需重启等补充提示。
2. 移除自动激活的账号范围／消耗／不重发，以及自动保存／检查周期两段补充提示，保留主要功能说明。
3. 新增账号轮换方式设置：队列轮换（额度耗尽后切换下一个账号）和平衡切换（均衡使用多个账号额度）。
4. 将“最近检查 … 个账号，发送 … 条消息，确认 … 个账号窗口已开始计时”改为已有全局提示样式。

## Goal

精简弹窗内容，持久化并实际应用 OpenAI 账号调度模式；自动激活反馈复用全局提示，不重复堆叠内联消息。

## Scope

- OpenAI 账号设置弹窗、中英文文案、全局通知。
- 账号调度模式的存储、接口及实际请求选择。
- 单元／集成／前端回归和必要文档。

## Out of Scope

- 不修改其他提供商调度策略、官方授权及既有故障切换安全边界。
- 不发布或安装生产包，不操作真实账号额度。

## Existing / Expected Behavior

当前只按稳定创建顺序及已知额度拒绝选择同池账号；设置仅含 WebSocket 与自动激活，检测摘要内联显示。
新增持久化 queue / balanced 模式，默认 queue 保持升级兼容；balanced 只改变同池首次选号，不突破固定账号、同池身份、模型权限及生成开始后不重发边界。

# Analysis

## Related Modules

- `gateway.CodexSettingsService`、`gateway.forwardWithFailover` 和 `subscription` 的版本化额度观察。
- OpenAI 设置弹窗、自动激活计数查询及现有 Radix Toast 风格。
- 方案比较：纯轮询不参考额度；只选最高剩余可能短期集中。采用平滑加权轮询，候选都有一分钟内的可信额度时按限制窗口剩余百分比加权，否则等权轮询；只读取已有内存观察，不在请求热路径启动官方组件。
- 全局提示由设置入口持有，离开弹窗后仍可展示；相同检测结果只提示一次，加载初始零值不弹出。

## Risks

- 多请求并发、会话粘性及备用账号选择不能破坏既有不重复发送规则。
- 无额度快照、过期快照或相同额度时必须有确定且安全的行为。
- 新模式默认保留队列语义，避免升级后静默改变既有调度。

# Implementation Plan

1. 阅读规范、订阅／路由文档和已有调度测试，比较平衡策略的可执行边界。
2. 定位实现并补充分析；先补回归，再完成设置持久化和真实选号。
3. 精简提示、复用全局通知，补齐中英文和前端验证。
4. 执行针对性测试及 `task verify`，审查差异、同步文档、本地提交并归档。

# Progress

- 已创建任务记录，尚未修改业务代码。
- 已确认复用现有候选计划与故障切换，模式读取对下一请求生效；不改 Core 选择算法和其他提供商行为。
- 已完成迁移 035、持久化与 Wails 绑定、真实请求轮换及三项设置共享保存锁。
- 已删除指定三段补充说明；检测摘要改为全局 5 秒提示，同一结果去重，关闭设置后仍可见。
- 已覆盖 HTTP／WebSocket 实际账号序列、加权／等权／耗尽、并发、固定账号、同身份去重及不可重放边界。
- 复核时修正大候选池的状态容量判断，避免把已有项重复计入容量导致每次重置轮询；3000 个候选及版本淘汰回归连续三轮通过。
- 两轮完整质量门通过，最终 109 个文件／629 项前端测试、全部 Go 测试与 Windows 构建通过。

# Implementation Result

## Changes

1. 精简 OpenAI 设置：移除 WebSocket 共用范围说明、自动激活范围／消耗说明及保存／检查周期说明；保留主要功能、保存进度及安全错误反馈。
2. 增加队列／平衡二选一，默认 queue；与 WebSocket、自动激活共享查询和保存互斥，关闭／重开不取消保存，失败保留已确认值。
3. 新增有约束的非敏感模式字段与薄 Wails 接口。真实转发按当前模式处理同池候选；queue 保留原顺序，balanced 采用平滑加权轮询，新鲜完整额度按限制窗口剩余百分比加权，否则等权分配。
4. 保留固定账号、模型目录范围、账号配置版本、计费／地区、最多三次尝试、生成后／结果不确定不重放边界；同一已知订阅身份不重复参与。全部耗尽不发探测，故障重试关闭时也不会向已知耗尽账号发送平衡探测。
5. 平衡分数按账号／版本隔离，容量有界，并发分配有锁，网络不持锁；账号列表每次只读取一次，不为每个候选重复列举数据库，不额外查询上游。
6. 检测摘要复用 Radix Toast 的全局悬浮风格，支持手动关闭、5 秒自动消失及结果去重；待确认／失败使用对应提示，不伪报为成功。

## Files Changed

- `migrations/035_codex_account_rotation.sql`。
- `internal/application/gateway/account_rotation*.go`、Codex 设置、服务与转发调用点。
- `internal/application/subscription/quota_balance*.go`。
- `internal/transport/wails/codex_settings.go`、`gateway.go` 及工具生成的两份绑定。
- OpenAI 设置、轮换选项、全局 Toast、订阅双语言字典与对应前端测试。
- PRD、路由／故障切换／订阅说明、架构概览、ADR-018、CURRENT 及本记录。

## Technical Notes

无新依赖、凭据格式或 Core API 变化。平衡的是请求分配，不能据此保证不同模型／上下文的实际 Token 或额度百分比完全相同。只读路由预览不消耗轮询位置；HTTP／SSE／WebSocket 每次真实请求共用策略。

# Verification

## Tests

- PASS：`go test ./internal/application/gateway ./internal/application/subscription ./internal/application/quotaactivation ./internal/transport/wails -count=1`。
- PASS：`go test ./internal/application/gateway ./internal/application/subscription -run 'Rotation|Balance' -count=3`。
- PASS：大池容量修正后的账号序列、加权、边界、并发和容量回归三轮。
- PASS：前端类型检查与三个设置测试文件，16 项；覆盖模式持久化、保存失败与互斥、隐藏静态文案、全局提示位置／保留／去重、中英文。
- PASS：034 → 035 迁移保留旧开关，默认 queue；模式保存／重新读取、其他开关不覆盖模式、非法值和缺失记录错误。

## Quality Checks

- PASS：`task verify BIN_DIR=tmp/openai-rotation-20261020/verify-bin`。
- PASS：容量边界修正后的最终 `task verify BIN_DIR=tmp/openai-rotation-20261020/final-bin`，exit 0；格式检查、Go vet／staticcheck、ESLint、TypeScript、全部 Go 测试、109 个文件／629 项前端测试、实际前端生产构建与 Windows amd64 production 构建通过。
- PASS：Wails 工具生成并检查新接口与字段，未手工编辑绑定。
- PASS：`git diff --check`。
- jsdom scrollTo 未实现及 Vite 大 chunk 为现有非阻断提示。

## Limitations

已尝试隔离 test profile 的原生 UI 自动验收，但窗口未暴露可操作的 UIA 控件，未能可靠导航到设置弹窗，不能将其他同名窗口截图当作本次通过证据。已关闭仅本任务启动的测试进程。原生布局仍需人工确认，真实订阅额度消耗未实测；所有调度网络回归使用隔离夹具，没有读取真实账号、发送付费消息或修改生产程序。

# Documentation

已同步 PRD、Architecture Overview、ROUTING、FAILOVER、SUBSCRIPTION、CURRENT，新增 ADR-018。
技术栈、构建和测试规范无变化。

# Deviations

使用两项单选而非有歧义的布尔开关表达互斥模式。平衡采用请求级加权，不猜测 Token 到额度的兑换比例，也不引入额外后台模型探测。原生自动验收未取得可靠结果，已明确保留人工验收限制。

# Git

- `58521d12569777cc02668dea0d5ccf990a3c0165 feat(account): 增加 OpenAI 轮换模式并精简设置反馈`
- 仅本地提交，未推送；任务归档和 CURRENT 更新使用后续文档提交。
- 已审查实际差异与状态，未纳入其他已有未跟踪任务记录，未更改版本或安装包资产。

# Result

指定文案精简、真实队列／平衡轮换、持久设置及全局检测提示已实现；回归和最终完整质量门通过。
原生布局及真实账号额度效果仍需人工验收，本次未发布安装包、自动安装、发送付费消息或替换生产程序。

# Completed

2026-10-20
