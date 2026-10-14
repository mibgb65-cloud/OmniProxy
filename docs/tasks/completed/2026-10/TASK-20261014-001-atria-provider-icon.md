# TASK-20261014-001: 补充 Atria 提供商品牌图标

## Status

Completed

## Created

2026-10-14 09:59（Asia/Singapore）

## Request

用户要求先修复 Atria 提供商图标，暂不接入额度获取。

## Goal

使用官方 Logo，使已有 Atria 自定义配置在提供商、账号和配额页面显示一致的本地品牌图标。

## Scope

- 保存官方品牌素材并记录来源。
- 在共享图标组件中支持 Atria 类型和官方 API 地址的自定义配置。
- 更新图标调用处、执行最小充分验证和项目质量门。

## Out of Scope

- 不接入额度查询或新增 Atria 内置提供商、推理能力。
- 不修改真实账号、凭据或数据库配置。
- 保留已有 Atria 接入规划任务及其他工作区文件。

# Analysis

现有图标只按 kind 匹配九家本地素材，自定义配置统一保存为 openai-compatible，故仅增加 atria 映射无法修复用户截图中的已有配置。

方案比较：只增加 kind 映射不能覆盖已有自定义配置；修改数据库种类会超出图标任务范围。采用仅影响展示的官方主机名识别，在 openai-compatible 配置指向 api.atria-asi.ai 时选用本地 Atria 图标，不按名称猜测品牌，也不从任意配置地址下载图片。

Logo 来源：https://www.atria-asi.ai/assets/atria-logo.png?v=20260909。先核实响应、格式和大小，再保留原始素材；使用 object-contain 保持比例。

风险较低。畸形 URL、相似域名和其他自定义服务商应继续显示通用图标；已有品牌映射优先。

# Implementation Plan

1. 保存官方 Logo，记录品牌素材来源。
2. 添加 Atria 映射及精确官方主机名识别，向共享组件传递已有 Base URL。
3. 验证已有品牌、自定义 Atria、其他域名及无效 URL 的显示；执行 task verify。
4. 完成任务记录、更新 CURRENT、审阅差异并按项目流程提交。

# Implementation Result

- 新增官方原始 `frontend/public/provider-icons/atria.png`（11,488 字节，640 × 602），记录来源和 SHA-256。
- 共享图标组件新增 Atria 映射；自定义配置只匹配 HTTPS、默认端口、无用户信息的 `api.atria-asi.ai`，地址长度限制与配置边界一致。
- 提供商卡片、账号总览、账号详情和配额分组传入已有 Base URL；品牌图标保持比例、固定尺寸和装饰语义。
- 保持原有品牌优先级和通用图标回退，无后端、数据库、凭据或额度查询逻辑变更。

# Verification

- 定向图标测试：23 项通过，覆盖显式 Atria、自定义官方地址、大小写／默认端口、相似域名、路径伪装、用户信息、非默认端口、HTTP、无效／空／过长地址及原有品牌优先级。
- `task verify BIN_DIR="$env:TEMP/omniproxy-atria-icon-verify"`：PASS，格式、lint、类型检查、Go 测试、69 个前端测试文件／379 项测试及生产构建全部通过。
- 非阻断提示：Windows Task 辅助命令提示缺少 uname／tail，jsdom 提示 scrollTo 未实现，Vite 提示已有主包大于 500 kB；质量门退出码为 0。
- 原生验收：从当前项目构建，在临时 test 数据目录及 3004／3894 端口完成自定义 Atria 创建，确认提供商深浅主题、账号总览、账号详情和配额分组均加载官方 Logo。原生图标为 24 × 24 CSS px，object-fit 为 contain，白色底块清晰可辨。
- 配额页使用隔离占位账号，仍显示“此账号的额度或余额查询尚未接入”，未发送推理或额度查询请求。
- Wails beta.28 会清除浏览器调试环境变量；原生 DOM 验收使用临时 Go overlay 添加浏览器调试参数和独立 WebView 数据目录，仓库启动代码未改动。
- UI/UX 技能数据库两次检索未命中；依据技能 Quick Reference 和仓库现有图标规范检查装饰语义、固定尺寸、原始比例和深浅主题。
- `git diff --check`：PASS。既有未跟踪 Atria 接入规划保留，不纳入本任务提交。

# Documentation

更新品牌素材来源、任务记录和 CURRENT；不涉及产品能力或架构变更，不需要 ADR。

# Deviations

None。

# Remaining Risks

按官方 API 主机名识别自定义配置；使用自建中转域名时仍显示通用图标。已有运行窗口来自 OmniProxy-main 工作区，本任务未修改或重启那份工作区的应用。

# Git

`b76af6a1e88bbc4a1ef7b4904bcda1ef0cdb9941 fix(frontend): 补充 Atria 提供商品牌图标`

仅提交本任务的前端、测试及品牌素材文件；任务归档与 CURRENT 单独同步，既有 Atria 接入规划保持未提交。

# Result

现有 Atria 自定义配置已使用官方本地图标，四处界面一致；完整质量门及深浅主题原生验收通过。额度查询维持未接入。隔离验收窗口已退出。

# Completed

2026-10-14（Asia/Singapore）
