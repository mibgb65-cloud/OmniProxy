# TASK-20261021-008: 排查 v2.0.0 高内存占用

## Status

In Progress

## Created

2026-10-21 16:00（Asia/Singapore）

## Request

用户提供任务管理器约 2,109 MB 占用截图，希望检查 v2.0.0 为什么占用高、应用内哪些服务最可能占用内存。

2026-10-21 追加：用户重启电脑后启动占用暂时正常，但开始使用中转就达到约 1 GB，明确要求优化。

用户已确认：不修改官方加密格式或降低工作因子，采用 Windows 保护的有界缓存、源文件／凭据变更失效和实际解密后的大工作区回收。

## Goal

区分桌面宿主、WebView、后台服务与缓存，复现并修复 Codex 授权解密的高内存重复分配／保留，明确冷读取峰值与已安装版本的边界。

## Scope

- 只读观察现有生产进程、相关子进程及本机监听端口。
- 核对启动服务、代理请求处理、日志／历史和缓存的内存边界。
- 必要时在隔离测试环境进行内存分配与保留验证。
- 保存实际排查结果和验证证据。
- 在隔离中转复现并采样定位后，对证据充分的分配／保留问题做最小修复和回归验证；不改变协议、安全或会话兼容边界。

## Out of Scope

- 不停止、重启、升级或替换用户正在使用的生产应用。
- 不读取／输出真实凭据或请求正文，不抓取可能含敏感内容的生产进程内存。
- 不把进程总内存臆测分配给单个 Go 服务，不进行无关重构。

# Analysis

## Existing Behavior

首次进程快照显示主进程 `OmniProxy.exe` PID 34484 工作集约 2.06 GiB，私有提交约 2.24 GiB；其 WebView 子进程显著低于主进程。需要继续确认统计口径及时间趋势。

## Related Modules

已检查桌面服务装配、Control API、网关并发与正文上限、WebSocket 会话历史、Google 工具签名缓存、日志环形缓存及 SQLite 连接配置。

## Progress

- `/healthz` 确认生产进程实际为 `2.0.0-beta.17`，13:04 启动。主进程约 2,118 MiB 工作集／2,295 MiB 私有提交，WebView 不是主要占用来源。
- 只读取 Windows 虚拟内存区域元数据（没有读取进程内存正文）：主进程存在两块约 1 GiB 的私有可读写提交区域，合计约 2 GiB；其余单块最大约 110 MiB。需要区分 Go 堆与原生分配。
- Control API 仅提供 healthz，没有运行时 heap／pprof 诊断入口，无法直接从线上精确测出每个 Go 服务的保留对象。

## Risks

生产进程保持只读，不向生产网关发送推理请求。缓存只能保留系统保护后的内容，不能留存明文授权；每次读取仍须核对源文件和系统凭据。冷读取仍需满足官方 KDF 的约 1 GiB 瞬时需求，不能宣称完全消除峰值。

# Implementation Plan

1. 核对项目边界、运行版本、进程树、内存趋势和诊断入口。
2. 沿服务装配检查有界缓存与大对象生命周期，优先检查 Go 主进程。
3. 针对具体疑点使用无真实数据的隔离验证；如需修复，先明确影响和验证范围。
4. 汇总已证实事实、疑似根因、采样限制和建议。
5. 按追加需求，先分别复现 Core 转发、提供商转发和请求记账，采集假数据分配栈；仅对复现根因做修复。
6. 回归覆盖资源生命周期及原协议边界，运行 `task verify`，记录实际效果和未解决项。

## 已复现根因与实施方案

