# TASK-20260928-002: Windows 凭证存储基础设施

## Status

Completed

## Created

2026-09-28 10:45

## Request

实现凭证存储。需要覆盖厂商 API Key（如 DeepSeek）、以 API Key 和专属地址使用的 Token Plan（如 MiMo），以及订阅登录后可能产生的较大、需轮换的凭证包（如 ChatGPT 订阅账号）。

## Goal

提供与厂商及方案无关的安全字节存储：写入返回不透明 `credential_ref`，可读取和删除；数据库以后只保存引用。生产、开发、测试环境严格隔离，凭证明文不落盘、不进入日志。

## Scope

- Windows 当前用户范围的 DPAPI 加密和本地密文文件存储。
- 固定格式的随机引用、大小限制、路径和损坏数据校验、原子性尽量充分的文件写入。
- Windows 真实加解密与文件生命周期测试；同步架构文档和设计决策。

## Out of Scope

- 获取厂商登录凭证、OAuth 流程、订阅权益判断、网关适配器、账号页面。
- 修改现有账号数据库结构或将订阅视为 API 平台额度。

## Existing Behavior

`accounts.credential_ref` 字段已存在，但 `internal/infrastructure/credential` 尚未实现；当前没有写入或读取凭证的路径。

## Expected Behavior

调用方可安全存取大小受限的不透明凭证包；磁盘只有 DPAPI 密文，引用无法转成任意文件路径，不同运行环境之间不能互读。

---

# Analysis

## Related Modules

- `internal/platform/profile`
- `internal/infrastructure/credential`（新建）
- `docs/architecture/OVERVIEW.md`、`docs/architecture/TECH_STACK.md`
- `docs/decisions/`、`docs/plans/CURRENT.md`

## Options

1. Windows Credential Manager：系统直接管理凭证，但单条 generic 凭证的 `CredentialBlob` 有 2560 字节上限，不适合尚未确定长度的订阅登录凭证包。
2. 当前用户范围 DPAPI + 应用数据目录下的密文文件：可控制大小上限，并随现有数据目录迁移；需自己管理文件生命周期和随机引用。

采用方案 2。存储层只处理字节，不把 API Key、Token Plan、订阅方案编码进加密格式；方案类型和认证方式由后续账号/提供商层定义。

## Risks

- DPAPI 密文一般依赖同一 Windows 用户和机器；跨机复制不构成可用备份。
- 调用方写入凭证后、保存数据库引用前若崩溃，可能遗留不可引用的密文文件；后续账号服务需在失败时回滚，清理机制留待真实账号流程。
- 现有文档将 DPAPI 称为 OS credential store；本任务需准确说明其实际落盘方式。

---

# Implementation Plan

1. 实现与平台无关的引用、大小、文件生命周期逻辑，以及 Windows DPAPI 适配；非 Windows 返回明确的不支持错误。
2. 增加真实 Windows DPAPI 回归测试，覆盖大于 Credential Manager 单条限制的凭证、跨环境隔离、损坏密文、异常引用与删除。
3. 更新 ADR 和架构文档，运行 `task verify`，复查差异并提交。

---

# Progress

- 已确认 MiMo Token Plan 使用专属 API Key 与 Base URL，存储层无需新增 Token Plan 专用密文格式。
- 已实现按运行环境隔离的随机引用文件存储和 Windows 当前用户范围 DPAPI 适配；`golang.org/x/sys/windows` 从现有间接依赖转为直接依赖。
- Windows 真实 DPAPI 测试已覆盖短密钥、超过 Credential Manager 单条上限的凭证、1 MiB 大小边界、数据目录迁移、篡改、路径格式与环境隔离。
- 新增 ADR-002，澄清当前架构中的 DPAPI 保护与密文文件落盘方式。
- `task verify` 全部通过。
- 开发环境已恢复：Wails 连接 Vite，9245/3891 端口正常监听，首页 HTTP 返回 200。

---

# Implementation Result

## Changes

- 新增 `credential.Store` 的 `Put/Get/Delete` 接口。`Put` 为每份凭证生成 128 位随机引用，按运行环境隔离保存 DPAPI 密文；`Get/Delete` 仅接受精确格式的引用。
- 对明文设定 1 MiB 上限，对密文读取设定有界上限；拒绝空值、过大文件、异常引用和损坏密文。写入使用独占创建、同步并在失败时清理部分文件。
- Windows 调用 `golang.org/x/sys/windows` 的 DPAPI 封装，使用用户范围和禁止 UI 的模式，并在释放原生输出缓冲区前清零。其他平台提供明确的不支持结果。
- 将既有的 `golang.org/x/sys` 从间接依赖提升为直接依赖；不新增外部模块或数据库迁移。

## Files Changed

- `internal/infrastructure/credential/{store.go,reference.go,dpapi_windows.go,dpapi_unsupported.go,store_windows_test.go}`
- `go.mod`
- `docs/decisions/ADR-002-dpapi-credential-files.md`、`docs/decisions/README.md`
- `docs/product/PRD.md`、`docs/architecture/OVERVIEW.md`、`docs/architecture/TECH_STACK.md`
- `docs/plans/CURRENT.md`、本任务记录

## Technical Notes

凭证文件位于 `<DataDir>/credentials/<profile>/cred-v1-<random>.bin`。调用方负责在保存数据库引用失败时调用 `Delete` 回滚，且必须管理 `Get` 返回的明文字节生命周期。存储层不暴露给 Wails 绑定或 HTTP API。

---

# Verification

## Tests

- [x] Windows 真实 DPAPI 的短 API Key、较大凭证和 1 MiB 边界往返测试。
- [x] 生产/开发环境隔离、数据目录迁移、删除与缺失、非法输入和密文篡改测试。

## Quality Checks

- [x] Format
- [x] Lint
- [x] Type Check
- [x] Test
- [x] Build
- [x] Verify

## Verification Result

`task verify` PASS：Go 测试全通过；前端 13 个测试文件、52 个测试通过；Windows 生产构建成功。文件拆分后，凭证包定向测试及 `go vet` 再次通过。凭证包测试使用真实 Windows DPAPI 和临时目录，密文文件不含测试明文；没有写入真实凭证或用户数据库。`git diff --check` PASS。

---

# Documentation

已新增 ADR-002，并同步 PRD、架构概览、技术栈与当前任务看板。已应用的 `migrations/001_initial.sql` 保持不变；其中关于 DPAPI 的旧注释由 ADR-002 澄清实际落盘方式。

---

# Deviations

None.

---

# Git

## Commit

`e5c9ee3759d52326b0f73ce670038251aa474b6c feat(credential): 增加 Windows DPAPI 凭证存储`

---

# Result

已具备面向三类账号所需的通用、本地加密凭证存储基础设施。后续账号服务负责凭证内容格式、认证/方案元数据及引用与数据库记录的一致性。

---

# Completed

2026-09-28 11:00
