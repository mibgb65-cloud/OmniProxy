# TASK-20261020-008: 网关启动设置、额度切换与内容兼容性

## Status

Completed（2026-10-27 随 v2.0.0 正式发布归档）

## Created

2026-10-20

## Request

1. 通用设置增加「软件启动时是否默认启动网关」。
2. 登录账号额度耗尽时，网关仍有可用账号却无法发消息，需要排查并改善体验。
3. 使用其他模型报 `unsupported_content`，需要定位原因。

用户补充确认：第二项由 Codex Desktop 在发送前直接阻止，因此没有请求进入网关。
不再要求用户以网关请求记录证明同一事实。

用户要求只读对照 `E:\go\new\OmniProxy`：旧版同样使用 openai 登录模式，但没有遇到相同发送拦截。

用户进一步授权「先尝试旧版的操作方式」。本轮先以临时 CODEX_HOME、合成授权和
本机实际 Codex 核心验证旧版 file 登录是否仍有效；若兼容，再实现带加密备份、冲突
检查和恢复的账号同步试验。保留 model_provider=openai，不改会话数据库／索引，
不直接照抄旧版无保护覆盖凭据，不自动重启当前聊天所在的 Desktop。

## Goal

沿用现有设置、启动生命周期、故障转移和协议适配实现最小修复；区分客户端额度拦截与网关内部切换，不将无法复现的问题宣称为已修复。

## Scope

- 启动设置持久化、桌面生命周期、设置界面及中英文文案。
- 额度错误识别、账号调度与会话恢复边界。
- Responses 内容兼容检查与模型适配。
- 必要回归测试、产品说明与质量门禁。

## Out of Scope

- 绕过官方客户端／工作区额度限制。
- 未经用户确认覆盖真实登录、重置生产账号或重放用户消息。此次旧方式试验以临时配置
  为起点；真实授权只能经后端受控回调使用，不输出秘密，且必须先验证备份／恢复。
- 无关重构、提交、发布或安装新版本。

# Analysis

## Related Modules

- profile 配置与 Wails ServiceStartup；通用设置已有 OS 自启动和关闭行为。
- application/gateway 已实现同池账号切换、明确额度拒绝识别和生成后不可重放保护。
- River 原生 Responses 校验只支持文本／函数工具；不应将包含图片的请求伪装为文本。

## Evidence

- 只读生产库：2026-10-20 12:41:13（新加坡时间），`DeepSeek-V4.1-Flash-1017` → River，Responses WebSocket，单次尝试返回 400 / unsupported_content。
- 仅统计本聊天本地事件的角色、时间、内容类型：12:40:55 用户输入含两个 input_image，未输出消息文本或图片数据。River 的 textContent 会明确拒绝这类历史；已确认可复现的拒绝条件，但未保存当时完整请求正文，不能唯一归因或排除工具／其他参数限制。
- 客户端接入代码有意保留内置 openai 身份与官方认证文件，因此不能凭网关账号池消除客户端自身的工作区额度提示。独立提供商分支已设置 requires_openai_auth=false；不擅自切换身份或改写真实客户端配置。
- 只读生产配置：账号故障切换开启，两个启用且已认证的 OpenAI 订阅账号，相关 GPT 模型无启用的手工固定账号路由。
- 生产请求 2026-10-20 12:02:20（新加坡时间），gpt-6.1-sol / WebSocket，首次尝试 429 / upstream_quota_exhausted，第二次尝试使用不同账号返回 200，最终 success。说明真实网关换号链路已成功工作；不将截图中的客户端工作区提示直接等同于该链路失败。
- 用户进一步明确第二项是 Desktop 第一层发送拦截，不存在对应网关请求。因此前述成功换号只能证明网关路径可用，不能用于反驳或解决本次前置拦截。
- 核对 `buildCodex` 与 ADR-005：内置 openai 模式只改 openai_base_url，特意保留提供商身份、登录与会话筛选；没有将 Desktop 切为独立提供商。该设计保留原聊天，但未解除 Desktop 自身登录额度约束。
- 官方配置参考确认自定义 provider 与 requires_openai_auth 配置存在，且内置 openai 不能覆盖；文档并不保证特定 Desktop 版本会据此解除其发送门禁，必须单独验证，不能沿用 CLI 成功结论。