- 300 轮假上游 Responses WebSocket＋真实隔离记账，工作集约 23–45 MiB，未复现 GiB 保留。
- `readEncryptedAuth` 每次读取官方 `codex_auth.age`；`UseTokenAt → prepareToken → codex.UseToken` 在每次中转准备凭据时进入这条路径。
- 官方 work factor 20 调用 `scrypt.Key(N=2^20,r=8,p=1)`，其 `v := make([]uint32,32*N*r)` 必需 1 GiB。完全虚构密文实验的第一次／第二次解密后工作集约 1,044／2,070 MiB，分配 profile 指向 `golang.org/x/crypto/scrypt.Key`，精准复现量级。并非已经证实的 WebView／日志／会话历史泄漏。
- 实施：最多 16 个、固定 5 分钟有效期的按源路径隔离缓存；只保存复用既有 DPAPI 保护后的授权快照。密文内容及每次新读到的系统 passphrase 摘要共同校验，不用 mtime 作为唯一依据。
- 所有实际解密串行，只有大工作因子解密完成后主动归还已失去引用的 KDF 工作区。保留官方刷新、账号版本／身份校验和消费后清零；注销清理缓存。
- 不新增依赖、数据库迁移、公开 API、明文文件或常开诊断端点；测试覆盖命中、更新、错误口令、删除、TTL、容量、并发、保护失败、缓存损坏及旧 token 身份边界，并做真实 factor 20 的假数据专项验证。

## 重启后复测

- 用户自行重启电脑与应用，新进程 PID 14716，16:41:22 启动，仍为 `2.0.0-beta.17`。
- 17:01:44 采样工作集约 1,294.5 MiB／私有提交 1,332.9 MiB；随后采样工作集约 2,417.2 MiB／私有提交 2,457.6 MiB，两块私有提交区域约 1,024／1,020.28 MiB。
- 与用户“开始中转后上 GB”的反馈一致，不能视为重启已解决。不同时间仍有真实流量，最终根因需隔离采样，不推断特定 DLL 或服务负责。

# Implementation Result

## 已确认的事实

- 本机健康检查返回 `2.0.0-beta.17`／production，主进程 PID 34484，启动时间约 13:04。
- 16:00 起主进程工作集先约 2,118 MiB／私有提交约 2,295 MiB；后续快照为工作集 2,320.5 MiB／私有提交 2,388.1 MiB。采样期间有真实代理请求，不能把两个快照当作空闲泄漏曲线，也不能把不同内存口径相减后归为泄漏。
- 后一快照中六个 WebView 子进程工作集合计约 171 MiB，私有提交合计约 308 MiB。GPU 子进程工作集约 28 MiB，renderer 约 71 MiB。因此截图的大头在宿主主进程，不是 renderer／GPU 独占约 2 GiB。
- 主进程虚拟内存区域元数据显示两块约 1 GiB 的私有可读写提交，分别约 1,024 和 1,023.52 MiB。`MEM_PRIVATE` 不能区分 Go 堆、原生堆和直接分配，不能据此宣称是某个 Go 服务或某个第三方模块。
- 线程数采样为 32、句柄约 985–988。Windows 线程数不等于 Go goroutine 数量。
- 只读 SQLite 聚合时，历史明细约 4,210 条、保留上限 10,000，pending 1 条。本次运行主要流量为 OpenAI Responses WebSocket（约 224 次），其次 River／Google Messages 和少量 OpenAI HTTP。聚合不包含真实凭据、账号身份或请求正文。

## 服务与资源边界核对

