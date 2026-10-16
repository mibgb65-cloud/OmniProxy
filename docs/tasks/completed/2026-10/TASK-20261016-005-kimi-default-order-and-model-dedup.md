# TASK-20261016-005: Kimi 默认使用顺序与模型去重优化

## Status

Completed

## Created

2026-10-16（Asia/Singapore）

## Request

用户询问 Kimi 是否需要同样处理，并要求「你都优化做了吧」。此前已说明当前只有国内／海外普通 API Key，需要默认地区优先及模型去重，Kimi Code 订阅需另外接入。

## Goal

用户已补充确认「也包括 Kimi Code 订阅 Key 接入」。完成普通 API／Kimi Code 订阅的国内／海外账号选项、原生文本调用、默认计费／地区顺序及模型去重，保留旧地区和固定账号兼容。

## Analysis

当前 routing.kimiAliases 自动公布国内／海外副本，kimiAmbiguous 阻止混合地区的裸模型名。MiMo／StepFun 已具备独立持久设置、候选过滤、共用前端设置及兼容目录，可在既有边界内扩展。Kimi Code 没有现有认证、账号选项或适配器，需先核实官方协议及与普通 API 模型的关系，不能直接复用订阅标签。保留五份既有未跟踪任务记录。

## Implementation Plan

官方公开文档已确认国内／海外 coding/v1 三协议及四个订阅模型 ID。迁移 028 为旧账号增加默认 payg 的计费类型，并保存默认订阅优先／国内优先设置。相同模型按已核实关系规范化，K2.8 Preview 与普通 K2.7 不混同；订阅 K3 按未验证套餐保守展示 256K。保留真实客户端 User-Agent，不伪装工具。没有已确认的订阅 Key 模型发现／额度接口时仅显示预设和官方控制台入口。Key 接入不包含 OAuth 登录。

官方依据（2026-10-16）：https://www.kimi.com/code/docs/ 、https://www.kimi.com/code/docs/kimi-code/models.html 、https://www.kimi.com/code/docs/kimi-code/membership.html 。

1. 核实 Kimi 普通账号路由、客户端目录及 Kimi Code 官方接入方式，明确实施范围。
2. 增加持久偏好、默认选择、公开目录去重及旧地区／Claude ID 兼容；复用现有设置交互。
3. 补充边界／回归测试和文档，执行完整质量门及正常桌面核验，审查提交并归档。

## Implementation Result

- 增加普通 API／Kimi Code 订阅 Key 与国内／海外账号组合，单个／批量／编辑保持保护凭据与乐观锁；旧账号迁移为 payg，凭据、地区、启用状态及旧目录身份保持。切换计费方式与地区一样使版本递增、旧目录失效、旧在途选择拒绝使用新配置。
- 原生订阅 Chat／Responses／Messages 文本与 SSE 使用固定 coding 端点，保留工具／思考历史。K3 与同款高速版精确映射上游 ID，K2.8 Preview／K3-256k 独立发布；真实 User-Agent 有界保留，不透传客户端认证、不伪装工具。
- 设置增加独立的计费／地区顺序，默认 coding／cn，选择即保存，任一保存期间两个选项互斥；关闭重开保持，失败显示安全错误并保留旧显示。先服务商顺序，再计费、地区、创建时间／ID；固定账号／显式通道优先，失败不跨通道重试。
- 公开目录合并为六款逻辑模型，不自动发布计费／地区副本；旧地区、旧 Claude 上下文／编码 ID 及官方订阅原生 ID 保持兼容。目录能力取自所选账号，未核实订阅档位的 K3 保守公布 256K。
- 订阅卡片／列表、配额页显示未知订阅额度与固定控制台入口，不借普通余额冒充订阅，不调用未确认的订阅发现或额度接口。普通 API 模型同步和余额继续保持原行为。
- 正常桌面开发环境由原有 Wails 监听器重建；原生设置与新增账号表单已验收。恢复 coding／cn 及原窗口／主题，未向用户库加入测试账号或录入 Key。

