# Current Development Status

当前状态看板。完整任务信息见 `docs/tasks/`（Source of Truth）。

## Current Version

0.1.0 — see `internal/version` and `build/config.yml`.

## In Progress

暂无当前任务。

## Recently Completed

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
