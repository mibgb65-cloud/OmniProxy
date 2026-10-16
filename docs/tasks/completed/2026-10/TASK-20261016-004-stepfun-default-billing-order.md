# TASK-20261016-004: StepFun 默认计费顺序与模型去重

## Status

Completed

## Created

2026-10-16（Asia/Singapore）

## Request

StepFun 也需要和 MiMo 一样处理：设置默认优先套餐还是普通 API Key，并合并同名模型展示。

## Goal

阶跃星辰账号页增加设置入口，支持 Step Plan／普通 API Key 默认顺序，选择后持久生效；公开目录去掉自动生成的通道副本，保留固定账号和旧显式通道兼容。

## Analysis

现有 StepFun 同时存在多个国内／海外计费池时，裸模型名报 ambiguous_stepfun_pool，目录自动生成四类通道别名。沿用 MiMo 已验证的策略：默认 Step Plan 优先，每个服务商内先筛选合格计费方式，再按创建时间／ID 稳定选择，无首选候选才用另一通道，请求失败不跨通道重试。复用设置外壳、查询／保存交互与账号选择过滤，MiMo 设置独立且行为不变。迁移 027 保存 StepFun 非敏感设置；旧通道及 Claude 编码 ID 继续通过兼容解析。保留五份既有未跟踪记录。

## Implementation Plan

1. 补充独立持久设置与绑定，复用计费顺序选择及前端设置内容。
2. StepFun 公开目录去重，扩展兼容解析与回归测试。
3. 同步产品／架构文档，运行定向与完整质量门，正常桌面窗口核验后提交归档。

## Implementation Result

- 阶跃星辰账号页新增设置按钮及局部浮层，支持 Step Plan／普通 API Key 两种默认顺序，初始 Step Plan 优先；选择即保存，读取失败可重试、保存失败保留原值。
- 迁移 027 独立保存 StepFun 默认偏好，枚举、缺失／损坏和保存失败均有明确校验，不影响 MiMo 设置；新增两项类型化 Wails 绑定并复用受管目录刷新。
- 提取 MiMo／StepFun 共用的合格计费候选筛选、前端读取／保存 Hook 与设置内容，分别使用独立查询／mutation key。候选仍遵循服务商优先级与创建时间／ID，固定账号／显式通道优先，请求失败不跨计费方式重试。
- 公共目录、网关模型发现及受管客户端目录移除自动生成的 StepFun 通道副本；兼容目录保留旧国内／海外通道与 Claude `[1m]`／桌面编码 ID。CN Step Plan 专有 Router、原生协议与账号配置版本限制不变。
- 正常桌面开发环境自动重建，原生设置双向保存、SQLite 持久值、关闭重开、深浅主题及 960×600 截图检查通过；测试后恢复 Step Plan 优先及原窗口／主题。

## Verification

- PASS：路由、网关及 Wails Go 包定向测试，含默认／持久值／非法枚举／缺失／损坏、MiMo 偏好独立、服务商优先级、停用／模型能力回落、固定账号、去重、协议元数据和旧 ID 兼容。
- PASS：网关 HTTP 集成在三种原生协议间核对实际国内 Step Plan／海外按量入口与伪凭据，设置切换无需重启；公开列表仅五款原模型，含 CN Step Plan Router，上游失败只有一次请求。
- PASS：MiMo／StepFun 设置和提供商账号页 3 文件／18 项前端定向测试，验证共用逻辑及缓存／服务绑定独立。
- PASS：正常桌面开发窗口 PID 36176，两种顺序选择后与实际开发 SQLite 值一致，关闭重开保持；深浅主题、960×600、边缘避让截图检查通过，恢复初始 step_plan。
- PASS：`VITEST_MAX_WORKERS=4 task verify BIN_DIR=E:/go/new/wails3-v/OmniProxy/tmp/stepfun-billing-order-20261016/quality-bin` 退出码 0；格式、Go vet／staticcheck、ESLint、TypeScript、全部 Go 测试、82 文件／444 项前端测试和 Windows 生产构建通过。验证产物没有替换运行中的开发版，既有工具探测／jsdom scrollTo／体积提示不影响通过。
- PASS：质量门结束后再次核对开发窗口 PID 36176，显示 Step Plan 优先，SQLite 实际值为 step_plan，3891 健康接口返回 development／ok。
- PASS：`git diff --check`，已审查设置存储、公共筛选、目录兼容和前端复用差异。

临时原生脚本第一次使用了带额外空格的控件标题，校正脚本选择器后验收通过，没有修改产品来适配脚本。质量日志、截图、原生核验脚本和报告位于忽略目录 `tmp/stepfun-billing-order-20261016/`。

## Documentation

已同步 STEPFUN、PRD、架构概览、技术基线、ADR-011 及 CURRENT 已完成概览。

## Deviations

None.

## Remaining Risks

验证使用伪凭据和受控上游，没有发送真实付费请求。官方 Step Plan 额度保持未知；外部客户端可能需要重新加载目录。

## Git

`6bbcc8eb616b8e99ce9cec1dad81bd90722c4652 feat(stepfun): 支持默认计费顺序并合并同名模型`

已审查并只提交本任务实现、生成绑定、测试及文档，保留五份既有未跟踪记录；任务归档与 CURRENT 更新另以文档提交保存。

## Result

StepFun 已增加同款默认顺序设置和公开模型去重，保留旧通道及固定账号兼容；正常桌面窗口已更新，完整质量门及原生验收通过。

## Completed

2026-10-16（Asia/Singapore）
