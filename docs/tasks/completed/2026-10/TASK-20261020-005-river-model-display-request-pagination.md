# TASK-20261020-005: 优化 River 模型显示与请求历史分页文案

## Status

Completed

## Created

2026-10-20

## Request

用户提供 River 官方文档和模型截图，要求优化 River 的
`DeepSeek-V4.1-Flash`、`DeepSeek-V4.1-Flash-1017` 模型显示，其余模型不更改。
请求历史底部分页应由“第 1 页 / 1017 条”改为“第 1 页 / 共 88 页”形式，
其中总页数根据实际记录数量和分页大小计算，不固定为 88。

## Goal

- 对齐上述两个 River DeepSeek 模型的显示与官方模型标识。
- 请求历史分页正确显示当前页与总页数。
- 保持其他模型和现有请求协议兼容。

## Background

参考：https://docs.river.ai/ 及用户附图。

## Scope

- 检查 River 模型预设、显示映射及相关测试。
- 修正请求历史分页的国际化文案与传参。
- 补充回归验证与必要文档。

## Out of Scope

- 更改其余模型、添加提供商或调整核心技术栈。
- 使用真实凭证或发起收费模型请求。
- 无关重构与未提交用户文件。

## Existing Behavior

River 仅预设三款模型，原版 DeepSeek 的公共 ID 为 `deepseek-flash`，
显示名为 `DeepSeek-Flash`，缺少 1017 独立版本。请求页每页 50 条，
但将 `total` 直接传给分页文案的 `count`。

## Expected Behavior

River 显示指定的两个 DeepSeek V4.1 Flash 模型；请求历史按实际分页显示
“第 {{page}} 页 / 共 {{totalPages}} 页”，并保留独立的记录总数提示。

---

# Analysis

## Related Modules

- `internal/provider/river`：模型白名单、元数据及公共／上游 ID 映射。
- `internal/application/modelcatalog`、`internal/application/routing`：复用上述映射。
- `frontend/src/features/activity/requests-view.tsx` 及中英文分页文案。

## Dependencies

预计无新增依赖。

## Risks

保留原版 `deepseek-flash` 逻辑 ID，避免打断与直连 DeepSeek 的共享优先级。
1017 使用独立公共 ID，不与原版合并；完整官方 ID 仍可直接解析。
已同步账号继续尊重官方发现快照，不擅自将新预设混入旧快照。

---

# Implementation Plan

1. 读取相关开发规范，定位 River 模型与请求分页实现。
2. 核对模型官方标识，确定最小兼容变更。
3. 实施修正，补充模型与分页回归测试。
4. 运行针对性测试和 `task verify`，核查 diff 并记录实际结果。

---

# Progress

- 已创建任务记录，尚未修改代码或配置。
- 已核对 2026-10-20 官方公开模型文档：两款 DeepSeek 分开提供，
  1017 的上游 ID 为 `deepseek-ai/DeepSeek-V4.1-Flash-1017`，
  两者上下文均为 1048576、输出上限均为 384000。
- 官方控制参数表尚未单列 1017 的推理档位；不猜测档位，暂用上游默认。
  BrowserOS 工具未连接，使用只读公开 HTTPS 获取文档，未登录或调用推理。
- 方案：原版显示 `DeepSeek-V4.1-Flash`，保留原逻辑 ID；
  1017 公共 ID／显示名为 `DeepSeek-V4.1-Flash-1017`。其他两款映射不变。
- 分页保留每页 50 条，统一分页常量，总页数向上取整且空列表显示 1 / 1。
  1017 条应为 21 页；4351～4400 条才是 88 页。
- 新增模型／分页测试先复现旧行为失败，再实施修正并通过定向测试。
- 原生 UIA 核对工作区 `bin/omniproxy.exe`：请求页显示
  “保留 117 条，上限 10000 条；第 1 页 / 共 3 页”，提供商页 River 预览显示
  `DeepSeek-V4.1-Flash · DeepSeek-V4.1-Flash-1017 +2`，核对后恢复仪表盘。
- 首轮工作区 `task verify` 在 Go 根包测试遇到临时资源
  `wails_windows_amd64.syso` 消失；当时有 Wails 开发进程运行。
  为避免共享构建产物干扰，已将 HEAD 和本任务变更复制到独立验证目录重跑完整门禁。

---

# Implementation Result

## Changes

