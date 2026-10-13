# TASK-20261012-006: Claude Code Desktop 一键启用本地网关

> 补录说明：开始本次修复时尚未创建记录，用户指出流程遗漏后立即补建。本记录如实保留这一偏差；后续实现、验证和交付从本记录继续跟踪。

## Status

Completed

## Created

2026-10-12（在实现中途补建）

## Completed

2026-10-12

## Request

“E:\go\new\OmniProxy 我原先的这个项目都可以一键写入这个本地网关啊，你看看为啥这个不行呢，我打开让我登录。”随后用户询问这些开发操作是否有标准 Markdown 记录。

## Goal

使一键配置与旧项目一样完整启用 Claude 本地网关，重启后进入第三方模式；保留原配置、会话和恢复能力。补齐本轮真实任务记录。

## Background

此前只写入独立网关配置和配置库条目，要求手动选择，未完成用户期待的一键启用。相关安装、写入兼容见 TASK-20261012-005。

## Scope

- 对比旧项目的启动模式和配置选择逻辑。
- 一键配置设置两份 `claude_desktop_config.json` 的 `deploymentMode=3p`，并设置配置库的 `appliedId`。
- 记录原模式与配置 ID，提供安全恢复及多文件失败回退。
- 兼容原生请求与客户端模型 ID 规则，保留实际显示名称并复用现有精确路由。
- 明确重启要求，验证配置和真实启动界面。
- 补录 TASK-20261012-002 至 005，并更新 CURRENT。

## Out of Scope

修改旧项目、清理用户历史会话、改写上游凭据、切换客户端安装包、重新设计网关协议。

## Existing Behavior / Root Cause

旧项目 `writeClaudeDesktopProfile` 写入普通与 3P 启动配置的 `deploymentMode=3p`，再将 `_meta.json.appliedId` 指向网关配置。新版 `desktopEntry` 原先只添加条目，没有完成这些激活字段及禁止登录选择器的写入。

后续实机排查发现两个独立问题：工具执行环境与实际客户端使用不同的用户配置视图，工具写入成功并不代表客户端读到了新配置；原生连接探针携带的 `Sec-Fetch-Site: none` 被误判为浏览器跨站请求。客户端另会对非 Claude 模型 ID 提示配置问题，旧项目已使用路由兼容名，而新版直接写入了上游模型名。

## Expected Behavior

预览说明将启用第三方模式；应用成功后激活本地网关。用户完全退出并重启 Claude 后进入网关模式。恢复时还原原模式和选择，保留接入后新增的非受管设置。

## Analysis

### Related Modules

`internal/application/agentconfig/`、`internal/core/proxy/`、`internal/core/modelcatalog/`、`internal/application/gateway/`、迁移、配置词条、LOCAL_AGENTS、BUILD 与 ADR-005。

### Dependencies

沿用现有 SQLite、JSON 合并、文件版本检查、DPAPI 配置备份及加密目录写入兼容；不新增外部依赖。

### Risks

- 多文件切换必须在失败时回退，避免配置条目和启动模式不一致。
- 仅模式和配置 ID 等非敏感字段写入新表；密钥和原客户端配置继续使用加密备份。
- 运行中的客户端不会立即使用磁盘新配置；重启验收前核对是否存在活动任务。
- 保留外部修改保护；恢复不能覆盖用户后来选择的其他配置。

## Implementation Plan

1. 核对旧实现和本机非敏感启动状态。
2. 准备三份文件的合并变更与原模式快照。
3. 将激活状态和主配置版本同事务提交；失败回退文件。
4. 恢复原模式、配置选择与主配置，并保留非受管字段。
5. 增加首次启用、重复应用、恢复、外部修改和失败回退测试。
6. 运行 `task verify`，核对原生界面，更新文档和记录。
7. 追加实机发现的原生 Fetch Metadata 兼容与稳定模型路由名，重新验证完整质量门和实际 Code 界面。

## Progress

