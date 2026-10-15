# TASK-20261015-006: Kimi 普通 API 余额显示

## Status

In Progress

## Created

2026-10-15

## Request

“kimi的余额能够接入显示吗？”

## Goal

接入当前 Kimi 普通 API Key 账号的官方余额，在账号卡片／列表、配额页和仪表盘资源概况中复用已有查询能力。

## Scope

- 按账号国内／海外选项查询固定官方余额接口，显示可用、代金券及现金余额。
- 保留官方金额精度，国内 CNY、海外 USD；现金允许欠费负数，余额不足按可用金额判断。
- 复用受保护凭据回调、超时、缓存、错误提示和并发队列。
- 合同、安全边界、配置变更隔离及页面显示测试，运行 `task verify`。

## Out of Scope

Kimi Code 订阅额度、充值、换汇、真实账号推理调用或大范围重构。

## Existing Behavior

Kimi 普通 API 账号已有地区选项和原生转发，但余额显示待接入。通用余额服务仅支持 DeepSeek 和 StepFun，配额资源清单也未包含 Kimi。

## Expected Behavior

账号详情和配额页显示对应地区官方余额，手动刷新期间保留旧值，错误不伪造零余额；切换地区／Key 后不返回旧配置结果。仪表盘继续仅由用户触发官方查询。

# Analysis

## Related Modules

`internal/provider/kimi`、`internal/application/balance`、前端账号／配额／仪表盘资源和生成的 Wails DTO。

## Dependencies

无新增依赖或数据库迁移。

## Official Contract

- [国内余额](https://platform.kimi.com/docs/api/balance)：`GET https://api.moonshot.cn/v1/users/me/balance`，CNY。
- [海外余额](https://platform.kimi.ai/docs/api/balance)：`GET https://api.moonshot.ai/v1/users/me/balance`，USD。
- Bearer API Key；成功 `code=0`、`status=true`，`data` 中三项 JSON number。可用余额包含现金和代金券，现金欠费时可用余额等于代金券，不能简单求和。代金券不可为负，可用余额小于等于零无法推理。
- 核对日期：2026-10-15。生成文档的通用示例 `code=123` 与字段说明冲突，采用明确的成功语义 `code=0`。

## Risks

必须避免凭据跨地区、误用 DeepSeek 的充值余额语义、浮点精度丢失、把未完成查询当零余额及配置更新后的过期响应。

# Implementation Plan

1. 增加 Kimi 余额传输与严格解析，使用独立 DTO 保留官方字段语义。
2. 余额服务按提供商分派，受控使用对应账号版本凭据并复核版本。
3. 接入现有展示和资源清单，增加中英文文案与回归测试，重新生成绑定。
4. 同步产品／架构／状态文档，运行针对性测试和完整质量门，审阅后提交。

# Progress

- 已核对现有代码及国内／海外官方余额协议。
- 已实现后端适配、账号版本校验、独立公开 DTO、前端卡片／列表与配额展示，并重新生成绑定。
- 针对性测试中修正了新增测试对刷新按钮可访问名称和重复状态文案的选择断言；未改变产品行为。
- 完整 `task verify` 已通过，含全部 Go 测试、76 个前端测试文件／415 项测试及 Windows 生产构建。

# Implementation Result

## Changes

- Kimi 官方余额按账号地区固定发往 `.cn`／`.ai`，禁止重定向、跨地区回退和自定义地址；校验 Key、10 秒超时、64 KiB 响应、成功状态及金额类型／长度／精度，错误不暴露上游正文。
- 可用／代金券／现金使用独立公开 DTO，保留负现金与十进制精度；可用余额精确大于零时显示充足，不重算分项。无依赖或迁移新增。
- 查询前绑定账号版本的受保护 Key 回调，查询后复核元数据与 Kimi 选项；换 Key／地区的在途旧结果不会返回。停用账号仍可只读查询。
- 账号卡片／列表与配额页共享 Kimi 展示、现有缓存及刷新队列；仪表盘通过共用资源分类自动纳入，仍只由用户主动发起官方检查。完善中英文金额标签及通用服务不可用文案。

## Files Changed

- `internal/provider/kimi/balance*.go`
- `internal/application/balance/{service,kimi,kimi_test}.go`
- `frontend/src/features/accounts/kimi-balance*.tsx`、`account-balance.tsx`、`provider-accounts-view*.tsx`
- `frontend/src/features/quota/{quota-model.ts,quota-resource.tsx,quota-view.test.tsx}`
- `frontend/src/services/kimi.ts`、中英文余额词典与生成绑定。

## Technical Notes

`AccountBalance` 保持原方法签名，仅新增可选 `kimi` 字段，旧 DeepSeek／StepFun 语义保持兼容。复用版本化缓存，无自动轮询或自动重试。现有页面测试文件已超过 200 行，本次在原有集成环境补充少量用户可见断言；新的组件及测试文件均低于 200 行，未进行无关拆分。

# Verification

## Tests

- `go test ./internal/provider/kimi ./internal/application/balance`：PASS。
- 传输契约测试覆盖两地区 URL／Bearer、认证头清理、查询期限、精度／欠费现金、必填字段和业务失败状态、金额格式／长度／精度、负代金券、响应上限、HTTP 错误、网络／读取异常、重定向和非法 Key／地区。
- 应用服务测试覆盖停用账号可读、精确正数／零／负数状态、受保护字节清理、安全错误、不跨地区重试，以及在途替换 Key／切换地区拒绝旧结果。
- 前端新增 7 项测试，覆盖卡片／列表的现金／代金券语义、精度、刷新期间／失败保留快照、新版本缓存隔离、真实零与首次失败、中英文标签、配额页耗尽统计与未确认平台不查询。完整 415 项测试通过。
- 真实 API Key 查询、原生桌面视觉验收：未执行，仍待实际账号验收；未声称已验证真实官方返回或桌面显示。

## Quality Checks

- [x] Format（gofmt／Prettier）
- [x] Lint（go vet／staticcheck／ESLint）
- [x] Type Check
- [x] Test
- [x] Build（Windows production）
- [x] Verify
- [x] `git diff --check`

## Verification Result

`task verify`：PASS（exit 0）。76 个前端测试文件／415 项测试通过，Go 全量测试通过，Windows 生产构建成功。输出包含本机既有 uname／tail 探测、jsdom scrollTo 和 Vite 大体积 chunk 非阻断提示；未为这些提示修改无关模块。未新增依赖或改变锁文件。

# Documentation

已同步 KIMI、BALANCE、PRD、QUOTA、架构概览及 CURRENT；技术栈、构建流程与 ADR 无需修改。

# Deviations

None.

# Git

待提交。

# Result

已接入 Kimi 普通 API 账号的国内／海外官方余额及账号／配额展示，并纳入仪表盘资源分类；完整质量门通过。真实 Key 和原生桌面验收仍待后续，Kimi Code 订阅额度不属于本次实现。