## 旧版对照（用户指定路径，只读）

- `OmniProxyBackend/codex_config.go:135`：一键配置调用 selectFreshCodexClientAuth，有合格账号时通过 writeCodexAccountAuth 写入默认 Codex auth.json，明确使用 chatgpt 登录；并非只修改请求地址。
- `codex_config.go:464`：按 Active、Low 状态依次选择账号，优先当前导入和选中账号；禁用、已标记 Exhausted／Invalid，以及缺邮箱／套餐／access token 的账号不参与客户端登录选择。根据的是账号池已有状态，不保证实时额度查询。
- `codex_config.go:417`：写 forced_login_method，并将 cli_auth_credentials_store 固定为 file，让客户端读取写入的 auth.json。GPT 配置仍保留内置 model_provider=openai。
- `codex_config.go:535`：写入真实池账号授权内容，移除 OPENAI_API_KEY，防止混用 API 登录。没有可用且包含套餐的账号时，另有 API 登录回退分支。
- 当前项目 `internal/application/agentconfig/patch.go:76` 和 ADR-005 明确不改客户端官方登录及凭据存储，只合并 openai_base_url 等受管配置。这是已确认的客户端认证策略差异，不能据“都叫 openai”视为等价。
- 源码中 selectFreshCodexClientAuth 和 writeCodexAccountAuth 的调用来自一键配置，未发现每次模型请求或额度耗尽后自动同步默认客户端登录的调用。因此不能把旧版实现描述为已证明能永久绕开 Desktop 门禁。
- 撤回此前“需要先切换独立提供商”作为首选路径的判断：保留 openai 并同步客户端登录是旧版已有的替代设计。其对当前 Desktop 凭据格式、登录重载和耗尽情形的实际效果仍需隔离验证。

## Risks

启动默认值须保持既有行为；流式响应已输出后不可盲目跨账号重放；不同提供商支持的内容类型不同，不应静默丢弃内容。

# Implementation Plan

1. 核对现有文档、实现和测试，定位三项需求的实际边界。
2. 增加启动网关设置并覆盖持久化和启动行为。
3. 修复证实存在的额度切换、内容兼容问题，补充回归测试。
4. 执行定向验证和 `task verify`，同步任务与相关文档。
5. 根据本机 Codex 0.163.0 对旧 file 登录的实际支持结果，决定是否实施受控同步；
   不再将旧版源码可运行直接视为当前 Desktop 兼容。

# Progress

- 用户补充第三项具体错误码为 `unsupported_content`。
- 选择默认关闭的 profile 布尔设置以保持历史行为；只在应用启动时读取，不把切换偏好等同于立即启停。
- 第二／三项优先做已有安全边界的定向验证和解释，不绕过客户端限制、不删除图片或虚构上游视觉能力。
- 已实现启动偏好与中英文界面，生成 Wails 绑定；保留默认手动启动。
- 已完成定向测试、原生 Codex CLI 合成上游换号验证及完整质量门。
- 原生开发窗口已核对新开关并截图；隔离 production 构建在 test profile 连续三次启动验证关闭／开启／关闭。

# Implementation Result

## Changes

- `profile.Config.GatewayAutoStart` 默认 false，沿用原子配置保存，修改不会覆盖其他设置。
- Wails `DesktopSettings` 和 `SetGatewayAutoStart` 提供读取／保存；仅 ServiceStartup 读取偏好并启动网关，失败记录日志，不退出桌面应用，允许手动重试。
- 通用设置增加独立开关、保存中／失败反馈和中英文说明，使用既有 Switch 组件及 Query 缓存。
- 新增配置损坏／其他字段保留、默认状态、启停偏好不立即改变状态、端口冲突及原生启动不退出测试。
- 新增 River 图片历史边界回归：最新问题即使纯文本，既有图片仍明确拒绝，不能静默删除图片再转发。
- 第二项没有发现可据此修复的网关换号缺陷；测试和一条真实生产记录均已验证成功换号。第三项定位到 River 当前协议适配限制，未扩展图片／专用工具能力。
- 用户补充后将第二项明确归入 Desktop 前置额度拦截；修复范围转为客户端接入模式与聊天身份兼容，不增加无效的网关重试。
- 已只读核对用户指定旧版的配置、账号选择和授权写入链路，并执行相应临时目录测试；未读取或替换真实 auth.json，也未修改旧项目。

