# TASK-20261017-019: 已接入模型卡片与百万上下文筛选

## Status

In Progress

## Created

2026-10-17（Asia/Singapore）

## Request

已接入模型增加一键筛选 [1m] 的按钮，列表改为紧凑卡片并按空间自动分配每行数量；移除接入配置区的网关未启动提示。

## Analysis

AgentConnectedModels 当前按行展示实际受管目录，仅支持文字搜索；AgentClientSettings 的 running 属性只用于停止提示，并通过多层组件透传。

## Implementation Plan

1. 增加可切换的「仅看 [1m]」按钮，与搜索取交集，按真实 ID 的精确后缀筛选，不推测模型能力。
2. 使用紧凑自适应卡片网格，保留名称、ID、默认模型和滚动；筛选后少量卡片不拉伸为整行。
3. 移除指定提示及孤立的 running 透传／翻译，同步现有行为测试，验证宽窄窗口、中英文和大字号，运行质量门并提交归档。

## Implementation Result

- 已接入模型改为紧凑卡片，保留名称、真实 ID 和默认模型标记；网格使用 auto-fill，根据容器与字号自动分配列数，少量结果仍保持原列宽，长文本换行且不横向溢出。
- 搜索旁新增可按下／取消的「仅看 [1m]」按钮，以真实 ID 的精确后缀筛选，与大小写不敏感搜索取交集，保留搜索输入；增加当前显示数／总数播报和无匹配状态。仅改变展示，不写入模型选择或客户端配置。
- 移除用户指定的网关未启动提示、中英文孤立翻译及只为该提示透传的 running 属性；停止状态下预览／保存及原安全限制保持正常。

## Verification

- PASS：已接入模型／客户端应用／工作区定向测试共 3 个文件／22 项及 TypeScript；新增中英文后缀精确筛选、搜索交集、取消保留输入、默认标记、无匹配与空格操作，保留九种客户端停止时配置能力验证。
- PASS：最新开发版原生真实 36 项目录，1920／2560／960／960+140% 字号分别核对为 3／5／2／1 列；深浅主题无横向越界。[1m] 快筛、搜索叠加、取消恢复全部、单卡筛选后不拉伸均通过；指定停止提示不存在。未写入真实客户端配置或发起模型请求。
- PASS：`VITEST_MAX_WORKERS=4 task --force verify BIN_DIR=tmp/connected-model-cards-20261017/verify-bin` 退出 0；格式、Go vet／staticcheck／ESLint、TypeScript、Go 测试、93 个文件／525 项前端测试、前端与 Windows 生产构建通过。只有既有 scrollTo 测试环境与 chunk 体积提示。
- PASS：`git diff --check`。验证日志与截图位于忽略目录 tmp/connected-model-cards-20261017/，保留开始前五份无关未跟踪记录。

## Documentation

已同步 LOCAL_AGENTS 与 CURRENT；未改变产品边界、接口、技术栈或构建流程，无需 ADR。

## Remaining Risks

[1m] 筛选只匹配该显式模型 ID 后缀，不推断其他模型的上下文容量；不表示上游实际调用已验收。

## Git

待提交。

## Result

紧凑卡片、[1m] 快筛和停止提示移除均已完成；开发版已更新，定向测试、完整质量门与原生验证全部通过。
