# TASK-20261027-003: 修复 HTTP 透明转发 Token 用量漏记

## Status

Completed

## Created

2026-10-27（Asia/Singapore）

## Request

先修复生产日志排查中确认的 Token 用量记录错误。

## Goal

HTTP 透明转发在消费上游 usage 后再结束尝试，使请求、尝试及累计用量正确持久化，同时保留透明转发、取消、中断与换号行为。

## Background

根因与生产只读证据见 TASK-20261027-002；该任务只完成排查，尚未修改正式业务代码。

## Scope

- 修复 HTTP 透明转发尝试级记账生命周期。
- 补充真实 activity／SQLite 的端到端回归与错误路径覆盖。
- 执行定向测试和 `task verify`，同步任务记录及 CURRENT。

## Out of Scope

- 不回填历史用量，不修改生产数据、凭据或客户端配置。
- 不自动安装、重启或发布生产程序。
- 不修改无关协议、依赖、技术栈或数据库结构。

## Analysis

- 选择调整 HTTP 透明转发生命周期，不放宽 activity 的 `pending`／账号归属校验，避免使已终止或被拒绝尝试重新接受用量。
- 成功或不可换号的响应消费完成后才结束尝试，并使用实际转发的取消／中断分类；父记录绑定失败时也要收尾尝试。记账错误必须可见，但不能重放上游或追加错误正文破坏已交付的透明响应。
- 现有透明流测试多为内存观察者，换号集成测试的成功响应未携带 usage；新增真实 SQLite 回归，断言请求／尝试／累计和汇总，并覆盖缓存、零用量与流式终结事件。
- 前一排查任务文档已存在未提交修改，属于既有工作；不覆盖其历史结论，也不把临时 overlay 当成正式实现。

## Implementation Plan

1. 检查转发、尝试收尾、用量事务及现有测试，确定最小修复边界。
2. 先以真实数据库测试复现 SSE／JSON 漏记，再修复收尾顺序并覆盖取消、中断、换号及记账错误。
3. 执行定向测试与完整质量门，审查差异、同步文档和实际交付状态。

## Implementation Result

### Root Cause

透明 HTTP 转发在读取响应正文之前调用 `EndAttempt`，而 activity 用量事务要求目标尝试仍为 `pending`，导致上游 usage 已解析但持久化整体回滚。

### Changes

- 保留严格的数据库状态／账号校验，改为 `Update → 消费响应与上游 usage → EndAttempt`。
- 尝试按实际成功／取消／上游流中断收尾，父记录绑定失败时也结束尝试；保留真实上游 HTTP 状态，不把网关记账错误伪装成上游状态。
- 尝试收尾出错通过受控 `accounting_unavailable` 和诊断标记显式报告，不改写已交付的响应、不重放请求，不覆盖已确定的取消／流中断原因。
- 统一已声明和推断内容类型的透明响应观察策略：用量入库失败只标记诊断，不丢弃响应字节；普通协议转换仍使用原严格模式。
- 三个新测试文件分别覆盖真实 SQLite 用量／汇总／换号、流式完成与中断、记账失败的透明性与可见性。

### Files Changed

- `internal/core/proxy/passthrough.go`
- `internal/core/proxy/passthrough_observe.go`
- `internal/core/proxy/usage_reader.go`
- `internal/core/proxy/passthrough_accounting_failure_test.go`
- `internal/application/gateway/passthrough_usage_test.go`
- `internal/application/gateway/passthrough_usage_stream_test.go`
- `docs/product/GATEWAY.md`
- `docs/plans/CURRENT.md`（仅本任务状态；保留前一排查任务既有修改）

### Regression Prevention

- 未声明类型 SSE、声明 SSE、JSON 的非零／真实零用量，断言缓存、请求、账号尝试、累计、账号及提供商模型汇总。
- 明确拒绝换号后只把用量记到被接受账号，一次逻辑请求只累计一次。
- 终结事件后上游异常断开仍保存成功与用量；终结前上游中断／客户端取消同步结束请求与尝试，无报告时保留未知。
- 记账观察失败保持透明字节；父账号绑定失败、尝试收尾失败、错误分类保留及无重放。

## Verification

- 修改前新增的六个用量形态子用例及换号回归均失败：成功请求及尝试仍为 unknown，复现根因；测试编写阶段发现并修正了 DTO 的 `Summary` 字段访问，不将编译错误算作缺陷复现。
- 修改后六项新增测试函数的所有用例通过，定向重复 10 次通过：proxy 0.203s，gateway 21.412s。
- 最终修改后的 gofmt 检查、`go vet ./internal/core/proxy ./internal/application/gateway`、`staticcheck ./internal/core/proxy ./internal/application/gateway` 通过。
- 原工作区 `task verify` 已通过格式、lint、typecheck、全部 Go 测试以及 132 文件／809 项前端测试；构建阶段的宽泛 sources 扫描长时间停留在检查旧临时副本，未进入绑定生成／正式编译。只停止了本任务的 Task 进程，未改 Taskfile 或删除已有临时目录。
- 工作区外的全新临时目录执行未修改的 `task verify` **PASS，退出码 0**：格式、lint、typecheck、全部 Go 测试、132 文件／809 项前端测试以及 Windows production 绑定生成、前端构建和桌面编译全部成功。复制全部受版本控制文件及本任务新增源码／测试，不携带 `.git`、旧临时副本、凭据、真实 `.env` 或生产数据；仓库公开的 `.env.example` 正常保留。
- 已逐字节核对六个变更源码／测试以及 Go、Task、前端依赖配置与验证副本一致。
- 验证产物位于临时目录 `omniproxy-verify-20261027-003-2l3nwiru/bin/omniproxy.exe`，23,115,264 bytes，SHA256 `e4850a4578170c5fa718973b19defee4ea98c674942f1464c0552fde9a9b1066`；`go version -m` 确认 production 编译标签。未替换安装目录或正在运行的应用。
- 非阻断输出包括既有 Windows `uname`／`tail` 探测提示、jsdom `scrollTo` 未实现提示及前端大块警告；没有修改无关工具或测试以掩盖它们。
- `git diff --check` 和 UTF-8 无 BOM 检查通过。

## Documentation

- 更新网关文档，说明尝试完成边界、记账失败诊断与透明响应保证。
- 无产品范围、公共 API、技术栈、数据库迁移或架构边界变化，不新增 ADR。

## Deviations

定向故障测试补充验证了已声明类型的透明响应也必须使用非严格记账观察模式。该路径此前误用普通协议转换的严格模式，可能因入库错误截断原始字节；一并做最小修复，保留转换路径原行为。

## Git

```text
04e053c43ce8007fe7f28b987ed31d0762420133 fix(gateway): 修复透明转发 Token 用量漏记
```

仅提交本次业务修复、回归测试及网关说明；前一排查任务的未提交记录及 CURRENT 原有排查段落保留，不纳入本次提交。未推送或发布。

## Result

HTTP 透明转发提前完成账号尝试的根因已修复；真实 SQLite 回归、故障路径、重复验证及干净副本的完整质量门均通过。
历史用量未回填，已安装／正在运行的生产程序未替换；修复需随新构建安装后才会在生产生效，不能把源码验证通过等同于实机部署完成。

## Completed

2026-10-27（Asia/Singapore）
