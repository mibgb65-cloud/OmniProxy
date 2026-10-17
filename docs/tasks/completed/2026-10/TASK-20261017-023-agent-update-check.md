# TASK-20261017-023: 客户端软件版本更新检查

## Status

Completed

## Created

2026-10-17（Asia/Singapore）

## Request

在客户端详情的软件安装区域增加检测版本更新功能。

## Goal

用户可按需检查已安装客户端的可用更新，清楚区分已是最新、有更新、无法确认和失败；检测不自动升级或改写客户端配置。

## Analysis

- 本次入口针对截图中的 Claude Code Desktop；其他客户端的独立发布通道不在本次范围。
- 本机 WinGet 1.29.380 的 `show` 对 Claude 商店包返回 `Unknown`，`list` 不显示更新不能证明已是最新，不能据此误报。
- [Claude 官方 Windows 部署文档](https://support.claude.com/en/articles/12622703-deploy-claude-desktop-for-windows) 提供固定 MSIX 最新发布重定向。只读取响应头、不跟随下载；实测返回官方 `downloads.claude.ai/releases/win32/x64/2.42304.0/Claude-….msix`，本机安装版本为 `2.37337.1.0`。
- 固定产品 ID、固定架构入口、严格校验 HTTPS／发布域／路径／数字版本；数字版本补齐四段比较，格式未知不判断最新。网络错误只返回稳定错误码。

## Implementation Plan

1. 在既有安装服务增加按需只读更新检查，重新确认真实安装版本；请求有超时并与安装互斥，不下载／执行安装包。
2. Claude Code Desktop 已安装时增加独立检查按钮，固定加载尺寸、柔和旋转、简洁可访问结果；检查不影响打开客户端。
3. 覆盖更新／无更新／未知／失败、重定向防护、非法输入及独立加载，完成原生验证与质量门，同步文档并提交归档。

## Implementation Result

- Claude Code Desktop 已安装时，在打开客户端旁增加「检查更新」，只由点击触发；结果在版本下方简洁展示，支持重试与重新检测清除旧结果。
- 后端重新确认安装版本并同步检测缓存，20 秒总超时；官方 HTTP 请求 15 秒超时，仅接受固定架构／HTTPS 域／MSIX 路径，拒绝跟随下载；不执行安装或改写配置。
- 采用三／四段数字版本补零比较；未知／越界／预发行格式不宣称暂无更新。HTTP 错误不暴露底层详情。
- 检查按钮为中英文全部状态预留宽度，1.4 秒旋转并支持减少动态效果；检查期间打开和官方说明独立可用。安装动作提取为小组件，手写实现文件均在 200 行以内。
- 无新增依赖、数据库迁移或其他客户端更新渠道；官方公开发布可能早于商店／组织分发。

## Verification

- Go 定向测试：基础设施／应用／Wails 三个包通过；覆盖 15 组版本／发布地址案例、非法 ID、未知版本、网络／HTTP／取消、响应正文不读取、安装互斥、关闭及真实版本缓存。
- 前端定向测试：17 项通过（既有安装 11 项、更新检查 6 项），覆盖按需检查、新版本／暂无／未知／失败重试、重复点击、独立加载、缓存版本、未安装和英文。
- `task --force verify BIN_DIR=tmp/agent-update-check-20261017/verify-bin`：PASS，格式／lint／类型／Go 测试／537 项前端测试／前端与 Windows 生产构建全部通过。日志：`tmp/agent-update-check-20261017/verify.log`。缓存同步补充后另行通过应用包测试、`go vet` 与 `staticcheck`，生产构建包含该补充。
- 正常开发窗口真实只读检查：当前 `2.37337.1.0`，发现官方 `2.42304.0`；18 次原生采样中 17 次处于加载，按钮宽高始终 `101×32`，打开和官方说明始终可用。
- 深浅主题、960×600 窄窗口按钮布局通过；恢复原窗口大小与主题。截图及采样：`tmp/agent-update-check-20261017/native-*`。未安装／升级软件或应用客户端配置。
- 修正测试环境不加载 Tailwind 时占位文本进入可访问名称的问题：占位显式 `aria-hidden` 后测试通过；JS DOM 既有 `scrollTo` 提示不影响质量门。

## Documentation

- 更新 `docs/product/LOCAL_AGENTS.md`、`docs/architecture/OVERVIEW.md` 与 `docs/plans/CURRENT.md`，说明范围、触发方式、官方公开版本及未知边界。
- Wails 绑定由生成器重新生成并纳入提交。

## Deviations

本机 WinGet 的商店源未提供可比较的最新版本，改用已核实的官方 MSIX 发布重定向；本次只实现截图中的 Claude Desktop，未推测其他产品的更新地址。

## Git

实现提交：`8d929daf0a25489696e21b2f172db0581a22717a` — `feat(gateway): 添加 Claude Desktop 按需版本更新检查`。任务记录随随后文档提交归档，未推送远端；无关未跟踪任务文件保持原样。

## Result

Claude Code Desktop 按需版本更新检查已完成、验证、提交并归档，正常开发窗口已重新构建更新。官方公开发布版本检查可用，其他客户端更新渠道不在本次范围。

## Completed

2026-10-17 21:33（Asia/Singapore）