- 已完成旧项目只读对比，定位启动模式和 `appliedId` 缺失。
- 已写入第一版自动启用、恢复与迁移 014；尚未完成编译、回归和原生验收，不能视为最终实现。
- 用户指出记录遗漏后补建本任务与前四份记录；继续开发前已创建这些记录。
- 第一轮完整 `task verify` 通过，但原生客户端仍显示登录，未将任务标记完成。
- 核对 MSIX 包缓存与目录链接，并用目标句柄解析受保护的目录；原生安装清单明确排除共享 `Claude-3p` 目录，不能将第三方配置盲目改到包目录。
- 客户端诊断仍报告第三方配置未读取；临时用户策略与目录镜像试验未解决问题，需清理这些诊断变更，不作为交付方案。
- 进一步逐字段对比旧项目，发现还缺少 `disableDeploymentModeChooser=true`；已补齐并继续验证。
- 用户重新用旧项目配置后，原生界面已进入 Gateway 并显示 DeepSeek 模型。工具侧仍读取到新版 `appliedId`，与原生实际生效状态矛盾；下一步区分开发进程运行上下文和配置写入算法，避免继续凭猜测修改客户端目录。
- 已用交互用户的临时 Windows 任务执行只读核对，读取到旧配置 ID `157210` 且文件无加密属性；工具侧同路径仍是新版 ID `309006` 并显示加密属性。确认先前原生失败发生在隔离配置视图，临时任务已删除。接下来删除未经验证的 MSIX 普通目录重定向，沿用旧项目的共享目录；通过正常用户上下文启动开发版做真实验收。
- 自动审批拦截了关闭旧开发进程并用临时 Windows 任务启动新版的命令，原因为 `blocked by policy`；该命令未执行。改用普通桌面窗口关闭与资源管理器打开临时启动器，不建立启动任务或使用管理员权限。
- 普通资源管理器打开启动器也被自动审批拦截，原因为 `blocked by policy`。用户随后手动启动新版，确认其进程来自本次 `native-bin` 构建，控制端口为 3891。用户应用配置时遇到版本冲突；继续通过现有“重新读取配置”流程刷新实际文件基准，再预览／应用并验收。
- 在用户启动的窗口中，已完成重新读取、预览、应用，原生设置页确认网关为 `127.0.0.1:3001`，Code 已进入 Gateway 模式。配置保护提示已消除；原配置保留为加密历史。
- 原生连接探针仍报告 HTTP 403。已核对安装包中的探针使用 Electron Session Fetch，携带原生请求的 Fetch Metadata；用已授权 Key 加 `Sec-Fetch-Site: none` 重现 `local_access_only`，无该头或 `same-origin` 返回 200。准备仅允许无 Origin 的 `none` 请求，继续拒绝浏览器跨来源及 DNS 重绑定；增加真实请求形态与错误 Key 的回归测试后重新验收。
- 403 修复版已通过实机探针：Code 的 `ConfigHealth` 为 `healthy`，真实 Messages 探针返回 200，目标模型为 `deepseek-flash`。原生界面仍有非 Claude 模型 ID 的配置提醒；安装包验证逻辑及旧项目实现确认需要客户端路由别名。下一步生成稳定的专用兼容别名，保留实际显示名称，以当前可用模型目录解析到原模型并复用原有精确路由；不写入用户路由表，不增数据库迁移。新增公开可用的兼容 ID 只作用于 Messages 解析，现有原模型 ID、Codex 目录和路由行为保持兼容。
- 稳定兼容别名已实现，应用和目录刷新使用同一格式；当前目录解析后继续使用原路由、账号与模型。新增真实 HTTP Messages 转发、未知／移除条目拒绝、停用账号拒绝及原路由／非受管请求字段保留测试，全部通过。最新完整 `task verify` 通过（64 个前端测试文件／340 项测试及 Go 检查、生产构建），最终临时构建为 `native-bin-final`，等待正常用户上下文复测配置提醒。
- 用户已启动最终构建；重新预览／应用成功。Claude 原生 Code 页面免官方登录进入 Gateway，配置警告消失，模型菜单显示 8 个实际模型名称。真实 Messages 探针将兼容名解析为 `deepseek-flash` 并返回 200，`ConfigHealth=healthy`。开发网关 3001／控制接口 3891 均由最终构建监听，旧项目 3000 未改动。

## Implementation Result

已实现主配置、两份共享启动模式、当前配置选择及禁止官方登录选择器的自动写入；模式与主配置版本同事务提交，并提供多文件失败回退和原模式恢复。已撤销未经验证的 MSIX 普通目录重定向，继续使用旧项目已验证的共享目录，并保留打开前配置检查。

用户从资源管理器启动开发版后，重新读取／应用已成功，Claude 设置显示 `127.0.0.1:3001`，Code 已免官方登录进入 Gateway 模式。此前“写入成功但仍登录”的多轮实机测试使用了隔离配置视图，不能代表实际客户端读取结果。

已修复网关误拒绝无 Origin 的 `Sec-Fetch-Site: none` 原生请求，实际探针返回 200 且状态为 `healthy`。进一步补齐稳定兼容模型 ID、当前目录解析和原路由转发，应用与刷新保留实际显示名称。最终构建的完整质量门及实机验收均通过：3001 生效、Code 免官方登录、配置提醒消失、8 个实际模型可选。

## Files Changed