- River 预设和严格模型白名单增至四款，新增 1017 精确模型 ID 及已公开上下文／输出限制。
- 原版显示名补全 V4.1，保持 `deepseek-flash`、旧完整 ID、共享服务商优先级兼容。
- 1017 采用独立公共模型 ID，不替换原版，也不回退直连原版。
- 已持久化 River 目录的显示名按映射刷新，但不自动扩展账号的实际可用模型。
- 请求分页统一使用 50 条常量计算请求、偏移、按钮边界及总页数；同步中英文。
- 补充目录精确映射、发现持久化、旧快照、路由和三协议网关／计量回归，
  以及分页空态、整页、末页、21／88 页、搜索重置和英文显示测试。
- 同步 PRD、架构、River／路由／请求产品文档与当前任务看板。

## Files Changed

- `internal/provider/river/catalog.go`、`identity.go` 及相邻测试。
- `internal/application/routing/river.go` 及路由回归测试。
- `internal/application/modelcatalog/river_test.go`。
- `internal/application/gateway/river_test.go`。
- `frontend/src/features/activity/requests-view.tsx` 及分页测试。
- `frontend/src/i18n/locales/activity-{zh,en}.ts`。
- `docs/product/{PRD,RIVER,ROUTING,ACTIVITY}.md`、`docs/architecture/OVERVIEW.md`。

## Technical Notes

无新增依赖、迁移、凭据读取、真实推理或 API 行为替换。
1017 未公开单列的推理档位不推断；暂用上游默认。
既有成功发现目录保持权威，需要用户再次「同步官方模型」后才加入新版本。

---

# Verification

## Tests

- PASS：`go test ./internal/provider/river ./internal/application/provider ./internal/application/modelcatalog ./internal/application/routing -count=1`。
- PASS：`go test ./internal/application/modelcatalog ./internal/application/routing ./internal/application/gateway -run River -count=1`。
- PASS：请求与用量组件及分页两个测试文件，共 9 项测试。
- PASS：原生开发窗口请求页总页数与 River 两款型号预览；仅导航与读取，
  不发起账号同步、模型请求或更改设置，完成后恢复仪表盘。

## Quality Checks

- 首轮工作区完整门禁未通过：根包编译期间临时 Windows 资源消失；
  格式、lint、typecheck、内部 Go 测试已通过，未跳过失败步骤。
- PASS：在 `tmp/river-model-display-20261020/verify-source` 中执行完整
  `task verify BIN_DIR=../verify-bin`，exit 0；格式、Go vet／staticcheck、
  ESLint、TypeScript、全部 Go 测试、111 个文件／646 项前端测试、
  Wails 绑定生成、前端生产构建及 Windows amd64 production 构建全部通过。
- 现有 jsdom `scrollTo` 未实现、Vite 大 chunk／插件耗时提示为非阻断信息。
- 验证副本由 HEAD 加本任务修改构成，不修改工作区或已安装程序的构建产物；
  未跳过门禁步骤或删除既有测试。

## Limitations

- 未使用真实 River Key 实测新版本授权、推理或收费。
- 原生 River 预览已确认；浮层按钮未提供可用的 InvokePattern，
  不将自动化展开失败记录为浮层内容验收通过。
- 本任务不发布、打安装包或替换已安装生产程序。

---

# Git

- `3dd7426ad44f07029e43d20da4fe70c52065b390 fix: 完善 River DeepSeek 型号显示与请求分页`。
- 已检查 `git diff --check`，15 个代码／测试文件与通过完整门禁的隔离源码
  SHA256 完全一致；22 个变更文件均核对为 UTF-8 无 BOM。
- 仅本地提交，未推送，保留工作区原有不相关的未跟踪任务记录。
- 完成状态、归档和 CURRENT 更新使用后续文档提交，不再更改已验证代码。

---

# Deviations

当前每页 50 条，用户的“共 88 页”作为文案形式示例处理，不固定页数。
官方未单列 1017 推理档位，不假定其与原版完全一致。
原生浮层自动展开受 UIA pattern 限制，仅确认预览；模型完整映射由后端回归覆盖。
共享目录首轮资源缺失后，改为隔离源码执行同一完整质量门。

---

# Result

源码实现、回归、完整质量门及原生请求页／模型预览核对均完成。
两款 River DeepSeek 版本独立且保留原路由兼容，其他型号不变；
请求页展示按实际记录数计算的总页数。
未发布、打包或替换生产版；已同步账号需手动重新同步以获取 1017 型号。

# Completed

2026-10-20
