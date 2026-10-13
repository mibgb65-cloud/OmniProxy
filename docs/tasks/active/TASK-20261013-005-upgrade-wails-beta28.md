# TASK-20261013-005: 升级 Wails 至 beta.28

## Status

In Progress

## Created

2026-10-13（Asia/Singapore）

## Request

用户查询 Wails 最新版本后，要求将当前 v3.0.0-beta.25 更新到 v3.0.0-beta.28。

## Goal

统一升级 Go 依赖、前端运行时和本机 CLI，验证构建、绑定与原生开发启动兼容性。

## Scope

- 检查 beta.26–beta.28 官方发布说明及依赖差异。
- 升级 Go 模块、前端运行时与对应锁文件、本机 CLI。
- 将 CI 与安装说明中的 CLI 版本固定为项目版本，同步技术基线。
- 运行完整 `task verify` 并验收普通 `task dev`、原生窗口和图标。

# Analysis

当前 Go 模块、前端运行时及 CLI 均为 beta.25，CI 与 README 使用浮动 latest。
beta.26 增加开发前端 HTTP 就绪检查，beta.28 修复 Windows WebView2 恢复与回调生命周期。
保留项目现有开发等待和回归测试，避免在依赖升级中混入无关重构。

## References

- https://github.com/wailsapp/wails/releases/tag/v3.0.0-beta.26
- https://github.com/wailsapp/wails/releases/tag/v3.0.0-beta.27
- https://github.com/wailsapp/wails/releases/tag/v3.0.0-beta.28

## Risks

预发布框架升级需要验证桌面生命周期和前后端桥接；CLI 更新前需正常退出当前开发任务，避免 Windows 可执行文件占用。

# Implementation Plan

1. 确认版本已发布，阅读相关变更并检查当前调用兼容性。
2. 正常退出当前仓库开发实例；升级 Go、JS 和 CLI，固定 CI 版本。
3. 核对依赖差异和生成绑定，处理真实兼容性问题。
4. 运行质量门、原生开发启动和关键界面操作验收。
5. 更新实际结果与文档，审查并提交修改，归档任务。

# Implementation Result

- Go 模块升级至 `v3.0.0-beta.28`，整理并更新 Go 校验记录；前端 `@wailsio/runtime` 精确锁定 `3.0.0-beta.28`，同步 pnpm 锁文件。
- 本机 `wails3` CLI 已安装并确认 `v3.0.0-beta.28`。
- CI 与 README 改为安装精确版本的 CLI，技术基线与构建说明同步。
- 依赖差异仅涉及 Wails 本身；未升级其他业务依赖、Go／Node 或应用版本。
- 保留已有开发等待行为及测试，仅更新其背景注释以说明与 CLI 探测的关系。

# Verification

- PASS：CLI、`go list -m` 和 pnpm 安装列表均确认 beta.28。
- PASS：`git diff --check`；Go／JS 锁文件仅包含目标 Wails 版本及校验值差异。
- PASS：`task verify` 退出码 0，格式、Go vet／staticcheck／ESLint、TypeScript、全部 Go 测试、64 个前端测试文件／340 项测试与生产构建通过。
- PASS：新 CLI 生成 72 个方法／66 个模型的绑定，没有新增已跟踪绑定差异；生产 EXE 的 Go 构建信息确认为 `v3.0.0-beta.28`。
- PASS：普通 `task dev` 启动成功，任务 PID 26556，当前仓库应用 PID 21284，窗口可见且正常响应，日志明确记录 Wails beta.28 与前端连接成功。
- PASS：原生“关于”页通过 UI Automation 读到实际版本 `v3.0.0-beta.28`，验证升级后的 Go／JS 桥接可用。
- PASS：3891 `/healthz` 返回 development／ok，9245 `/@vite/client` 返回 HTTP 200，HMR 服务正常。
- PASS：原生 EXE／窗口／快捷方式仍采用 C 图标，系统托盘注册正常；未出现相关图标错误。
- 验证日志和原生快照保存在 `tmp/wails-beta28/`，仅作为本机验收产物，不纳入 Git。

## Remaining Risks

beta.28 仍是预发布版本；本任务验证了正常构建与桌面运行，未做 WebView2 崩溃故障注入。既有 uname／tail、jsdom scrollTo、Vite 大包及受管客户端目录刷新提示未影响质量门和原生启动。

# Documentation

已同步 README、TECH_STACK、BUILD 和 CURRENT。既有任务记录与测试中的旧版本字符串保留其历史／测试数据意义，不作为当前版本来源。

# Git

实现提交后填写提交哈希。

# Result

Wails Go 模块、前端运行时、本机 CLI 和 CI 安装目标统一为 beta.28，完整质量门与原生验收通过，新开发版保持运行。
