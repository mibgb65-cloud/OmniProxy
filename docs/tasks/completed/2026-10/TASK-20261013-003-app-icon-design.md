# TASK-20261013-003: 桌面与任务栏图标设计

## Status

Completed

## Created

2026-10-13 09:48（Asia/Singapore）

## Request

帮我设计一下我的这个软件的桌面图标，还有任务栏的图标，要求高级简洁，先去参考大厂的图标。

后续反馈：用户认为 C 不错，本轮将 C「钴蓝汇聚」记录为选定设计方向。

## Goal

参考大厂官方图标资料，为 OmniProxy 提供原创图标方案与桌面、任务栏尺寸预览。

## Scope

- 调研 Microsoft、Google、Adobe 的官方图标设计资料。
- 结合本地 AI 网关与多服务汇聚的产品定位制作三个视觉方向。
- 保存设计资产、参考链接与生成提示词，检查透明背景及小尺寸效果。

## Existing Behavior

`build/appicon.png` 为 Wails 默认图标；Windows 使用 `build/windows/icon.ico`。

# Analysis

## Related Modules

设计交付保存在 `design/app-icon/`，本任务先提供设计稿。

## Dependencies

使用内置 imagegen；不新增应用运行依赖。

## Risks

大尺寸效果不代表任务栏可读性，需要实际缩放检查；视觉方案属于本轮设计建议。

# Implementation Plan

1. 阅读项目定位与现有图标，查看官方大厂设计资料。
2. 生成三个独立原创方案，保持简单轮廓、少量色彩、充足负空间。
3. 提供桌面与任务栏实际尺寸预览，保存提示词与参考来源。
4. 检查交付文件、编码与 Git diff，完成任务记录。
5. 根据用户反馈更新选定方案与预览标记。

# Implementation Result

## Changes

- 阅读 Microsoft Windows、Google Chrome 与 Adobe 官方设计资料，参考简单轮廓、品牌色与小尺寸适配原则。
- 使用内置 imagegen 生成 A「翡翠连接环」、B「曜石入口」、C「钴蓝汇聚」三个独立原创 PNG，均为 1254×1254、透明外底。
- 使用项目现有 Wails CLI 导出各方案的 Windows ICO，包含 16／20／24／32／40／48／64／96／128／256 共 10 个尺寸。
- 增加深浅背景对比图、可切换展示主题的本地 HTML 预览、各尺寸 PNG 及验证记录。
- 在 `design-spec.json` 保存参考来源、完整生成提示词和导出方式，基于实际预览推荐 B。
- 用户后续选择 C「钴蓝汇聚」；更新设计说明的选定方向与推荐标记，并在 HTML 预览标出 C。

## Files Changed

- `design/app-icon/`：三款 PNG／ICO、尺寸资产、对比图、HTML 预览、导出脚本、设计说明与验证记录。
- 本任务记录与 `docs/plans/CURRENT.md` 的完成概览。

## Technical Notes

桌面和任务栏共享同一视觉标识，ICO 为系统提供不同尺寸。只交付方案，未替换 `build/appicon.png`、`build/windows/icon.ico` 或其他应用资源。

# Verification

## Verification Result

- PASS：三款图像尺寸为正方形，四个画布角的 alpha 均为 0，缩放结果保留透明通道。
- PASS：人工检查 `comparison.png`，核对大图与深浅底 16／24／32 像素样本，无文字遮挡和布局溢出。
- PASS：读取三个 ICO 的实际目录，核对文件头、10 个帧尺寸与数据边界。
- PASS：使用项目现有 jsdom 检查 HTML 脚本、三张方案卡片、图像路径与深浅展示切换。
- PASS：JSON 解析、UTF-8 无 BOM 与 `git diff --check`。
- PASS：后续选定方案更新核对，C 为唯一选定与推荐方向；预览选定标记、资源路径和主题切换正确。
- `task verify` 未运行：本任务仅新增独立设计提案与任务文档，未修改应用代码、依赖、构建配置或当前使用的图标；以资产格式、尺寸、透明背景与视觉检查作为最小充分验证。

## Remaining Limitations

尚未执行 Windows 原生桌面／任务栏接入验收。16 像素下细小切面和开口的辨识度下降，定稿接入时可进一步做像素对齐优化。

# Documentation

设计说明随资产保存；完成后同步 CURRENT 概览。PRD、架构与技术栈无变化。

# Git

`119d24d chore(design): 添加 OmniProxy 桌面与任务栏图标方案`

`cd60522 chore(design): 将钴蓝汇聚标为用户选定方案`

# Result

已完成三款桌面／任务栏图标设计提案、Windows 多尺寸图标和预览；用户选定 C「钴蓝汇聚」作为后续定稿方向。

# Completed

2026-10-13 09:58（Asia/Singapore）

后续方案选择记录完成：2026-10-13 10:00（Asia/Singapore）。
