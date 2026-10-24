# TASK-20261024-019: 修复 Codex WebSocket 握手超时与凭据准备

## Status

Completed

## Created

2026-10-24（Asia/Singapore）

## Request

用户确认实施 [Codex WebSocket 修复计划](../../plans/PLAN-20261024-codex-websocket-handshake.md)：“你做吧”。

## Goal

保持客户端原生 WebSocket 与网关接收开关，修复握手准备超时、取消误分类及不必要的授权准备开销，补齐安全诊断与回归验证。

## Background

现场客户端约每 15 秒握手超时，重试到 5/5 后回退 HTTP；开关实际开启。
已确认透明入口先准备凭据和上游再确认握手、订阅准备共享串行锁、取消会被折叠为凭据失败，以及隔离授权续期未传播请求上下文。
具体慢阶段仍需诊断证据，不预先宣称唯一根因。

## Scope

- 透明 WebSocket 握手预算、生命周期、策略复核与安全诊断。
- 取号取消语义、凭据准备快路径与续期取消传播。
- 在现有身份保护边界内优化并发，清理已移除的客户端传输标记。
- 回归测试、相关产品／决策／计划文档、完整质量门。

## Out of Scope

- 修改真实客户端配置、账号授权或运行中数据库。
- 自动安装、重启现有应用、公开发布或替用户发送真实模型请求。
- WebSocket 转 HTTP/SSE、逐轮计费、大型连接池或无边界后台预热。
- 覆盖／提交其他任务既有未提交改动。

# Analysis

- 保留上游先握手的透明中继，不提前返回伪成功 101；独立握手上下文不得约束中继寿命。
- 订阅健康路径改为读取受保护授权并核对成员／工作区，无需每次启动官方组件强刷；不能只核对同工作区。
- 授权相关操作采用可取消的账号级门，登录提交仍保留全局排他；避免仅替换 token 锁而绕过注销／替换互斥。
- 保留官方 KDF 的串行内存上限，明确本地不可中断步骤不具备严格实时取消保证。
- 旧客户端标记只停止新增；已有受管配置清理须满足修订与归属保护，不操作真实用户配置。

# Implementation Plan

1. 记录工作区基线，检查握手、凭据读取、身份校验与已有测试。
2. 优先补取消语义、握手总预算与阶段日志；回归慢准备和长连接。
3. 消除健康请求不必要的官方强刷，安全复用现有授权读取，审计互斥和刷新回写。
4. 清理受管配置旧标记，同步恢复／兼容测试和文档。
5. 运行相关完整包与 `task verify`，记录实际结果、未验证事项与剩余风险。

# Implementation Result

## 握手与协议边界

- 透明入口保留真实上游先握手；总预算默认 10 秒，全部凭据准备累计最多 5 秒，单候选最多 3 秒，单次拨号最多 5 秒且服从剩余总期限。
- 建连结束即释放握手计时器，中继使用原请求上下文；没有给长会话新增 10 秒寿命。
- 在上游准备完成、确认客户端握手前及每个后续客户端帧重新核对开关与原授权方式；不将官方长令牌交给接入 Key 比较，不允许撤销 Key 后降级为前缀授权或改变客户端归属。
- 请求取消停止候选遍历，归还健康租约／并发槽，不污染凭据失效状态，也不向已离开的客户端补写伪交付结果。
- 增加 `credential_prepare_timeout`／`upstream_socket_timeout`，内部准备期限与父请求取消分别处理；前端复用现有超时文案。
- 阶段日志仅记录连接关联 ID、固定阶段、尝试序号、耗时、稳定结果与可证实的关闭码／方向，不输出认证材料、URL、原始错误或关闭原因。

## 凭据与并发

