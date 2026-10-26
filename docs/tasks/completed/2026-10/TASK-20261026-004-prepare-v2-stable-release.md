# TASK-20261026-004: 修复发布阻塞并准备 v2.0.0

## Status

Completed

## Created

2026-10-26（Asia/Singapore）

## Request

“你做吧，做完后我要准备正式发布 2.0.0 版本了。”

## Goal

修复 TASK-20261026-003 中的安装器数据目录与用户身份问题，补充验证并准备 2.0.0 本地正式发布资产。

## Scope

- 兼容真实旧安装目录的 data-dir.txt，防止升级改选目录后遗留现有数据。
- 在实际交互用户身份下校验／读写安装数据位置，不使用其他提权管理员的配置或权限。
- 增加回归测试，执行完整质量门和发布资产校验。
- 同步 2.0.0 版本、发布说明、相关开发文档和 CURRENT。

## Out of Scope

- 不创建标签、推送或公开发布。
- 不覆盖用户真实安装、读取真实凭据或发送付费上游请求。
- 不把未执行的 UAC／真实客户端验收记为通过，不自行选择开源许可证或承诺代码签名。

## Analysis

- 回归测试已先复现：旧标记指向已存在目录时，安装辅助解析返回错误的默认目录。
- 不修改进程全局 APPDATA，也不只切换线程令牌后继续使用 os.UserConfigDir（它仍读进程环境）。
- 采用显式安装位置上下文：原始安装目录由 NSIS 传入，配置／默认数据根由真实用户令牌的 KnownFolder 查询；读写在专属锁定线程上使用普通用户模拟令牌，按该用户权限验证文件。
- 相比从管理员临时目录再启动普通用户子进程，此方案不需要复制可执行文件、共享结果文件或暴露管道；模拟失败直接报错，不回退管理员写入。
- 读取历史标记／数据库权限错误时失败关闭；已有数据不得被新位置覆盖。同位置的历史标记可安全固化为 app.json，支持程序目录改变。
- 原始审查记录及 CURRENT 存在本聊天上一轮的未提交文档变更，保留。

## Implementation Plan

1. 比较 Windows 用户身份与历史目录解析方案，确定最小安全实现。
2. 先补回归测试，再修复安装器／profile 调用链。
3. 进行隔离辅助程序及安装器验证，运行完整质量门。
4. 同步版本、发布说明并生成本地正式安装包，明确人工验收边界。
5. 审查 diff，更新记录，不自动推送或发布。

## Implementation Result

### Changes

- `profile.InstallerLocation` 显式分离用户配置目录、默认数据目录与原安装目录。
  app.json 优先、旧标记兼容、默认位置兜底；中断搬迁保留旧位置，读取／权限异常失败关闭。
- NSIS 在初始化时保留原安装目录，不受程序目录页后续修改影响；旧位置有根数据库或
  `v2/omniproxy.db` 时拒绝切换空目录；保持位置可固化 app.json，不移动数据。
- Windows 提权助手获取交互桌面普通用户令牌，KnownFolder 定位其配置；提权文件操作在
  专属锁定线程中模拟该用户，恢复失败不复用线程。普通用户助手直接使用自身身份。
  不修改全局环境、不继承管理员的 APPDATA／OMNIPROXY 配置、不回退为管理员写入。
- 原生向导改用固定、独占的 UTF-16LE 结果文件，使用后删除；普通命令行仍保留 UTF-8 输出。
  超出 NSIS 默认字符串容量的结果明确拒绝，不能截断后保存错误路径。
- 配置读取限制 1 MiB、旧标记限制 16 KiB；目录探针与配置临时文件使用唯一名称，
  使用同一个父目录句柄原子替换配置，不使用非原子复制兜底。
- 修正正式版／Beta 渠道测试的版本假设，显式覆盖两种默认渠道及保存的 Beta 偏好。
- 发布脚本在质量门失败时立即停止，不再继续打包。
- 源码与前端版本同步为 `2.0.0`，准备正式发布说明及本地 Windows amd64 安装包。

### Verification Evidence

- 先添加历史目录回归测试，确认旧实现返回错误的默认位置，再实施修复。
- Go profile／appinstall 多轮无缓存专项通过，覆盖历史标记、两种数据库布局、重复选择、
  程序目录改变、损坏配置／标记、路径边界、环境隔离、模拟线程隔离与真实文件权限读写。
- 使用 production 源码、独立链接时产品名构建辅助程序：管理员环境陷阱未生效；
  中文旧目录正确解析，已有数据库改选返回 3，保持旧位置／更换程序目录／新建及替换配置通过，
  非法命令和路径返回 2，旧假数据库哈希保持不变。
- 原生 NSIS 最小探针先复现 nsExec 中文输出乱码，改成 UTF-16LE 文件协议后原生往返通过。
- 用户级原生安装器四次向导操作通过：实际显示中文旧目录、修改程序目录、禁止自动启动测试应用；
  交互安装、静默覆盖升级、卸载退出码均为 0。配置／旧数据库／未知文件保留，
  删除过的桌面快捷方式未重建；测试产品、注册键、目录和可执行程序均独立于真实安装。
- 本次测试没有启动真实生产应用、读取真实账号凭据或调用上游模型。

### Files Changed

