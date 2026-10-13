# TASK-20261012-004: 重装客户端后重新读取配置

> Retroactive record：已完成实现的补录；此前未在修改代码前创建 Task Record。

## Status

Completed

## Created / Completed

2026-10-12（补录日期）

## Request / Goal

“我卸载了 claudecode 后重装了……文件已被其他工具修改，请重新读取……但是我没找到重新读取的入口呢？”提供明确入口，读取当前版本后安全合并接入配置。

## Analysis / Root Cause

外部修改保护依据原受管哈希，但 UI 缺少重新读取入口；单纯刷新页面也不能明确授权接管新文件并更新恢复基准。

## Implementation Plan

追溯实际步骤：详情与弹窗增加入口 → 清除过期预览 → 明确接入当前版本 → 暂存新的加密恢复基准 → 同事务切换版本与备份 → 失败回退。

## Implementation Result

- 详情页和一键配置弹窗均增加“重新读取配置”。
- 重新读取获得当前版本、清除旧预览；应用继续核对文件和模型目录版本。
- 只有明确重新读取后才接入已变化文件，保留当前非网关设置。
- 新恢复基准先加密暂存，应用成功后与文件版本同事务提交。
- 迁移 013 的 `agent_backup_history` 保留旧加密备份引用；失败恢复当前文件并保留原备份。

## Files Changed

- `frontend/src/features/gateway/agent-config-dialog.tsx`、`use-agent-config-form.ts`
- `frontend/src/services/agents.ts`、配置详情及中英文词条
- `internal/application/agentconfig/rebase.go`、`apply.go` 及回归测试
- `migrations/013_agent_backup_history.sql`

## Verification

- 重装后新设置保留、旧备份归档、恢复新基准、事务失败回退等测试通过。
- 原生开发窗口确认重新读取入口可操作；该次原生验收只读取配置，没有应用用户真实配置。
- 完整 `task verify` 通过，后续最近一次完整验证为 64 个前端测试文件、340 项测试及全部 Go 检查、构建。

## Documentation

已更新 LOCAL_AGENTS 的重新读取与恢复基准说明；补录同步 CURRENT。

## Risks / Deviations

重新读取后再次发生外部修改仍会拒绝覆盖。原先遗漏事前记录，本文件为补录。

## Git

Not committed by Agent. 本次模块与已有未跟踪实现交织，未混入其他工作区变更提交。

## Result

用户有可见的重新读取入口，并可在保留新设置和历史备份的前提下重新接入。
