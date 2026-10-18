# TASK-20261018-003: 评估 Beta 发布准备并升级 Wails

## Status

Completed

## Created

2026-10-18（Asia/Singapore）

## Request

用户询问当前功能是否已适合发布测试版，并要求检查 Wails 更新，有更新则升级。

## Goal

依据现有实现与验证结果评估 Windows Beta 的发布条件，升级到已发布的 Wails 最新 v3 版本并验证兼容性。

## Scope

- 核对产品主流程、尚未完成能力、应用更新与 Windows 打包配置。
- 核对官方 Wails 发布记录及 Go／npm 已发布版本。
- 同步升级 Go 模块、前端运行时、CLI、CI 和当前版本文档。
- 执行完整质量门及隔离原生启动验证，记录测试版发布前的剩余工作。
- 修复打包验证暴露的 NSIS 脚本输入编码问题，验证 Windows 安装包能够编译。

## Out of Scope

- 不扩展提供商、调度器或客户端功能。
- 不发布远端 Release，不确定未经确认的正式应用版本号与发布渠道。
- 不修改现有用户数据、凭据或客户端接入配置。

# Analysis

## Existing Behavior

应用版本为 0.1.0；Wails Go／JS／CI 固定 beta.28。近期任务显示账号、路由、网关、统计、客户端配置已实现，部分真实客户端调用及自动调度仍待完成。工作区存在五份既有未跟踪任务记录，保持原样且不纳入本任务提交。

Go 模块代理的 `@latest` 与 npm 的运行时版本均确认 beta.30（2026-10-15 发布）。beta.29 修复 Windows WebView2 COM 引用计数及 Deferral 释放，beta.30 修复托盘右键菜单、菜单更新及 InvokeSync panic 导致的阻塞。继续使用同一 Wails 3 技术栈。

## References

- https://github.com/wailsapp/wails/releases/tag/v3.0.0-beta.29
- https://github.com/wailsapp/wails/releases/tag/v3.0.0-beta.30
- https://v3.wails.io/changelog/

## Risks

Wails 仍为预发布框架，需要验证生成绑定、Windows 构建、桥接与桌面生命周期；生产包启动验证必须使用隔离数据目录，不连接真实模型。

# Implementation Plan

1. 阅读发布说明并核对官方模块／运行时版本，检查发布相关实现。
2. 将 Wails Go／JS／CLI／CI 统一到 beta.30，保持其他依赖与应用版本不变。
3. 运行 task verify，审查依赖、绑定和构建差异；按需验证隔离原生启动。
4. 记录 Beta 准备情况、实际限制与验证结果，同步文档，审查提交并归档任务。

# Implementation Result

- Go 模块、前端 `@wailsio/runtime` 与本机 CLI 统一升级为 beta.30，CI 的 CLI 安装目标及 README／TECH_STACK／BUILD 同步固定该版本。
- Go／pnpm 锁文件只变更 Wails 版本及校验值；没有升级其他业务依赖、工具链或应用版本。新版 CLI 生成 99 个方法／92 个模型／5 个事件，未引入已跟踪绑定差异。
- Windows NSIS 命令在宿主 Windows 和跨平台分支均显式指定 UTF-8 输入；构建文档说明 NSIS 的 PATH 与编码要求。
- 依据现有代码、近期任务和构建配置完成 Beta 准备评估，区分已实现能力、实测范围及发布收尾；未创建远端 Release 或修改发布策略。

## Root Cause（安装包编码）

仓库脚本为 UTF-8 无 BOM，NSIS 编译器默认按本机 ACP 读取；中文 Windows 上遇到 `©` 等 UTF-8 字节时报告 `Bad text encoding`，原打包命令未声明字符集。采用显式 `-INPUTCHARSET UTF8` 读取脚本，不改变仓库编码规范或安装行为。

# Beta 发布评估

## 结论

现有主流程已适合进入 Windows 小范围 Beta，建议冻结新增功能，转向真实客户端回归、打包与发布收尾；不能将产品描述为所有 PRD 能力均已完成。正式发布时间、测试人群和版本号仍待用户确认。

## 已有基础

- 账号与受保护凭据、模型路由、已适配上游文本／工具转发、请求与用量统计、本地客户端配置／备份／恢复均已有实现与测试。
- 近期任务包含实际 CLI 的流式／工具续接与原生界面验收，不能据此推断所有客户端和提供商均已实测。
- 开发与生产数据隔离；已有 Windows 生产构建、NSIS 打包入口及 Windows CI 质量门。

## 发布前收尾

1. **实机回归**：从干净 Windows 用户环境安装，核对添加账号、启动网关、真实客户端的流式／工具调用、重启后的配置及凭据持久化、配置恢复和卸载；新接入 Kimi Code／Pi／Factory Droid／Goose 的真实调用仍待验收。
2. **发布包与元信息**：当前应用版本为 0.1.0，未确认 Beta 版本号；`build/windows/info.json` 与 NSIS 辅助文件仍有 `My Company` 等模板内容，Windows 资源语言仍为 `0000`。确定发布版本后统一应用、Windows 数字资源及安装器版本，验证安装／覆盖升级／卸载和校验和；签名状态必须如实说明。
3. **更新行为**：应用仅实现只读检查，默认 `UpdateRepo` 为空，未实现下载／安装。检查器使用 GitHub `/releases/latest`，该端点不返回 prerelease；`internal/version.Compare` 还会忽略预发布后缀，使同一基础版本的 beta.1 与 beta.2 相等。首轮若手动分发，需明确手动更新；若承诺应用内 Beta 更新，必须先补齐渠道选择与 SemVer 比较并验证。
4. **对外分发信息**：README 许可证仍为 TBD；官网、仓库、协议入口按已有用户要求保持待配置。确定下载／反馈渠道、许可证和已知限制，准备与实际能力一致的说明；当前 `.github/workflows` 仅含 CI，不含发布工作流。

