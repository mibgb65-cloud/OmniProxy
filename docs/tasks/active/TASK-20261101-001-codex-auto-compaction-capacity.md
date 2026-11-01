# TASK-20261101-001: 修复 Codex 模型容量丢失导致的自动压缩失效

## Status

In Progress

## Created

2026-11-01（Asia/Singapore）

## Request

用户遇到 Codex 上下文窗口耗尽错误，要求在只读诊断基础上进行根本修复，而不是仅调整本机阈值。

后续要求：暂不进行真实客户端人工验收；修复后提交代码、完善任务记录，并在 GitHub 发布补丁版本。当前 stable 为 `2.2.1`，目标版本为 `2.2.2`。

## Goal

修复订阅模型容量获取及受管 Codex 目录链路，避免依赖 App Server `model/list` 未提供的窗口字段；以可信元数据恢复自动压缩触发依据，并保留用户设置和模型隔离。

## Scope

- 订阅模型发现、容量元数据校验及 Codex 模型目录。
- 缺失／非法容量、账号隔离、既有配置兼容和原生核心自动压缩回归。
- 产品说明、任务看板与质量门验证。
- `2.2.2` 版本映射、双语发布说明、代码提交与 GitHub Actions 正式发布。

## Out of Scope

- 不扩大未经确认的上游容量，不统一硬编码 1M 窗口。
- 不读取或输出凭据、真实对话正文，不用真实大上下文消耗上游额度。
- 不修改其他任务文件，不自动安装或重启用户应用。

## Existing Behavior

当前受管 `gpt-6.1-sol` 目录缺失容量，用户全局也没有显式窗口和阈值；失败聊天窗口事件为 `null`。模型发现尝试读取 `model/list` 的 `contextWindow`，但实际 Desktop 核心协议未提供该字段。

## Analysis

- 官方 App Server 的 `model/list` 没有 `contextWindow` 字段，现有解析的空值一路进入受管目录；当前三个失败聊天均缺失窗口且未完成压缩。
- 方案比较：读取用户缓存会跨越账号边界且可能过期；读取隔离缓存仍依赖私有缓存格式、有效期与身份摘要。选择复用受保护订阅凭据访问固定官方原生 `/models`，只提取有界公开容量，并与 `model/list` 精确合并。
- 保留现有模型可见性、档位、默认项、授权版本检查和受管文件外部改动保护；不使用 `max_context_window` 扩大默认值。目录缺失或非法时明确失败，不替换旧快照。
- 发布沿用既有 GitHub Actions 构建／签名／发布流程，不更改签名凭据、历史版本或已共享标签。

## Implementation Plan

1. 阅读规范、检查订阅模型发现和隔离核心的目录来源。
2. 比较可信容量来源与缺失值处理方案，实施最小闭环。
3. 增加容量校验、目录和原生核心自动压缩回归。
4. 执行定向测试与 `task verify`，同步文档并审查提交。

## Implementation Result

- 删除对 `model/list.contextWindow` 的无效依赖。公开模型、名称、默认项和档位继续由隔离官方核心发现。
- 新增 `chatgpt.ModelContextWindows`，仅访问固定官方原生 `/models`，15 秒超时／2 MiB／1000 模型／容量数值边界，拒绝重复模型 ID；只采用默认 `context_window`，不借用用户缓存、不使用 `max_context_window` 扩大容量。
- 应用层先释放发现锁，再通过统一 `UseTokenAt` 凭据入口校验同一账号版本并读取容量，保留提供商代理作用域；按精确 ID 合并，不增加不可见模型。容量未知／非法或上游失败统一返回安全错误，既有数据库事务保留上次成功快照。
- 新增官方响应投影、安全边界、账号变更、旧空容量快照恢复及失败后保留容量回归；扩展本机 CLI／Desktop 原生离线契约，从目录容量和合成 usage 验证自动压缩，不依赖全局窗口／阈值覆盖。
- 版本源和 Windows 映射提升至 `2.2.2`／`2.2.2.0`，补齐双语发布说明及同步模型／重新加载的升级步骤。不修改用户当前配置、数据或运行中的应用。
- GitHub 发布待提交与标签推送后执行；按用户要求不做真实长聊天、真实上游摘要质量或覆盖安装验收。

## Verification

- PASS：provider/chatgpt、subscription、modelcatalog、core/modelcatalog、agentconfig 定向回归；新增提供商代理作用域断言已纳入完整质量门。
- PASS（非跳过）：CLI `0.163.0` 和实际 Desktop 核心 `0.166.0-alpha.5.1` 分别执行 `TestInstalledCodexRemoteCompactionV2ThroughProxy`，手动与自动两项全部通过。所有消息、usage、账号及上游均为隔离合成数据。
- PASS（退出码 0）：`VITEST_MAX_WORKERS=4 task --force verify BIN_DIR=bin/verify-compaction-222`，格式、lint、类型检查、51 个 Go 测试包、146 文件／890 项前端测试及 Windows production 构建全部通过。保留既有 ESLint react-refresh、jsdom scrollTo 与 Vite 大 chunk 提示。
- PASS：通过仓库既有 Windows 原生版本 API 读取本地 quality-gate 应用，`ProductVersion=2.2.2`、数字版本 `2.2.2.0`、产品与公司均为 OmniProxy；未把本地构建当作公开安装器。
- PASS：`git diff --check`；新增主要源码及发布说明 UTF-8 无 BOM、LF 检查。
- 真实客户端／真实大上下文／覆盖安装：按用户要求暂不执行，不能将离线合成 usage 契约等同于真实窗口容量验证。

## Documentation

- 已更新：Architecture、Testing、LOCAL_AGENTS、UPGRADE、CURRENT 及 `docs/releases/v2.2.2.md`。
- 不新增 ADR：只修正既有订阅目录的容量来源并复用原凭据／代理／持久化边界，无新依赖、迁移或核心架构方向变化。

## Deviations

- 升级修复沿用显式「同步模型」和既有受管文件刷新，不新增自动联网迁移或覆盖用户配置的后台任务。
- 用户后续明确要求公开发布，原仅修复范围扩展为 `2.2.2` 正式补丁发布；人工验收按要求延期。

## Git

待完成差异审查及提交。

## Result

进行中。
