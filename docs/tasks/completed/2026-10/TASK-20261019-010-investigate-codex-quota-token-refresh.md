# TASK-20261019-010: 排查 Codex 额度查询失败与令牌刷新机制

## Status

Completed

## Created

2026-10-19 17:22（Asia/Singapore）

## Request

额度查询经常出现“额度查询失败，请稍后刷新。”是什么原因？token 自动刷新续期和手动刷新是否已经实现？

## Goal

依据现有实现、测试和脱敏运行证据定位额度失败的具体阶段，核对自动令牌续期与手动刷新入口的真实行为；若确认本任务范围内缺陷，最小修复并验证。不将额度失败直接等同于登录过期，不臆测官方组件行为。

## Analysis

已核对生产／开发均为 2.0.0-beta.7。本次通过只读数据库、已有应用服务与隔离官方组件查询两个生产订阅账号，均成功，耗时 5876／5532 ms，其中子进程初始化约 3.5～3.8 秒；开发环境没有 authenticated 的 OpenAI 订阅账号可查询。未复现历史失败，不据此认定唯一根因。

代码确认：网关 UseTokenAt 与手动 Refresh 均调用 account/read(refreshToken=true)，但 ReadQuota 直接读取额度后才 Account(false) 核验邮箱；没有独立后台续期任务。10 秒预算同时覆盖子进程初始化和额度读取，所有读取错误统一变成 quota_unavailable，且没有额度失败阶段日志。本机 codex-cli 0.163.0 导出协议确认 refreshToken=true 的 managed auth 语义；usedPercent 为 int32，与现有模型一致，不修改正确的整数契约。

方案比较：不增加定时续期、无限重试或大依赖；沿用官方组件，在额度读取前主动刷新并核验成员身份，将读取预算对齐现有 30 秒单次操作上限，保留不修改账号状态及旧快照的原则。仅输出阶段、稳定错误码和耗时，不输出身份、授权定位、令牌或原始响应。诊断读取没有主动调用 Refresh／Account(true)、没有发送模型请求或更改账号元数据。

## Implementation Plan

1. 读取相关规范和订阅架构／产品说明，创建记录后再修改项目文件。
2. 沿前端额度查询和手动刷新入口追踪到服务与官方组件，查阅已有回归测试。
3. ReadQuota 前调用 Account(true) 并核验成员；维持账号归属校验与错误不注销的行为，采用既有 30 秒操作预算。
4. 增加超时／授权刷新失败文案与脱敏阶段日志，覆盖刷新顺序、身份隔离、超时／取消、失败不破坏授权和日志不泄漏的回归。
5. 执行定向测试和 task verify（输出到独立 BIN_DIR，避免覆盖运行中的开发程序），更新订阅说明及任务看板；历史失败未复现与生产版本尚未部署必须如实说明。

## Implementation Result

- ReadQuota 在同一官方组件中先 Account(true) 续期并核验成员邮箱，再读取 RateLimits 并核对工作区 ID；仍使用已有操作锁，不引入定时任务、自动重试、额外依赖或账号元数据写入。
- 读取预算由 10 秒调整为既有 operationTimeout（30 秒），继承调用方更短的总预算；排队后上下文已失效时不再打开官方组件。
- 新增 quota_timeout／quota_refresh_failed 的中英文提示，保留额度接口失败、明确缺失授权和身份变化各自的安全错误。额度失败不注销账号、不覆盖最后成功快照，也不将网络错误判定为 Token 永久失效。
- 新增额度失败诊断：prepare／queue／account／open／refresh／rate_limits／identity／normalize 固定阶段，以及 code／cause_code。日志不包含账号 ID、邮箱、配置定位、授权令牌、原始错误或官方响应。
- 增加刷新顺序与 30 秒预算、身份隔离、缺失凭据、协议／请求失败、规范化失败、超时／取消、失败不改变持久授权和日志／界面不泄漏的回归。订阅产品说明同步按需续期与两个手动刷新入口的区别。
- 原生产 PID 30124 保持运行，本次没有安装、发布或替换生产程序；开发 watcher 自行重载为 PID 1392，不将此当作真实账号额度／过期续期验收。验证构建输出位于忽略的 tmp/quota-token-20261019/verify-bin，不提交临时诊断工具、构建或协议文件。

## Verification

- PASS：本次修改前，生产两个 authenticated 账号分别通过原 ReadQuota 查询，5876／5532 ms；仅匿名别名、阶段、结果码及耗时输出。未复现原失败，也没有主动强刷真实账号授权或发送模型请求。
- PASS：本机 codex-cli 0.163.0 协议导出；全新隔离 CODEX_HOME 的 Account(true) 调用被接受且账号为空，未继承现有登录。这只验证参数／隔离契约，不声称完成真实过期令牌续期。
- PASS：go test ./internal/application/subscription ./internal/infrastructure/codex ./internal/transport/wails。
- PASS：四个定向前端文件（subscription-quota、subscription-quota-refresh、subscription-login、quota-view），28 项测试。
- PASS：task verify BIN_DIR=tmp/quota-token-20261019/verify-bin；format check → Go／前端 lint → typecheck → 全部 Go 测试 → 107 个文件／610 项前端测试 → Wails 绑定生成／前端生产构建／Windows amd64 构建，exit 0。输出 EXE 为 22,024,192 bytes。
- PASS：git diff --check；修改代码 UTF-8 无 BOM，无依赖或生成绑定变更。现有 jsdom scrollTo 未实现和 Vite chunk 体积警告不影响通过。

未执行：在生产进程中部署修改后的版本、真实令牌到期／撤销场景或原故障复现。后续若仍失败，需依据新阶段日志继续区分官方网络／接口／凭据原因，不能将本轮防护和模拟回归等同于历史问题已全部修复。

## Git

2616efee603681ee19351731cd8034db8c4a7f79 fix(subscription): 补齐额度查询续期与失败阶段诊断

仅本地提交，未推送。

## Result

已确认自动（按需）和手动 Token 刷新原本存在；额度链路显式续期缺口、较紧预算及不可诊断的通用失败已补齐，并通过完整质量门。历史失败的唯一原因仍未确定，生产安装与复现验收留待用户安排。
