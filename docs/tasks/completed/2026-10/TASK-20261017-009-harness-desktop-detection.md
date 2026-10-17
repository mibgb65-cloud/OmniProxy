# TASK-20261017-009: 修复 Harness Desktop 安装漏检

## Status

Completed

## Created

2026-10-17（Asia/Singapore）

## Request

用户已将 DeepSeek Harness Desktop 安装在 `D:/DSH`，客户端接入页未检测到。

## Goal

根据实际安装记录正确识别软件、版本与安装路径，使现有固定打开入口可定位主程序。

## Analysis

本机卸载注册表为 `DeepSeek Harness 0.2.0-rc.2`，DisplayVersion 为 `0.2.0-rc.2`，
Publisher 为空，InstallLocation 为 `D:/DSH`，DisplayIcon 指向该目录的固定主程序。
旧规则要求精确无版本名称及 DeepSeek 发布者前缀，因而漏检。
现有 MiMo NSIS 适配已有名称去版本、图标路径和主程序存在性校验，可扩展同一规则。

## Implementation Plan

1. 扩展受控 NSIS 元数据适配，识别 Harness 的版本后缀和空发布者安装记录。
2. 继续校验精确产品名、合法版本、绝对路径与固定现存主程序，不写死盘符或执行注册项命令。
3. 增加漏检回归和错误产品／路径边界，完成本机检测、打开与质量门验证。
4. 更新产品说明、审查差异、提交并归档。

## Implementation Result

- 将现有 NSIS 精确软件名适配扩展为 Harness／MiMo 共用，兼容 Harness 名称后的版本
  和空发布者字段，使用注册项指向的固定现存主程序确定目录。
- 保持已知发布者的安装形式兼容，主程序名称、参数和现有打开实现不变，无用户目录
  硬编码、全盘扫描、安装或配置写入。

## Verification

- `go test ./internal/infrastructure/agentinstall` 通过，包括新增带版本名称／空发布者／
  引号图标路径、旧发布者记录及错误名称、版本、命令参数、相对路径／URL、缺失文件等边界。
- 本机实际检测返回 `installed`、`0.2.0-rc.2`、`D:/DSH`，固定打开主程序存在。
  证据 `tmp/harness-desktop-detection-20261017/actual-detection.json`。
- `VITEST_MAX_WORKERS=4 task verify BIN_DIR=tmp/harness-desktop-detection-20261017/verify-bin`
  退出码 0：格式、Go vet／staticcheck、ESLint、TypeScript、全部 Go 测试、89 个前端文件／
  500 项测试及 Windows 原生构建通过。证据同目录 `verify.log`。
- 正常桌面重建并启动开发版后，卡片已显示“已安装”，详情显示 `0.2.0-rc.2`，打开按钮
  可用；实际点击后观察到 `D:/DSH/DeepSeek Harness.exe` 进程，未提交模型任务或修改
  Harness 配置。原生截图及 `native-qa.log` 保存在同一证据目录，开发版保持运行。

## Git

实现提交：`8f2d06f71877e09b425d77537710eef0306d9638`
（`fix(gateway): 修复 Harness Desktop 安装漏检`）。
分支：`codex/river-builtin-provider`；未推送。

## Result

已正确识别用户安装在 `D:/DSH` 的 Harness Desktop `0.2.0-rc.2`，开发界面与实际打开
验证通过。修复针对安装记录格式，适用于同格式的自定义目录安装；未硬编码用户路径。
完整质量门通过，原有无关未跟踪任务记录保持原样。
