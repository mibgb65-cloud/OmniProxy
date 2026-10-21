# TASK-20261022-002: 恢复 Codex GPT 模型 Fast 闪电入口

## Status

Completed

## Created

2026-10-22

## Request

用户要求修复 TASK-20261022-001 已定位的缺陷：新版 OmniProxy 重建模型目录
丢失服务档位，导致 Codex GPT 模型旁的 Fast 闪电按钮消失。

## Goal

保留官方确认的 GPT 服务档位，使受管 Codex 目录恢复 Fast 能力声明；
不伪造第三方能力，不强制开启 Fast，不修改真实客户端配置或凭据。

## Scope

- 官方模型能力采集、持久元数据与 Codex 目录输出。
- 原生模型、别名及订阅 Responses 转发回归测试。
- 文档、任务记录与质量门禁。

## Out of Scope

- 真实付费 Fast 调用、账号／授权变更、真实客户端配置写入。
- 自动安装／重启正在运行的客户端或发布版本。
- 工作区已有的其他任务改动。

# Analysis

继承 TASK-20261022-001 的结论。本机官方核心生成的
`v2/ModelListResponse.json` 确认 `serviceTiers` 是
`{id,name,description}` 数组；旧 `additionalSpeedTiers` 已标记 deprecated。
采用官方 RPC 字段，不读取用户默认缓存、不硬编码模型名单、不推断账号权限。

复用现有 `metadata_json` 持久化，无新增迁移或依赖；仅在原生 OpenAI
Responses 目录输出确认的服务档位。验证档位数量、ID、展示名称和描述边界，
旧数据缺失字段继续作为未知能力，不自动启用付费模式。

# Implementation Plan

1. 核对目录来源与字段契约，选择最小兼容修复。
2. 增加有界能力模型并贯通采集、持久化及输出。
3. 覆盖能力保留、未知／第三方隔离、别名和请求参数透传回归。
4. 执行定向测试、真实内核离线契约验证和 `task verify`。
5. 更新受影响文档，检查差异并记录交付限制。

# Implementation Result

## Changes

- 增加有界 `ServiceTier` 元数据，校验最多 8 档、唯一的 ASCII ID
  （32 字节）、名称（80 字节）和描述（500 字节），拒绝非法类型及不安全控制字符。
- 官方 RPC 解析 `serviceTiers`，复用账号版本校验和 `metadata_json` 快照；
  同步失败保留旧能力，无迁移、依赖或默认缓存读取。
- 原生 OpenAI Codex 目录输出 `service_tiers`，第三方和未知能力不声明 Fast；
  固定账号别名保留实际目标账号的能力，不改变默认模型或服务档位选择。
- 重新生成两个 core/modelcatalog Wails 类型绑定；其他任务的 appservice
  绑定保持字节不变。
- 回归覆盖目录导出、输入边界、RPC、数据库快照、别名、受管目录刷新和原生
  Responses 中 `priority`／`default`／缺省档位透传。
- 补充本地 Agent 使用说明：旧快照需要重新「同步官方模型」，运行中客户端
  需重载模型菜单；没有改写真实客户端配置、运行程序或发布版本。

# Verification

## Tests

- PASS：六个受影响 Go 包定向测试。
- PASS：Fast／ServiceTier 相关测试重复 10 轮。
- PASS：用 Go overlay 临时恢复旧版采集／导出实现，新回归稳定复现
  “Fast capability was lost”及“official service tiers dropped”；不修改工作区源码。
- PASS：`OMNIPROXY_CODEX_CATALOG_QA=1` 的真实内核隔离契约测试，
  CLI `0.163.0` 和 Desktop 包 `26.1016.6573.0` 内核
  `0.163.0-alpha.11` 均保留 Fast 档位名称、ID、描述，未知模型不增加档位。
  仅临时 HOME／虚假 Key／本地目录，未发推理请求或继承真实登录。

## Quality Checks

- 当前工作区 `task verify`：格式、Go／前端 lint 通过，前端 typecheck
  被原有未提交的关于页测试阻断。`about-view.test.tsx:118` 及
  `project-resources.test.tsx:40,52,54,72,85` 的 `getByRole`
  传入不属于 `ByRoleOptions` 的 `exact`；未修改这些无关文件。
- PASS：在 `tmp/codex-fast-clean-verify-20261022/` 导出 HEAD，只复制本次
  14 个产品／测试／生成绑定／说明文件，执行完整 `task verify`：
  format → lint → typecheck → 全部 Go 测试 → 118 文件／706 项前端测试 →
  Windows production 构建全部通过。
- PASS：逐文件 SHA256 确认验证副本与本次修复一致；原有其他任务改动、
  真实 Codex 配置和受管目录的受保护基线均未变化；`git diff --check` 通过。
- 构建仅位于隔离目录 `bin/omniproxy.exe`，22358016 bytes；未替换原 bin、
  制作安装包、自动运行、安装、推送或公开发布。

## Remaining Limitations

未重启真实 Desktop UI 验收闪电按钮，也未发起真实 Fast 调用。
离线契约证明目录能力被真实内核保留，不证明上游实际提速、账号授权或额度比例。
当前工作区完整门禁的无关类型错误仍保留，不能将隔离验证描述为整个脏工作区已通过。

# Documentation

- 更新 `docs/product/LOCAL_AGENTS.md` 的 Fast 能力来源、兼容边界和升级后同步步骤。
- CURRENT 已添加完成摘要；现有其他任务状态不覆盖。
- 无技术栈、架构边界或依赖变化，不新增 ADR。

# Deviations

主工作区门禁因原有其他任务的类型错误停止，改用 HEAD 加本任务精确文件清单
的干净副本完成全门禁；未删除测试、跳过检查或顺手修复无关改动。

# Git

`4420c051f45c3ecbbcd43f0b4c1189ff771e02a7`
`fix(codex): 恢复原生模型 Fast 服务档位`

本地提交只包含本任务的 16 个文件；既有其他任务改动未暂存或提交。未推送。

# Result

Fast 能力采集与导出链路已修复，专项回归和两种官方内核契约通过。
运行中的旧程序和真实受管目录未自动更新；使用修复版本后重新同步官方模型。

# Completed

2026-10-22
