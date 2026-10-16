# TASK-20261016-009: Google Antigravity 订阅剩余额度

## Status

In Progress

## Created

2026-10-16 23:02

## Request

用户要求把 Google 订阅账号中“订阅剩余额度暂未接入”的能力也接入。

2026-10-17 补充：用户在额度卡片截图指出，此处还应显示账号有哪些模型可以使用。
在原额度任务内补齐模型目录展示，区分上游目录和网关已接入能力。

## Goal

通过已保存的独立 Google 授权读取真实订阅剩余额度与重置时间，账号详情和配额总览
共享缓存与刷新；缺失或过期数据明确显示，不根据套餐名或本机用量推算。
同一卡片展示当前账号真实模型目录，明确本机网关支持范围，不由额度组名称猜模型权限。

## Existing Behavior

Google 订阅登录、模型发现、Chat 推理与用量已接入，额度为占位文字；配额页把它作为
未支持资源。现有 Codex 额度 DTO 只适合固定窗口，不能直接套用 Google 不同额度组。

# Analysis

## Related Modules

`provider/antigravity`、`application/googlesubscription`、Wails 绑定、账号额度组件、
配额模型／总览以及仪表盘资源缓存。

## Risks

Antigravity 内部只读接口格式与实际账号权限需核实；不同额度组可能共享模型资源。
不能把模型级剩余当作周额度、不能合计共享桶、不能把缺失比例默认为零或满额。
令牌、身份和原始响应不能进入日志／前端／任务记录。

## Confirmed Protocol

已使用现有开发账号进行两次只读查询，未发送推理：
`retrieveUserQuotaSummary` 返回 HTTP 200，Gemini Models 与 Claude and GPT models
各有 weekly／5h 两个 bucket，包含官方 remainingFraction 与 resetTime；当前四桶为 1。
`fetchAvailableModels` 亦包含 quotaInfo，但其模型级值不能当作周窗口。
因此仅采用独立 summary 接口，按官方分组与窗口展示，不额外聚合模型值。

补充模型需求：模型目录使用独立只读查询与缓存，不将模型发现失败冒充额度查询失败。
展示完整官方目录及本机适配状态，不把尚未接入的第三方／多模态模型标为网关可调用。
实测原始字典有 22 项，官方 `agentModelSorts[].groups[].modelIds` 仅引用 6 个可选型号；
选用后者排除内部补全与历史别名。现有路由目录保留 14 项，保持旧配置兼容。

官方说明：https://antigravity.google/docs/cli/commands/usage/
参考协议：https://github.com/router-for-me/CLIProxyAPI/issues/3876
对照：https://github.com/router-for-me/Cli-Proxy-API-Management-Center/issues/369

## Implementation Choice

新增独立 Google 额度 DTO／Wails 方法，复用既有 DPAPI 授权与串行查询队列。
后端查询前后检查授权版本，前端按账号身份／状态缓存 60 秒、保留 10 分钟；
只手动刷新，不自动轮询。账号详情、配额页与仪表盘读取共享缓存；仪表盘仍默认不发请求。
缺失比例／时间保持未知，过期不会自动恢复满额；不套用 Codex 未开始计时判断。

# Implementation Plan

1. 核对固定参考源码与官方额度说明，使用已有开发账号做最小只读结构验证。
2. 确定真实接口及字段后补充分析并实现类型化额度 DTO、边界校验与授权版本保护。
3. 接入账号卡片／列表、配额总览共享缓存、状态与手动刷新，复用现有队列与样式。
4. 验证缺失／异常／耗尽／过期、取消与账号隔离，真实只读查询验收并运行 `task verify`。
5. 根据补充要求增加账号模型目录及适配状态，验证目录刷新、失败与身份隔离。
6. 同步文档，检查差异，提交并归档任务；保留已有其他任务记录。

# Implementation Result

- 新增官方额度 summary DTO、比例／重置／数量边界校验，保留未知与真实零值。
- 额度和模型共用 10 秒超时与授权版本保护，退出／重新登录拒绝在途旧结果；Wails
  仅发布白名单字段，错误脱敏，继续使用 DPAPI，不新增依赖或持久化迁移。
- 账号详情／列表与配额页展示官方分组、独立窗口百分比、重置时间和手动刷新；
  同身份刷新失败保留旧值，仪表盘默认只订阅缓存，需主动检查才读取上游。
- 卡片增加可选模型预览与独立浮层，列出具体名称／ID，分组区分现有 Chat 适配与
  上游待接入型号，支持独立刷新、键盘关闭和错误保留，不根据额度池推测模型权限。
- 复用现有串行查询队列，额度与模型独立缓存，按账号／身份／状态隔离；兼容原 Codex
  重置组件，Google 缺失重置时间显示待确认，不使用未开始计时推断。

# Verification

- `go test ./internal/provider/antigravity ./internal/application/googlesubscription -count=1`
  通过：summary 契约、比例／缺失／零值、异常响应、选择器过滤和在途授权变更保护。
- 5 个相关前端文件的 31 项测试通过，覆盖额度共享／身份隔离、刷新失败保留、可选模型
  独立失败、浮层 Portal、键盘 Escape 与焦点恢复。
- 最终 `task verify` 通过：gofmt／Prettier、go vet／staticcheck／ESLint、TypeScript、
  全部 Go 测试、88 个前端文件／469 项测试、生产前端与 Windows 桌面构建。
  已有 jsdom scrollTo、Vite 大包以及 Windows 缺少 uname／tail 提示不影响成功退出。
- 真实开发账号通过正式 `ReadQuota` 返回两组四窗口，剩余百分比和重置时间正常；
  `ReadAccountModels` 返回 6 个官方可选型号，3 个 Gemini Chat 已适配、3 个第三方待适配。
  只执行只读查询，未用推理激活窗口，未消耗付费积分验证。
- 正常桌面 `task dev` 已从当前分支重建启动，开发 profile 健康接口返回 ok；实际暗色
  窗口显示模型预览、已接入 3／上游可选 6、四个官方额度窗口与倒计时，浮层展示名称／
  ID 和两类适配状态，关闭后卡片布局正常。前期浅色额度亦由用户截图确认。
  原生截图仅保存在忽略目录，未提交身份信息；未更换正在运行的生产版程序。

# Documentation

已更新 Google 订阅说明、配额说明、PRD、架构及 CURRENT。

# Deviations

用户在额度完成后补充模型列表需求，纳入同一卡片闭环；使用官方选择器而非原始模型
字典，避免把内部条目当成可选模型。第三方模型推理适配不在本次目录展示范围内。

# Git

待提交。

# Result

额度占位已替换为真实官方分组窗口；用户补充的可选模型入口已完成，具体型号和网关
适配边界清晰。完整质量门、真实只读服务及开发窗口验证通过。