## Files Changed

- `internal/platform/profile/config.go`、`desktop.go` 与启动偏好测试。
- `internal/transport/wails/app.go`、`desktop_settings.go`、`gateway_startup.go` 与相邻测试。
- 设置组件、查询 Hook、服务包装、中英文文案、设置测试及生成绑定。
- `internal/provider/river/image_boundary_test.go` 与相关产品说明。

## Remaining Work

- 第二项已由用户确认为 Desktop 发送前拦截，无需继续索取对应网关请求。优先沿旧版同 openai 提供商的客户端登录同步方向，在隔离 Desktop 中验证凭据存储、重新加载及额度门禁；验收必须覆盖当前客户端登录账号耗尽、网关另有可用账号时的发送和原聊天续接。
- ADR-005 同时保护提供商身份和官方登录文件，不能直接照抄旧版覆盖真实 auth.json。若引入客户端登录同步，应设计显式用户确认、加密备份、版本冲突检查及恢复，并单独评估当前 Desktop 的凭据格式／工作区和插件行为；本轮只读对照不是写入真实授权的许可。
- 截至只读对照阶段，未迁移提供商身份、删除会话历史或改写真实认证；用户随后授权
  试用旧方式的实际操作和边界见后续结果。
- 用户授权旧方式试验后，已实现独立受控工具：只读查询官方额度，选择完整窗口均可用的
  池账号，核验身份后使用 ChatGPT/file 登录，保留 openai 与原网关地址／模型。不会回退
  到 API 登录，也不自动同步后台登录或改动系统加密授权命名空间。
- 原 config.toml／原 auth.json 以 DPAPI 加密备份，目标目录只保存恢复引用；写入检查
  文件版本，失败回滚自有修改，恢复拒绝覆盖外部变化，备份绑定准确目标路径。
- Windows 命名 Mutex 阻止同一目标并发试验；没有通过 Wails 暴露完整授权。
- 新增 `cmd/codex-login-trial` 与说明、agentconfig 试验／恢复及边界测试、
  infrastructure/codex 后端完整授权回调和独立 OpenFile 验证入口；默认 Open 仍用 keyring。
- River 图片／namespace／custom 工具透传仍未接入；本轮交付是原因与边界诊断，不是完整多模态兼容修复。
- 当前 Codex 0.163.0 的 file 登录格式识别已通过，真实池账号在独立 CODEX_HOME 的
  app-server account/read 校验也已通过；该临时目录随后成功恢复，auth.json 与恢复
  引用均已移除，未留下临时明文授权。
- 实际 Desktop 的发送门禁仍须在应用旧方式并完全重启 Desktop 后验收。当前聊天所在
  Desktop 不自动重启；若其门禁依据全局工作区而非该文件登录，仍可能需要进一步排查。
- 文件登录与池内授权是两个存储副本，后续 refresh token 更新／缓存重载兼容尚未验证；
  试验期间普通接入配置可能显示外部变更，不能同时运行普通一键配置／恢复。

## 旧版登录试验实际操作结果

- 用户授权后，在 production 账号池只读筛选官方额度，跳过三个未能确认完整额度的候选，
  不将其当作可用；对已确认有额度且身份核验通过的账号进行同步。
- 已对实际 `C:\Users\mimanchi\.codex` 执行旧方式试验：保留 openai、原网关地址及模型，
  设置 ChatGPT/file，写入选定池账号的 auth.json；没有采用 API 登录回退。
- 原 config.toml 和原 auth.json 已通过 production profile 的 DPAPI 存储备份；
  默认目录存在 `omniproxy-login-trial.json` 非敏感恢复引用。
- 实际默认配置下 `codex login status` 已识别 ChatGPT 文件登录，输出只记录布尔结果，
  没有打印账号身份、凭据或原配置。
- 默认 `secrets/codex_auth.age` 的 SHA256 在操作前、操作后及 CLI 状态核验后均相同，
  系统加密登录未改。未改写聊天／会话索引，未发送真实模型请求，未重启 Desktop。
- 为避免中断本聊天，Desktop 的完全退出／重开和发送验收由用户随后进行。第二项仍未
  标记为已修复；不把文件登录已生效当成 Desktop 发送门禁已解除。
