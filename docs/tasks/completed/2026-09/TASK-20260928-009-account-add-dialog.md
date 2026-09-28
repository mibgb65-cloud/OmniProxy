# TASK-20260928-009: 使用弹窗添加提供商账号

## Status

Completed

## Created

2026-09-28 17:30

## Request

点击提供商卡片的“添加账号”后，应弹出悬浮窗录入账号，而不是在页面正文中展开表单；同一提供商需要能添加多个 API Key。

## Goal

账号录入使用可通过键盘操作的弹窗；保存后关闭弹窗并刷新所属提供商的账号列表，同一家提供商可重复添加多个独立账号。

## Existing Behavior

表单作为页面正文的一张卡片渲染在提供商列表上方；后端支持多账号，但前端测试只覆盖单次新增。

## Expected Behavior

点击任意提供商的添加按钮打开对话框，取消或按 Esc 关闭；保存后卡片显示新账号，再次点击仍可添加第二个 API Key。

---

# Analysis

## Related Modules

- `frontend/src/features/accounts/account-form.tsx`
- `frontend/src/features/accounts/accounts-view.tsx`
- `frontend/src/features/accounts/accounts-view.test.tsx`

## Risks

弹窗需要正确处理焦点、Esc 和等待保存时的关闭行为；账号密钥不能因弹窗关闭而出现在页面其他位置。

---

# Implementation Plan

1. 使用项目已安装的 Radix Dialog 包裹账号表单，沿用现有页面风格。
2. 保留每个提供商自己的添加入口，提交成功关闭弹窗；等待保存时阻止意外关闭。
3. 扩展交互测试覆盖弹窗打开/取消/连续添加两个不同 API Key，运行 `task verify` 并在实际窗口检查。

---

# Progress

- 每张提供商卡片现以 Radix Dialog 管理自己的添加入口，表单不再占据正文空间。
- 已验证同一提供商连续添加两个独立 API Key 后，账号数量及列表正确更新。

---

# Implementation Result

- 添加按钮打开带遮罩的居中弹窗，表单保持现有名称、API Key 校验与错误提示。
- 保存成功关闭弹窗并刷新账号列表；再次打开时获得全新表单，可继续添加同一提供商的账号。
- 弹窗支持 Esc、取消按钮和右上角关闭按钮；保存期间阻止关闭，关闭后焦点返回原添加按钮。
- 新增中英文的多账号提示和关闭按钮标签。

---

# Verification

- `task verify` 通过：格式、Lint、类型检查、Go 测试、前端 16 个测试文件共 73 个测试和生产构建均通过。
- 随后补充并通过定向账号页测试（5 个）：连续添加两个不同 API Key、Esc 关闭与焦点返回、保存中禁止关闭。
- 已在实际 Wails 窗口目视检查弹窗居中、遮罩和表单布局。
- `git diff --check` 通过。

---

# Documentation

已更新 `docs/plans/CURRENT.md`；产品和架构文档无需修改。

---

# Deviations

None.

---

# Git

## Commit

`292f7ffe06e9f1e54b0b23319f388f79c57cf7aa` feat(account): 使用弹窗添加多个账号

---

# Result

账号添加已改为弹窗；同一提供商可以连续添加多个独立 API Key 账号。

---

# Completed

2026-09-28 18:59
