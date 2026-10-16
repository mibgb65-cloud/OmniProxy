# TASK-20261017-006: 接入 DeepSeek Harness 与 OpenCode

## Status

Completed

## Created

2026-10-17（Asia/Singapore）

## Request

用户要求实施 DeepSeek Harness 接入，同时接入 OpenCode CLI 和 Desktop。

## Goal

在本地 Agent 页面提供两款产品的入口，支持模型目录、独立接入 Key、配置预览／应用、
备份恢复及适当的安装检测。遵循官方配置共享与 profile 边界，验证实际请求和工具续接。

## Scope

- 官方配置与 Windows 安装／检测方式核实。
- 既有 Agent 配置、保护备份、模型目录和鉴权扩展。
- 前端入口、安装状态、配置与中英文文案。
- 必要回归、质量门、原生窗口和文档。

## Out of Scope

未经验证的自动安装渠道、覆盖第三方会话／插件／模型配置、实现客户端自身的 Agent 运行时。

# Analysis

官方已确认：OpenCode CLI／Desktop 共用用户全局 JSON(C) 模型配置；Harness 使用
`profiles/web`、`profiles/desktop` 的 YAML patch 和共享 version-1 凭据文件。
选择为 OpenCode 建一个客户端、Harness 建两个 profile 客户端，统一使用网关 Chat 文本／工具
目录；保留原提供商和未知容量，不声明图片能力。Harness 的备份版本包含本 profile 配置
和本 profile 的单个凭据引用，避免恢复时覆盖另一 profile 或其他凭据。
新增两个小型配置解析依赖：现有 go.sum 中的 yaml.v3 用于保留 YAML 节点／注释，HuJSON
用于正确读取并增量修改 OpenCode JSONC，避免自己实现配置语法。

# Implementation Plan

1. 核实 Harness 的 profile YAML 与凭据引用、OpenCode 的 CLI／Desktop 共享 JSON(C) 配置。
2. 基于现有服务扩展受管字段、模型目录、接入 Key、备份与恢复；保护外部改动。
3. 接入两款产品的卡片／详情、检测与安全官方入口。
4. 完成契约与回归测试，核对真实客户端／原生开发窗口，运行 task verify。
5. 同步文档、审查差异、提交和归档。

# Implementation Result

- 新增 `opencode` 共用客户端、`harness-web`／`harness-desktop` 两个 profile 客户端；
  卡片归组、详情、独立 Key、预览／应用／恢复、目录刷新及重新读取均沿用既有工作流。
- OpenCode 使用 HuJSON 增量合并 JSONC 的 OmniProxy 提供商，保留注释和其他设置；
  支持显式文件／配置目录／XDG 路径、已有 JSONC 优先与同名提供商冲突保护。
  已受管提供商允许网关端口变化后刷新地址；未受管同名配置继续拒绝覆盖。
- Harness 使用 YAML 节点修改目标 profile；凭据文件只修改本 profile 的独立引用。
  本机实测 rc.7 后增加扁平凭据兼容，保留已有 version-1 布局，格式迁移不使受管版本失效。
- Harness 配置／本引用作为一个备份版本，分步写入并做版本比较；配置或数据库写入失败
  回滚已写文件，恢复不会覆盖另一 profile 或其他凭据。继续使用已有 DPAPI 备份表，
  无迁移，不修改上游适配器或核心代理协议；遵循 ADR-005 的会话与外部改动保护。
- 安装目录新增四个固定检测目标，未核实的安装渠道只提供官方说明；已安装时可打开
  固定本机入口。新增 `AgentOpenClient` 不接收路径、任务、命令或凭据参数。
- 统一 Claude／Harness 分组组件，补齐双语文案、OpenCode 本地品牌 SVG 与来源记录。
- 两个小型配置解析依赖固定版本；未引入 Harness／OpenCode SDK 或运行时到项目依赖。

# Verification

- 定向 Go：agentconfig、agentinstall 应用／基础设施及 Wails 通过；覆盖独立 Key、协议筛选、
  无副作用预览、JSONC／YAML 设置保留、刷新／恢复、双 profile 隔离、外部修改拒绝、
  数据库提交失败的双文件回滚、旧凭据迁移、路径优先级、重复键／别名／多文档拒绝与
  非白名单启动目标拒绝。旧有合理测试保留，安装器回归增加手动渠道不得执行的检查。
- 前端定向四文件 35 项测试全部修正后通过对应全量检查；新增客户端沿用网关停止时的
  预览／应用流程，验证手动安装入口与打开客户端只传固定 ID。
- 最终 `task verify BIN_DIR=tmp/harness-opencode-20261017/verify-bin`：PASS，退出码 0。
  格式、go vet／staticcheck／ESLint、类型检查、全部 Go 测试、88 文件／484 项前端测试、
  前端生产及 Windows 构建通过；95 个 Wails 方法的绑定由工具生成并提交。
- 真实客户端隔离验收：本机 Harness `0.1.0-rc.7`、临时安装的 OpenCode CLI `1.19.3`
  分别读取生成配置，通过本实现的实际网关与独立 Key 访问本机模拟上游。均完成 SSE、
  读取隔离标记文件、工具结果续接和最终回答；强制要求完整 `reasoning_content` 回传后
  两者仍成功。每客户端 3 个请求（含客户端内部辅助请求）、1 次文件读取工具及续接。
  未发送真实付费推理、未读取或修改真实客户端会话，QA 运行时位于忽略目录。
- 原生项目窗口按进程路径核对：三个新增入口、真实默认配置路径的三个配置预览均可
  应用（只预览、未实际应用），深浅主题、960×600 和返回导航通过；窗口尺寸及主题恢复。
  未点击真实外部客户端打开按钮，打开动作由固定 ID／安全边界和前端调用回归覆盖。
- 证据：忽略目录 `tmp/harness-opencode-20261017/` 的 `verify-complete.log`、
  `acceptance-reasoning.jsonl`、`native-verification.json` 与原生截图。最终复核端口更新兼容后
  再次完整质量门通过，最终日志为 `verify-release.log`。
- `git diff --check`：PASS；其他五份未跟踪任务记录保留。

# Documentation

新增 `docs/product/HARNESS_OPENCODE.md`，同步 PRD、本地 Agent、架构、技术栈与 CURRENT。

# Deviations

未使用未经核实的 WinGet 包或静默安装脚本；新产品提供官方安装和已安装本机打开入口。
本机 Harness 使用旧版扁平凭据，增加与新版自动迁移兼容的双布局读写，避免只支持最新
仓库而破坏已有安装。配置能力继续复用现有服务，不新增客户端运行时或独立网关。

# Remaining Risks

Desktop 配置路径／profile 与 OmniProxy 原生预览已验证，两款产品实际 Desktop 界面调用
尚未独立验收。真实上游权限、多模态、高级参数、长上下文及未来客户端私有格式仍需按
实际版本验证。Harness 两个文件不具备操作系统级崩溃原子性；未来未知格式明确拒绝覆盖。
OpenCode 项目／环境／组织配置或远程后端、Harness 主目录／启动覆盖可能优先于受管文件。

# Git

实现提交：`a49f2347febd00176a72841c6c0d5d19c92e1bda feat(gateway): 接入 Harness 与 OpenCode 本地客户端`。

分支 `codex/river-builtin-provider`；归档记录与 CURRENT 由后续文档提交保存。

# Result

两个产品的本地接入功能已完成，配置全生命周期、关键隔离保护和真实客户端流式／工具
续接通过，开发窗口已更新。完整质量门通过；用户可在客户端接入页预览并应用。

# Completed

2026-10-17（Asia/Singapore）
