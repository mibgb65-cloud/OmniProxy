# TASK-20261026-003: v2 正式发布就绪审查

## Status

Completed

## Created

2026-10-26

## Request

“你看看目前这个 v2 版本可不可以正式发布了？我感觉我做的都差不多了。”

## Goal

依据当前代码、测试、构建和发布流程，区分正式发布阻塞项、待验收项与非阻塞改进，给出有证据的发布建议。

## Scope

- 核对产品范围、当前版本、活动任务与发布检查文档。
- 审查发布、升级、凭证安全和关键请求路径。
- 执行适当的本地验证并记录限制。

## Out of Scope

- 不修改业务代码、版本号或发布配置。
- 不发布、不安装、不推送、不自动提交。
- 不访问真实用户凭证或调用付费上游接口。

## Analysis

- 当前版本为 `2.0.0-beta.44`，看板记录本地 production 安装包已生成、未签名且未公开发布。
- 仓库有明确的安装生命周期验收要求；本地构建通过不能替代真实安装／升级验收。
- 以代码与测试为准，不将旧任务未归档或尚未实现的非目标功能自动视作发布阻塞项。
- 本机 Go、Node、pnpm、Task、staticcheck 与 Wails CLI 已可用，执行当前工作区完整质量门。

## Implementation Plan

1. 阅读产品、架构、开发及发布规范。
2. 核实关键功能与未闭环的验收证据。
3. 运行质量门及针对性发布校验。
4. 记录发现、验证结果和发布建议。

## Implementation Result

本次只做审查、隔离复现与验证，不修改业务代码。

### 已复现的发布阻塞项

**F1 / P1：旧安装器数据目录兼容路径缺失，升级时可能将现有数据留在旧位置。**

- `profile.Detect` 会读取安装目录中的历史 `data-dir.txt`；`InstallerDataDir` 却只读取 `New` 和 `app.json`，未走这条兼容路径。
- `InstallerSetDataDir` 仅对上述错误的“当前位置”检查数据库，旧目录存在数据库时仍可能允许改写新位置。
- NSIS 从 `$PLUGINSDIR` 调用提取出的辅助程序，也没有传入真实旧安装路径；只在辅助程序里补读自身旁边的标记文件并不能完整解决安装现场的问题。
- 使用当前源码 `go build -tags production` 构建隔离辅助程序，创建旧 `data-dir.txt`、旧目录中的假数据库及空白配置。读取结果错误地指向隔离的默认 AppData；向新目录写入返回退出码 **0**，而不是已有数据应返回的 **3**；`app.json` 已指向新目录。
- 旧假数据库仍存在，问题是启动位置改变、用户看到空账号／历史，不是文件被删除。触发条件是历史自定义目录用户在升级向导中改选新的数据位置，不代表所有覆盖升级都会出错。
- 应统一应用与安装器对“实际数据位置”的解析，并在写入前对真实旧目录做保护；补充历史标记、`app.json`、独立 v2 数据库布局的回归用例。

隔离复现结果保存在本次工作区外：

`C:/Users/mimanchi/.codex/visualizations/2026/10/26/01a17824-8bec-7e20-8fee-78aedc6c123e/installer-review/legacy-reproduction.json`

### 高风险但未做真实 UAC 复现的路径

**F2：跨用户提权时，安装器数据目录读写可能使用管理员身份，而非桌面使用者身份。**

- 默认机器级包要求 `admin`，数据目录页通过 `nsExec` 直接启动辅助程序。
- 数据目录 read/write 分支直接使用当前进程的 profile／AppData，没有像 `startInstalled` 那样取交互桌面用户令牌及其环境。
- 当标准用户输入另一管理员的凭据完成 UAC 时，应验证读写的究竟是哪一用户的 `app.json`，以及新目录是否对实际运行用户可写。
- 此项依据代码调用链提出；本次没有触发提权、切换真实用户或修改任何真实配置，不能称为 UAC 实测通过或已复现。

### 当前候选版仍需补齐的验收

- 2026-10-18 的用户级／机器级真实新装、静默升级、重启、卸载已有成功记录，不能说项目从未验收。
- 后续安装向导已经改动；`TASK-20261024-013` 明确写明没有执行真实向导点击验收。当前正式候选应补测默认／自定义数据目录、旧版覆盖及取消提权。
- 新代理层及 Beta.44 弹窗有自动化覆盖，但真实 mixed／HTTP／SOCKS 端口与原生窗口验收仍在发布说明中标记待做。
- 固定客户端入口、effort 菜单等活动任务存在真实客户端待验收项。应选择承诺支持的主客户端做请求、流式、工具调用、断线／换号回归，不必把所有历史活动记录的未归档状态都当成阻塞。
- E2E 框架仍标记 planned；本次质量门不能代替安装后的端到端验收，可以先用可追溯的人工验收闭环，不要求为发布新增大型测试体系。

