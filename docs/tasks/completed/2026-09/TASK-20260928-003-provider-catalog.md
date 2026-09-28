# TASK-20260928-003: 九家提供商预设与配置管理

## Status

Completed

## Created

2026-09-28 11:23

## Request

添加九家提供商：OpenAI（API Key、订阅登录）、Anthropic、Google、DeepSeek、智谱、月之暗面、MiniMax、小米、阶跃星辰（后八家目前预设 API Key）。用户确认本次 OpenAI 订阅登录仅显示为待接入，先完成九家提供商配置。

## Goal

用户可在提供商页面看到九家预设，新增、查看、编辑、启停和删除提供商配置；配置持久化于现有 SQLite `providers` 表。同一家厂商可建立多条配置，以支持不同 Base URL。认证方式预设真实展示，未实现的订阅登录不可误操作。

## Scope

- 九家提供商静态目录及认证能力信息。
- 提供商配置的应用服务、Wails 绑定与前端页面。
- 名称、Base URL、厂商种类和删除边界的校验及回归测试。
- 相关产品、架构、任务文档更新。

## Out of Scope

- 厂商 API 请求适配器、健康检测、模型列表和网关调用。
- 账号/API Key 录入及 OpenAI 订阅登录授权流程。
- 自动向数据库插入九条未配置的提供商实例。

## Existing Behavior

`providers` 表已有 `id/kind/name/base_url/enabled` 等字段，但无应用服务和 Wails 方法；前端提供商页是占位内容。

## Expected Behavior

页面始终展示九家可配置的预设；新建配置写入数据库并能在重启后读取。允许同厂商多配置；已有账号的配置不可直接删除，以避免遗留凭证引用。OpenAI 订阅登录标注待接入。

---

# Analysis

## Related Modules

- `migrations/001_initial.sql`（只读取，不修改已应用迁移）
- `internal/application/provider`（新建）
- `internal/transport/wails`
- `frontend/src/app/routes/providers.tsx`、`frontend/src/services/omniproxy.ts`
- `frontend/src/i18n/locales/{zh,en}.ts`

## Dependencies

使用既有 SQLite、Wails 绑定、React Query 和表单组件；不新增外部依赖。

## Risks

- 配置管理不能暗示网关已经能调用这九家；页面应清楚说明当前仅管理配置。
- 删除配置可能触发数据库外键级联并遗留密文，须先检查是否关联账号。
- Base URL 属于外部输入，必须校验长度、协议、主机和禁止凭证型 URL。

---

# Implementation Plan

1. 建立九家提供商目录、配置数据模型与 SQLite CRUD 应用服务，并完成校验与删除保护。
2. 接入 Wails 绑定和前端提供商页面：预设目录、配置列表、新增/编辑/启停/删除与中英文文案。
3. 增加服务和页面测试，运行 `task verify`，检查差异并提交、归档任务记录。

---

# Progress

- 九家预设、SQLite 配置服务和提供商页面已经接通，定向测试通过。
- 实际窗口发现提供商列表加载失败；根因是 `main.go` 把含有 `migrations/` 子目录的 embed FS 直接传给只读取根目录 SQL 的迁移器，导致真实数据库从未创建 `providers`/`accounts` 表。已将迁移输入改为嵌入的 SQL 子目录，并增加实际嵌入迁移测试。
- 修复后在真实 Wails 窗口验证九家预设和已配置列表；通过页面创建一条不含凭证的开发测试配置，确认 SQLite 持久化，并清理该测试记录（当前开发数据库提供商记录数为 0）。
- `task verify` 全部通过。
- 开发环境已恢复：Wails 已连接 Vite，9245/3891 端口监听正常，首页 HTTP 返回 200。

---

# Implementation Result

## Changes

- 建立九家代码定义的提供商目录。九家均展示 API Key 方式；OpenAI 另展示“订阅登录 · 待接入”。月之暗面沿用既有 `kimi` kind，小米沿用 `mimo` kind。
- 新增 `internal/application/provider` 配置服务，使用现有 SQLite 表完成列表、新增、编辑、启停与删除；允许同厂商多条配置。名称、ID、Base URL 做长度、格式和协议校验；远程自定义地址要求 HTTPS，本机 HTTP 可用。
- 删除操作通过原子 SQL 条件拒绝有关联账号的配置，防止外键级联后留下凭证文件。
- Wails 绑定和前端提供商页面接入真实数据；新增配置表单、状态操作、删除确认及中英文文案。账号凭证和网关能力明确标为后续接入。
- 修复主程序嵌入迁移目录的路径问题，使实际应用启动时能创建 `providers`/`accounts` 表。

## Files Changed

- `internal/application/provider/{catalog.go,validation.go,service.go,service_test.go}`
- `internal/transport/wails/providers.go`、`main.go`、`main_test.go`
- `frontend/src/features/providers/*`、`frontend/src/app/routes/providers.tsx`
- `frontend/src/services/omniproxy.ts`、`frontend/src/i18n/locales/{zh,en}.ts`
- `frontend/bindings/omniproxy/internal/application/provider/*`、生成的 Wails AppService 绑定
- `docs/product/PRD.md`、`docs/architecture/{OVERVIEW,TECH_STACK}.md`、`docs/development/CODING_RULES.md`、`docs/plans/CURRENT.md`、本任务记录

## Technical Notes

预设目录与用户保存的配置是两种数据：预设代码定义且始终可见，配置以 128 位随机 ID 写入 `providers` 表。Base URL 留空表示后续适配器的厂商默认值。本次没有网络请求适配器，也没有账号凭证写入。

---

# Verification

## Tests

- [x] 九家目录、OpenAI 待接入认证方式、同厂商多配置及配置 CRUD。
- [x] 非法输入和已有账号时的删除保护。
- [x] 实际嵌入的数据库迁移回归测试。
- [x] 页面新增/编辑/停用/删除及错误提示测试。
- [x] 真实 Wails 窗口显示和开发数据库写入检查。

## Quality Checks

- [x] Format
- [x] Lint
- [x] Type Check
- [x] Test
- [x] Build
- [x] Verify

## Verification Result

`task verify` PASS：Go 测试通过；前端 14 个测试文件、56 个测试通过；Windows 生产构建成功。最后的端口边界校验及翻译清理后，定向 Go 测试、前端类型检查、格式检查和页面测试再次通过。实际开发窗口显示九家预设且无加载错误，新增配置写入 SQLite；测试记录已清理。`git diff --check` PASS。

---

# Documentation

已同步 PRD 中的厂商范围、架构中的首个应用服务与提供商页面状态、技术栈和代码目录说明、CURRENT 看板。没有新增 ADR：该实现遵循既有应用服务和 Wails 边界。

---

# Deviations

原计划只接入既有 `providers` 表；实际窗口暴露主程序未正确定位嵌入迁移文件，增加了迁移入口修复和回归测试。没有修改已应用的 SQL 迁移文件。

---

# Git

## Commit

`70d6840a08837d9351d6e0c232818e5bd500314b feat(provider): 添加九家提供商预设与配置管理`

---

# Result

九家提供商预设和可持久化配置已可在桌面应用中使用；OpenAI 订阅登录保持明确的待接入状态。

---

# Completed

2026-09-28 11:53