## Verification

- PASS：Kimi 提供商、账号、路由、网关、目录、余额、Wails、Core 代理、数据库九个 Go 包定向测试。包含旧账号迁移／目录身份、计费版本／凭据保护、非法值、互相独立的偏好、停用和能力回落、固定通道、六款目录与旧 ID 兼容。
- PASS：提供商 48 个地区／模型／协议／流式组合核对固定端点、逻辑到上游 ID 映射及真实客户端标识；订阅发现／余额和非法身份／参数在网络前拒绝。额外收紧模型映射断言后，`go test ./internal/provider/kimi` 再次通过。
- PASS：HTTP 集成覆盖四种计费／地区组合 × 三协议 × JSON／SSE，共 24 次成功请求，核对端点、Key、真实身份及请求历史计费归属；上游失败只发一次请求，无跨通道重试。
- PASS：Wails 假 Key 经真实 Windows 保护存储，元数据不含 Key／credential_ref；官方控制台入口固定，普通账号／任意 URL／缺少浏览器均不误报成功。
- PASS：6 文件／21 项前端定向测试，覆盖计费／地区互锁、关闭重开、失败保留与错误脱敏、明确订阅批量选择、卡片／列表及配额不查普通余额，以及 MiMo／StepFun／普通 Kimi 回归。
- PASS：`VITEST_MAX_WORKERS=4 task verify BIN_DIR=E:/go/new/wails3-v/OmniProxy/tmp/kimi-optimization-20261016/quality-bin` 退出码 0；格式、Go vet／staticcheck、ESLint、TypeScript、全部 Go 测试、84 文件／449 项前端测试及 Windows 生产构建通过。验证产物未替换运行中的开发版，既有工具探测、jsdom scrollTo 及体积提示不影响通过。
- PASS：正常桌面原生窗口 PID 44468 核对计费／地区双向保存、实际 SQLite 持久值和重开，深浅主题／960×600 内部滚动及边界显示正常；单个与批量订阅／海外表单可选，表单最小窗口可滚动，未提交任何用户数据。
- PASS：`git diff --check`，已检查生成绑定及本任务差异。

首轮前端测试夹具缺少 Dialog 外壳、使用了不匹配的控件角色且控制台 Mock 未返回 Promise；改用现有 AddAccountDialog、实际 Tab／按钮和异步 Mock 后通过，未修改产品来适配测试。原生检查先等待开发实例重建完成，并修正临时脚本的窗口辅助类型名称后通过。日志、截图、校验脚本及报告位于忽略目录 `tmp/kimi-optimization-20261016/`。

## Documentation

已同步 KIMI、PRD、QUOTA、架构概览、技术基线、ADR-012 和 CURRENT 已完成概览。`quota-model.ts` 因新增订阅类型约 212 行，已评估：保留现有集中类型／分类／汇总入口，避免为此次单一分类做无关模块拆分。

## Deviations

按用户补充确认新增 Kimi Code 订阅 Key。官方协议核查后进一步保留原生订阅 ID、不同型号及上下文权限边界，未扩大到 OAuth 或未经核实的订阅额度接口。

## Remaining Risks

没有真实订阅 Key，未发送真实付费模型请求；真实套餐权限／长输入及实际第三方客户端重载仍待用户账号验证。订阅官方剩余保持未知；K3 档位未知时目录保守为 256K，外部客户端可能需要重载模型目录。

## Git

`4337082d338cf0995b832340b5b77b36bc489497 feat(kimi): 接入 Code 订阅 Key 与默认计费地区顺序`

本任务实现、生成绑定、测试及文档已经审查并单独提交，UTF-8 无 BOM／LF 检查通过；五份既有未跟踪任务记录保持原样。任务归档及 CURRENT 以单独文档提交保存。

## Result

Kimi 国内／海外普通 API 与 Code 订阅 Key、默认计费／地区顺序及模型去重已实现，旧配置兼容；完整质量门及原生界面验收通过，开发版保持运行。

## Completed

2026-10-16（Asia/Singapore）
