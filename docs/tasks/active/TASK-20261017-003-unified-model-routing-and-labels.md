# TASK-20261017-003: 统一 River 模型展示与路由归组

## Status

In Progress

## Created

2026-10-17（Asia/Singapore）

## Request

- 将长型号 `nvidia/NVIDIA-Nemotron-3.5-Super-VL-120B-A12B-NVFP4` 简化展示为 `NVIDIA-Nemotron-3.5`。
- River DeepSeek V4.1 Flash 与直连 DeepSeek Flash 合并为同一模型，在路由页配置服务商优先级。
- 直连 DeepSeek Flash、Pro 恢复普通和 `[1m]` 双选项。
- 用户补充：GLM-5.3-Flash 也需要处理。

## Goal

让模型列表更易读，同款模型共用逻辑路由及优先级；实际请求仍使用所选服务商的官方模型 ID，旧 ID／别名／客户端配置保持可解析。

# Analysis

- Nemotron 统一短名称／逻辑 ID 为 `NVIDIA-Nemotron-3.5`；GLM 为 `GLM-5.3-Flash`；River Flash 归入既有 `deepseek-flash`。精确白名单映射，实际发送官方原始 ID。
- 仅改标签不能让默认映射共用优先级，因此在既有路由选择完成后再还原所选服务商型号；必须同时修复网关只在显式别名命中时改写正文的问题。
- 公共目录去重，兼容目录保留旧 River 完整 ID、旧 `[1m]` 与 Desktop 摘要；显式规则及固定账号继续优先。既有账号快照原始 ID 保持不变，读取时派生短名称，不需要重新同步账号。
- 追加迁移 030 将旧 River 优先级归入新逻辑模型，已有逻辑配置优先；不覆盖账号、凭据或显式规则。
- DeepSeek 普通选项被 `ClaudeEntries` 的仅 1M 条件移除；本次恢复普通／1M 双选，已实测订阅目标的独立规则保持不变。
- 智谱直连适配器尚未实现，不虚构 GLM 的直连候选；此次仅统一 River GLM 的短名称与逻辑目录，为实际已有候选提供优先级。

# Implementation Plan

1. 核对现有逻辑模型归一化和长上下文选项规则，明确兼容边界。
2. 最小扩展模型名称及 River 到逻辑模型的映射，贯通优先级、实际转发、客户端目录。
3. 恢复 DeepSeek 双选项，保留已验证订阅目标的独立长上下文规则。
4. 增加路由／网关／客户端菜单回归，运行质量门并完成原生验收。

# Implementation Result

## Changes

- River 精确型号映射集中在 `identity.go`：公共 Flash 为 `deepseek-flash`、Nemotron 为 `NVIDIA-Nemotron-3.5`、GLM 为 `GLM-5.3-Flash`；提供商卡片和账号发现／旧缓存读取使用短名称。
- 默认映射和可调用目录以逻辑型号归组，Flash 可同时选择直连 DeepSeek 和 River；选定账号后还原上游原始 ID，并在网关默认映射中改写请求正文。
- 追加迁移 030 合并旧优先级，保留用户已有逻辑配置；显式规则、固定账号、原始型号请求及 Claude 旧长上下文／Desktop 摘要继续解析，兼容条目不重复发布。
- 恢复 DeepSeek Flash／Pro 普通与 `[1m]` 双选，已验证订阅目标仅 1M 行为维持；受管 Claude／Desktop／Codex 菜单刷新保留同款默认，避免归一化后选中无关模型。
- 前端目录／新增规则使用逻辑型号；既有 River 原始 ID 规则在不改变原目标时仍可编辑保存，后端继续验证账号资格，避免归并后旧规则被界面阻断。
- 沿用账号保护、缓存版本、请求统计及无隐式重试语义，无新依赖或新数据表。

## Files Changed

- `internal/provider/river/identity.go`、`migrations/030_river_logical_models.sql`。
- `internal/application/{provider,modelcatalog,routing,gateway,agentconfig}/` 及相关回归测试。
- `internal/core/modelcatalog/claude.go`。
- 前端路由编辑器及模型目录／路由回归测试。
- River、路由、本地客户端、PRD、架构说明、CURRENT 与本任务记录。

# Verification

- 定向 Go：routing、modelcatalog、agentconfig、gateway、Core modelcatalog、River 适配器全部通过；覆盖公共去重、两个服务商排序、停用后选次序候选、固定账号优先、真实上游 ID 改写、原始／1M／Desktop 摘要兼容、普通／长上下文双选，以及三个客户端受管菜单刷新后默认型号保持。
- 迁移回归：从 029 升级，旧 GLM／NVIDIA 偏好归入短 ID、已有逻辑 Flash 偏好优先、重复启动不恢复旧条目或覆盖配置。
- 前端定向 2 文件／16 测试通过，覆盖逻辑型号创建规则、GLM 短名称目录同步、旧 River 原始规则编辑保存；补充该兼容路径后再次执行完整质量门。
- 最终 `task verify BIN_DIR=tmp/model-identity-20261017/verify-bin`：PASS，退出码 0；格式、Go vet／staticcheck／ESLint、类型检查、Go 全套、前端 88 文件／474 测试、Windows 生产构建全部通过。绑定重新生成后无变化，无新依赖。
- 原生开发窗口按进程路径核对：NVIDIA／GLM 短名称已显示；Flash 默认映射只保留一行，展示 DeepSeek／River 两个可用服务商与优先级入口。仅核对公开目录，不发起推理；随后打开客户端接入页触发现有受管目录刷新流程，不强制覆盖外部改动。
- 本机证据位于忽略目录 `tmp/model-identity-20261017/`：最终质量门日志、`native-verification.json` 与 `flash-unified-route.png`。真实 Claude 进程未重启，客户端缓存菜单需重新加载。
- `git diff --check`：PASS；修改使用 UTF-8 无 BOM／LF，其他五份未跟踪任务记录保留原样。

# Documentation

同步 River、路由、客户端及受影响规范和任务看板。

# Remaining Risks

智谱直连仍是未实现适配器，不宣称 GLM 已有两个可用服务商。真实长窗口、图片和推理费用并未通过本次目录调整完成验收；不改变这些既有能力边界。运行中的客户端可能缓存模型菜单，需重新加载；已有非受管／外部改动配置仍受保护，不强制覆盖。

# Git

待提交。

# Result

待完成。
