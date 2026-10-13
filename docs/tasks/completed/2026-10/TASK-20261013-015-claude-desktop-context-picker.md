# TASK-20261013-015: Claude Desktop 长上下文模型菜单验收与修复

## Status

Completed

## Created

2026-10-13（Asia/Singapore）

## Request

用户在 TASK-20261013-014 后反馈：模型菜单仍只有 8 项，没有 DeepSeek `[1m]` 选项；确认已完全退出并重开 Claude。

后续明确调整要求：DeepSeek 默认只保留 1M，不再提供 200K。此要求优先，普通／1M 双选项目标取消；其他提供商保持原配置。

## Goal

找出磁盘配置 10 项与实际桌面菜单 8 项的差异，修复实际客户端兼容问题，并核对原生菜单及上下文显示。

最终目标调整为：Claude CLI / Desktop 的已确认 DeepSeek 模型及其路由只提供 1M，既有普通 DeepSeek 默认升级到 1M，实际菜单和调用均生效。

## Scope

Claude Desktop 配置加载、模型列表处理、兼容 ID 与网关映射；必要测试、文档与实机验收。

## Out of Scope

修改 Claude 安装包、清理用户会话、无关重构、暴露客户端密钥或提示词。

# Analysis

现有受管配置已包含 Flash / Pro 普通及 1M 共四个选项，官方 CLI 隔离验证已确认 200K / 1M；该结果不能代表 Desktop 菜单渲染。截图与用户确认说明重开后的原生菜单仍不完整，需要核对客户端实际配置和处理逻辑，不继续将问题归因于未重启。

本机官方 Desktop 2.31226.1 安装包的配置转换会去掉 `[1m]` 并合并同基础 ID 的选项，因此旧方案虽然写入 10 项，启动诊断和 Code 菜单只剩 8 项。其原生 `supports1m` / `prefer1m` 字段与 `modelIdForCli` 可表达独立的长上下文选项。修复使用独立基础兼容 ID、清晰的长上下文标签与原生标志，网关同时识别后缀移除后的 ID，并保留前一方案的旧 ID 解析。

## Risks

保留已公开兼容 ID、原默认、路由和历史会话；修复需兼容客户端发送前去除后缀。若需重新应用配置，继续使用已有加密备份和版本保护。

# Implementation Plan

1. 只读核对原生配置加载和菜单代码，确认过滤／去重或配置视图问题。
2. 确认影响后修复目录适配与解析，补齐兼容回归。
3. 原生菜单与上下文验收，必要真实调用及 `task verify`。
4. 更新任务、LOCAL_AGENTS 与 CURRENT，审查并提交任务相关文件。

# Progress

- 用户已确认完全退出重开后仍只有 8 项。此前只依据磁盘配置与 CLI 验证判断菜单可用不充分。
- 已只读核对客户端安装包配置归一化、模型菜单与 CLI 模型选择逻辑；不修改客户端安装包。原生窗口已打开，准备修复后验证真实菜单。

# Implementation Result

## Changes

- Claude DeepSeek 目录及继承其能力的路由只提供 `[1m]`，旧普通默认升级为对应长上下文模型；Codex 默认及其他提供商不受该选择规则影响。
- 长上下文兼容 ID 有独立基础摘要；写入 `supports1m` / `prefer1m` 原生字段，应用与刷新使用同一类型和生成函数。
- 网关识别新兼容 ID 的完整及去后缀形式，旧普通／共享基础长上下文请求继续解析；目标仍经过原用户路由和账号选择。
- 目录、旧默认迁移、原生字段、归一化唯一性、应用／刷新／恢复与真实 HTTP 转发回归已补齐。
- 已通过 OmniProxy 现有备份／校验流程应用 CLI 默认 `deepseek-flash[1m]`。原生编辑器仍加载旧 ID 列表，与工具侧文件不同；通过 Claude 自身界面同步两条模型记录，并使用“保存并重启”生效，不修改安装包或会话文件。

## Files Changed

`internal/core/modelcatalog/claude*.go`、`internal/application/agentconfig/` 目录过滤／默认／配置序列化及测试、网关目录／转发回归、路由目录回归；LOCAL_AGENTS、CURRENT 与任务记录。

# Verification

- PASS：针对性 Core、配置服务、网关及路由测试。
- PASS：最终 `VITEST_MAX_WORKERS=4 task verify BIN_DIR=E:/go/new/wails3-v/OmniProxy/tmp/claude-context-picker/quality-bin` 退出码 0，格式、Go vet / staticcheck、ESLint、TypeScript、全部 Go 测试、69 个前端测试文件 / 366 项测试及 Windows 构建通过。用户调整为仅 1M 后重新运行了最终门。
- PASS：本机官方 CLI 2.1.295 和真实 DeepSeek，Flash / Pro 长上下文以及兼容请求均正常，客户端报告 1000000。
- PASS：实际 Claude Desktop 重启后菜单只出现 `deepseek-flash[1m]`、`deepseek-v4-pro[1m]` 两个 DeepSeek 选项，默认选中 Flash 1M。
- PASS：原生新会话发送最小验证消息，实际收到 `OK`，上下文按钮与弹层均显示 `49.4k / 1M (5%)`；验证过程中未请求读文件或调用工具。
- PASS：实际 CLI 受管设置的 `ANTHROPIC_MODEL` 为 `deepseek-flash[1m]`，原生预览显示八个模型／路由、DeepSeek 仅两种 1M。
- 真实 Pro 新 ID 通过 CLI 集成验证；原生完整对话验收为 Flash，未逐一对其他模型进行任务压测。
- 截图、质量日志与临时 UIA 脚本位于忽略目录 `tmp/claude-context-picker/`。原生菜单和上下文截图不含密钥；只读取和修改模型字段，未展示密钥。
- 开启 Claude 远程调试端口的验收命令被自动审批拒绝，理由为 `blocked by policy`，未执行；改用正常配置界面、UIA 和原生保存／重启完成验收。

# Documentation

LOCAL_AGENTS 与 CURRENT 已同步，TASK-20261013-014 追加澄清 Desktop 验证边界。无新依赖、数据库迁移或架构边界改变。

# Deviations

用户将双选项要求调整为 DeepSeek 默认仅 1M，按最新明确要求实施。原生验收额外发现客户端编辑器与工具侧文件的版本不同，不能只依据磁盘文件数量判断；最终通过原生保存／重启和真实上下文数值确认。

# Git

分支 `codex/deepseek-claude-context`。最终 diff、UTF-8 无 BOM / LF、文件长度与质量门审查通过，提交本任务代码及文档后补记实现提交号。

# Result

DeepSeek 仅 1M 的目录、默认和实际桌面调用已生效；CLI 默认已更新，原生显示 49.4K / 1M，完整质量门通过。

## Remaining Risks

未发送真实百万 token 长提示进行压测。历史会话上一次回复的用量显示可保留旧值；新会话及新的 1M 调用已实测。工具与交互用户配置视图不同的环境仍需从正常用户终端运行应用并核对实际客户端，不能只看工具侧文件。

## Completed

2026-10-13 22:17（Asia/Singapore）
