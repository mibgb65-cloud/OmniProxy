# TASK-20261014-008: 修复 Claude Desktop 一键配置写入位置

## Status

In Progress

## Created

2026-10-14 11:24（Asia/Singapore）

## Request

用户反馈 Claude Desktop 一键接入未成功，实际仍使用旧项目的配置；旧项目 `E:\go\new\OmniProxy` 可以成功。排查是否读取配置位置错误，或开发版未安装造成。

## Goal

确认客户端实际配置入口，修复错误定位，使一键配置写入与 Claude 使用的配置位置一致，保留原配置及恢复保护。

## Scope

- 对比新旧项目路径及激活实现，只读核对本机客户端清单和脱敏配置状态。
- 修复 Claude 第三方配置位置并增加路径回归验证。
- 执行质量门，更新相关文档及记录。
- 核对运行中的开发版是否使用修复，验证实际配置位置。

## Out of Scope

修改旧项目、删除包缓存、清理用户会话、修改上游凭据或模型路由、变更技术栈。

## Existing Behavior / Root Cause

当前路径选择只要在商店包 LocalCache 发现 `_meta.json`，就改写该包缓存；共享 `Local\Claude-3p` 不再更新。该逻辑由 `cbd7176` 引入。

本机 Claude `2.31226.1.0` 的安装清单明确将 `LocalAppData\Claude-3p` 排除出文件系统虚拟化；实际运行日志使用共享目录及旧项目的 3000 地址。截图中的 `D:\WpSystem\...\LocalCache\Local\Claude-3p` 是包缓存，而不是该客户端的第三方配置入口。

两处配置确实同时存在，共享配置 9 个模型、包缓存 10 个模型。开发网关 3001 正常监听，读取两份配置中的接入 Key 请求 `/v1/models` 均返回 200；两份 Key 一致。没有证据说明必须安装 OmniProxy 才能接入。

## Analysis

采用客户端清单和旧项目已验证的共享入口；不复制写入两份目录，不依据残留包缓存猜测当前入口。路径变化后沿用现有重新读取和备份归档保护。

运行中的开发版位于 `E:\go\new\wails3-v\OmniProxy-main\bin\omniproxy.exe`，当前工作区为另一个分支；交付需明确运行版与源码的区别。

## Implementation Plan

1. 恢复统一共享第三方配置目录，验证空目录、双目录、损坏包缓存都不误选。
2. 检查路径变更后的重新读取、激活及恢复逻辑，执行针对测试及 `task verify`。
3. 更新接入说明与 CURRENT，核对 Git 差异，仅提交本任务改动。
4. 检查实际运行版并记录原生验证结果与限制。

## Progress

- 已完成只读定位；用户进一步确认未写入实际配置，仍显示旧软件配置。
- 已修复路径选择；配置服务全部测试通过，包括激活、恢复、重新读取及备份归档。任务编号按 main 已存在的 001–007 调整为 008，避免与并行工作记录重复。
- 新增路径纠正后的应用／恢复回归测试：未重新读取时拒绝覆盖，重新读取后激活共享配置，恢复使用共享目录的原状态，旧包缓存保持不变；测试通过。
- 源工作区完整 `task verify` 退出码 0；接下来同步至实际运行的 main worktree，并执行该版本的质量门及原生验收。

## Implementation Result

- 已恢复共享第三方配置入口，删除错误的包缓存自动选择，保留现有路径安全检查。
- 覆盖新旧包名、共享配置存在／尚未创建、损坏包缓存，以及路径纠正后的安全应用和恢复。
- 实际运行版同步与原生验收进行中。

## Verification

- PASS：`go test ./internal/application/agentconfig -count=1`。
- PASS：`go test ./internal/application/agentconfig -run 'TestDesktop(Path|Paths)' -count=1`，包含新增路径恢复测试。
- PASS：源工作区 `VITEST_MAX_WORKERS=4 task verify BIN_DIR=E:/go/new/wails3-v/OmniProxy/tmp/claude-desktop-shared-config/quality-bin`，退出码 0；完整检查与生产构建通过。
- PASS：`git diff --check`。

## Documentation

计划更新 LOCAL_AGENTS 和 CURRENT。

## Deviations

None.

## Git

待填写。

## Result

待填写。
