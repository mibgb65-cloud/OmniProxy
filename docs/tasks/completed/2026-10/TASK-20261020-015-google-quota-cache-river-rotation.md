# TASK-20261020-015: Google 额度快照持久化与 River 调度设置

## Status

Completed

## Created

2026-10-20

## Request

- Google 订阅额度读取结果需要像 OpenAI 一样保存，切换页面、重启及读取失败时保留显示。
- River 账号页增加设置按钮，可以选择队列使用或均衡使用。
- 某个 River API Key 持续失败时，能够切换到另一个可用 API Key。

## Goal

复用现有持久快照和调度机制，在保持账号身份隔离及不重复生成安全边界的前提下完成两项能力。

## Scope

Google 公开额度显示快照、River 使用方式持久化／设置入口／失败切换，以及必要迁移、测试和产品文档。

## Out of Scope

- 不读取或输出真实凭证、请求正文，不修改生产账号和真实用量。
- 不修改其他提供商的策略，不打包安装程序、安装或发布。
- 不触碰既有 Atria 草稿及其他任务的未提交改动。

# Analysis

- Google 已有网络读取前后授权版本保护，但只有前端内存缓存。新增独立公开快照表，按身份／授权版本保存；普通令牌轮换在账号事务内重绑定快照，重新授权或退出不复用旧授权快照。本地恢复不等待联网刷新锁，以 SQL 联查当前身份／授权版本避免旧数据跨域。
- Google 额度与模型目录独立；恢复快照不调用上游，不参与实时调度。前端共用选项同时覆盖账号页与配额总览。
- River 复用既有设置弹出面板、平滑轮询和同池失败机制，默认 queue，balanced 按可用账号等权分配；设置持久化且热生效。
- 401／429 等明确拒绝沿用安全切换。River 连续两次网络／5xx 失败后暂时冷却，影响后续请求，不重放结果不确定的当前请求。固定账号、关闭故障切换、三次尝试和已输出后的不重放边界不变。

# Implementation Plan

1. 明确现有身份／授权版本、缓存显示与调度边界，确定最小兼容方案。
2. 实现 Google 公开额度持久化与恢复显示，覆盖重启／失败／身份变化。
3. 实现 River 队列／均衡模式持久设置和安全失败切换，覆盖固定路由及响应已开始等边界。
4. 运行关键回归和完整质量门，进行可行的隔离原生验收，同步文档、审查并本地提交归档。

# Implementation Result

## Changes

- 迁移 038 保存 Google 每账号一条公开额度分组快照；严格 JSON／比例／时间／数量／长度校验，按账号身份摘要和当前授权版本隔离，账号删除级联清理。
- Google 成功读取后保存，重启／切页先本地恢复再进入网络队列；失败保留原值，暂缺字段合并，新重置窗口不继承旧百分比。普通令牌轮换在账号事务内重绑定，重新授权不恢复旧版本。
- 本地恢复不解密授权、不争用可能联网的刷新锁；SQL 同时核对当前身份和版本，较旧同域响应不能覆盖新快照，新授权可覆盖旧域记录，不受系统时钟回拨阻挡。
- 新增 Google 本地恢复 RPC 和可选 `fromCache` 标志，账号页与配额总览共用查询；留存值不计作当前已确认额度，不参与调度和权限判断。
- 迁移 039 增加独立 River 设置，默认 queue；标题栏复用局部设置面板，两项单选自动持久保存，保存期间互斥、失败保留旧值，支持中英文。
- River balanced 复用有界平滑轮询，按同池健康 API Key 等权分配真实请求；固定账号不重排，配置／版本／计费／地区边界不变。
- 401／402／429 等明确拒绝沿用有限安全切换；一分钟内同账号／版本／模型连续两次网络或 5xx 失败后冷却 30 秒，后续请求跳过，恢复采用既有单个探测机制。
- 网络或结果不确定的当前请求不重放；保留最多三次尝试、全局切换开关及取消保护。修正 River 5xx 尝试记录为失败，避免 HTTP 失败被标为成功。
- 原生验收发现“留存待刷新”误触发“已到官方重置时间”的提示，已将概览留存状态与真实时间到期判定分开，并补回归。

## Files Changed

- `migrations/038_google_subscription_quota_snapshots.sql`、`039_river_account_rotation.sql`
- `internal/application/googlesubscription/quota*.go`、`account/google_subscription.go`
- `internal/provider/antigravity/quota_snapshot.go`、额度 DTO 和稳定错误码白名单
- `internal/application/gateway/river*.go`、共享轮换入口和失败健康状态
- `internal/transport/wails/google_subscription.go`、River 设置适配及网关初始化
- Google 查询／留存提示、River 设置组件／服务／文案和对应回归测试
- 工具生成的 Wails 绑定，产品说明、架构概览、ADR-018／019 扩展及 CURRENT

