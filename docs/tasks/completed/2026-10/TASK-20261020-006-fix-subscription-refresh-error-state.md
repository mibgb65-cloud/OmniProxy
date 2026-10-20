# TASK-20261020-006: 修复订阅账号刷新时的取消误报与持久错误提示

## Status

Completed

## Created

2026-10-20

## Request

用户反馈 ChatGPT／Codex 账号刷新后间歇显示“授权操作已取消，请重新开始”、
“操作未完成，请检查网络和登录组件后重试”，影响使用体验。
要求对照 `E:\go\new\OmniProxy` 的 v1 实现定位并修复，不仅隐藏错误。

## Goal

区分授权生命周期、额度刷新和临时网络失败，修复取消误判及过时错误状态，
保持真实失效授权和安全错误可见，正常刷新无需重复登录。

## Scope

- 只读对照 v1 订阅授权与额度刷新实现。
- 检查当前前后端刷新、取消、并发、错误映射及账号卡片状态。
- 根因修复、回归测试、完整质量门及必要文档。

## Out of Scope

- 修改 v1 参考仓库、发布或替换生产安装版。
- 输出真实 token、授权文件、账号内容或未脱敏日志。
- 未经验证移除身份检查、吞掉错误、伪造刷新成功。

## Existing Behavior

手动刷新调用 `account/read(refreshToken=true)`，每次强制访问认证服务；
RPC 将 DeadlineExceeded 与主动取消都映射成 `operation_cancelled`。
身份查询出错后仍尝试把账号状态写为 `unknown`，原始错误又可能落到通用文案。
前端 useMutation 的错误长期内嵌卡片，其他操作成功不会自动清除。

## Expected Behavior

普通额度刷新与主动取消授权互不误报；临时失败保留最近额度并给出准确提示，
恢复成功后清除对应的过时错误；真实认证失效保留明确的重新登录指引。

---

# Analysis

## Related Modules

- `internal/infrastructure/codex/client.go`：RPC 超时／取消分类。
- `internal/application/subscription/operations.go`：手动刷新、状态写入及可选 account_id。
- `frontend/src/features/accounts/subscription-account-status.tsx`：操作反馈生命周期。
- `frontend/src/services/subscription.ts`、中英文订阅文案及相关测试。
- v1 只读参考：`OmniProxyBackend/internal/proxy/codex_refresh.go`。

## Dependencies

预计无新增依赖。

## Risks

不得破坏授权版本保护、成员／工作区身份隔离、凭据锁和退出／删除语义。

---

# Implementation Plan

1. 阅读相关规范、历史修复及当前订阅实现，对照 v1。
2. 用隔离组件／模拟登录组件复现实际根因，比较最小修复方案。
3. 实施修正，覆盖取消、超时、并发与恢复边界。
4. 执行定向测试与完整 `task verify`，核对实际 UI，更新文档与任务记录。
5. 审查 diff，只提交本任务变更并归档记录。

---

# Progress

- 已创建任务记录，尚未修改实现。
- 对照 v1：JWT／auth 元数据本地读取，接近过期才续期；不将每次刷新都作为强制 OAuth。
- 比较方案后不移植 v1 自管 token，不更改凭据格式、网关强制核验或登录完成后的授权缓存同步。
  手动刷新只读托管身份，已有工作区 ID 不重复触发附加用量网络读取；
  真正用量与模型请求仍按原有逻辑核对工作区及成员。
- 修复 RPC 的截止时间误分类，普通刷新增加包含排队的总预算；
  临时错误不改已保存授权状态，只提供有界非阻断反馈；确切缺失授权与身份变化继续明确提示。
- 前端操作错误与提交时账号版本关联，后续成功重新登录／身份更新后不保留旧红字。
- 官方公开 app-server 文档再次确认 `refreshToken=true` 在托管 ChatGPT 模式下强制续期：
  https://learn.chatgpt.com/docs/app-server 。保留其正常托管续期，不照搬 v1 私有 token 管理。
- 新增离线测试先复现强制续期、取消／超时混淆及失败改写状态，再修复。
  订阅服务与官方组件适配层全包测试通过；关键回归连续 3 轮通过。
- 前端类型检查与四个测试文件／38 项定向回归通过，包括 5 秒自动消失、
  关闭、重试恢复、真实身份错误及中英文。
- 正在隔离源码执行完整质量门，避免影响其他 Wails 开发构建产物。

---

# Implementation Result

## Changes

- RPC 使用可解包的稳定 context 错误，分别保留超时与主动取消；
  请求已过期时不向官方组件写入 RPC。
- 普通账号刷新使用 `refreshToken=false`，30 秒预算包含排队；
  已有工作区标识不重复请求附加用量元数据。