- `internal/platform/profile/{installer,installer_path,config,config_file}.go` 及相关测试。
- `internal/infrastructure/appinstall/{helper_windows,start_windows,installer_user_windows,installer_result_windows}.go` 及相关测试。
- `build/windows/nsis/{data_dir_page,installer_runtime,strings}.nsh`。
- `internal/transport/wails/{about,automatic_download}_test.go`、`scripts/build-release.ps1`。
- 应用／前端／Windows 版本资源、`docs/releases/v2.0.0.md` 及相关发布文档。

## Verification

- 最终执行 `scripts/build-release.ps1 -Version v2.0.0 -OutputDirectory release-assets/v2.0.0`：
  PASS，退出码 0。包含最新结果长度保护的源码通过完整 `task --force verify`、
  `task --force package`、Windows 版本资源及 SHA256 资产校验。
- 格式、Go vet／staticcheck、ESLint、TypeScript、Go 全包测试、132 文件／809 项前端测试、
  Windows production 构建均 PASS；部分 Go 包使用缓存，另行运行下述无缓存专项。
- `go test -count=1 ./internal/platform/profile ./internal/infrastructure/appinstall`：PASS。
- `go test ./internal/transport/wails -run 'Test(DefaultUpdatePreferences|UpdateChannelCannotChangeDuringCheck)'`：PASS。
- 临时目录 EFS 特测：本机返回“不支持”，该专项显式 SKIP；没有据此声称通过 EFS 专项。
  实际带 Encrypted 属性的 Roaming 目录中隔离 QA 配置的新建与原子替换另行通过。
- 用户级原生安装、覆盖升级、卸载及中文路径协议：PASS。
- 最终安装器：`release-assets/v2.0.0/OmniProxy-Setup-v2.0.0-windows-amd64.exe`，
  20,242,608 bytes，SHA256 `f17e2a5709cbd459cff7f82b28768574fce75839a22e28df1c371ad9e3dd0560`。
  同目录生成配对 `.sha256`；Authenticode 为 `NotSigned`。
- `git diff --check`：PASS。构建未引入依赖、绑定或数据库迁移变化。
- **未执行**：真实跨用户／机器级 UAC、真实 Codex／Claude 请求和付费提供商调用。
  令牌隔离与用户级安装结果不冒充这些项目通过。

本地证据（工作区外，不纳入提交）：

- `C:/Users/mimanchi/.codex/visualizations/2026/10/26/01a17824-8bec-7e20-8fee-78aedc6c123e/stable-installer-helper-result.json`
- `C:/Users/mimanchi/.codex/visualizations/2026/10/26/01a17824-8bec-7e20-8fee-78aedc6c123e/stable-installer-native-result.json`
- `C:/Users/mimanchi/.codex/visualizations/2026/10/26/01a17824-8bec-7e20-8fee-78aedc6c123e/exec-encoding-result.txt`
- `C:/Users/mimanchi/.codex/visualizations/2026/10/26/01a17824-8bec-7e20-8fee-78aedc6c123e/v2-stable-release-final.log`

## Deviations

- 首次正式版质量门发现两个旧测试写死 Beta 默认渠道；改为显式 stable／Beta 契约测试，
  未修改正确的正式版默认渠道行为，也未跳过测试。
- 本机 Roaming 带 Encrypted 属性，路径式原子改名报跨卷；改用 Go 标准库父目录句柄原子改名，
  真实配置新建和替换通过，没有放宽为非原子复制。
- 原生 NSIS 检查发现独立于 Go 测试的中文代码页乱码，增加文件结果协议及长度保护。
- 初期隐藏向导验收脚本有同步窗口消息等待、按钮通知参数及 PowerShell 重载问题；
  失败未计通过。改用有界读取与正确异步按钮通知后完成真实用户级生命周期。
- 两份中间安装资产保留在忽略目录 `release-assets/pre-*-v2.0.0-local-only/`，
  均未公开发布，不能代替最终 `release-assets/v2.0.0/` 的产物。

## Remaining Release Checks

- 最终发布前仍需实际使用者确认机器级／跨用户 UAC 和主要客户端真实请求。
- 安装器未签名，SmartScreen／未知发布者体验需接受或另行完成签名。
- 开源许可证仍未选择，本任务不代替用户决定授权方式。
- 不创建版本标签、不推送、不执行公开发布。

## Git

`bc49edd384e56b521a0ab328541a7d334669436e` — `fix(release): 修复安装数据保护并准备 2.0.0`。

实现、测试、版本、发行说明及本聊天的前置审查记录已本地提交；随后归档本任务记录。
未创建标签、未推送、未执行远端发布。

## Result

安装器发布阻塞修复、回归验证及 `2.0.0` 本地正式版本资产准备完成。最终产物通过完整质量门、
打包与资产校验；用户级原生生命周期和中文旧目录通过隔离验收。可以交付给使用者做最后的
机器级／跨用户 UAC 与真实客户端确认，然后按既有发布流程创建标签并发布；本任务没有自动发布，
不把尚未执行的人工项目写成通过。

## Documentation

已同步 Architecture、BUILD、RELEASE、UPDATE、CURRENT 与 v2.0.0 发行说明。
遵循原有 Windows／Go／Wails 技术栈，无新增依赖或 ADR。主 helper 文件 203 行，
新增身份及结果协议已拆为独立文件，其余入口保持连续分派，暂不为三行超出指导值做无关拆分。

## Completed

2026-10-26（Asia/Singapore）
