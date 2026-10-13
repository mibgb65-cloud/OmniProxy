# TASK-20261012-005: Claude 桌面客户端检测与配置写入兼容

> Retroactive record：本记录是已发生工作的补录；实现前未创建 Task Record。

## Status

Completed

## Created / Completed

2026-10-12（补录日期）

## Request

- 安装结束却显示“尚未检测到可用程序”。
- 一键配置显示“操作失败，请检查客户端配置与网关状态”。
- 用户明确需要 Claude Code Desktop 的编程入口。

## Goal

正确识别已安装客户端，区分实际写入错误，提供清晰的 Code 编程入口。

## Analysis / Root Cause

- 本机商店新版包名为 `AnthropicPBC.Claude`，检测仅查询旧名 `Claude`。
- 工具视图中的配置目录显示加密属性并拒绝同目录重命名，导致当时后台 `agent: write_failed`；后续已确认该视图与真实客户端不同，详见 006 的澄清和实机验收。
- 前端未识别该错误代码，统一提示客户端或网关异常。
- 官方 Code 入口与 Chat 共用安装包，但安装卡片没有说明清楚入口关系。

## Implementation Plan

追溯实际步骤：精确包名兼容 → 多包健康状态选择 → 定位原生写入失败 → 加密目录写入兼容 → 稳定错误文案 → Code 入口与说明 → 验证。

## Implementation Result

- 精确兼容新旧包名，识别健康包；不扩大到任意通配软件。
- 加密目录发生对应重命名错误时，以独占句柄重新核对版本、保留加密与 ACL，写入失败恢复原字节。
- 明确展示写入、读取、占用和备份错误；原始诊断不进入 UI。
- 增加“打开 Claude Code”，使用固定 `claude://code/new`；官方说明指向 Code 桌面文档。
- 安装卡片明确 Code 与 Chat 共用安装包。

## Files Changed

- `internal/infrastructure/agentinstall/catalog.go`、`desktop.go`、`desktop_windows.go` 及测试
- `internal/application/agentconfig/replace_windows.go`、`replace_other.go`、写入与故障回退测试
- `internal/transport/wails/agentinstall.go`、启动入口测试
- `frontend/src/services/agents.ts`、安装卡片、安装 Hook、词条及回归测试

## Verification

- 原生开发窗口曾正确显示“已安装 · 2.31226.1.0”，旧检测错误消失。
- 加密目录中的独立合成配置完成首次应用、重复应用、恢复；确认加密保留，未写入真实用户配置或残留测试文件。
- 文件版本冲突、文件占用、部分写入回退和回退失败分类测试通过。
- `task verify` 最终通过：64 个前端测试文件、340 项测试，Go 测试、格式、lint、类型检查和生产构建全部通过。
- 一次完整检查因重复标题导致定位歧义失败；测试随后明确定位二级详情标题，重新执行完整检查通过。

## Documentation

已更新 LOCAL_AGENTS；补录同步 CURRENT。

## Risks / Deviations

- 后续 TASK-20261012-006 的正常用户上下文核对确认：本机先前诊断中的加密属性与重命名错误来自工具隔离视图，真实配置文件并未加密。因此不能把本记录的合成目录验证当作真实 Claude 配置已被写入的证据；完整接入结果以 006 的原生验收为准。
- 加密目录兼容分支提供进程内失败回退，不能保证断电或强制结束时的写入原子性。
- 本次只解决识别、写入和 Code 入口，尚未自动启用第三方模式；随后用户报告登录页，继续由 TASK-20261012-006 修复，不能将本次验证宣称为完整网关接入验收。
- 事前任务记录遗漏，现据实际结果补录。

## Git

Not committed by Agent. 已检查差异与 `git diff --check`；未将已有未提交模块和其他工作区修改混合提交。

## Result

客户端识别与加密目录写入问题已修复；自动启用本地网关的缺失步骤另行跟踪。
