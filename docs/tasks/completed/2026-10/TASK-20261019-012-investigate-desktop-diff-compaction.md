# TASK-20261019-012: 排查 Codex Desktop 变更摘要缺失与上下文压缩

## Status

Completed

## Created

2026-10-19（Asia/Singapore）

## Request

用户反馈：v1 和官方接入时，Codex Desktop 对话中会显示本次新增／修改／删除的代码文件摘要；当前 Beta 不再显示。另外接近上下文上限后的压缩可能异常，要求排查原因。

## Goal / Scope

分别核对 Desktop 的结构化文件变更事件与工具调用，以及上下文阈值、模型能力、压缩请求端点和返回内容的完整链路。以代码、脱敏实际记录、官方协议及隔离回归定位；发现本任务范围内的确定缺陷时最小修复。不把两种现象直接判为同一原因，不用纯文本摘要伪装官方文件变更事件。

不读取或输出凭据／对话正文／真实补丁内容，不改写真实 Codex 配置、不重放用户会话、不擅自切换账号或安装版本。暂不生成新安装包，待具体修复及验证结果决定。

## Analysis

已查阅官方 App Server、配置和 Compaction 文档作为协议对照。当前 production 为 Beta.7、development 为 Beta.8。只读白名单摘要确认：当前 Codex 使用内置 openai 身份和受管本地目录，7 个 GPT 条目的 apply_patch_tool_type 均为 null；全局 model_context_window=1050000、model_auto_compact_token_limit=900000，WebSocket 与请求压缩关闭。未输出接入 URL／Key、提示词、对话或补丁。

代码确认 CodexJSON 未赋值原生补丁工具字段；gateway 只允许 responses／chat／messages，scopedAgentPath 仅接受六段路径，responses/compact 的七段路径被直接拒绝。固定 ChatGPT 适配器也只支持 Responses 推理，强制 stream=true，不能简单把 compact 当普通推理复用。已有 850K 任务主要是 Claude 客户端逐模型策略，不能套用到本次 Codex 压缩。

原生契约进一步确认：CLI 0.163.0 与 Desktop 内核 0.163.0-alpha.10.2 的压缩实际使用 Responses V2，并携带 X-Codex-Beta-Features=remote_compaction_v2 与 X-Codex-Turn-Metadata.compaction；旧网关均丢弃。只补 legacy /compact 不能解决当前核心的问题。新增真实核心→Core 代理→固定订阅适配器→假上游契约，修复前压缩失败，保留协议标记后收到 contextCompaction 完成项。模型目录 null 的原生对照无补丁工具／0 变更，修复后实际新增／修改／删除三项，均有结构化 file_change。

修复同时保留有界且校验的 V2 协议元数据，以及显式受信 Compaction 标志与固定订阅 legacy compact 端点，仅支持原生 ChatGPT Responses，保留不透明压缩项；不为第三方模型伪造压缩结果、不扩大普通 URL 转发、不改用户的全局窗口与既有会话。模型目录仅为原生 OpenAI Responses 开启 freeform apply_patch，其他提供商维持原状。先验证失败回归，再实现。

## Implementation Plan

1. 阅读相关项目规范、LOCAL_AGENTS、长上下文既有任务与 ADR，核对当前生产健康状态及版本。
2. 追踪 Codex 配置／模型目录、工具调用与流事件处理，检查文件变更项是否有不兼容或丢失。
3. 追踪自动压缩阈值、使用量及 /responses/compact 端点支持，使用只含固定元数据的只读诊断核对请求记录／安全日志。
4. 对已证实缺陷补回归、最小修复；按变更执行定向测试和完整质量门，记录结论、证据边界、剩余风险与提交。

## Continuation（2026-10-19）

用户要求继续完成本任务。续办范围限定现有 Desktop 文件变更与上下文压缩修复，不清理其他任务记录、不修改版本、不安装或重启生产程序。

- 已复核现有代码、协议边界、独立压缩用量统计与未提交文件归属；五个相关 Go 包的非缓存定向测试通过。
- 将重新执行两个已安装原生核心的显式离线契约、补齐必要边界回归、运行独立输出目录的完整质量门，再记录实际结果并提交归档。
- 原分析中的进程／版本属于前次调查快照；续办时另行记录配置哈希、现有生产与开发进程基线，只比较是否被本次工作改动，不据旧 PID 宣称当前运行状态。

## Implementation Result

- 受管 Codex 模型目录仅为原生 OpenAI Responses 条目显式声明 `apply_patch_tool_type=freeform`，补充手工编辑优先使用原生补丁工具的通用提示；第三方模型不推断此能力。
- 普通 HTTP Responses 保留经功能白名单、长度、JSON 结构／重复字段及控制字符校验的 Compaction V2 元数据；WebSocket 可覆盖当前帧回合元数据，转发后清空旧回合信息，避免压缩标记污染下一回合。
- 为旧核心增加固定 `/v1/responses/compact` 及 Codex scoped 路径。由端点设置受信 Compaction 标志，要求数组输入／非流式请求，经原生订阅固定 HTTPS 端点返回 JSON，不走 Chat／Messages 转换，不修改不透明压缩项。
- 沿用鉴权、正文容量、请求与尝试生命周期、用量统计、安全错误及不重放已接受请求的规则；不支持压缩的第三方适配器明确返回 501，不伪造压缩成功。
- 补齐目录、元数据 HTTP／WebSocket、legacy 压缩、应用层用量／失败不泄露／不重放及固定订阅适配器测试；增加两个显式启用的真实本机核心离线契约，不继承用户登录与规则。
- 无新依赖、数据库迁移、版本变更或架构边界调整。未清理旧 Atria 规划稿或其他任务记录，未生成安装包、推送、安装或主动重启应用。