- 隔离授权的健康路径直接核对受保护材料中的成员邮箱、工作区与有效期，准备前后校验账号版本／profile／启用状态，不再无条件启动官方组件强制联网。
- 仅临近过期时调用官方组件续期，由官方存储持久化旋转结果，再重读核验；移除只缓存访问令牌却不保存新 refresh token 的影子续期路径。
- 新增小型 `core/syncgate`：账号级可取消互斥、登录提交全局排他、优先等待中的全局提交及释放后清理账号键；订阅读取／刷新／兑换／注销／删除和网关自管凭据复用。
- 自管授权刷新按账号串行；回写校验成员、版本、原内容摘要及原引用，SQL 比较交换阻止旧结果覆盖重新导入、停用或已刷新授权。
- OAuth 请求遵循调用上下文；普通 HTTP 的授权准备另有 30 秒边界，不把准备期限带到模型网络流。
- 保留原 DPAPI 快照、容量／期限／指纹失效及 KDF 内存上限；等待昂贵 KDF 可以取消，实际已开始的同步计算完成后复核取消并清理结果。

## 客户端配置

- 停止新增已移除的旧 WebSocket feature 标记；明确指向本机 Codex 受管入口的自定义提供商声明 WS 支持。
- 受管刷新在 applied hash 与备份归属匹配时清理旧字段，保留原用户 false、恢复被旧版本覆盖的用户 true；不修改外部编辑文件，恢复仍逐字还原原配置。
- 未操作真实客户端配置、凭据、运行数据库或正在运行的应用。

## 文档与主要文件

- 核心：`internal/core/proxy/{passthrough_socket,socket_diagnostic,passthrough,handler,types}.go`、`internal/core/syncgate/`。
- 应用：网关取号／借凭据、订阅 token／互斥调用者、账号自管授权 CAS、Agent 配置写入／受管刷新及相邻回归测试。
- 基础设施：`internal/infrastructure/codex` 的有上下文授权读取、必要续期信号、KDF 排队取消及测试。
- 前端：`gateway-errors.ts` 与新增四项分类测试，无新 UI 文案或绑定契约。
- 同步 GATEWAY／LOCAL_AGENTS／SUBSCRIPTION、架构概览、ADR-020／023／024、修复计划及看板。

# Verification

## 已执行的定向验证

- 握手总预算、慢候选换号、长连接跨握手期限、多轮文本／二进制帧、准备中关闭开关、原授权撤销／归属变化、正常关闭诊断及敏感原因不输出。
- 取消不继续取号、内部准备超时不标为凭据损坏、健康读取不启动组件、必要时仅刷新一次、读取后版本变化拒绝发布、准备计时器不约束模型消费。
- 同工作区不同成员拒绝、OAuth 网络取消、KDF 等候取消、账号互斥与全局排他、CAS 防止重新导入／异成员／旧刷新／停用覆盖。
- 受管旧标记、用户 false／true、恢复原配置、自定义入口 WS 声明；前端错误分类 4 项通过。

## 相关包回归（`-count=1`，不用缓存）

```text
go test ./internal/core/proxy ./internal/core/syncgate ./internal/application/gateway \
  ./internal/application/subscription ./internal/application/agentconfig \
  ./internal/application/account ./internal/application/quotaactivation \
  ./internal/infrastructure/codex -count=1
PASS proxy 2.909s｜syncgate 0.254s｜gateway 53.521s｜subscription 24.422s
     agentconfig 52.062s｜account 20.732s｜quotaactivation 18.675s｜codex 0.667s
```

## 隔离真实 CLI 契约（现场同版本 0.163.0）

设置 `OMNIPROXY_CODEX_TEST_EXECUTABLE=C:\Users\…\OpenAI\Codex\bin\codex.exe` 后运行 `TestCodexCLIIsolatedWebSocket`：**PASS（0.66s）**。真实客户端二进制经 `…/codex/v1` 按自定义提供商的 `supports_websockets=true` 建立 WebSocket，命中 1 次请求，记账 `Transport=websocket`／状态 `success`，预热未被计入模型请求。仅使用 loopback 假上游与临时 `CODEX_HOME`，未接触真实账号、真实配置或公网。