- `internal/application/agentconfig/desktop_activation.go`、`desktop_activation_files.go`
- `internal/application/agentconfig/apply.go`、`restore.go`、`plan.go`
- `migrations/014_agent_desktop_activation.sql`
- `frontend/src/i18n/locales/agents-zh.ts`、`agents-en.ts`
- 激活、恢复、文件／数据库失败回退及原生请求／模型映射回归测试
- `internal/application/agentconfig/desktop_ready.go`、`canonical_windows.go`、`canonical_other.go`、`paths.go` 及测试
- `internal/application/agentconfig/patch.go` 及 Desktop 配置回归测试
- `internal/transport/wails/agentinstall.go`：未激活时拒绝打开到官方登录
- `internal/core/proxy/handler_helpers.go`、`native_fetch_test.go`：原生请求兼容与安全边界回归
- `internal/core/modelcatalog/claude_desktop.go` 及测试、`internal/application/gateway/routing.go`、`desktop_alias_test.go`：兼容路由名解析与实际模型转发
- `internal/application/agentconfig/catalog_patch.go`、`refresh.go`、`catalog_test.go`：兼容目录应用与更新

## Verification

已执行针对本任务的验证，未沿用前一任务结果代替本次检查。

- [x] 激活与恢复单测
- [x] 外部修改与失败回退测试
- [x] 原生启动验收
- [x] 原生连接探针及无配置提醒、实际模型列表验收
- [x] `task verify`
- [x] `git diff --check`

完整 `task verify` 通过：Go 测试、格式、lint、类型检查、64 个前端测试文件／340 项测试及生产构建。加密目录合成配置测试通过首次应用、重复应用和恢复。

原生应用确认模式文件和配置 ID 已写入，但 Claude 重开后仍登录，客户端诊断未读取到这些配置。临时 HKCU 策略试验没有解决问题，已清除新增策略值；临时启用的开发者模式已恢复为关闭。诊断文件只保留在临时目录，没有放入任务文档或提交用户凭据。

用户已确认：同一 Claude 客户端版本下，旧项目现在仍可一键接入。继续核对旧项目的实际文件写入方式、当前生效配置及新版差异；不把问题归因于客户端版本。

后续正常用户上下文验收已确认新版地址 3001 生效并进入 Code Gateway 模式，以上登录失败结果属于隔离视图。最新代理单测已通过原生请求、跨来源／跨站拒绝及错误／缺失 Key 检查；403 修复后的完整 `task verify` 再次通过（64 个前端测试文件／340 项测试及 Go 检查、生产构建）。修复版已生成到临时 `native-bin-next`；原生连接复测等待用户从资源管理器启动。

该修复版随后已正常启动：真实 Messages 探针返回 200、目标为 `deepseek-flash`，客户端 `ConfigHealth=healthy`。模型兼容名的完整 `task verify` 也已通过；最终 `native-bin-final` 构建重新应用并重开客户端后，兼容名正确解析为实际模型，探针再次返回 200，配置提醒消失，模型菜单显示 8 个实际名称。

正常用户上下文只读核对确认：开发者工具设置已关闭，临时策略值已清除，生效配置 ID 为 309006、地址为 3001，保留了其他配置条目。临时只读核对任务均已删除。

## Documentation

LOCAL_AGENTS、ADR-005 与 BUILD 已更新，包含一键启用、兼容模型名和正常用户上下文的实机验收要求。此前 TASK-20261012-005 的诊断误判已追加澄清。完成后归档本记录并同步 CURRENT。

## Deviations

任务记录并非事前创建，已如实标明补录。用户明确要求旧项目的一键启用行为，本次将替换此前手动选择的行为约定，同时继续保留原文件和会话。

实机推理验证范围为 DeepSeek Flash 的原生连接探针；其他模型未逐一运行完整编程任务。当前运行的是开发 profile 的临时验收构建，后续正常开发可从用户终端运行 `task dev`。

## Git

Not committed by Agent. 已复查 `git diff --check` 与相关实现。本任务涉及的应用层／Core 模块在仓库中均为既有未跟踪实现（相关目录的 tracked 文件数为 0）；单独提交本次片段会缺少依赖，整体提交则会纳入其他既有工作。因此保留未提交状态，未覆盖、删除或提交无关改动。

后续用户明确要求提交当前代码，TASK-20261012-007 已将完整代码快照提交为 `44fdbc49b8958d5a8785ba1ab27ef6a0eafd021c`。按用户“只提交代码”的要求，文档继续保留在工作区。

## Result

Completed. 一键配置、重新读取后的应用、3001 网关激活、Code 免官方登录、无配置提醒及实际模型列表均已通过验收。完整质量门通过，记录归档，开发版与网关保持运行。四份此前工作的补录已完成；后续开发继续先记录、再实现。