- `internal/core/proxy/server.go`：HTTP 在读取正文前获取名额，默认并发 16；WebSocket 另有 16 个连接名额。
- `internal/core/proxy/types.go`：普通 Chat 正文 2 MiB，Responses／Messages 16 MiB，流式总量上限 32 MiB。正文上限不是总内存预算，多轮 JSON 解码、协议转换和并发仍可能产生显著峰值。
- `internal/core/proxy/websocket.go`：连接保留上一轮完整输入／输出，用于 `previous_response_id`；重建后的请求检查 16 MiB。它是进一步采样的优先路径，但没有证据证明当前两块 1 GiB 区域就是会话历史。
- `internal/provider/antigravity/tool_signatures.go`：签名缓存最多 256 条，每条最多 64 KiB、1 小时有效；值的理论上限约 16 MiB，另外有索引等开销，不能单独解释约 2 GiB。
- `internal/infrastructure/logger`：日志环形缓存最多 5,000 条，分批最多读取 200 条，文本字段有长度边界。当前运行日志为数百条；没有发现启动时把整个磁盘日志装进内存的实现。
- `internal/infrastructure/database/database.go`：SQLite 单连接；请求历史存磁盘而非常驻完整请求正文。
- 更新检查、Google 模型同步、额度自动激活由后台生命周期管理；额度激活是单循环、合并唤醒，不是每个界面组件单独启动一个服务。没有证据支持“每个提供商启动一个高内存常驻进程”。

## 隔离实验

临时探针放在 Git 忽略的 `tmp/memory-investigation/main.go`，只使用随机临时数据目录、假数据和独立 WebView 用户目录：

| 场景 | 工作集 MiB | 私有提交 MiB | Go HeapAlloc MiB |
| --- | ---: | ---: | ---: |
| 仅加载相同后端依赖 | 16.41 | 48.86 | 1.01 |
| 新建隔离数据库并执行全部迁移 | 20.20 | 51.09 | 1.25 |
| 10,000 条每条 2,048 字符假数据，20 次聚合 | 27.91 | 59.37 | 0.84 |
| 关闭数据库并 GC／FreeOSMemory（仅探针进程） | 24.43 | 55.84 | 0.79 |
| 独立空白隐藏 Wails 窗口，3／6／9 秒采样 | 64.72 | 83.59 | 1.28 |

这些实验排除了“空白桌面壳或空数据库初始化必然占 2 GiB”的解释，但不覆盖真实长会话、实际页面运行、全部后台功能或生产进程已加载的原生组件，不能直接据此给生产服务分配 MB。

## 未修改的内容

- 没有修改业务协议、用户配置、前端或生产可执行文件，没有停止／重启生产网关；源码修改仅限基础设施凭据读取与相关测试。
- 没有读取生产内存正文、生成进程 dump 或读取真实凭据。
- 工作区已有的 CURRENT、Gemini 排查任务及 Atria 任务改动均保留，未纳入本任务。

## 实际修复

- `credential.MemorySecret` 复用既有 Windows DPAPI，仅在内存保存保护后的密文；返回独立明文副本，消费后沿用原有清零，清理与并发读取安全。
- `codex.encryptedAuthCache` 最多 16 条／固定 5 分钟有效期；每次重新核对完整密文和新读系统 passphrase 的摘要，源内容变化立即使旧条目失效，不依赖大小／mtime。过期惰性清理，命中不续期。
- Windows 保护失败或缓存损坏返回稳定错误；不降级成明文缓存，不新增授权磁盘副本。注销删除官方凭据时清除该隔离 home 的缓存。
- 所有实际解密受同一缓存锁串行保护；官方工作因子保持 20。仅在工作因子 18–20 的解密尝试完成后归还临时工作区，不在热路径或定时器中盲目 GC。
- 官方 `account/read` 刷新、账号启用／授权状态／版本和身份检查完全保留；锁不延伸到网络响应消费。

## 修复后的官方因子专项

`BenchmarkEncryptedAuthOfficialFactor20` 使用随机临时目录与完全虚构授权，`-benchtime=1x`：

| 项目 | cold | cached |
| --- | ---: | ---: |
| 单次耗时 | 2.262 秒 | 7.447 毫秒 |
| Go 分配量 | 1,073,921,912 B | 10,808 B |
| 读取完成后 Go 活跃堆 | 0.584 MiB | 0.599 MiB |
| 读取完成后进程私有提交 | 50.18 MiB | 50.37 MiB |

冷读取仍分配官方算法必需的约 1 GiB，但完成后已归还；缓存命中不重复 KDF。上述数字属于独立测试进程，不是完整桌面应用的运行内存保证；官方外部组件自身的瞬时开销不在本修改控制范围。