## Technical Notes

无新增依赖、凭据格式或 Core 到桌面／前端的依赖。Google 仅持久显示白名单 DTO，不保存原始响应或真实令牌；River 健康状态和轮询分数仅在内存有界保存，均衡请求数不保证均衡 Token。共享 failover 主循环保持既有结构，只增加 River 分支，未为文件长度进行无关重构。

# Verification

## Tests

- `go test ./internal/application/googlesubscription ./internal/application/gateway ./internal/application/account ./internal/provider/antigravity ./internal/transport/wails` 通过。
- `go test ./internal/application/googlesubscription ./internal/application/gateway -run 'Google.*Quota|River|QuotaRejects' -count=2` 通过。
- 新授权／时钟边界补充后，Google `Quota|Authorization` 关键回归两轮通过。
- 真实 SQLite 重开、失败保留、缺失字段、新窗口、旧响应、普通令牌轮换、重新授权／退出／邮箱变化／删除、损坏快照及联网锁占用下本地读取均有回归。
- River 实际 HTTP 验证队列／均衡、持久设置、非法设置、401／402／429、连续网络／500／503、冷却和恢复；固定账号、关闭切换、400／403 和取消不因该规则重放。
- 并发旧成功不得清除新冷却，成功重置失败计数，失败尝试正确记账且不增加逻辑请求重放。
- 前端验证冷缓存恢复、读取失败保留、等待网络时可见、退出不恢复、旧值不计作当前额度，以及 River 保存／失败／互斥／重开／中英文。

## Quality Checks

- [x] Format
- [x] Lint
- [x] Type Check
- [x] Test
- [x] Build
- [x] Final Verify

## Verification Result

三轮 `task --force verify BIN_DIR=tmp/task-015-build` 完整通过，最终轮 exit 0：格式、Go vet／staticcheck、ESLint、TypeScript、全部 Go 测试、118 个前端测试文件／700 项测试及 Windows production 构建通过。最终日志为忽略目录内的 `tmp/task-015-verify-final-complete.log`，包含新授权时间域边界回归及留存／到期文案修正。

使用 `--force` 避免既有临时目录导致的增量缓存检查拖延，执行相同格式／lint／类型／测试／构建步骤，并非跳过检查。输出至独立目录，不覆盖开发中的 `bin/omniproxy.exe` 或正式安装程序。

## Native Acceptance

- 独立 test 数据目录／配置／WebView，Google 无效合成授权和两个 River 合成 Key；不读取真实账号，不执行真实推理。
- River 设置在原生深浅主题、1200×800／960×720 检查可见性、保存均衡、关闭重开及多次进程重启；数据库回读确认 `balanced` 持久保留。
- Google 冷启动时从 SQLite 恢复 Gemini 周／5h 的 75%／88%、第三方周／5h 的 64.5%／40%，数据库保持一条快照。
- 无效合成授权导致模型／额度联网读取在本地授权检查处失败，四个百分比及原重置时间仍可显示；未伪造联网成功。
- 已查看实际截图；Google 完整卡片通过 1200×1100 高窗口复核，留存标记存在且修复后不再误报到达重置时间。最小窗口中的卡片需正常纵向滚动，未将 UIA 未成功滚到底部的截图当作完整底部可见证据。
- 验收进程按精确路径关闭，未停止生产进程。临时截图／日志在忽略目录 `tmp/task-015-native/`。
- 现有 jsdom `scrollTo` 未实现及 Vite 大 chunk 为非阻断提示。

# Documentation

已同步 PRD、GOOGLE_SUBSCRIPTION、RIVER、FAILOVER、Architecture Overview、ADR-018／019 扩展和 CURRENT。没有技术栈、构建或测试规则变更。

# Deviations

- Google 的目录授权版本会随普通令牌轮换改变，因此增加同事务快照重绑定；不能直接复制 OpenAI 的隔离配置摘要规则。
- 对结果不确定的失败采用后续请求冷却，而不是盲目重放当前请求；符合既有安全边界。
- 真实账号官方读取及真实多 Key 消费不做试验；使用本地契约服务、真实 SQLite／DPAPI 和合成账号验收。

# Git

`9149578f8bb0d498388a981340310e3fd0044d30` — `feat(account): 持久化 Google 额度并完善 River 调度与切换`

仅提交本任务 60 个文件；既有未跟踪 Atria 草稿未修改、未提交。未推送远端。

# Result

功能、完整质量门、产品文档及隔离原生验收已完成并本地提交归档。未打包安装程序、安装或发布。

# Completed

2026-10-20