### 非阻塞事项与发布边界

- 本地 Beta.44 安装包资产／SHA256 校验通过，Authenticode 状态为 `NotSigned`。未签名是公开分发的信任与 SmartScreen 风险，不自动等同于业务功能不可用；SHA256 不是数字签名。
- `README.md` 的许可证仍是 `TBD`，如计划开源分发需先明确授权。不能从这一点推断产品不允许以闭源方式发布。
- Control API 当前只有 `/healthz`；架构文档也明确后续才控制网关。正式发布文案不能把它宣传为完整管理 API，但无须为桌面 v2 首发强行补齐。
- 完整候选仍是 `2.0.0-beta.44`。正式发版时需按现有脚本同步正式版本、发布说明并明确已知限制；不建议通过只改版本号来替代验收。
- 本次没有执行公开发布、真实用户安装或上游付费请求，也不是完整漏洞／许可证合规审计。

## Verification

- 当前工作区 `task --force verify`：PASS，退出码 0；格式、Go vet／staticcheck、ESLint、TypeScript、Go 全包测试、132 文件／809 项前端测试与 Windows production 构建均通过。Go 全包结果包含缓存命中，另执行下述 8 包无缓存专项。
- 不使用缓存的专项测试：`go test -count=1 ./internal/platform/profile ./internal/infrastructure/appinstall ./internal/infrastructure/database ./internal/infrastructure/credential ./internal/infrastructure/logger ./internal/infrastructure/update ./internal/application/updater ./internal/core/netproxy`，8 个包全部 PASS。
- `scripts/verify-release-assets.ps1 -Version v2.0.0-beta.44 -AssetDirectory release-assets/v2.0.0-beta.44`：PASS；安装器 20,198,610 bytes。
- `Get-AuthenticodeSignature`：`NotSigned`。
- `scripts/test-windows-metadata.ps1` 对本次新构建应用及已有 Beta.44 安装器校验：PASS。
- 隔离辅助程序复现 F1：成功复现错误行为，仅使用合成目录／数据库存在性夹具，不读取真实账号或凭据。
- 未执行：真实 NSIS／UAC／外部客户端与提供商请求验收；未把历史验收冒充本次验证。
- 非阻塞告警：Task 的 `uname`／`tail` 工具探测、Vite 大包、jsdom `scrollTo` 未实现与环境性能提示；质量门最终成功，未为隐藏告警修改代码或测试。
- `git diff --check`：PASS；完整质量门未造成受跟踪业务代码、依赖、绑定或配置变化。

完整质量门与专项日志：

- `C:/Users/mimanchi/.codex/visualizations/2026/10/26/01a17824-8bec-7e20-8fee-78aedc6c123e/v2-release-readiness-verify.log`
- `C:/Users/mimanchi/.codex/visualizations/2026/10/26/01a17824-8bec-7e20-8fee-78aedc6c123e/v2-release-readiness-targeted-tests.log`

## Documentation

仅新增本审查记录并同步 CURRENT 发布评估摘要。不修改产品要求、版本、业务实现或发布配置。

## Deviations

额外执行了隔离辅助程序复现，以区分一般性的验收建议和真实代码问题；未扩大到修复或实际安装。

## Git

审查结束时未提交；后续随 TASK-20261026-004 的本地发布准备收尾提交，提交信息见该记录。

## Result

**功能已接近发布收尾，可冻结功能并准备 RC；不建议当前直接发布稳定正式版。**

最小闭环：

1. 修复 F1，并在同一次安装器安全审查中确认 F2 的原使用者身份与目录权限。
2. 对最新候选执行隔离的新装／旧自定义目录升级／自动更新重启／卸载保留数据及主客户端请求回归；记录实际结果。
3. 质量门保持通过后，补正式发布说明与已知边界，按既有正式版本脚本发布；签名及开源授权根据实际分发方式明确决策。

不要求为首发补齐所有未来功能、全量 E2E 框架或机械清空活动任务列表。审查任务已完成，发现的问题尚未修复；没有创建标签、安装、推送、发布或提交。

## Completed

2026-10-26（Asia/Singapore）
