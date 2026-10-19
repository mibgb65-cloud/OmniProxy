# TASK-20261019-007: 排查 ChatGPT 订阅登录通用失败

## Status

Completed

## Created

2026-10-19 13:02

## Request

用户在本地 Beta.6 的 ChatGPT 账号登录窗口持续看到“操作未完成，请检查网络和登录组件后重试”，要求排查。

## Goal / Scope

定位实际生产日志、登录组件发现／启动及授权错误映射，修复已证实的原因并进行隔离验证。保护现有账号、系统凭据、当前 Agent 和进行中的真实授权；不记录令牌、授权码、完整授权链接或账号身份。

## Analysis

当前 Control API 确认 production / 2.0.0-beta.6。提示属于通用 operation_failed，不能仅凭文案认定网络异常。需先确认当前实例实际数据／日志目录；已知默认 v2 日志最后更新时间早于当前实例启动，不能用旧日志推断新登录失败。

## Implementation Plan

1. 只读定位生产进程的环境与日志，关联具体授权阶段及稳定错误码。
2. 检查组件解析、官方子进程协议及前端错误映射，在独立目录复现。
3. 对已确认缺口最小修复、测试，必要时生成后续本地测试包；不发布 GitHub。

## Implementation Result

### Root Cause

- 当前生产元数据为 3 个 Google 订阅、1 个 OpenAI 订阅，均使用 subscription_login 类型。
- recoverPending 原先遍历所有订阅账号，直接调用仅适用于 codex-keyring 格式的 SubscriptionProfiles；遇到 Google 引用即返回 account_unavailable，发生在创建授权会话及启动官方组件之前。
- 前端没有识别 account_unavailable 等已存在稳定错误码，导致显示通用 operation_failed 文案。
- 已只读确认生产进程 PID 16040 的 stderr 句柄缺失；实际打开日志路径仍为默认 v2/logs/omniproxy.log。中央 fanoutHandler 在控制台失败后立即返回，故新版启动及错误没有写到文件／UI，不能用旧日志反推新故障。

### Changes

- account 增加纯元数据的提供商类型判定，区分其他提供商与无效／不可读取的配置；Codex 清理只处理自己的提供商引用，不读取 Google 凭据，仍保护自身不可读取引用。
- 补充开始授权前的安全错误日志，以及前端已有稳定错误码的分类和中英文提示。
- 日志 fanout 独立尝试所有出口并汇总错误，单个控制台失败不阻断文件和 UI。
- 版本更新为本地 Beta.7，保留 Beta.6 功能；不改 OAuth 参数、登录端点、真实客户端配置、账号或系统凭据，不发起真实用户授权、不重启生产。


## Verification

- 已先增加回归并确认旧实现失败：混合 Google 账号无法开始 Codex 登录，控制台错误后文件／UI 无记录。
- PASS：修复后浏览器／设备码的独立模拟授权均完成，既有账号与 Google 引用保持不变；自身不可读取引用仍阻断危险清理。
- PASS：日志出口故障与错误码不泄露原始内容的后端回归；登录界面针对账号读取错误显示具体提示，原始消息不显示（该前端文件 9 项通过）。
- PASS：实际 PATH 解析为 codex.exe，官方组件在独立 CODEX_HOME 中初始化成功；仅 initialize／关闭，没有真实登录、回调占用或模型请求。
- PASS：build-release.ps1 -Version v2.0.0-beta.7 -OutputDirectory release-assets/v2.0.0-beta.7，完整 task --force verify、task --force package、全部 Go、格式／静态／类型检查及 104 文件／577 前端测试通过。
- PASS：Windows 应用与安装器版本资源及安装包 SHA256 一致性校验；完整版本 2.0.0-beta.7，数字版本 2.0.0.0。
- PASS：真实 Windows GUI 二进制以 test profile、独立 HOME／配置／数据目录和随机回环端口启动；只读确认 stderr_missing=true，文件中仍存在 starting OmniProxy 记录，健康接口为 beta.7/test。测试进程正常退出，生产 PID 16040／beta.6 未重启。
- 冒烟检查初版 .NET FileVersionInfo 返回空版本，改用仓库已有的原生 Windows 版本资源读取器后通过；未放宽或跳过产物版本核验。
- 构建日志与测试收据仅保存在忽略的 tmp/subscription-login-20261019/；没有真实网页登录或替用户完成授权。


## Local Artifact

- `release-assets/v2.0.0-beta.7/OmniProxy-Setup-v2.0.0-beta.7-windows-amd64.exe`
- 大小：19305220 bytes。
- SHA256：`0d56116d6137b6bfc857b355131bd7535f859a323561f9e4cb4727ff92c5c9fa`，同目录保存 `.sha256`。
- 本地测试版，未发布、未自动安装，Beta.6 资产保留。

## Git

9d001ec5f11b01416f014d45d780a1f25a9ef555 fix(subscription): 修复混合订阅登录阻断和 GUI 日志丢失

仅本地提交，未推送或发布 GitHub；未改动既有用户未跟踪文件。

## Result

已定位并复现混合提供商订阅引用被误读造成的登录前阻断，以及通用文案和 GUI 日志丢失问题；修复、回归、完整质量门和本地 Beta.7 安装包均完成。既有账号／授权与生产实例保持不变；真实官方网页授权由用户更新后完成，不能将模拟流程等同于代用户真实登录成功。

## Completed

2026-10-19 13:27