- 临时组件／读取／取消错误不修改授权状态、更新时间及工作区；
  真正缺失授权仍标为需要重新登录，邮箱变化仍拒绝。
- 刷新失败按繁忙、超时、临时不可用等稳定码分类，不暴露原始组件内容。
- 临时失败采用可关闭的 5 秒 StatusToast；身份、组件缺失、存储故障仍内联显示。
  反馈与操作提交时的账号版本关联，后续重新登录不继承旧错误。
- 同步订阅说明及架构文档，无新增依赖、迁移或公开 DTO。

## Files Changed

- `internal/infrastructure/codex/client.go`、`context_error.go` 及回归测试。
- `internal/application/subscription/status_refresh.go`、`operations.go`、
  `diagnostic.go` 及刷新／排队／可选身份测试。
- `frontend/src/features/accounts/subscription-account-status.tsx` 及反馈测试。
- `frontend/src/services/subscription.ts` 及错误映射测试。
- `frontend/src/i18n/locales/subscription-{zh,en}.ts`。
- `docs/product/SUBSCRIPTION.md`、`docs/architecture/OVERVIEW.md` 和任务看板。

## Technical Notes

网关取令牌前及授权完成后的强制同步保持原样，实际额度／推理的成员和工作区
校验不变。对 v1 仅作只读参考，未修改 v1 代码或移植其自管 OAuth。
已确认并复现代码缺陷；未读取用户真实授权或完整原始日志，不将所有历史失败
宣称为同一种网络根因。

---

# Verification

## Tests

- PASS：`go test ./internal/application/subscription ./internal/infrastructure/codex -count=1`。
- PASS：刷新／RPC 截止时间／已取消请求／缺失账号 ID 的关键回归 `-count=3`。
- PASS：前端 `pnpm typecheck`。
- PASS：订阅错误映射、刷新反馈、登录生命周期及额度四个文件，共 38 项测试。

## Quality Checks

- PASS：在 `tmp/subscription-refresh-state-20261020/verify-source` 中执行
  `task verify BIN_DIR=../verify-bin`，exit 0。格式检查、Go vet／staticcheck、
  ESLint、TypeScript、全部 Go 测试、113 个文件／666 项前端测试、
  Wails 绑定生成、前端生产构建与 Windows amd64 production 构建全部通过。
- 现有 jsdom `scrollTo` 未实现、Vite 大 chunk／插件耗时提示非阻断；
  未删测试、跳过门禁或修改公开类型来绕过验证。

## Native Verification

- 使用独立 test profile、工作区内 SQLite／DPAPI 定位夹具、隔离的
  LOCALAPPDATA／APPDATA 和无网络 CLI 替身尝试原生验收。
- 两次尝试中，首次未通过 MainWindowHandle 取得隐藏窗口，
  改用 Win32 按目标 PID 枚举后取得窗口句柄，但 UIA 未取得“账号”导航内容。
  **原生交互验收未通过，不宣称原生提示与重试流程已验收。**
- 已关闭且只关闭本任务启动的隔离测试进程；未读取真实账号授权，
  未修改生产数据、默认 Codex 配置或已安装程序。
- 临时反馈自动消失、关闭、重试恢复和真实身份警告由实际 React／Radix
  组件测试覆盖；原生可见窗口及真实网络仍待人工验收。

---

# Git

- `ac9d5a74c0bf7e22e99927cb0fda83e28a722f2f fix(subscription): 避免刷新误报授权取消和持久错误`。
- `git diff --check` 通过；15 个代码／测试文件与完整门禁隔离源码 SHA256 一致，
  本任务 19 个变更文件已核对为 UTF-8 无 BOM。
- 仅本地提交，未推送；未纳入原有无关的未跟踪任务文件。
- 完成状态、任务归档与 CURRENT 更新使用后续文档提交。

---

# Deviations

普通刷新借鉴 v1 的按需授权思想，但仍保留官方组件与隔离凭据体系，
不重写为自管 OAuth，也不增加网络重试或延长原有 30 秒预算。
原生 UIA 验收未取得导航内容，已记录限制；以组件回归与完整门禁交付源码，
不以模拟测试宣称真实账号网络问题已全部消失。

---

# Result

已修复并覆盖普通刷新强制续期、截止时间误报授权取消、临时失败改写状态、
操作错误长期留在卡片及账号更新后残留旧错误的问题。
完整质量门与 666 项前端测试通过；真实身份／授权缺失仍可见。
原生 UIA 和真实账号网络验收受限，已明确记录，不作为已通过项。
未发布、打安装包或替换已安装生产程序；v1 参考仓库未修改。

# Completed

2026-10-20
