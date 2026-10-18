# TASK-20261018-018: 修复真实订阅转发的流式响应失败

## Status

Completed

## Created

2026-10-18 22:43

## Request

用户要求修复当前 Beta.3 的网关转发失败；在隔离环境处理，避免更改共享 Codex 配置和影响当前 Agent。

## Goal

定位真实 200 / invalid_upstream_response 的原因，最小修复并验证，不把模拟成功等同于生产成功。

## Scope

- 上游订阅响应、HTTP / WebSocket 流处理及安全诊断。
- 按需进行有界真实诊断；仅使用应用隔离账号，不读取默认 Codex 登录，不记录正文、凭据或身份。
- 回归测试、task verify、实际验收说明。

## Out of Scope

- 不修改真实 Codex 配置、不重启生产软件或当前 Agent。
- 不覆盖已发布 Beta.3 资产或标签，不自动推送或发布新版本。

# Analysis

TASK-017 已确认：网关监听和客户端接入正常，真实 HTTP / WebSocket 流式请求上游状态 200，随后响应前置校验返回 502。现有落盘缺少媒体类型和失败分支，需避免猜测为特定错误页。

## Risks

真实诊断需保护隔离账号凭据和提示词；只输出有界枚举元数据。修复必须保留对恶意／非预期响应的拒绝，避免任意 200 被视为成功。

# Implementation Plan

1. 复查订阅与传输实现，安全确认实际返回协议。
2. 隔离复现并为根因增加失败回归；必要时增加无敏感数据的诊断。
3. 最小修复，执行定向与完整质量门，明确真实验收边界。
4. 更新任务记录与相关文档，审查 diff 并本地提交。

# Progress

- 已使用固定官方端点与应用隔离账号完成一次有界小请求：上游 HTTP 200、Content-Type 缺失、无 Content-Encoding，前 4096 字节识别为 SSE，包含 response.created；未输出正文或凭据，未刷新登录或改写生产库。
- 根因已确认：合法 Responses SSE 缺少媒体类型头，被两种入口前置校验拒绝。兼容仅放在固定订阅提供商边界，200 + 缺失类型时有界验证首事件后补全，不放松显式类型或通用 Core 校验。
- 增加枚举化媒体类型与是否推断的诊断，不落盘原始头部或正文。

# Implementation Result

## Changes

- 固定订阅适配器对 HTTP 200 且缺失 Content-Type 的响应，最多预读 512 KiB，并验证首个完整 Responses SSE 事件。通过后仅补全媒体类型；显式其他类型、HTML／JSON／空／坏数据仍拒绝。
- 前缀保存覆盖 bufio 预读，逐字节回放给原统计及转发链；不等待整个上游结束，不缓存整段响应，不重复发送真实模型请求或跨账号重放。
- 核心类型校验不变；增加枚举 upstream_media_type 与 upstream_content_type_inferred 布尔日志，禁止落盘原始头值与正文。
- 增加提供商边界、分片／首事件完整性／前缀上限、两种入口及错误不泄露／不重放、日志分类测试；更新客户端接入文档。

## Real Validation

真实诊断与验收使用只读生产库获取指定应用隔离账号的定位信息；凭据仅通过现有 codex.UseToken 回调使用，没有调用登录刷新、读取默认 Codex 登录或改写生产数据。实际服务运行在独立临时 SQLite、随机 loopback 端口，账号展示身份为虚构 QA 数据。

- 修复前：一次有界小请求确认上游状态 200、Content-Type 缺失、无内容编码、正文为 Responses SSE；只输出固定元数据。
- 修复后：gpt-5.6-luna 的 HTTP 和 WebSocket 各一次；实际安装的 CLI 0.163.0 与 Desktop 内核 0.163.0-alpha.10.2 各发一次 WebSocket，均 exit=0 / turn.completed。
- 另验证 gpt-6.1-sol 的 HTTP 与 WebSocket 各一次。共六次修复后真实请求，全部 success / 200、usage_complete=1、输入／输出已知且首字时间存在。
- 隔离客户端无真实登录、使用只针对测试监听有效的假接入 Key、临时 CODEX_HOME／HOME／USERPROFILE、忽略用户配置与规则、只读工作目录，重试设置为零。两种内核都未打开 Desktop GUI；不宣称外壳界面已验收。
- 首次隔离网关夹具省略虚构订阅邮箱和 workspace，账号统计身份校验在调用上游前返回 503；修正夹具身份后重建独立临时库，再完成上述六次测试。未将该夹具问题作为产品修复。
- 当前共享 Codex 配置 SHA256 前后相同，生产 PID 42784／Beta.3 保持运行；没有替换、安装、推送或发布。

## Remaining Scope

本地构建包含修复，但当前运行的官方 Beta.3 不会自动载入。需后续部署／重启新构建才会让用户现有生产入口生效；已发布 Beta.3 标签和资产保持不动。

# Verification

- PASS：先添加回归，旧实现实际失败（类型头为空）；修复后通过。
- PASS：提供商与 Core 全包十次重复；订阅网关／预热拒绝与冷却／200 后不重放相关回归十次重复。
- PASS：新增最终分片与嵌套错误覆盖后，提供商／Core／网关全包重跑。
- PASS：真实上游六次请求与两个实际 Codex 内核，均完整结束且统计有效；用户配置哈希未变化。
- PASS：首次 task --force verify，全 Go、前端 104 文件／575 测试、格式、静态检查、类型检查及 Windows 生产构建。
- PASS：最终 task --force verify（含最后补充的分片／嵌套错误测试）：格式、go vet／staticcheck／ESLint、类型、全部 Go 测试、104 文件／575 前端测试、Windows 生产构建。既有分包体积和 jsdom scrollTo 提示不影响通过。


# Git

ad105c12df49820b69ed72eb018296508271f50b fix(gateway): 兼容订阅响应缺失 SSE 类型头

仅本地提交，未推送／发布。


# Documentation

已更新 docs/product/LOCAL_AGENTS.md 的订阅 SSE 兼容边界及真实验收说明；CURRENT 更新本地修复状态。无数据库迁移、新依赖或公共错误码变更，不需要新增 ADR。

# Result

已确认并修复固定官方订阅端点合法 SSE 缺失 Content-Type 导致的误拒绝。六次真实请求及双 Codex 内核验收通过，完整质量门通过；运行中的生产进程保持不动，部署仍待后续执行。

# Completed

2026-10-18 22:57