## 热路径握手延迟（临时探针，测后已删除）

本机 loopback 假上游、单账号热路径，连续 20 次「握手 + 一帧往返」为一轮，共四轮：

```text
20/20  p50=587µs  p95=1.255ms  max=1.584ms
20/20  p50=955µs  p95=1.702ms  max=2.022ms
20/20  p50=531µs  p95=1.577ms  max=1.612ms
20/20  p50=895µs  p95=1.594ms  max=2.323ms
```

四轮均 20/20 成功，P95 约 1.3–1.7 ms，远低于计划建议的 2 s 目标。探针文件未进入仓库；该数值只代表网关传输层开销，不含真实授权解密、官方组件启动和公网上游。

## 假授权冷／热专项

```text
go test ./internal/infrastructure/codex -run '^$' -bench '^BenchmarkEncryptedAuthOfficialFactor20$' -benchtime=1x -count=1 -timeout=90s
PASS
cold:   1,910,644,800 ns/op，约 1.91 秒
cached: 5,248,300 ns/op，约 5.25 毫秒
完成后的 Go 活跃堆约 0.58／0.60 MiB，进程私有提交约 50 MiB
```

仅使用临时目录假授权，未读取真实凭据；冷解密约 1 GiB 分配是算法必需峰值，不等同于整个桌面程序内存。

## 完整质量门

```text
task verify
EXIT=0
format:check → lint（gofmt / go vet / staticcheck / eslint）→ typecheck → test → build 全部通过
go test . ./internal/...  全部 ok（proxy 3.115s、gateway 42.378s、subscription 18.504s、wails 16.686s，其余 cached）
frontend：127 文件 / 757 项测试通过，95.94s
windows production build：go build -tags production 生成 bin/omniproxy.exe 成功
```

输出写入本机临时日志 `%TEMP%\omniproxy-verify-019.log`，未覆盖生产安装或项目既有安装包；质量门内 `wails3 generate bindings` 重新生成后 `frontend/bindings` 无内容差异（工作区变更数前后同为 167 项）。

## 尚未执行

- 用户真实账号的冷／热握手、开关切换及 CLI／Desktop 端到端实机验收：任务 Out of Scope 明确不操作真实配置、账号授权与运行中数据库，需用户自行执行。
- 安装、替换或重启正在运行的应用；正式 NSIS 安装包与签名发布。
- 阶段日志在用户真实环境中的慢点归因；历史日志缺少阶段耗时，不能追溯证明唯一慢步骤。

# Deviations

- 单候选 2 秒是计划初建议；根据假授权 factor-20 冷读约 1.91 秒调整为 3 秒，仍保留 5／10 秒上限。
- 隔离登录的必要刷新由官方持久化，而非自行旋转后只保存访问令牌；否则进程重启或缓存到期会丢失新 refresh token。自管账号仍自行刷新并 CAS 回写。
- 并发成功刷新通过账号门与重读最新存储合并，不引入脱离请求生命周期的后台 singleflight 或失败缓存。
- 本地 KDF 本身不可实时中断；只保证排队可取消和计算结束后的状态检查，不用后台 goroutine 掩盖这一限制。

# Git

Not committed by Agent. 工作区包含多个既有未提交任务；不得夹带提交。

# Result

代码修复、文档同步与自动验证已完成：`task verify` exit 0（127 文件／757 项前端测试、Go 全部包 ok、Windows production 构建成功），八个相关包禁用缓存回归通过，现场同版本 Codex CLI 0.163.0 的隔离契约以 WebSocket 通过，热路径 20 连握手 P95 约 1.3–1.7 ms。真实账号的 CLI／Desktop 实机验收属于本任务 Out of Scope，仍待用户执行；因此不宣称现场约 15 秒握手超时已在用户机器上消失。工作区其他任务的未提交改动未被夹带，本任务未提交。

---

# Completed

2026-10-24 21:39