## Beta 应明确声明的能力边界

- `internal/application/gateway/service.go` 每次只选择一个账号，不进行请求失败后的自动重试或故障切换；完整健康／负载调度尚未实现。
- Anthropic、Zhipu 及通用 OpenAI-compatible 的完整网关适配并非均已实现；内置提供商卡片不等于推理链路已接通。
- 部分套餐额度、真实 Key 与长上下文、部分桌面客户端调用仍待验收；跨协议 SSE 存在缓冲完整结果后输出的路径，多模态不在当前承诺范围。
- 当前产品目标为 Windows；Wails v3 beta.30 本身仍为预发布版本。

GitHub 更新端点依据：https://docs.github.com/en/rest/releases/releases#get-the-latest-release 。本节只记录检查结果，不改写用户尚未确认的发布策略或补做这些独立功能。

# Verification

- PASS：官方 Go 模块 `@latest`、npm 查询、本机 `wails3 version`、项目 `go list -m` 与前端依赖列表均确认 beta.30。
- PASS：`task --force verify BIN_DIR=tmp/wails-beta30-20261018/verify-bin`，退出码 0；格式、Go vet／staticcheck／ESLint、TypeScript、全部 Go 测试、100 文件／564 项前端测试、生产前端与 Windows amd64 构建通过。
- PASS：生产 EXE 构建信息包含 Wails `v3.0.0-beta.30` 和 `production` 标签。
- PASS：隔离 test profile 使用生产构建的内嵌前端，原生“关于”实际读取 beta.30；43892 健康接口返回 test，43002 网关从 UI 成功启动／停止；托盘 `WM_CONTEXTMENU` 能打开正确菜单，结束菜单跟踪后执行 Quit，Windows 原生查询退出码为 0。未调用真实模型、未应用真实客户端配置。
- PASS：先在完整副本上复现默认 ACP 的 `Bad text encoding`，增加 `-INPUTCHARSET UTF8` 后，相同 NSIS 脚本、图标、bootstrapper 与生产 EXE 编译通过，安装器为 10420156 字节；`task --dry windows:create:nsis:installer` 确认实际任务展开带相同字符集参数。仅编译，未实际安装。
- PASS：`git diff --check`；锁文件无其他依赖变化，生成绑定与图标无额外差异。
- PASS：通过正常桌面启动器恢复当前分支开发版，确认共享配置实体路径没有 Codex 包重定向。新原生应用 PID 9328，可见且响应；3891 健康接口返回 development，9245 首页及 HMR 均 HTTP 200，日志明确连接前端开发服务；原生“关于”和运行 EXE 均确认 beta.30，保留生产实例，开发网关保持停止。
- 原生隐藏启动与菜单跟踪的验收脚本已校正，最终结果为正常退出；脚本、完整质量门／打包日志及 JSON 位于忽略目录 `tmp/wails-beta30-20261018/`，不纳入 Git。

## Remaining Risks

Wails 仍为预发布框架；未做 WebView2 崩溃故障注入和长时间压力测试，未执行干净机器安装／覆盖升级／卸载。测试包保持当前 0.1.0 与模板元信息，Authenticode 状态为 NotSigned，不作为已定稿发布包。既有 jsdom scrollTo、Vite 大包和开发启动 uname／tail 提示不影响本轮检查。产品及发布限制详见上述 Beta 发布评估。

# Documentation

已同步 README、TECH_STACK、BUILD、CURRENT；发布准备结论保留在本任务记录。无业务 API／架构边界或 PRD 功能变更，无新增 ADR。

# Deviations

在发布准备的实际打包验证中发现既有 NSIS 默认代码页问题，追加两处命令参数的最小修复并复现／验证；未扩大到发布渠道、元信息定稿或自动更新开发。恢复开发环境时遇到本机既有 Task 指纹扫描耗时，改用忽略目录临时开发配置的 `task --force build DEV=true` 完成构建，未修改仓库的默认开发配置。

# Git

实现提交：`695ae196747b9cfe6f4c51c2d10c89f74253dbe6` — `build(wails): 升级至 beta.30 并修复安装包编码`。任务评估与验证记录随后归档。未推送，既有五份用户任务记录保持原样。

# Result

Wails 相关版本已统一为 beta.30，完整质量门、隔离原生运行与 NSIS 编译通过。现有功能适合进入 Windows 小范围 Beta，正式分发前仍需完成实际安装／升级／卸载与客户端调用回归、版本元信息和发布说明；自动更新及调度边界必须明确。

# Completed

2026-10-18（Asia/Singapore）
