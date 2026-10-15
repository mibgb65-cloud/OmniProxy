# TASK-20261015-007: 小米 MiMo Token Plan 多区域接入

## Status

Completed

## Created

2026-10-15（Asia/Singapore）

## Request

“Token Plan 也接入一下吧，我记得tokenplan有不同的区域吧，调用的接口也不一样”

## Goal

在现有小米 MiMo 普通 API 基础上支持 Token Plan 个人／团队专属 Key 和中国／新加坡／欧洲集群，并保持账号、目录、路由与调用计费通道一致。

## Background

当前 MiMo 只支持普通按量入口，安全白名单明确拒绝 Token Plan 域名。已有 StepFun／MiniMax 账号选项及版本化目录、通道路由模式可复用。余额与 Token Plan Credits 是不同资源，不能相加或根据本机 Token 伪造官方剩余。

## Scope

- 按账号保存计费类型与 Token Plan 集群，新增／批量／编辑及卡片／列表显示。
- 三区域固定官方端点，个人 `tp-`／团队 `ttp-` Key；普通按量行为保持兼容。
- 网关原生 Chat／Responses／Messages、模型目录、路由与客户端菜单按通道隔离，版本变化使旧目录及在途配置失效。
- Token Plan 只公布官方确认的现役文本模型；本机用量沿用现有观察器，官方套餐剩余未经验证时保持未知并提供平台入口。
- 回归测试、文档、`task verify`、Git 提交和任务归档。

## Out of Scope

自动采集浏览器 Cookie、额外网页登录、现金余额、充值、套餐购买、未确认的 Key 额度接口，以及无关重构。

## Existing Behavior

普通按量 Key／地址与三种原生协议已实现；Token Plan、账号级 MiMo 选项及官方剩余额度尚未接入。

## Expected Behavior

用户选择按量或 Token Plan；Token Plan 选择中国、新加坡、欧洲集群，并使用对应固定地址与专属 Key。区域或 Key 不匹配时明确失败，不跨地区或切回按量消费。套餐额度未知时如实显示。

# Analysis

## Official Sources