### Files Changed

- `internal/core/modelcatalog/codex.go`、`codex_patch_test.go`：原生补丁能力及第三方边界。
- `internal/core/proxy/`：固定路径／请求标志、V2 元数据校验和转发、压缩结果验证及相应回归。
- `internal/application/gateway/agent.go`、`compaction.go`、`compaction_test.go`：原生订阅分流、用量与安全失败验证。
- `internal/provider/chatgpt/client.go`、`compaction.go`、`compaction_test.go`：共享固定端点发送逻辑及 JSON 压缩操作。
- `internal/infrastructure/codex/native_patch_integration_test.go`、`compaction_integration_test.go`：真实核心离线契约。
- `docs/product/LOCAL_AGENTS.md`、`docs/architecture/OVERVIEW.md`、`docs/plans/CURRENT.md` 与本任务记录。

## Verification

- PASS：`go test ./internal/core/modelcatalog ./internal/core/proxy ./internal/application/gateway ./internal/provider/chatgpt ./internal/infrastructure/codex -count=1`，五个相关包全部通过。
- PASS：HTTP 元数据边界与既有 V2／WebSocket／Compaction 回归 `-count=3`；重复协议头、重复 JSON 字段、错误类型及超长元数据在到达上游前拒绝，错误响应不回显原始元数据。
- PASS（预期失败验证）：Go overlay 仅在忽略的临时目录模拟撤去补丁能力、丢弃 V2 头两种缺陷；目录和应用网关回归分别按预期失败，未替换工作区源文件。
- PASS：CLI `0.163.0` 与 Desktop 内核 `0.163.0-alpha.10.2` 分别显式设置 `OMNIPROXY_CODEX_TEST_EXECUTABLE`、`OMNIPROXY_CODEX_PATCH_QA=1`、`OMNIPROXY_CODEX_COMPACT_QA=1`，运行两个本机契约 `-count=3 -v -timeout=180s`。每个核心每轮均验证目录能力关闭时无补丁工具／0 文件变更，开启后实际新增／修改／删除三个临时文件并完成 turn；真实核心经 Core 代理与固定订阅适配器收到一次 V2 压缩请求，返回完整 `contextCompaction` 项。
- PASS：`task --force verify BIN_DIR=tmp/desktop-compat-20261019/finish-verify-bin`，退出码 0；gofmt／Prettier、go vet／staticcheck／ESLint、类型检查、全包 Go 测试、107 文件／613 项前端测试、绑定生成及 Windows production 构建全部通过。二进制输出到独立临时目录，不覆盖正在使用的 `bin/` 或生产程序。
- PASS：真实 Codex 配置 SHA256 与续办基线相同；生产 PID 5336、启动时间及二进制 SHA256 未变。开发基线 PID 35384 在工作期间退出，后出现同路径 PID 47028，`bin/omniproxy.exe` 哈希未变；本次没有主动启动／终止该进程，不把开发进程声称为未变化。
- PASS：本任务手写 Go 文件均不超过 200 行；提交前核对 UTF-8 无 BOM／LF、`git diff --check`、差异归属与敏感信息。无生成绑定、go.mod／go.sum 或构建版本文件的实质差异。
- 非阻塞告警：Vite 主包超过 500 KiB；Vitest 给出运行性能建议。未为通过门禁隐藏告警或改动无关前端。

本地证据位于忽略目录 `tmp/desktop-compat-20261019/`：`finish-targeted.log`、`finish-native-cli.log`、`finish-native-desktop.log`、`negative-catalog.log`、`negative-v2-headers.log`、`finish-verify.log` 与退出码文件。该目录、基线摘要、模拟凭据和 QA 二进制均不提交。

## Documentation

已补充本地 Agent 产品说明和架构概览，说明目录刷新／客户端重载、新旧压缩协议、固定端点及不可伪造的能力边界；归档时同步当前看板。遵循 ADR 005／008，不引入新架构决策。

协议对照来源：

- [Codex App Server](https://developers.openai.com/codex/app-server)：原生文件变更、回合 diff 和手动压缩的生命周期。
- [Compaction](https://developers.openai.com/api/docs/guides/compaction)：压缩项不可解释，应保留规范输出作为后续上下文。

公开 API 文档不能证明订阅私有 V2 协议细节；本任务关于 V2 标记的结论以已安装核心的离线契约为依据。

## Remaining Risks

- 未调用真实订阅上游、未复现用户历史会话、未执行 Desktop GUI 外壳视觉验收；不把离线核心契约等同于生产端到端验收。
- 本次通过 `thread/compact/start` 验证原生 V2 压缩链路，没有用真实 900K／1.05M 长会话验证自动阈值、摘要质量或后续会话质量；未改变用户全局阈值。
- 修复尚未重新打包／发布／安装。需要先运行包含修复的版本，再通过已有接入流程刷新受管目录并重新加载客户端，才能让新工具能力生效；不得绕过外部修改保护。不会补生成旧会话中终端写文件的历史变更摘要。
- 仅验证上述两个已安装核心版本。官方私有协议变化、第三方模型自定义工具及压缩支持不能由此泛化。

## Git

实现提交：`697942bc512737413955375e6b983f07ae0b153b` — `fix(codex): 修复原生文件变更能力与上下文压缩链路`。

仅本地提交，未推送远端；完成状态、提交哈希与归档由后续文档提交保存。

## Result

两个已证实兼容缺陷的最小修复、必要回归、完整质量门及本地实现提交已完成，任务归档为 Completed。生产程序与真实 Codex 配置保持原样，发布／安装及真实上游／GUI 验收属于后续交付阶段，不纳入本次完成范围。

## Completed

2026-10-19 20:32（Asia/Singapore）
