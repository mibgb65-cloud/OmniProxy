# TASK-20260928-008: 账号页面与 API Key 账号管理

## Status

Completed

## Created

2026-09-28 16:33

## Request

为目前已有的提供商做出账号页面；用户后续会自行调整页面 UI。

## Goal

账号页面展示已配置的内置及自定义提供商，能为其创建、查看、启停和删除 API Key 账号。密钥仅保存在现有 Windows 凭证存储中，不在列表或错误中回显。

## Scope

- 复用现有提供商目录、SQLite 账号表与 DPAPI 凭证存储。
- 完成账号应用服务、Wails 绑定和基础页面交互。
- 提供必要测试和文档更新。

## Out of Scope

- OpenAI 订阅登录和 Token Plan 授权流程。
- 上游 API Key 有效性测试、网关调用与账号调度。

## Existing Behavior

账号页是占位页面；SQLite 已有 `accounts` 表，凭证存储尚未由账号服务使用。

## Expected Behavior

用户能在账号页按提供商管理多个 API Key 账号；密钥只在创建时输入，账号列表仅显示非敏感元数据。

---

# Analysis

## Related Modules

- `internal/application/account/`
- `internal/infrastructure/credential/`
- `internal/transport/wails/`
- `frontend/src/features/accounts/`
- `frontend/src/services/omniproxy.ts`

## Risks

文件凭证存储与 SQLite 不是同一事务；创建失败需清理刚写入的密文，删除时应保证不会留下可用账号却丢失密钥。

---

# Implementation Plan

1. 实现受校验的 API Key 账号创建、列表、启停与删除；安全处理密文和元数据的写入次序。
2. 通过 Wails 绑定接入账号页，用现有提供商配置驱动选择列表与分组。
3. 覆盖凭证不泄露、输入边界和账号生命周期，运行 `task verify` 并目视检查页面。

---

# Progress

- 已完成账号应用服务、Wails 绑定与按提供商分组的基础页面。
- 新增 API Key 账号、启停、删除和凭证清理测试，涵盖真实 Windows DPAPI 存储。

---

# Implementation Result

- 新增 `internal/application/account`，对账号名称、提供商 ID 与 API Key 做边界校验；支持 API Key 账号的创建、列表、启停及删除。
- API Key 存入已有的 Windows DPAPI 凭证存储，SQLite 仅存 `credential_ref`。Wails 列表和操作返回值只包含账号元数据。
- 将账号页占位内容替换为九家内置提供商和自定义提供商的账号卡片，支持录入多个账号、启停与删除。
- 复用现有提供商图标和滑块样式，加入中英文文案；OpenAI 订阅登录仍明确显示为待接入。

---

# Verification

- `task verify` 通过：格式、Lint、类型检查、Go 测试、前端 16 个测试文件共 71 个测试及生产构建均通过。
- 账号应用服务测试覆盖生命周期、输入校验、数据库写入失败后的密文清理、删除清理失败；Windows 绑定测试覆盖真实 DPAPI 存取。
- 前端测试覆盖十个提供商卡片（九家内置及一家自定义）、新增、启停、删除及 API Key 不回显。
- 已在实际 Wails 开发窗口目视检查账号页及新增账号表单。
- `git diff --check` 通过。

---

# Documentation

已更新 `docs/architecture/OVERVIEW.md`、`docs/architecture/TECH_STACK.md` 与 `docs/plans/CURRENT.md`。

---

# Deviations

None. SQLite 与凭证文件无法组成同一事务；创建写库失败时清理刚写入的密文，删除先移除账号元数据再清理密文。极端情况下密文删除失败会留下不可从页面访问的加密孤儿文件，同时向调用方报错；后续可加入专门的清理重试机制。

---

# Git

## Commit

`59be07ec3a5341b23572b6c2b3cd6460c5488b49` feat(account): 添加 API Key 账号管理页面

---

# Result

账号页面已可管理当前提供商的 API Key 账号；订阅登录、Token Plan、上游校验和网关使用留待后续任务。

---

# Completed

2026-09-28 16:49
