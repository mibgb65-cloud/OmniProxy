# Current Development Status

当前状态看板。完整任务信息见 `docs/tasks/`（Source of Truth）。

## Current Version

2.0.0-beta.2 — 已公开 Windows amd64 [Pre-release](https://github.com/mibgb65-cloud/OmniProxy/releases/tag/v2.0.0-beta.2)，包含历史统计库启动冲突、图标和正式渠道修复；CI／Release／公开下载及 SHA256 验证通过。应用／前端使用完整 SemVer，Windows 数字版本 2.0.0；详见 `product/UPGRADE.md`。

## In Progress

- [TASK-20261018-013](../tasks/active/TASK-20261018-013-fix-codex-isolated-forwarding.md)：隔离修复 Codex WebSocket 预热被误转发、上游错误状态丢失及完成事件等待 EOF；不修改真实 Agent 配置或生产运行状态，正在回归验证。

## Recently Completed

- [TASK-20261018-011](../tasks/completed/2026-10/TASK-20261018-011-publish-v2-beta2.md)：`5de94cf`／v2.0.0-beta.2 已推送并公开预发布；原生 Beta.2 安装／升级／重启／卸载、CI `39119451562` 与 Release `39119455141` 均成功。公开安装器 19,255,511 bytes／SHA256 `0b4f4e56dc7f1c8a781a1c13fa0913e6f0ea124f8de2c837858346b658e4ce97`，默认更新客户端确认 Beta.1 可发现并下载校验 Beta.2；正式 Latest v1.3.1 与 Beta.1 原资产保留。
- [TASK-20261018-010](../tasks/completed/2026-10/TASK-20261018-010-fix-installed-startup-and-shortcuts.md)：历史统计库与迁移 008 冲突已定位，本机备份后隔离数据恢复；已实现只读布局识别／旧库保护／有效 v2 沿用／可见失败及日志落盘／独立快捷方式 ICO。最终完整质量门、575 项前端测试、用户级隔离安装／运行中升级／重启／卸载及错误提示通过；生产安装器原生资源校验通过。修复 `7085fe4` 已推送，公开 Beta.1 添加已知问题提示且资产未变，Beta.2 继续交付。
- [TASK-20261018-009](../tasks/completed/2026-10/TASK-20261018-009-publish-v2-beta1.md)：首个 v2.0.0-beta.1 已公开预发布，标签应用代码 `18cd42f`；干净环境质量门、远端 CI、源码一致性守卫、自动 NSIS／SHA256／发布及实际下载校验通过。旧 master／v1.3.1 Latest 保留。修复 `14edf51`／`18cd42f`／`ce89b68`；正式渠道编码问题在 Beta.1 说明中列出，main 修复 `38184e9` 通过定向／在线／完整验证，随后续测试版交付；默认 Beta 渠道正常。
- [TASK-20261018-008](../tasks/completed/2026-10/TASK-20261018-008-installer-and-in-app-updates.md)：安装器／下载进度／SHA256／取消重试／静默重启、侧栏升级标记和版本浮窗、Beta Release 工作流完成。完整质量门／575 项前端测试、用户级和机器级安装／运行中升级／重启／卸载、v1 覆盖、首次完成页普通用户启动与原生深浅／960 验收通过，最终生产安装包资源／SHA256 校验通过。实现 `cae895f`，启动身份修复 `5107c8f`；未操作真实账号／正式安装或发布远端，代码签名尚未接入。
- [TASK-20261018-007](../tasks/completed/2026-10/TASK-20261018-007-v1-import-and-beta-updates.md)：旧版只读 API Key 预览／导入、快照复核／去重重试及正式／Beta 检查；应用版本 2.0.0-beta.1，Windows 数字资源同步。完整质量门／571 项前端测试、隔离原生深浅／960、最终 Windows 构建与安装器编译通过。实现提交 `c77f3d1`；订阅、未知计费等迁移及手动安装限制见 UPGRADE，未操作真实账号或发布。
- [TASK-20261018-006](../tasks/completed/2026-10/TASK-20261018-006-sync-latest-main.md)：main 从 `558cdba` 快进至 `a1b4275`，完整纳入当前开发分支 68 个提交并切换至 main。完整质量门、102 文件／567 项前端测试与 Windows 生产构建通过；无额外业务／生成文件变更，五份既有未跟踪记录保持不变。本地同步，无远端推送。
- [TASK-20261018-005](../tasks/completed/2026-10/TASK-20261018-005-account-failover.md)：同服务商／认证／计费／地区账号切换，默认开启、最多三次；固定账号／200 后不重放、冷却与凭据保护、一次请求／多次尝试记录及原生开关／详情。完整质量门、102 文件／567 项前端测试、最终 Windows 构建、四组隔离原生宽窄／深浅主题验收通过，普通开发版已更新。实现提交 `6d6f8d0`；Google 429／未识别失败保守返回，未用真实付费账号验收。
- [TASK-20261018-004](../tasks/completed/2026-10/TASK-20261018-004-failover-design.md)：完成自动故障切换方案分析，用户确认首版只在同服务商同计费通道内切换；明确候选来源、固定账号、200 后不重放、冷却与尝试级记账、分期及测试清单。分析提交 `8c49d2c`；功能实施见 TASK-20261018-005。
- [TASK-20261018-003](../tasks/completed/2026-10/TASK-20261018-003-beta-readiness-upgrade-wails.md)：Wails Go／JS／CLI／CI 统一 beta.30，修复 NSIS 输入编码；完整质量门／564 项前端测试、Windows 构建、隔离原生与安装包编译通过，普通开发窗口已更新。实现提交 `695ae19`；建议进入小范围 Beta，实际安装／升级／卸载、客户端回归、版本／发布信息及更新渠道边界见任务记录。
- [TASK-20261018-002](../tasks/completed/2026-10/TASK-20261018-002-add-four-agents.md)：接入 Kimi Code、Pi、Factory Droid、Goose CLI／Desktop，增加模型选择／默认值、独立 Key、双文件备份回滚和固定官方软件入口；兼容 Kimi 旧 Python 配置。完整质量门／564 项前端测试／Windows 构建通过，原生深浅／960 预览和同名冲突保护通过。实现提交 `8a8679e`；实际客户端调用待验收，Pi 既有同名配置保持受保护。
- [TASK-20261018-001](../tasks/completed/2026-10/TASK-20261018-001-time-range-selection.md)：网关／仪表盘时间范围统一为高对比选中卡片，补齐间距、阴影和换行；559 项前端测试、完整质量门与 Windows 构建通过，正常原生深浅主题／三时段／960 窗口及可访问焦点共 14 项核对通过。实现提交 `c2b317e`。
- [TASK-20261017-026](../tasks/completed/2026-10/TASK-20261017-026-gateway-request-metrics.md)：网关新增六项指标、趋势、最近记录筛选／详情、累计上游分布和接入信息；迁移 032 记录实际下游首字及独立聚合，旧数据／非流式未知、真实零有效。559 项前端测试、Go／转发集成、完整质量门和 Windows 构建通过；原生深浅主题、1920／960、时段和详情验收通过，未调用真实模型。实现提交 `8351157`。
- [TASK-20261017-025](../tasks/completed/2026-10/TASK-20261017-025-fix-agent-update-failure.md)：定位更新前第二次查询失败，增加五分钟服务端目标复用和实际错误解析；保留 WinGet 退出码，真实重试确认商店无适用更新后提供固定官方下载入口。551 项前端测试、完整质量门及 Windows 构建通过，原生后备按钮 `101×32`；Claude 仍为 `2.37337.1.0`，未误报升级。实现提交 `dc169d7`。
- [TASK-20261017-024](../tasks/completed/2026-10/TASK-20261017-024-agent-update-action.md)：发现新版后同一按钮切换「更新」，显式点击后执行固定商店包后台升级，复用进度／互斥／取消并核对实际版本；失败和旧版保留不会误报成功。541 项前端测试、完整质量门和 Windows 构建通过，原生深浅主题确认按钮切换及 `101×32` 稳定尺寸，未实际升级本机软件。实现提交 `e35b294`。
- [TASK-20261017-023](../tasks/completed/2026-10/TASK-20261017-023-agent-update-check.md)：Claude Code Desktop 增加按需官方版本检查，固定加载尺寸并独立显示新版本／暂无／未知／失败，不下载安装包。537 项前端测试、完整质量门、Windows 构建和正常桌面深浅主题／960 窄窗口通过，实测发现 `2.42304.0`；其他产品更新渠道未扩展。实现提交 `8d929da`。
- [TASK-20261017-022](../tasks/completed/2026-10/TASK-20261017-022-remove-config-hints.md)：移除配置底部重启提示与目录刷新说明，清理对应中英文翻译；7 项定向测试、类型／格式检查及原生 36 项配置页面核对通过，开发版已更新。实现提交 `1623511`。
- [TASK-20261017-021](../tasks/completed/2026-10/TASK-20261017-021-config-model-cards.md)：配置和展示共用紧凑模型卡片与网格，配置增加整卡勾选、默认标记、上下文和 [1m] 快筛；隐藏选择与默认模型保留，全选／清空仍作用完整目录。531 项前端测试、完整质量门、Windows 构建及开发版原生宽窄窗口／深浅主题／勾选与筛选核对通过，未应用真实客户端配置。实现提交 `4e21a35`。
- [TASK-20261017-020](../tasks/completed/2026-10/TASK-20261017-020-gateway-action-feedback.md)：后台状态同步与刷新按钮反馈解耦；主按钮／侧栏共享约 650ms 成功反馈、220ms 淡入淡出与慢速圆环，实际请求／缓存立即执行，失败不等待动画。529 项前端测试、完整质量门和 Windows 构建通过；原生双入口采样 362 次，刷新全程不闪且可用，启停可见 622–639ms，尺寸位置不变并恢复原停止状态。实现提交 `1625cba`。
- [TASK-20261017-019](../tasks/completed/2026-10/TASK-20261017-019-connected-model-cards.md)：已接入模型改为紧凑自适应卡片，新增可叠加搜索的 [1m] 精确后缀快筛与显示数量；移除接入配置区的停止提示及孤立透传。完整质量门／525 项前端测试与 Windows 构建通过，开发版原生 1920／2560／960／140% 对应 3／5／2／1 列、单卡不拉伸、深浅主题和筛选验证通过。实现提交 `40af711`。
- [TASK-20261017-018](../tasks/completed/2026-10/TASK-20261017-018-stable-gateway-buttons.md)：网关启停／刷新按钮按全部状态文案预留尺寸，局部限制过渡属性，消除点击时大小变化；中英文延迟操作、522 项前端测试、完整质量门与 Windows 构建通过。开发版原生采样 312 次覆盖全部启停／刷新状态，宽高与相邻水平位置变化均为 0px，网关恢复原停止状态。实现提交 `d10db4e`。
- [TASK-20261017-017](../tasks/completed/2026-10/TASK-20261017-017-agent-detail-model-panel.md)：Agent 详情页左侧安装／接入管理，右侧展示实际受管模型目录并内联配置；自动预览、保存／取消返回列表、搜索／刷新／默认标记、空状态和停用提示，窄容器上下排列。九种客户端读取／版本保护、93 个文件／520 项前端测试、完整质量门及 Windows 构建、开发版原生深浅主题／960×600／140% 验收通过。实现提交 `896b3ee`。
- [TASK-20261017-016](../tasks/completed/2026-10/TASK-20261017-016-routing-view-toggle-position.md)：默认映射视图切换移到右上角模型数量下方，保持右对齐；宽窄窗口／深浅主题／140% 字号无重叠或越界，完整质量门／514 项前端测试及 Windows 构建通过，开发窗口已更新。实现提交 `ab0dadf`。
- [TASK-20261017-015](../tasks/completed/2026-10/TASK-20261017-015-routing-card-grid.md)：默认映射模型／服务商视图使用自适应并排卡片，1920／2560 实测 4／5 列、收起侧栏可到 6 列；筛选不拉伸单卡、宽窄窗口／深浅主题／140% 字号／原生键盘及完整质量门／514 项前端测试通过，开发窗口已更新。实现提交 `1a5adf7`。
- [TASK-20261017-014](../tasks/completed/2026-10/TASK-20261017-014-agent-model-selection.md)：一键配置默认全选，支持模型搜索、逐项选择、全选／清空、已选数量及默认模型；保存选择并在目录刷新／重启后保留，恢复时清除。九种客户端与边界／事务回滚测试、两轮完整质量门／514 项前端测试、Windows 生产构建及最新开发版原生深浅主题／960×600／键盘验收通过，未改写用户真实客户端配置。实现提交 `e698738`。
- [TASK-20261017-013](../tasks/completed/2026-10/TASK-20261017-013-update-menu-about-resources.md)：检查更新独立一级分类，关于页新增官网／开源仓库／免责／使用／隐私五项待配置占位；底部间距移入滚动内容，消除边框上方提前截断。完整强制质量门／511 项前端测试、最终 Windows 构建、开发版与隔离生产版原生深浅主题／960×600／140% 验收通过，四次底部间距均为 1px 边框，开发窗口已更新。实现提交 `ab69094`。
- [TASK-20261017-012](../tasks/completed/2026-10/TASK-20261017-012-settings-sidebar-layout.md)：设置窗口改为通用／请求与日志／数据存储／关于四分类，默认页紧凑展示启动、关闭与字号；存储按面板宽度适配。508 项前端测试、质量门各阶段、Windows 生产构建及原生深浅主题／960×600／140% 字号／草稿和键盘验收通过；Task 指纹扫描使用强制构建完成，开发窗口已更新。实现提交 `e608438`。
- [TASK-20261017-011](../tasks/completed/2026-10/TASK-20261017-011-settings-about-dialog.md)：设置／关于合并到侧栏“折叠”下方，通过模糊背景悬浮窗口切换页签；保留草稿、工作页面、旧链接和关闭焦点。完整质量门／507 项前端测试、原生深浅主题／960×600／滚动及三种关闭方式验收通过，开发版已更新。实现提交 `4bbd7bc`。
- [TASK-20261017-010](../tasks/completed/2026-10/TASK-20261017-010-responsive-agent-card-grid.md)：Agent 卡片使用内容区宽度驱动的自适应网格，实测 1440／1920／2560 为 3／4／5 列，侧栏收起继续增列。16 个原生布局场景无横向越界、末行等宽，深浅主题／最小窗口／详情返回及完整质量门／500 项前端测试通过。实现提交 `df5ef3f`。
- [TASK-20261017-009](../tasks/completed/2026-10/TASK-20261017-009-harness-desktop-detection.md)：兼容 Harness Desktop 带版本名称／空发布者的 NSIS 安装记录，正确识别 `D:/DSH` 的 `0.2.0-rc.2`，原生已安装状态与实际打开验证通过。完整质量门／500 项前端测试及后端边界回归通过，开发版已重建。实现提交 `8f2d06f`。
- [TASK-20261017-008](../tasks/completed/2026-10/TASK-20261017-008-unify-agent-client-cards.md)：七个产品统一为分组卡片、15 个软件入口分别展示安装／接入状态，共用详情并保留原入口可见焦点。完整质量门／500 项前端测试、原生十组导航和返回焦点、深浅主题／960×600 验收通过；开发版已更新。实现提交 `b4d3657`。
- [TASK-20261017-007](../tasks/completed/2026-10/TASK-20261017-007-kilo-qwen-mimo-code-clients.md)：接入 Kilo／Qwen／MiMo Code 本机模型配置、独立 Key、预览／备份恢复与软件入口；MiMo CLI／国内桌面版／国际桌面版同一卡片独立检测和打开。完整质量门／491 项前端测试、三个实际 CLI 的 SSE／工具／思考续接与恢复、原生两版状态／深浅主题／960×600 验收通过；实际桌面对话与 Qwen 后台记忆边界已注明。实现提交 `c58d931`。
- [TASK-20261017-006](../tasks/completed/2026-10/TASK-20261017-006-harness-and-opencode-clients.md)：接入 OpenCode CLI／Desktop 共享配置、Harness Web／Desktop 独立 profile，支持 Key、模型目录、预览／备份恢复、检测与固定打开入口。完整质量门／484 项前端测试、真实 Harness／OpenCode CLI 隔离 SSE／工具／思考续接及原生三个预览、深浅主题／960×600 验证通过；实际 Desktop 界面调用待验收。实现提交 `a49f234`。
- [TASK-20261017-005](../tasks/completed/2026-10/TASK-20261017-005-group-claude-client-entries.md)：Claude Code CLI／Desktop 合并为同一张入口卡片，各自保留安装／接入状态、详情导航与返回焦点。完整质量门／479 项前端测试、原生深浅主题及 960×600 验收通过；实现提交 `fc1fa01`。
- [TASK-20261017-004](../tasks/completed/2026-10/TASK-20261017-004-google-account-resources-and-agent-models.md)：Gemini 项目资源移入对应 API Key 卡片／列表；六款 Antigravity 订阅模型接入 Agent 三协议目录与同步入口。完整质量门／476 项前端测试、原生深浅主题／960×600、Opus／Sonnet／GPT-OSS 真实 Responses 及 Gemini／Sonnet 两轮工具续接通过。实现提交 `dc11eca`。
- [TASK-20261017-003](../tasks/completed/2026-10/TASK-20261017-003-unified-model-routing-and-labels.md)：River NVIDIA／GLM 短名称、Flash 与直连 DeepSeek 共用优先级、DeepSeek 普通／1M 双选及旧 ID／规则／菜单默认兼容；迁移 030 保留已有逻辑偏好。完整质量门／474 项前端测试及当前开发窗口归组验收通过；实现提交 `88714b7`。
- [TASK-20261017-002](../tasks/completed/2026-10/TASK-20261017-002-river-provider-icon.md)：补齐 River 官方本地图标，保留品牌底色、比例与圆角尺寸，修复共享入口的通用图标回退。完整质量门／473 项前端测试及当前开发窗口深浅主题验收通过；实现提交 `288c748`。
- [TASK-20261017-001](../tasks/completed/2026-10/TASK-20261017-001-river-builtin-provider.md)：River 第十一家内置服务商，三款官方模型、账号目录同步、路由、原生 Chat／Responses 和 Messages 文本工具转换。完整质量门／472 项前端测试与隔离原生窗口验收通过；实现提交 `ad8ada2`。真实 Key、图片及长窗口仍待验证。
- [TASK-20261016-009](../tasks/completed/2026-10/TASK-20261016-009-google-subscription-quota.md)：Google Antigravity 真实分组额度、重置时间、共享缓存与可选模型浮层；真实只读查询返回两组四窗口／6 个型号，区分 3 个 Gemini Chat 适配与 3 个第三方待接入型号。完整质量门／469 项前端测试及开发窗口验收通过，未发送推理。实现提交 `662361b`。
- [TASK-20261016-008](../tasks/completed/2026-10/TASK-20261016-008-google-subscription-label.md)：修复 Google 提供商与配额卡片误标 ChatGPT / Codex，提供商／账号标签共用品牌选择，未知类型使用中性订阅标签。完整质量门／456 项前端测试及实际提供商页标签核对通过。实现提交 `0c5c10e`。
- [TASK-20261016-007](../tasks/completed/2026-10/TASK-20261016-007-google-antigravity-subscription.md)：独立 Google / Antigravity 订阅登录、DPAPI 凭证、真实模型发现、文本／工具 Chat JSON/SSE、路由与用量。完整质量门／452 项前端测试通过，协作真实授权后发现 14 个模型，固定账号调用 Gemini 2.5 Flash 返回 HTTP 200（输入 6／输出 1 Token），开发窗口视觉核对通过。实现提交 `80e6ba8`；官方剩余额度、其他协议及高级模型／长窗口仍待验证。
- [TASK-20261016-006](../tasks/completed/2026-10/TASK-20261016-006-claude-cli-model-picker.md) 修复 Claude CLI 只配置默认档位、未发布完整模型菜单；补齐显式菜单与受管刷新，保留加密备份和外部编辑保护。完整质量门／450 项前端测试、原生预览应用及 Claude Code 2.1.292 实际 18 模型菜单验证通过。实现提交 `c1ccced`；原 CLI 重启后生效，未发送上游推理请求。
- [TASK-20261016-005](../tasks/completed/2026-10/TASK-20261016-005-kimi-default-order-and-model-dedup.md) Kimi 普通／Code 订阅 Key × 国内／海外、默认计费／地区顺序与六款逻辑模型去重，旧配置／Claude ID 兼容、真实身份保留，订阅未知额度独立展示。完整质量门／449 项前端测试、原生三协议／流式集成及正常桌面设置／表单／深浅主题／960×600 验收通过。实现提交 `4337082`；真实订阅 Key／套餐及长窗口待验证。
- [TASK-20261016-004](../tasks/completed/2026-10/TASK-20261016-004-stepfun-default-billing-order.md) 阶跃星辰账号设置支持 Step Plan／API Key 默认顺序与同名模型去重，复用 MiMo 设置交互及候选筛选，偏好独立；旧通道／Claude ID／固定账号兼容。完整质量门／444 项前端测试、三协议集成及正常桌面保存／深浅主题／960×600 验收通过。实现提交 `6bbcc8e`。
- [TASK-20261016-003](../tasks/completed/2026-10/TASK-20261016-003-mimo-default-billing-order.md) MiMo 设置保存 Token Plan／API Key 默认使用顺序，同名模型统一发布；保留固定账号、服务商顺序、旧通道及 Claude 上下文 ID 兼容，失败不跨通道重试。完整质量门／443 项前端测试、三协议网关集成及正常桌面持久值／深浅主题／960×600 验收通过。实现提交 `90fdeb0`。
- [TASK-20261016-002](../tasks/completed/2026-10/TASK-20261016-002-mimo-account-settings.md) 小米 MiMo 账号页增加与 OpenAI 共用的设置入口和局部空面板；完整质量门／439 项前端测试及原生深浅主题、960×600 和关闭／焦点检查通过，开发窗口已更新。实现提交 `2b94ac0`。
- [TASK-20261015-010](../tasks/completed/2026-10/TASK-20261015-010-mimo-provider-access-info.md) 小米外层提供商卡片补充 Token Plan 入口和三集群 OpenAI／Anthropic 基础地址、个人／团队 Key 及模型范围，明确按量默认入口和探测范围；浮层不撑高卡片，账号表单共用地区常量。完整质量门／439 项前端测试及原生深浅主题最小窗口、布局和焦点检查通过。实现提交 `19ffc59`。
- [TASK-20261015-009](../tasks/completed/2026-10/TASK-20261015-009-sidebar-link-drag.md) 关闭侧栏链接／品牌图片默认拖放及导航文字选中，修复误拖出现标题／本地 URL 拖影；点击和键盘导航保持可用。完整质量门／433 项前端测试及原生展开／收起拖动验证通过。实现提交 `8461b1a`。
- [TASK-20261015-008](../tasks/completed/2026-10/TASK-20261015-008-agent-config-success-toast.md) 客户端配置保存成功改为弹窗顶部悬浮通知，5 秒收起、可手动关闭、悬停／聚焦暂停，脱离正文滚动且不改变保存状态。完整质量门／431 项前端测试、960×600 原生深浅主题与滚动验收通过；只写临时 Codex 验收夹具。实现提交 `2dfe07a`。
- [TASK-20261015-007](../tasks/completed/2026-10/TASK-20261015-007-mimo-token-plan.md) 小米 MiMo 普通／Token Plan、个人／团队 Key 与中国／新加坡／欧洲集群，账号单个／批量／编辑、原生三协议、目录／通道路由和当次计费归属。完整质量门／428 项前端测试通过。实现提交 `f5b0591`；官方剩余 Credits、真实套餐 Key／长窗口和原生桌面显示待验收。
- [TASK-20261015-006](../tasks/completed/2026-10/TASK-20261015-006-kimi-balance.md) Kimi 国内／海外普通 API 官方可用、代金券与现金余额，账号卡片／列表及配额展示、仪表盘资源分类，复用版本隔离和共享缓存。完整质量门／415 项前端测试通过。实现提交 `e4958db`；真实 Key／原生桌面显示待验收，Code 订阅额度未接入。
- [TASK-20261015-005](../tasks/completed/2026-10/TASK-20261015-005-subscription-login-dialog-ui.md) ChatGPT 登录弹窗品牌标题、授权方式卡片、固定操作区和状态面板，移除过时转发提示；完整质量门／408 项前端测试、原生深浅主题和 960×600 最小窗口通过，未触发真实授权。实现提交 `4086f19`。
- [TASK-20261015-004](../tasks/completed/2026-10/TASK-20261015-004-minimax-api-implementation.md) MiniMax 国内／海外 × 按量／订阅账号、原生文本三协议／思考／函数工具／SSE、目录与隔离路由、当次用量及官方订阅窗口；完整质量门／407 项前端测试、实际开发与隔离原生深浅主题通过。实现提交 `0014521`；普通余额、高级 Responses 工具及真实 Key／长窗口仍待验证。
- [TASK-20261015-003](../tasks/completed/2026-10/TASK-20261015-003-minimax-api-plan.md) MiniMax 国内／海外 × 按量／订阅接入规划，采用账号级选项和通道隔离；核对现行 `.cn` 域名、原生三协议、模型发现及订阅额度入口，保留旧域名／套餐和资源契约待验证项。规划提交 `199867a`，接入代码尚未实施。
- [TASK-20261015-002](../tasks/completed/2026-10/TASK-20261015-002-remove-agent-config-intro.md) 移除一键配置弹窗的两段通用说明及中英文孤立翻译，将可访问描述关联到现有重启提示；原生窗口和完整质量门／401 项前端测试通过。实现提交 `34f855f`。
- [TASK-20261014-017](../tasks/completed/2026-10/TASK-20261014-017-kimi-api-implementation.md) Kimi 四款现役模型、国内／海外账号配置、模型同步与地区路由、原生思考／工具／SSE／用量；K3 三协议、K2.7 Code／K2.6 Chat／Messages、高速版 Chat。完整质量门／401 项前端测试及原生深浅主题通过。实现提交 `a20b0ab`，真实 Key／长窗口待验证。
- [TASK-20261014-016](../tasks/completed/2026-10/TASK-20261014-016-stepfun-api-implementation.md) 阶跃星辰国内／海外 × 按量 API／Step Plan 四组合、账号选项、隔离路由、原生三协议及普通余额；国内人民币／海外币种待确认，完整质量门／397 项前端测试及原生深浅主题通过。实现提交 `77b11d5`，真实 Key／长窗口和 Step Plan 剩余查询待验证。
- [TASK-20261014-015](../tasks/completed/2026-10/TASK-20261014-015-stepfun-api-plan.md) 阶跃星辰国内／海外 × 按量 API／Step Plan 四组合规划，明确账号级选项、计费池隔离、原生三协议及余额／Credit 边界；官方资料和文档检查通过。规划提交 `3f3212d`，接入代码尚未实施。
- [TASK-20261014-014](../tasks/completed/2026-10/TASK-20261014-014-consolidate-worktrees.md) 最新 main 统一到原 OmniProxy 目录，清理六个附加工作树，保留 Git 历史／18 个分支／独有资料及 OmniMail-win；正常桌面开发版从唯一目录启动，构建、原生界面、热更新和健康接口验收通过。整理记录提交 `7e80c35`。
- [TASK-20261014-013](../tasks/completed/2026-10/TASK-20261014-013-provider-model-popover.md) 提供商模型列表改为浮层，保持当前与相邻卡片高度；移除小米两个 V2.5 旧模型，main 原生深浅主题／960 与 1717 px 及完整质量门／392 项前端测试通过。实现提交 `ac9d340`。
- [TASK-20261014-012](../tasks/completed/2026-10/TASK-20261014-012-accounts-overview-full-width.md) 账号总览弹性等宽网格，宽屏五列两行并填满可用宽度；窄窗口／字号缩放／末行对齐／原生窗口及完整质量门通过，详情连续列宽保持原样。实现提交 `6070f83`。

- [TASK-20261014-011](../tasks/completed/2026-10/TASK-20261014-011-xiaomi-mimo-api.md) 小米 MiMo 普通 API Key、五个对话模型、账号目录同步及原生三协议／SSE／思考／函数调用；完整质量门／386 项前端测试通过，合入 main 并正常桌面启动核对。实现提交 `184af39`；真实 Key 调用待验证。

- [TASK-20261014-008](../tasks/completed/2026-10/TASK-20261014-008-claude-desktop-shared-config.md) 修复共享目录选择与重新读取恢复基准；正常桌面启动开发版避开 Codex 文件重定向，真实文件更新为 3001／10 模型，Claude 重启后 Code／连接 healthy 已验收，最终 main 完整质量门通过。main 修复提交 `fa69dc6`。
- [TASK-20261014-007](../tasks/completed/2026-10/TASK-20261014-007-merge-agent-scrollbars-main.md) 滚动条优化合入本地 main，保留既有 Atria／Gemini 实现并统一任务编号；合并后完整质量门／384 项前端测试通过。合并提交 `b47dfa6`。
- [TASK-20261014-006](../tasks/completed/2026-10/TASK-20261014-006-agent-config-scrollbars.md) 一键配置弹窗正文与模型目录统一细圆角悬停滚动条，去掉系统箭头、固定占位；完整质量门和深浅主题原生显隐／独立滚动／拖动验收通过。实现提交 `b078db0`。
- [TASK-20261014-005](../tasks/completed/2026-10/TASK-20261014-005-merge-main-and-restart-dev.md) Gemini 与 Atria 图标合入本地 main，保留双方配额界面并消除任务编号冲突；完整质量门／384 项前端测试通过，已重新运行正常 task dev，原生图标和 Gemini 资源模块验收通过，3001 网关恢复运行。合并提交 `1311430`。
- [TASK-20261014-004](../tasks/completed/2026-10/TASK-20261014-004-atria-provider-icon.md) 补充 Atria 官方本地图标，识别已有官方 API 地址的自定义配置，统一提供商／账号／配额页面；完整质量门与深浅主题原生验收通过，Atria 额度查询维持未接入。实现提交 `b76af6a`。
- [TASK-20261014-003](../tasks/completed/2026-10/TASK-20261014-003-gemini-api-quota-implementation.md)：先重建最新开发环境，完成 Gemini Chat JSON/SSE、官方模型限制、用量和项目资源观察；真实 Flash-Lite 小请求、原生窗口及完整质量门／371 项前端测试通过。官方剩余资源与 Agent 桥接保留后续。实现提交 `bf050a2`。

- [TASK-20261014-002](../tasks/completed/2026-10/TASK-20261014-002-gemini-api-quota-plan.md)：Gemini API、用量、项目共享限流与官方额度的四阶段规划；明确只读 Cloud 授权、预付余额入口与 Agent 签名桥接边界；P1/P2 落地见 TASK-20261014-003。规划提交 `efb4972`。
- [TASK-20261014-001](../tasks/completed/2026-10/TASK-20261014-001-merge-atria-and-gpt-context.md)：Atria 与 GPT 长上下文两分支合入 main；保留预设目录与账号确认能力，冲突已解决，合并后完整质量门／369 项前端测试／生产构建通过。合并提交 `7dce1b0`，真实 Atria Key 调用仍待验收。
- [TASK-20261013-017](../tasks/completed/2026-10/TASK-20261013-017-atria-builtin-provider.md)：Atria 第十家内置提供商、原生三协议 HTTP／SSE、预设目录／路由与用量统计；分支质量门、369 项前端测试及隔离原生验收通过。实现提交 `015cd6a`，真实 Atria Key 调用尚未验证。
- [TASK-20261013-016](../tasks/completed/2026-10/TASK-20261013-016-gpt-subscription-long-context.md) 六款 GPT 900019 输入 token 实测通过、按授权版本保存 1M、补充 GPT-6.1 Sol；原生九项菜单已保存生效，DeepSeek 实际 1M、GPT 独立 850K 压缩阈值与完整质量门通过。Terra 过载／429 未确认，原生旧缓存的自动刷新仍有限制。实现提交 `cbd7176`。
- [TASK-20261013-015](../tasks/completed/2026-10/TASK-20261013-015-claude-desktop-context-picker.md) DeepSeek 默认仅 1M、旧默认升级与原生长上下文字段；菜单仅两个 DeepSeek 1M 选项，原生实际回复显示 49.4K / 1M，CLI 默认及完整质量门通过。实现提交 `7b02540`。
- [TASK-20261013-014](../tasks/completed/2026-10/TASK-20261013-014-deepseek-claude-context-and-requests.md) DeepSeek Claude Code 普通／1M 选项、beta Messages 与 system 提醒兼容，正文容量适配；真实四组合、默认工具、开发网关和完整质量门通过，Desktop 目录已更新。实现提交 `06a3378`。
- [TASK-20261013-013](../tasks/completed/2026-10/TASK-20261013-013-continuous-account-card-width.md) 账号详情与总览采用连续列宽，消除窗口拉宽跨断点时卡片突然缩小；37,914 次浏览器尺寸检查、原生窗口验收与完整质量门通过。实现提交 `6a5726d`。
- [TASK-20261013-012](../tasks/completed/2026-10/TASK-20261013-012-popover-scrollbars.md) 重置卡和 OpenAI 设置使用细圆角悬停滚动条，保留占位、默认隐藏、淡入淡出；完整质量门与深浅主题显隐／布局／拖拽原生验收通过。实现提交 `cfe1a8b`。
- [TASK-20261013-011](../tasks/completed/2026-10/TASK-20261013-011-openai-settings-popover.md) OpenAI 设置改为局部面板，自动激活开关直接持久保存生效，移除二次确认；完整质量门、隔离原生保存及深浅主题／最小窗口开发验收通过。实现提交 `f91ca15`。
- [TASK-20261013-010](../tasks/completed/2026-10/TASK-20261013-010-reset-credits-popover.md) 重置卡改为入口下方局部面板、取消整页遮罩、保留确认／核对及关闭保护；完整质量门与深浅主题／最小窗口原生验收通过。实现提交 `91fbea0`。
- [TASK-20261013-009](../tasks/completed/2026-10/TASK-20261013-009-codex-quota-auto-activation.md) 自动激活 Codex 额度窗口：默认关闭、明确保存授权、原账号最小消息、复查与周期持久去重；完整质量门与原生验收通过，真实设置保持关闭。实现提交 `3084778`。
- [TASK-20261013-008](../tasks/completed/2026-10/TASK-20261013-008-codex-websocket-settings.md) OpenAI 账号设置新增网关级 Codex WebSocket 开关，默认开启、持久保存、动态生效，保留客户端身份；完整质量门与原生验收通过。实现提交 `50b309c`。
- [TASK-20261013-007](../tasks/completed/2026-10/TASK-20261013-007-model-provider-priorities.md) 同名模型服务商优先级排序、旧偏好兼容迁移及配置弹窗；完整质量门、三服务商原生真实保存与开发窗口验收通过。实现提交 `baa1621`。
- [TASK-20261013-006](../tasks/completed/2026-10/TASK-20261013-006-routing-model-provider-views.md) 路由默认映射与模型目录增加模型／服务商双向视图，移除指定目录说明；完整质量门、深浅主题与最小窗口原生验收通过，开发窗口已更新。实现提交 `dda694d`。
- [TASK-20261013-005](../tasks/completed/2026-10/TASK-20261013-005-upgrade-wails-beta28.md) Wails Go／JS／CLI 统一升级 beta.28，CI 固定同版；完整质量门通过，原生“关于”页显示 beta.28，开发版保持运行。实现提交 `d4848c3`。
- [TASK-20261013-004](../tasks/completed/2026-10/TASK-20261013-004-integrate-app-icons.md) C「钴蓝汇聚」已接入桌面／任务栏、托盘及标题栏／侧栏／favicon；10 个 ICO 尺寸、完整质量门和原生验收通过，开发版保持运行。实现提交 `a001b14`。
- [TASK-20261013-003](../tasks/completed/2026-10/TASK-20261013-003-app-icon-design.md) 参考微软／Google／Adobe，完成三款图标提案、多尺寸 ICO 与深浅预览；用户选定 C「钴蓝汇聚」，接入见 TASK-20261013-004。设计提交 `119d24d`。
- [TASK-20261013-002](../tasks/completed/2026-10/TASK-20261013-002-sync-main-fix-dev-startup.md) 文档提交 `fdfd29e`、本地 main 快进同步及开发启动等待修复 `1ecb6bb`；完整质量门与普通 task dev 原生验收通过。
- [TASK-20261012-007](../tasks/completed/2026-10/TASK-20261012-007-commit-current-implementation.md) 按用户要求只提交当前代码快照：`44fdbc4`；537 个代码文件，文档保留工作区，实际对话 400 尚未修改。
- [TASK-20261012-006](../tasks/completed/2026-10/TASK-20261012-006-claude-desktop-gateway-activation.md) Claude Code Desktop 一键启用 3001 网关、原生请求兼容、稳定模型映射与实际 Code 免登录／无配置提醒验收；完整检查通过，未混合提交既有工作区改动。
- [TASK-20261012-005](../tasks/completed/2026-10/TASK-20261012-005-claude-desktop-detection-and-write.md) Claude 新旧包名检测、加密目录写入兼容与 Code 入口（补录）
- [TASK-20261012-004](../tasks/completed/2026-10/TASK-20261012-004-agent-configuration-reread.md) 重装后重新读取配置、保留新设置与加密备份历史（补录）
- [TASK-20261012-003](../tasks/completed/2026-10/TASK-20261012-003-agent-installation-progress.md) 下载和安装真实进度、阶段与终态核对（补录）
- [TASK-20261012-002](../tasks/completed/2026-10/TASK-20261012-002-agent-cards-and-installation.md) 本地 Agent 品牌卡片、详情配置与固定客户端安装（补录）
- TASK-20261012-001 Claude Code Desktop 显示名称统一（既有记录）

- TASK-20261011-001 路由页四个选项卡、移除顶栏说明、导航与状态保留及原生验收

- TASK-20261010-006 默认模型自动映射、每个模型服务商优先选择与实际请求／目录一致

- TASK-20261010-005 每个页面独立记录和恢复主内容滚动位置，兼容异步内容与历史导航

- TASK-20261010-004 统一模型目录、按客户端格式自动配置与目录更新／恢复

- TASK-20261010-003 配额卡片资源状态移到 API Key／套餐同行右侧

- TASK-20261010-002 网关停止时允许本地 Agent 配置，保留模型校验与调用前启动提示

- TASK-20261009-019 网关公共控制栏、四选项卡、路由定位、延迟加载与切换状态保留

- TASK-20261009-018 稳定网关卡片收展、按钮移动与详情排版，消除侧栏动画跳动

- TASK-20261009-017 重建仪表盘运行总览、独立时间桶统计、趋势／异常／资源／接入与原生验收

- TASK-20261009-016 提供商电源启停按钮，统一账号样式与深浅主题

- TASK-20261009-015 重启标准开发环境并原生确认新配额页，恢复网关及窗口状态

- TASK-20261009-014 配额页全局额度／余额概况、分组筛选、共享缓存与受控刷新

- TASK-20261009-013 标题栏语言入口仅显示图标，与主题按钮尺寸统一

- TASK-20261009-012 语言按钮点击展开中英文浮层菜单、选择标记与键盘操作

- TASK-20261009-011 语言选择移到标题栏主题按钮左侧，移除侧栏入口

- TASK-20261009-010 侧栏网关运行状态、端口、账号总数与共享启停控制

- TASK-20261009-009 全部 9 处下拉选框统一 Radix 主题菜单、空值／表单／键盘及边缘布局

- TASK-20261009-008 每个提供商／模型的独立 Token 累计、历史回填与用量页分组模型表

- TASK-20261009-007 代理输入／输出 Token 累计、账号删除保留历史、请求／日志关联、保留设置与 Team 双身份校验

- TASK-20261009-006 本地 Agent 配置与原会话保留、ChatGPT／DeepSeek 模型同步、Responses／Messages／WebSocket 转发

- TASK-20261009-005 API Key 状态同行右对齐、余额刷新旋转与全局字号设置

- TASK-20261009-004 DeepSeek 真实余额查询、精确金额／币种明细与刷新

- TASK-20261009-003 额度后台刷新保留内容、稳定布局与信息更新动画

- TASK-20261009-002 Codex 重置卡数量／详情、使用确认与持久化幂等核对

- TASK-20261009-001 额度剩余百分比、精简重置提示与全局 OpenAI 设置弹窗

- TASK-20261008-010 Codex 订阅真实额度查询、进度条／重置计时、紧凑时间与 OpenAI 空设置弹窗

- TASK-20261008-009 收起侧栏选中背景、图标居中与覆盖滚动轨道

- TASK-20261008-008 订阅账号 ID、首尾缩略提示与账号电源启停按钮

- TASK-20261008-007 订阅账号邮箱标题、套餐配色、图标操作与全局悬浮提示

- TASK-20261008-006 Codex 授权会话恢复、读取失败保留授权与真实账号登录验证

- TASK-20261008-005 ChatGPT / Codex 订阅账号授权与生命周期管理（推理待接入）

- TASK-20261008-004 实时运行日志页、筛选与安全导出

- TASK-20261008-003 Codex 与 Claude 客户端接入预览（协议待实现）

- TASK-20261008-002 大屏账号与提供商卡片最多四列

- TASK-20261008-001 滚动条与窗口边缘缩放区域分离

- TASK-20261007-005 精确匹配模型路由与管理页面

- TASK-20261007-004 DeepSeek 网关最小调用闭环（真实账号认证待验证）

- TASK-20261007-003 编辑账号名称与 API Key

- TASK-20261007-002 网关页面与接入准备（代理后端待接入）

- TASK-20261007-001 账号详情卡片与列表切换

- TASK-20261006-004 账号自动命名、批量添加与删除确认弹窗

- TASK-20261006-003 账号卡片固定高度与删除图标

- TASK-20261006-002 标题栏双击最大化与还原

- TASK-20260928-010 账号卡片滚动、厂商详情页与全局内容宽度

- TASK-20260928-009 使用弹窗添加提供商账号

- TASK-20260928-008 账号页面与 API Key 账号管理

- TASK-20260928-007 精简提供商页面说明文案

- TASK-20260928-006 测试结果悬浮提示
- TASK-20260928-005 使用真实提供商品牌图标
- TASK-20260928-004 内置提供商默认启用与自定义提供商
- TASK-20260928-003 九家提供商预设与配置管理
- TASK-20260928-002 Windows 凭证存储基础设施
- TASK-20260928-001 修复主界面滚动布局与滚动条交互
- TASK-20260926-008 开机启动与关闭到托盘设置
- TASK-20260926-007 按参考稿优化存储设置页面
- TASK-20260926-006 设置页数据存储占用图与完整目录统计
- TASK-20260926-005 为侧栏文字增加逐字收展动画
- TASK-20260926-004 修复侧栏收展时图标位移与垂直跳动
- TASK-20260926-003 恢复收缩侧栏的展开按钮并居中图标
- TASK-20260926-002 修复侧边栏展开时短暂出现滚动条
- TASK-20260926-001 自绘标题栏与亮暗模式
- TASK-20260924-007 About page (version, runtime info, update check)
- TASK-20260924-006 Settings page: data storage management
- TASK-20260924-005 Task Record workflow
- TASK-20260924-004 Migrate i18n to i18next + react-i18n
- TASK-20260924-003 Frontend i18n (Chinese/English)
- TASK-20260924-002 State file/function size limits in coding rules
- TASK-20260924-001 Project initialization foundation

## Known Issues

- Dashboard, Settings, About, Providers, Accounts, Quota, Gateway, and Routing have live Wails data.
  Gateway supports local DeepSeek text Chat Completions (JSON/SSE), lifecycle
  controls and configuration checks. Other feature pages remain
  placeholders until their backends land. See `docs/product/GATEWAY.md`.
- Routing persists exact model aliases and optional fixed accounts, previews
  configuration without reading credentials, and dynamically rewrites gateway
  model names and discovery. Targets currently support DeepSeek only.
- Nine built-in providers are enabled by default and can be switched off;
  custom OpenAI-compatible endpoints can be added. API Key accounts are stored
  locally; DeepSeek request forwarding is implemented. Other provider adapters
  are pending. OpenAI ChatGPT / Codex subscription login now supports browser
  and device authorization, isolated accounts, refresh and logout; subscription
  inference remains pending. Real user authorization and persisted account
  identity / plan have been verified.
  Per-account subscription quotas show official remaining percentages and
  reset times; the quota page groups subscription quotas and DeepSeek balances
  with shared caches and controlled refresh. Quota history remains pending.
  See `docs/product/SUBSCRIPTION.md`.
- Gateway uses one enabled DeepSeek account per request; scheduler, retries,
  failover, Token Plan authentication and balance queries are pending. Current
  live verification reached DeepSeek but returned HTTP 401; the user chose to
  configure a valid key and verify real responses later.
- Provider model names are curated examples rather than account-specific live
  results. The Test action checks network reachability and latency without
  authenticating or invoking a model.
- Logs displays bounded structured runtime records with batched Wails updates,
  filtering, pause/follow controls, copy and JSONL export. Historical log-file
  search remains pending. See `docs/product/LOGS.md`.
- Single-instance protection is not wired yet; a manual launch after autostart
  can start a second process.
- Proxy HTTP integration tests cover the core flow with isolated providers;
  full desktop E2E (Playwright) remains pending.

## Next Steps

1. Connect Codex subscription inference and the Responses gateway using the
   verified subscription account authorization.
2. Add Claude CLI/Desktop Code protocol support and verify DeepSeek with a
   valid stored key.
3. Account scheduler (eligibility, health, cooldown, retry) and automatic
   failover.
4. Expand request-history and usage capabilities, then historical log search.