## 剩余边界

本轮未对生产内存生成 dump 或安装诊断端点，也未把其他原生组件认定为根因。正式运行版仍需用户用包含修复的新构建重新启动后验收；没有读取真实授权或重放真实会话。冷启动、官方授权变化、到期或容量淘汰仍可能短暂出现约 1 GiB 峰值。

# Verification

## Verification Result

- 生产进程树／内存区域元数据／本机 healthz：完成只读核对。
- SQLite：通过 `mode=ro` 连接，仅查询请求数量、状态、协议、提供商 ID 和保留上限，没有执行写语句。
- `go run ./tmp/memory-investigation`：通过，使用假数据，结果见上表。
- `go run ./tmp/memory-investigation shell`：通过，隐藏独立窗口自行退出；没有打开生产 WebView 数据目录。
- `go test ./internal/infrastructure/logger ./internal/core/proxy -run 'TestRequestedLogCapacity5000|TestSocketLongHistoryReconnectAfterPartialStream|TestResponsesLongHistoryHTTPAndCompaction|TestMessageProtocolsPreserveLongHistoryWithinByteBounds|TestConversationCountUsesBoundedProtocolScale' -count=1`：通过。
- `go test ./internal/infrastructure/credential ./internal/infrastructure/codex ./internal/application/subscription ./internal/application/gateway -count=1`：通过。
- 新增回归覆盖受保护内存副本／清零／并发／损坏，缓存命中、完整源内容和系统 passphrase 变化、固定到期／容量、跨 home 并发串行、保护失败／非法大小，以及文件删除／注销和官方工作因子上限。
- `go test ./internal/infrastructure/codex -run '^$' -bench '^BenchmarkEncryptedAuthOfficialFactor20$' -benchtime=1x -count=1`：通过，修复后实测见上表。
- `task --force verify BIN_DIR=tmp/codex-memory-verify-20261021`：完整通过格式、Go vet／staticcheck／ESLint、类型检查、Go 测试、118 文件／703 项前端测试、Wails 绑定生成和 Windows production 构建。
- 最终构建：`tmp/codex-memory-verify-20261021/omniproxy.exe`，22,297,088 bytes，SHA256 `b0fb24c77e0f894014bad0da51b056594f99dc7726b723074dc6f1709bbb3443`。未启动、安装或替换生产程序。
- `git diff --check`：通过。绑定、图标、依赖文件没有额外差异。构建保留原生产进程 PID 14716／`E:\codexproxy\OmniProxy\OmniProxy.exe`。
- 已知非阻断提示：既有 jsdom `scrollTo` 未实现、Vite 大 chunk 与 Windows 环境缺少 `uname`／`tail` 的探测提示；完整质量门退出码 0。

# Deviations

首次 `task verify` 的格式、静态检查、类型检查、Go 测试和 703 项前端测试通过，但构建前的增量扫描长时间未推进，只有本任务 Task 进程、没有构建子进程。核对精确 PID、可执行路径、父命令和本次 verify 参数后，仅停止该校验进程，使用 `--force` 重跑同一完整质量门并通过。该参数只绕过增量缓存判断，不跳过任何检查或测试；输出到独立临时目录。

# Documentation

更新架构概览、技术栈中的进程内／持久凭据边界、测试专项说明；新增 ADR-020。CURRENT 的其他任务已有改动保持不变。

# Git

Not committed by Agent.

# Result

已通过假数据精确复现官方 scrypt 单次 1 GiB 与重复解密后约 2 GiB 的保留量，并实现受系统保护的有界缓存和真实高成本工作区回收。针对性回归、官方因子专项和完整质量门通过；冷读取完成后独立测试进程私有提交约 50 MiB，缓存读取分配约 10.6 KiB。不降低安全强度，不修改已安装应用或生产数据，不宣称当前旧进程已生效。
