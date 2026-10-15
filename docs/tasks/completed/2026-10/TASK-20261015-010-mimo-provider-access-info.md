# TASK-20261015-010: 同步小米提供商卡片的 Token Plan 接入信息

## Status

Completed

## Created

2026-10-15（Asia/Singapore）

## Request

用户提供小米 MiMo 提供商卡片截图，反馈外层卡片仍只显示 API Key 与按量地址，没有同步已接入的 Token Plan 信息。

## Analysis

提供商卡片只根据认证方式和默认基础地址渲染；Token Plan 同样使用 API Key，不能把套餐错误建模为新的认证方式。账号层已经支持套餐及三个集群，外层展示遗漏了计费通道及地址说明。

最小方案在共享 `ProviderCard` 增加 MiMo 套餐详情浮层，沿用现有 Popover／ScrollRegion，不修改正确的认证 DTO 或实际账号转发配置。地区与地址展示复用账号表单的 MiMo service；卡片主地址及连通性测试明确标为按量，套餐模型限制和账号选择入口在浮层中说明。

## Goal

卡片明确支持 Token Plan，可查看中国／新加坡／欧洲的接入地址、个人／团队 Key 与模型边界；默认地址明确是普通按量入口，实际转发继续由账号选项决定。

## Implementation Plan

1. 补充小米卡片的套餐标识和详情浮层，复用现有 MiMo 地址生成逻辑和地区常量，保持卡片及相邻卡片布局稳定。
2. 明确普通默认入口与探测范围，保留现有认证、启停、测试及账号路由行为。
3. 回归键盘／关闭和地址显示，同步文档，运行 `task verify`，检查并提交归档。

## Implementation Result

- 共享提供商卡片在 API Key 认证标识旁新增可点击的 Token Plan 标识，展开独立 Popover 显示 cn／sgp／ams 三集群的 OpenAI、Anthropic 兼容基础地址；说明个人 `tp-`、团队 `ttp-` Key、账号选择及已确认的 Pro／Flash 模型范围。
- 明确常驻 URL 为按量入口，按钮改为「测试按量入口」，浮层说明探测范围；不修改探测回调、实际账号通道或转发配置。
- 地区列表抽为 MiMo service 的共享常量，账号选择、选项校验和外层信息复用；Anthropic 基址从现有地区地址转换，避免外层再维护独立主机列表。
- 浮层使用既有 Popover／ScrollRegion、Portal、碰撞避让和可用高度限制，打开时不向卡片布局插入明细；支持键盘打开、Esc／关闭按钮／外部点击关闭、焦点恢复与深浅主题。中英文说明同步。
- 无后端认证 DTO、接口、数据库或依赖变化；Token Plan 仍使用正确的 API Key 认证语义。

## Verification

- 针对性测试 PASS：6 文件／54 项，包含新增 6 项套餐信息测试和原提供商／模型浮层／账号表单回归。验证三集群六个基础地址、认证及模型说明、明细位于卡片外、键盘开关／焦点恢复、外部点击响应、按量测试回调保持、其他提供商不展示套餐以及英文说明。
- 完整质量门 PASS：`task verify BIN_DIR=tmp/mimo-provider-info-20261015/verify-bin`，格式、Go vet／staticcheck／ESLint、TypeScript、全部 Go 测试、80 个前端测试文件／439 项测试及 Windows 生产构建均通过。无依赖或锁文件变更，原有工具探测、jsdom scrollTo 和 Vite 大包提示不阻断。
- 原生验证 PASS：使用独立 test 数据、配置、WebView 存储和端口，在 960×600 窗口验证深浅主题。Token Plan 标识、按量入口及探测范围明确可见；浮层尺寸 420×290、边界在窗口内，滚动可查看欧洲最后一个协议地址；打开前后小米卡片电源按钮位置不变，关闭后焦点回到标识。
- 已逐张查看卡片、浮层顶部／底部和浅色浮层的关键截图，测试窗口已正常退出。没有提交 Key、修改用户账号配置、触发连通性探测或模型请求。证据位于忽略目录 `tmp/mimo-provider-info-20261015/`。
- `git diff --check`：PASS。

## Documentation

已更新 MIMO 的提供商卡片使用说明及 CURRENT。公开产品能力与后端边界未改变，无需新 ADR 或构建流程调整。

## Git

`19ffc59f774126adc141aac41659a792e265d774 fix(providers): 同步 MiMo Token Plan 接入信息`

随后补交任务归档和 CURRENT 完成状态。原有四份无关未跟踪任务文档保持原样，未纳入提交。

## Result

小米外层提供商卡片已同步 Token Plan 支持及三集群协议地址，按量默认入口与探测范围清晰区分，账号配置与实际调用行为保持；完整质量门及原生小窗口检查通过。

## Completed

2026-10-15 21:37（Asia/Singapore）