- 恢复命令：`bin/codex-login-trial.exe -profile production -home C:\Users\mimanchi\.codex -restore`。
  恢复拒绝覆盖试验后新的外部修改；不要删除恢复引用或 DPAPI 备份。
- 不含正文／凭据的前置文件摘要保存在本聊天 visualization 目录的
  `legacy-login-default-baseline.json`；临时真实授权试验目录已恢复并移除 auth.json。

# Verification

## Tests

- PASS：`go test ./internal/platform/profile ./internal/transport/wails -run 'GatewayStartup|GatewayAutoStart|DesktopSettings' -count=1`。
- PASS：`go test ./internal/provider/river ./internal/provider/chatgpt ./internal/application/gateway -run 'ResponsesImageHistory|Flash1017NativeRequests|QuotaDenial|SubscriptionQuota|CodexSocketFailover' -count=1`。
- PASS：设置 `OMNIPROXY_CODEX_TEST_EXECUTABLE` 为本机 Codex，执行 `TestCodexNativeQuotaFailoverAndLargeEvent`。只访问合成上游，使用临时 CODEX_HOME／工作区／配置／凭据，无真实上游请求。
- PASS：完整 `task verify BIN_DIR=bin/verify-task-20261020-008`，退出码 0；格式、Go vet／staticcheck／ESLint、TypeScript、全部 Go 测试、113 文件／671 项前端测试及 Windows production 构建通过。
- PASS：项目 `bin/omniproxy.exe` 对应的原生开发窗口可见新开关，截图已查看，布局无截断；仅打开后关闭设置，未更改其偏好。
- PASS：最终验证构建用独立配置／数据／WebView／随机 loopback 端口，以 test profile 连续启动 false／true／false，实际网关监听状态符合预期；测试进程已退出。
- PASS：`git diff --check`；保留用户既有未跟踪记录，未提交／安装／发布。

## Verification Notes

- 首轮定向前端测试暴露 mutationFn 直传会收到 Query context；按既有模式改为只传 enabled，完整门禁中 671 项全部通过。
- 隐藏隔离窗口不能读取设置无障碍入口，未把该尝试记为通过；改用现有项目原生开发窗口检查界面，隔离隐藏进程只核验真实启动生命周期。
- 门禁存在既有 jsdom scrollTo 未实现提示及前端大 chunk 警告，均未导致失败，不在本轮扩展修复。
- 门禁日志：`%TEMP%/omniproxy-task-20261020-008-verify.log`。
- 原生截图与隔离验证脚本／运行数据在本聊天 visualization 目录，未放入仓库或读取真实凭据。
- 较早的只读补充仅核对文档、代码与 ADR，没有产品代码变更；本轮授权试验已重新执行
  完整质量门及上述专用验证。第二项的 Desktop 门禁解除仍无实测，不列为 PASS。
- PASS：旧版在 `E:\go\new\OmniProxy\OmniProxyBackend` 执行 `go test -mod=readonly . -run '^(TestConfigureCodexWritesPoolAccountForChatGPTLogin|TestConfigureCodexCreatesAPIKeyFallbackWithoutAccount|TestConfigureCodexCreatesAPIKeyFallbackWhenAccountPlanTypeIsMissing|TestSelectCodexClientAuthPrefersImportedAccount|TestConfigureCodexWritesSelectedModelProfiles)$' -count=1`。测试使用临时 HOME／USERPROFILE、伪凭据和临时账号池，证明旧配置分支存在且契约成立，不等同于真实 Desktop 额度门禁测试。
- PASS：本轮 `go test ./internal/application/agentconfig ./internal/infrastructure/codex ./cmd/codex-login-trial -count=1`，覆盖 OAuth 字段校验、API 字段移除、原文件／不存在状态恢复、加密备份失败不写入、外部变化保护、配置失败回滚授权、跨目录恢复拒绝、工作区／提供商／入口冲突。
- PASS：显式启用 `OMNIPROXY_CODEX_LEGACY_QA=1`，当前安装核心 0.163.0 识别临时
  ChatGPT/file 登录；仅合成凭据，不代表账号可用性已验证。
- PASS：真实账号池只读额度筛选 → 后端回调读取 → 临时目录试验 → 隔离 app-server
  识别预期池账号 → 恢复。未调用模型，日志只含计数／状态，不输出令牌或身份。
