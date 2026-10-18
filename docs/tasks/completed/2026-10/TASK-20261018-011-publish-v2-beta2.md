# TASK-20261018-011: 构建并发布修复后的 v2 Beta.2

## Status

Completed

## Created

2026-10-18（Asia/Singapore）

## Request

用户要求完成后推送代码到 GitHub，并构建修复后的新 Beta.2。延续已授权的 Beta 发布流程。

## Goal

提供包含历史统计库启动隔离、可见错误日志、安装图标及正式更新渠道修复的 `v2.0.0-beta.2` Windows amd64 安装包和 SHA256，推送 main 并通过既有 GitHub Release 流程发布新的 Pre-release。

## Analysis

修复提交 7085fe4 已推送 main，最终本地完整质量门／575 项前端测试／隔离安装升级卸载通过。Beta.1 公开资产不能覆盖。新版本仅提升完整 SemVer 为 2.0.0-beta.2，Windows 数字资源继续 2.0.0.0；正式 Latest v1.3.1 和旧 master 不变。沿用现有最小权限工作流，标签触发质量门、打包、摘要校验及 draft 完整发布。Beta.1 特殊恢复分支不适用于新标签。

## Implementation Plan

1. 同步应用／前端完整版本，运行既有版本映射，整理 Beta.2 发布说明及相关状态文档。
2. 最终版本生产构建、安装器／原生资源／SHA256 校验及独立 Beta.2 安装、历史布局、运行中升级、重启、卸载验证。
3. 审查版本差异并提交、推送 main 和新的不可变版本标签，跟踪 CI 与 Release 完整成功。
4. 回读公开 Pre-release／资产／标签；用实际更新客户端验证 Beta.1 可发现 Beta.2，并下载校验公开安装包。归档结果及提交记录。

## Implementation Result

完整应用／前端／Windows 产品版本已统一为 2.0.0-beta.2，数字资源保持 2.0.0.0。发布说明覆盖启动隔离、日志提示、快捷方式图标及正式渠道修复；不承诺旧统计整库迁移。TASK-010 已归档，Beta.1 资产保持原样。Beta.2 已公开预发布。

main 与新标签均已推送，源提交 `5de94cfa8951d88666990694ea2fb4a33c99990a`，注释标签对象 `a8ff686b637bb9e11a86fec4dae687e471b1923a`；远端回读 peeled SHA 与源提交一致。CI `39119451562`、Release `39119455141` 均 success。未复用 Beta.1 特殊恢复分支或替换其资产。

公开 Release ID `416387563`，发布于 2026-10-18 10:39:21 UTC（18:39:21 Asia/Singapore），draft=false／prerelease=true。安装器 19,255,511 bytes，SHA256 `0b4f4e56dc7f1c8a781a1c13fa0913e6f0ea124f8de2c837858346b658e4ce97`；同名校验文件 114 bytes。正式 Latest 仍为 v1.3.1（392117295）。本地与 CI 构建摘要不同，公开下载已按 CI 发布摘要独立核对。

## Verification

- 修复代码已通过本地最终完整 `task --force verify`（104 文件／575 项前端测试及全部 Go／格式／静态／类型／生产构建）；Beta.2 版本提升后定向 `go test ./internal/version ./internal/infrastructure/update` PASS。最终标签在 Release 工作流再次运行完整质量门。
- Beta.2 `task --force build`、机器级生产 NSIS 候选包、原生 Windows 应用／安装器版本资源及发布资产验证全部 PASS。候选 19,238,099 bytes，SHA256 `41bd2990de3ced2c99c785dd360e6e479b7a6232288360283b4a59b60f7e49ef`；这是本地产物，公开 CI 产物摘要另行核对。
- 独立 OmniProxy Beta2 QA／独立卸载键／目录／配置／端口：首次安装、显式图标、历史库哈希保留、v2 全量迁移及重启、健康接口精确版本 2.0.0-beta.2、运行中静默覆盖／重启、保留已移除的桌面图标、运行中卸载、自有 ICO 清理、未知文件及两库保留全部 PASS；生产健康接口仍正常。
- CI `39119451562` 和 Release `39119455141` 完整质量门／打包／上传／发布全部 PASS，源标签不可变。
- 使用实际 update.Checker 读取公开正式／Beta 渠道，分别获得 v1.3.1／v2.0.0-beta.2；实际版本比较确认比 Beta.1 新。使用 SelectPackage／DownloadClient／Download／VerifyFile 真实下载公开安装器及校验文件，GitHub → release-assets CDN 重定向、精确大小／SHA256／文件复验全部 PASS。
- 公开资产验证脚本和 Windows 原生版本资源校验 PASS，下载摘要与 GitHub API 的 sha256 digest 及同名校验文件一致。
- Beta.1 公开说明已指向已发布的 Beta.2；回读确认 Beta.1 仍为公开预发布，原资产 ID／名称／大小／摘要完全相同。文档差异检查通过，五份既有用户任务记录未编辑或纳入提交。

## Git

`5de94cfa8951d88666990694ea2fb4a33c99990a` chore(release): 准备 v2.0.0-beta.2 修复版，已推送 main 和 v2.0.0-beta.2。

包含修复提交 `7085fe47ddf37c772e27a0b3add4e3886e1ea636`。

## Result

Beta.2 已推送并公开，可从 GitHub Release 下载或在 Beta.1 默认 Beta 渠道检查升级。原库／已有 v2 数据保护、图标和可见诊断已随包交付。未替用户安装到正式目录，也未触发正式数据导入；安装和下载由用户决定。

## Completed

2026-10-18（Asia/Singapore）
