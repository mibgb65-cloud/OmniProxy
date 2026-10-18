# TASK-20261018-002: 接入 Kimi Code、Pi、Factory Droid 和 Goose

## Status

In Progress

## Created

2026-10-18（Asia/Singapore）

## Request

用户希望多接入几个 Agent：Kimi Code、Pi、Factory Droid、Goose。

## Goal

按现有本地 Agent 管理流程增加四个客户端，提供准确的检测、配置预览、备份／应用／恢复及网关模型接入，复用现有品牌卡片和详情页面。

## Scope

官方配置和安装方式核对、现有 Agent 后端适配与前端目录扩展、必要测试、文档和质量门验证。

## Analysis

- Pi 按 pi.dev 的 Pi coding agent（badlogic/pi-mono）处理。
- 复用现有 agentconfig／agentinstall 与产品分组卡；四者采用已有 Chat Completions 网关，避免新增运行时或客户端 SDK。
- Kimi Code 当前官方版本使用 `KIMI_CODE_HOME`／`~/.kimi-code/config.toml`；兼容已有旧 Python kimi-cli 的 `KIMI_SHARE_DIR`／`~/.kimi/config.toml`，其 Chat 类型必须为 `openai_legacy`，不混用新版能力枚举。当前版目录优先，保存路径在预览中明确展示。
- Pi 需要 `models.json` 和 `settings.json`；Goose 需要 YAML 设置和独立自定义 provider JSON。选择有界双文件快照、逐文件版本校验和失败回滚，沿用现有加密备份记录；不改写其他客户端的多文件实现。
- Factory Droid 使用 `settings.json` 的 `customModels`，追加受管条目保留原模型顺序，按官方 CLI 规则生成自定义默认模型引用。
- 安装沿用固定官方入口／本机检测，不增加未经确认的安装脚本执行；Goose CLI／Desktop 共享配置，安装状态分开检测。
- 主要风险：外部配置覆盖、同名提供商冲突、多文件部分写入和客户端版本差异；分别通过白名单路径、所有权校验、失败回滚、回归测试及明确版本边界控制。

## Implementation Plan

1. 核对四个客户端的官方配置格式、协议、配置路径、安装／版本检测与兼容边界。
2. 复用现有检测、受管配置、备份／恢复和模型选择机制，增加四个客户端及中英文展示。
3. 验证配置安全、已有字段保留、恢复和网关协议，执行前后端测试、原生检查及 `task verify`。
4. 更新文档、任务记录和 CURRENT，检查变更并提交归档。

## Implementation Result

- 增加 `kimi-code`、`pi`、`factory-droid`、`goose` 四个接入 ID，接入现有精确白名单、独立 Key 鉴权、Chat 模型目录、所选模型持久化、刷新及真实已接入列表。
- Kimi Code 当前／旧 Python 版按已存在目录及显式环境变量区分，分别写 `openai`／`openai_legacy`；保留其他提供商与权限。Pi 管理模型和用户设置，保留原模型循环范围并补入所选新模型，显式发送 Bearer。
- Droid 保留原模型顺序，追加受管模型，按官方 CLI 的展示名／数组下标生成默认引用。Goose 的 CLI／Desktop 共用配置，采用独立 provider JSON、固定鉴权头和静态模型列表，不覆盖原 OpenAI Key、Keyring、认证或会话文件。
- Pi／Goose 使用固定关联路径的有界双文件快照；全组版本摘要、加密原字节备份、逐文件写入校验及部分失败回滚复用现有基础能力。预览目录版本同时绑定配置路径与 Kimi 变体，阻止把旧预览应用到另一数据上下文。
- 前端新增四张品牌分组卡和五个软件入口（Goose 两项），沿用详情、模型选择、默认模型、保存／取消及返回焦点；增加中英文标题、目录方式和通用配置冲突提示。
- 安装适配增加固定官方说明／下载入口、CLI 版本解析和 Droid 用户安装目录查找、Goose 桌面精确检测及打开入口。未增加安装脚本执行、SDK、依赖或数据库迁移。
- 新增实现和测试按职责拆为小文件，均不超过 200 行；既有 `plan.go` 仅增加必要分发与预览上下文，保留集中准备流程，避免扩大无关重构。

## Verification

- 后端定向：`go test ./internal/application/agentconfig ./internal/infrastructure/agentinstall ./internal/application/agentinstall` PASS。覆盖四者预览／应用／刷新／原字节恢复、其他字段保留、默认模型与官方协议／URL／鉴权字段、部分模型选择、外部关联文件修改、同名冲突、非法 JSON／TOML／YAML、数据库失败协调回滚与 Windows 第二文件占用回滚；补充 Kimi 目录优先级、旧版协议、版本输出及精确安装目标。
- 将既有 Key 白名单测试扩为全部 13 个客户端，同时启用时验证新增 Key 不被旧 LIMIT 排除；未知历史客户端仍拒绝。新增 Kimi 协议／Pi 数据目录变化使旧预览失效的回归。
- 前端定向 4 文件共 50 项 PASS；新增分组、五个入口、Goose 共享详情／返回焦点，以及四者预览／应用复用既有参数化行为测试。
- 完整质量门最终 PASS（退出码 0）：`task --force verify BIN_DIR=tmp/four-agents-20261018/verify-bin`，100 文件 564 项前端测试、全部 Go 测试、格式／lint／类型、生产前端及 Windows 构建均通过；最终补充的配置上下文保护和冲突文案已包含在本轮。日志 `tmp/four-agents-20261018/verify-final.log`。已有 jsdom scrollTo 和 bundle 大小提示不影响结果。
- 正常开发原生窗口中完成四个产品 × 深浅主题、1440／960 宽度共 8 组检查：Kimi／Droid／Goose 各提供 74 个实际可用模型的预览和可用应用按钮；Pi 检出本机已有未接管的同名 provider，按设计拒绝覆盖。未点击真实应用按钮、安装或调用模型；Goose Desktop 导航进入共享详情。原生脚本／结果／截图位于 `tmp/four-agents-20261018/native-*`，主题和尺寸恢复。
- Windows 安装检测未发现这四个客户端，本任务没有进行真实客户端重新加载或模型请求验收。Pi 实机成功应用需要先处理现有同名配置；保留其原配置。
- 图标检查根节点及无脚本／事件／foreignObject／外部引用；来源和 SHA-256 已记录。`git diff --check` 通过。

## Documentation

- 新增 `docs/product/KIMI_PI_DROID_GOOSE.md`，记录配置位置、格式、默认模型、环境覆盖、双文件保护、官方依据和验证边界。
- 更新 PRD、LOCAL_AGENTS、架构概览、图标来源和 CURRENT；技术栈／模块边界不变，无新增 ADR。

## Deviations

- Pi 原生成功预览受本机已有同名配置阻止；保留保护行为，以隔离临时目录测试验证成功应用／刷新／恢复。未为通过验收而覆盖真实配置。
- Kimi 官方当前版与旧开源 Python 版存在目录／协议差异，实际扩展兼容已有旧 TOML 配置，预览仍明确显示单一生效路径。

## Git

待提交。

## Result

四个 Agent 的产品接入已实现并通过最终质量门；客户端重载／调用仍待实机验收，Pi 既有同名配置继续受保护。
