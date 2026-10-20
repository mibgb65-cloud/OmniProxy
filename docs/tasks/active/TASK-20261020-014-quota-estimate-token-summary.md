# TASK-20261020-014: 精简额度估算并展示账号 Token 用量

## Status

In Progress

## Created

2026-10-20

## Request

移除额度卡片中的“本机样本参考，非官方余额”折叠内容；美元估算只显示一个具体值，不显示范围；5h 与一周估算同行展示；在下方新增账号 Token 用量块，包含输入、输出、缓存；核对模型官方价格。

## Goal

在现有账号资源卡片内提供紧凑、清晰且不误导的美元估算和真实 Token 统计，不将本机统计当成账号全平台用量。

## Scope

- 账号额度卡片、单值估算、Token 聚合和必要的数据契约。
- 相关模型官方价格核对、关键回归测试及任务文档。

## Out of Scope

- 不读取凭证或真实请求正文，不修改生产账号和生产用量。
- 不变更核心技术栈，不新增大型依赖，不修改 Atria 草稿和其他任务。
- 不自动打包安装程序、部署或发布。

# Analysis

- 现有美元估算已按账号／成员／工作区和独立窗口时刻隔离，并提供最小／最大值。保留旧字段保证兼容，新增明确的单值字段；缓存明细未知时不计缓存折扣，不采用缺乏依据的中点。
- 请求表留存上限默认 10,000 条，长期聚合表没有缓存明细；本次使用同身份留存请求聚合输入／输出／缓存，明确标注“本机留存”，不伪称全生命周期或全客户端统计，不新增迁移。
- 复用现有额度估算 RPC 返回 Token 聚合；不依赖上游额度成功，以免无额度快照时看不到本机用量。
- 已实际读取 OpenAI 与 Anthropic 官方价格页。当前 Codex 核心模型价格与现有表一致；其他厂商价格不影响本次仅 OpenAI 的估算入口，不进行无关批量改价。
- 保持现有样本门槛、未知模型／不完整用量状态和过期窗口保护；移除指定折叠说明，简短估算口径保留于悬浮提示和产品文档。

# Implementation Plan

1. 检查 TASK-012 的卡片、价格目录、缓存统计及窗口隔离实现。
2. 确认官方价格，以最小兼容改动实现紧凑估算和账号 Token 用量块。
3. 补充回归测试，运行质量门并同步受影响文档。

# Implementation Result

## Changes

- 移除指定折叠说明；5h／一周美元估算同行分两列显示具体总额和剩余值，保留 `≈` 及简短标题提示。
- 新增单值 `total`／`remaining` 字段，缓存明细未知时不计折扣；旧区间字段保留兼容，原有窗口／重置／观察时刻和 5% 门槛不变。
- 下方新增输入、输出、缓存三列 Token 块，紧凑单位配合完整整数提示；明确缓存已包含于输入和“本机留存”范围。
- 账号用量按提供商、本地账号、规范邮箱和工作区隔离，复用成员索引；覆盖留存请求而非仅当前额度窗口，不依赖模型定价。
- 保留未知、零和部分缓存的区别；无明细显示 `—`，部分缓存显示已知量的下界，未完成请求提示统计不完整。
- 复用现有 RPC，首次没有官方快照时也返回本机用量；退出登录不显示统计。本地请求变化触发读取，后台跳过、恢复可见后补读。
- 核对 OpenAI 主要 Codex 型号官方标准短上下文价，与现有参考价一致，增加固定回归；未进行无关厂商批量调价。

## Files Changed

- `internal/application/activity/quota_estimate.go`
- `internal/application/activity/account_token_usage.go` 及对应测试
- `internal/transport/wails/subscription_estimate.go` 及对应契约测试
- `internal/core/tokenprice/prices_test.go`
- `frontend/src/features/accounts/subscription-quota-usd.tsx` 及对应测试
- `frontend/src/features/accounts/account-token-summary.tsx` 及对应测试
- 中英文额度文案、自动生成的 activity 类型绑定
- `docs/product/PRD.md`、`docs/product/SUBSCRIPTION.md`、`docs/plans/CURRENT.md`

## Technical Notes

不新增迁移、依赖、上游模型调用或公开方法。Token 块不是全客户端／全生命周期累计账单；清理留存请求会减少这些数值。缺失缓存时的单值可能高于真实 API 等值成本，这是一致的估算规则而非已知缓存为零。

# Verification

## Tests

- Go：`go test ./internal/application/activity ./internal/core/tokenprice ./internal/transport/wails` 通过。
- 聚合回归覆盖身份／提供商隔离、窗口外请求、未知价格、历史清理、未知／零缓存、未完成请求和部分输出。
- RPC 契约覆盖无官方快照、无真实授权、已退出登录及非法 ID。
- 前端覆盖单值／双列、移除折叠内容、未知价格／过期窗口、快照错配、NaN／Infinity／负值、Token 精确大整数及缓存缺失、后台更新与恢复可见。

## Quality Checks

- [x] Format
- [x] Lint
- [x] Type Check
- [x] Test
- [x] Build
- [x] Verify

## Verification Result

`task --force verify BIN_DIR=tmp/task-014-build` 完整通过：格式、lint、类型检查、Go 测试、116 个前端测试文件／693 项测试及 Windows production 构建。

最初 `task verify BIN_DIR=tmp/task-014-build` 已通过检查和测试，但在构建准备阶段长时间未进入绑定生成；核对 PID 和命令后停止本任务进程，改用 `--force` 重跑同一质量门完整通过。该参数跳过增量缓存判断，不跳过检查或测试。最终可见性补读改动另做 Prettier／ESLint 检查，且包含在最终完整前端测试及构建中。

原生合成账号验收：

- 独立 test profile／数据目录／WebView 目录；两笔合成用量，无真实授权令牌、无真实推理请求。
- 深浅主题，1200×800 与 960×720；人工查看截图并核对原生控件位置，5h／周标题同一水平线，Token 三列无横向溢出。
- 5h 总额约 `$4.35`／剩余约 `$2.87`；周总额约 `$37.44`／剩余约 `$34.07`。
- 输入 `2M`、输出 `200K`、缓存 `1.6M`，缓存未重复计入输入。
- 合成账号缺少真实授权，官方额度读取显示登录提示并保留缓存，符合预期；Token 区块继续显示本地统计。
- 截图过程曾遇窗口最小化，已增加恢复与尺寸校验后重跑并逐张查看有效截图；未将 160×28 的无效截图视为通过。
- 验收进程已按精确路径关闭，未停止生产进程，未替换正式安装程序。临时日志和截图位于忽略目录 `tmp/task-014-native/`。

现有非阻断提示：jsdom 的 `scrollTo` 未实现、前端大 chunk 警告；本次未进行无关优化。

# Documentation

已同步 PRD、订阅产品说明和 CURRENT；无技术栈、架构边界、构建规则或 ADR 变更。

# Deviations

官方核对的现有主要 Codex 参考价无需改变数值；用价格回归固定核对结果。为保护正在运行的开发版，质量门输出至独立构建目录，不覆盖 `bin/omniproxy.exe`。

# Git

已审查本任务差异，待本地实现提交；不提交既有未跟踪的 Atria 草稿，不推送远端。

# Result

实现、完整验证、原生深浅主题／最小窗口验收及产品文档已完成，等待本地提交后归档；未打包安装程序、安装或发布。