- PASS：命令 `go test ./cmd/codex-login-trial -count=1`、`go vet`、`staticcheck`、
  `go build -o bin/codex-login-trial.exe ./cmd/codex-login-trial`；命名锁并发／释放测试通过。
- PASS：再次执行 `task --force verify BIN_DIR=bin/verify-task-20261020-008-legacy`，
  退出码 0；格式、lint、类型检查、全部既有 Go 测试、113 文件／671 项前端测试及
  Windows production 构建通过。命令包另行执行上列检查，未假称原门禁包含 cmd。
- 合成授权的 app-server account/read 返回 request_failed，不能当真实授权成功；
  离线测试只断言 CLI 能识别文件格式，真正身份识别由上述真实池账号隔离验证负责。
- 命名锁回归测试发现 CreateMutex 返回有效句柄并同时报告 ERROR_ALREADY_EXISTS 的
  Windows 语义；已按可用已有对象继续等待／互斥处理，未删除或弱化测试。

# Documentation

- 已更新 PRD、GATEWAY：默认值、设置位置、下一次软件启动生效及失败重试。
- 已更新 FAILOVER：客户端工作区额度与网关账号池的区别和诊断步骤。
- 已更新 RIVER：图片历史、函数工具和会话引用限制，强调官方视觉能力不等于当前适配器能力。
- CURRENT 保留本任务为进行中，记录实际默认客户端试验已应用，等待 Desktop 重启后发送
  和旧聊天兼容验收，而不是继续要求用户查找不存在的网关请求。
- 用户授权试旧方式后，ADR-005 增补显式诊断试验例外；默认一键接入仍保持原策略。
  FAILOVER 与命令 README 已说明授权、备份恢复、重启验收、并发保护及刷新副本风险。
  无新增依赖、数据库迁移、提供商身份或聊天数据库变更。
- 官方配置参考：<https://learn.chatgpt.com/docs/config-file/config-reference>，仅用于确认配置契约，不当作 Desktop 门禁修复证据。

# Deviations

原计划对证实的换号／内容缺陷实施修复；实际发现网关已有成功切换证据，River 内容限制属于尚未接入能力。用户补充已确认第二项是 Desktop 前置拦截，网关修复不能解决。旧版对照又确认同 openai 模式下有客户端登录同步，不应过早要求独立提供商。没有通过删除输入、盲目重试或覆盖真实授权改变安全边界，保留安全登录同步与身份兼容方案待验证。

# Git

用户已授权提交，以下代码已直接提交在本地 main，无需重复 merge：

- `b6f33987f1abd3d3910c9dddd9b90adbccf7b0f8 feat(settings): 支持启动软件时自动开启网关`
- `3b7e7392030fa98b394d4ce8ae5d65ce4af4aecb feat(codex): 增加带加密备份和恢复的旧版登录试验`
- `dced0551e016ca814f9f28cc0bf33513f7d25252 test(river): 明确图片历史兼容边界并补充回归验证`

不推送远端，Desktop 发送验收仍未完成，任务保持 In Progress。分支整理见 TASK-20261020-011。

# Result

第 1 项实现并验证完成。第 2 项在用户授权后，已完成带 DPAPI 备份／恢复的旧方式工具、
隔离真实池账号验证与临时恢复，并已应用到实际 Codex 默认配置；CLI 已识别 ChatGPT
文件登录，原 openai／模型／网关及系统加密登录保留，但实际 Desktop 重启后发送仍待用户
验收，不标记已修复。第 3 项仍为已定位的内容兼容限制，没有扩展图片／专用工具。
继续保留为 In Progress。

---

# Archival (v2.0.0)

- 原状态：In Progress
- 2026-10-27：v2.0.0 正式版已公开（标签源码 `c9c48b4a31988a6cc3fa2f4b1d423e61bd10f71b`，
  Actions 完整质量门与 Windows 构建通过），本任务交付内容已随该版本交付。
- 归档依据：归档时工作区无未提交代码改动，本记录引用的实现提交均为上述发布提交的祖先提交。
- 未执行项（实机客户端／原生窗口视觉／真实覆盖安装验收等）不伪造为已验收，转为发布后
  跟进事项，见 `docs/plans/CURRENT.md` 的 Post-release Follow-ups。