- [团队版及三集群地址](https://mimo.mi.com/docs/zh-CN/tokenplan/Token%20Plan/team)
- [普通／Token Plan Key 和协议入口](https://mimo.mi.com/docs/zh-CN/quick-start/summary/first-api-call)
- [官方 OpenCode 接入](https://github.com/XiaomiMiMo/awesome-mimo-agent/blob/main/docs/opencode.md)
- [Key 用量接口需求](https://github.com/XiaomiMiMo/MiMo-Code/issues/2495)：功能请求不能当已发布接口。
- 核对日期：2026-10-15。官方当前列出 cn／sgp／ams，OpenAI 基址 `/v1`，Anthropic 基址 `/anthropic`；团队版按席位计量。

## Related Modules

MiMo provider、账号应用服务和迁移、模型目录、路由、网关、请求归属、前端账号和配额、Wails 生成绑定。

## Dependencies

无新增依赖；新增迁移 025，扩展已有账号配置版本视图，不改变请求统计表结构。

## Risks

地区与计费池混用、同模型路由歧义、旧账号迁移兼容、版本并发、凭证安全、套餐不支持高速版、把本机用量误认为官方额度。

# Implementation Plan

1. 阅读现有账号选项／目录版本／地区路由实现，确认最小兼容方案。
2. 增加 MiMo 账号选项、迁移与受控凭据校验，扩展固定端点及套餐模型能力。
3. 接通目录、路由、网关及用量归属，保留原生协议能力。
4. 增加账号选择／标识及未知套餐资源显示，补充核心与界面回归测试。
5. 同步产品和架构文档、生成绑定、运行完整质量门、审阅并提交。

# Progress

- 已创建任务记录，已确认官方三个 Token Plan 集群地址。
- 采用账号级 `billingMode=payg/token_plan`，按量地区为空、套餐地区 cn／sgp／ams；保留现有官方／回环按量入口，套餐固定主机。旧官方及回环按量账号迁移为 payg，未知地址保持需确认；通道别名使用 `mimo-api/模型` 与 `mimo-区域-plan/模型`，多通道裸名要求固定账号。
- 已实现三集群原生三协议、个人／团队 Key 校验、账号选项及迁移、目录版本隔离、通道别名、当次请求计费／区域记录、前端单个／批量／编辑／卡片／列表和未知套餐资源卡片。未引入额外网页登录流程。
- 针对性 Go 测试已通过；前端 8 个文件／54 项测试通过，包含新增套餐表单和资源测试。初次测试暴露原有手写数据库夹具未加载新配置表，已补入迁移；原普通 MiMo 测试增加公开选项及新版目录版本，并保留原调用断言。安全白名单测试更新为拒绝未知套餐集群，不删除既有验证。
- 审阅时补齐前端 `ambiguous_mimo_pool` 本地错误映射；补充后该文件格式检查、ESLint、类型检查和全量测试通过。完整 `task verify` 已通过，78 个前端文件／428 项测试及 Windows 生产构建成功。

# Implementation Result

## Changes

- 账号级 MiMo 配置支持普通按量和 Token Plan（cn／sgp／ams）；个人／团队专属 Key 均可用，普通与套餐 Key 不能混用。单个／批量／编辑均有前后端校验，批量事务失败回滚元数据并清理凭据，编辑不回填完整 Key。
- 原生客户端精确允许三个 HTTPS 套餐主机，沿用禁止重定向、输入边界、原生思考／工具／JSON／SSE 和错误处理。套餐不采纳提供商按量默认地址，不跨集群／计费池回退。
- 套餐只公布官方确认的 Pro／Flash，普通高速版继续保留。发现模型成功后以账号快照为准，未同步为明确 preset；切换通道与 Key 后隔离旧目录，发现期间配置改变拒绝写回。
- 通道别名与多池裸名歧义、手工固定规则优先、客户端模型目录沿用既有路由系统；网关通过 `UseAPIKeyAt` 保护选择快照，历史保存当次地域与计费方式。
- 账号与配额显示地域标识及未知官方 Credits，提供后端固定官方套餐页入口；不新增额度 RPC、不伪造零或 100%，不把本机累计用量当官方剩余。仪表盘将其归入待确认资源，不自动查询。
- 迁移 025 保存公开选项和版本；旧官方／已识别回环按量地址兼容迁移，未知自定义地址要求确认，不猜 Key。其他提供商目录版本保持。

## Files Changed

MiMo provider、账号／目录／路由／网关、迁移 025、Wails 官方入口与生成绑定、前端账号表单／资源／配额、中文和英文词典及对应测试；详见本次 Git diff。

## Technical Notes

无新增依赖。复用已存在的账号选项／目录版本／通道隔离实现方式，没有新的核心架构决策，故不新增 ADR。部分既有账号／网关源文件和集成测试已接近或超过 200 行，本次仅添加当前功能接入点，未混入无关拆分；新文件均控制在 200 行内。普通账号原始模型名在单通道下继续工作，新增显式通道名；多通道裸名现在明确要求选择，防止误选地区或计费池。

# Verification

## Tests

- `go test ./internal/provider/mimo ./internal/application/account ./internal/application/modelcatalog ./internal/application/routing ./internal/application/gateway ./internal/infrastructure/database`：PASS。
- Provider 合同测试：cn／sgp／ams × 个人／团队 Key × Chat／Responses／Messages × JSON／SSE 共 36 组合；验证精确主机／路径、认证、原生正文／输出和请求后头部清理。覆盖未知／相似域名、HTTPS、安全参数、普通／套餐 Key 混用、套餐高速版拒绝、模型发现过滤及失败不回退。
- 账号／迁移测试：字段校验、版本冲突、区域修改与旧目录清理、保留 Key、按量／套餐不能只改配置复用不匹配 Key、混合类型批量预校验、事务失败凭证与元数据回滚、旧按量迁移和其他提供商版本保持。
- 目录／路由测试：preset 不伪造查询时间、账号发现优先、查询失败保留旧结果、在途区域修改拒绝写回、三集群别名、裸名歧义、普通高速版权益、手工规则优先。网关端到端通过伪上游执行三个集群 × 三协议 × JSON／SSE 共 18 次请求，验证原生 usage、当次区域和计费身份及凭证清理；路由选中后配置改变拒绝转发。
- Windows Wails 绑定测试：受保护套餐凭证不出现在公开账号 DTO，官方入口只接受已配置的套餐账号，普通账号／URL 输入／缺失浏览器均明确失败。
- 前端针对性 8 文件／54 项测试 PASS；新增 13 项测试覆盖单个／批量／编辑、三区域对应地址、个人／团队 Key、混合 Key 拒绝、版本保持、未确认账号、卡片／列表、未知套餐资源／概况和安全打开错误。
- 真实套餐 Key、真实区域权限与长窗口、原生桌面视觉验收：未执行；本次仅宣称代码与伪上游合同通过。

## Quality Checks

- [x] Format（gofmt／Prettier）
- [x] Lint（go vet／staticcheck／ESLint）
- [x] Type Check
- [x] Test（Go 全量，前端 78 文件／428 项）
- [x] Build（Windows production）
- [x] Verify
- [x] `git diff --check`

## Verification Result

`task verify`：PASS，exit 0。前端生产构建和 Windows 可执行文件生成成功，生成绑定包含 MiMo 公开选项和 `OpenMiMoConsole`。原有 uname／tail 探测、jsdom scrollTo 和 Vite chunk 大小提示不阻断；未修改依赖或锁文件。

# Documentation

已同步 MIMO、PRD、QUOTA、ROUTING、架构概览、技术栈当前实现及 CURRENT；编码／构建流程没有变化。

# Deviations

None.

# Git

## Commit

`f5b059166bd23f369cd9ca4e1029d89e3e65b4ed feat(mimo): 接入 Token Plan 三地区账号与原生调用`

随后提交任务归档和 CURRENT 完成状态。原有四份无关未跟踪任务文档保持原样，未纳入提交。

# Result

已完成 MiMo Token Plan 个人／团队 Key 和中国／新加坡／欧洲集群接入，支持账号配置、原生调用、模型目录、通道路由及当次用量归属；完整质量门通过。官方套餐剩余尚无已确认的 Key 接口，显示未知并提供固定平台入口；真实账号与原生桌面验收仍待后续。

# Completed

2026-10-15 17:11（Asia/Singapore）
