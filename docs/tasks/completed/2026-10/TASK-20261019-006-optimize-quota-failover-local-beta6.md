# TASK-20261019-006: 优化订阅额度耗尽切换并生成本地 Beta.6

## Status

Completed

## Created

2026-10-19 12:31

## Request

用户登录两个账号，其中一个 5h 额度用完后请求没有自动切换到有额度的账号。要求与 TASK-20261019-005 的 Desktop 失败问题一起优化，生成新的本地 Beta 供更新安装测试，不发布 GitHub。

用户追加明确要求：5h 额度和一周额度用尽都必须能够轮换。

## Goal / Scope

- 核对订阅额度、路由候选与故障切换的衔接，修复可证实的缺口。
- 保留显式固定账号、同池切换、幂等和不确定执行不重放等安全边界。
- 合并 Desktop 安全诊断及经验证的兼容修正，生成本地 2.0.0-beta.6 安装包及摘要。
- 不发布、推送或创建远端标签，不覆盖用户已安装应用、不重启生产、不修改真实客户端配置或凭据。

## Analysis

用户追加明确要求同时覆盖 5h 和一周额度。已确认：额度查询结果未参与网关候选选择；HTTP 200 内的执行前额度错误不进入原有状态码重试分支；未同步模型目录的备用账号被直接排除。Desktop 另有 Responses 首事件 512 KiB 与 WebSocket 2 MiB 不一致的可复现边界。


## Implementation Plan

1. 读取候选筛选、额度读取与拒绝分类代码，确认关键失败路径。
2. 增加回归，最小修复已知额度耗尽的选择与切换；继续 Desktop 协议诊断验证。
3. 更新版本、相关文档及本地发布说明，运行定向与完整质量门。
4. 生成独立本地产物目录，校验应用／安装器版本与 SHA256，交付用户测试。

## Implementation Result

- 订阅服务以有界内存保存身份校验后的额度，按账号版本保护；任一窗口为 0 时跳过，双耗尽取较晚重置，支持周窗口、重置卡刷新和缺失／陈旧窗口保护。
- 固定订阅适配器仅识别首个完整 `usage_limit_reached` 执行前拒绝；Core 返回安全 429，网关可继续同池候选，保留真实 HTTP 状态。生成后的错误、取消、不确定网络失败均不扩大重试。
- 允许未同步目录的同池订阅作为已知模型备用，不改变主模型发现或明确不支持的目录。
- 合并 TASK-005 的安全诊断与 2 MiB Responses 事件兼容修正；版本更新为本地 Beta.6，无新依赖或迁移。
- 新增共享窗口／重置／版本／快照、拒绝完整性、HTTP／WebSocket 切换与记账、大首事件及双原生内核回归。
- 首次双内核契约的模拟完成事件省略 total_tokens，导致客户端解析失败；修正合成协议夹具后，两种实际内核均通过。未将夹具问题作为产品修复。


## Verification

- PASS：5h／周／同时耗尽、窗口重置、过期／缺失窗口、版本保护、缓存有界的单元验证。
- PASS：HTTP／WebSocket 执行前切换、固定／关闭／全部耗尽／生成后不重放、跨模型共享拒绝、未同步备用及大首事件集成验证。
- PASS：CLI 0.163.0 与 Desktop 0.163.0-alpha.10.2 的真实原生进程对独立模拟服务契约，两账号尝试、600 KiB 首事件、turn.completed 与一次记账。
- PASS：最终 build-release.ps1 -Version v2.0.0-beta.6 -OutputDirectory release-assets/v2.0.0-beta.6，执行完整 task --force verify 与 task --force package。
- PASS：全部 Go、104 个前端测试文件／576 项测试、格式、lint、类型检查及 Windows 生产构建。
- PASS：Windows 应用／安装器版本资源为 2.0.0-beta.6，数字版本为 2.0.0.0；安装器大小 19,300,902 bytes，SHA256 与同名校验文件一致。
- 初次门禁指出版本文件的换行格式；统一 gofmt／Prettier 后重新完成最终质量门和打包。既有 Vite 分包体积、jsdom scrollTo 和本机可选 POSIX 工具警告未导致最终构建失败。
- 构建日志：忽略的 tmp/desktop-difference-20261019/beta6-build.log。未修改安装器业务流程，因此本轮不对真实生产安装目录演练覆盖安装。


## Local Artifact

- `release-assets/v2.0.0-beta.6/OmniProxy-Setup-v2.0.0-beta.6-windows-amd64.exe`
- 大小：19,300,902 bytes。
- SHA256：`fd60945e3373727c5b5fe2847e00ba3226207ab516590516dfde04b76844e638`。
- 同目录生成 `.sha256`；发布说明：`docs/releases/v2.0.0-beta.6.md`。

## Git

f31918dbf55ddcce1a6f9215f15aaf741a964f23 fix(gateway): 修复订阅额度轮换并兼容 Desktop 流响应

仅本地提交，未推送、创建远端标签或发布 GitHub；既有未跟踪记录保持不变。

## Result

已交付 5h／周窗口耗尽后的同池轮换、严格执行前额度拒绝分类、未同步备用账号参与，以及 Desktop 大首事件兼容和安全诊断。本地 Beta.6 安装包和校验文件可供用户手动更新测试；未自动安装、重启生产或修改共享客户端配置。无新增依赖或数据库迁移。

## Completed

2026-10-19 12:56