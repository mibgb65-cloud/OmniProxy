# TASK-20261023-017: 数据存储占用拆分更新缓存与登录凭据并清理更新残留

## Status

In Progress

## Created

2026-10-23 22:10

## Request

用户提供「设置与关于 → 数据存储」截图并指出：这个数据库显示的占用，其他里面是什么情况，怎么占用的这么多，不能分类出来吗？这样看起来好累。

## Goal

- 「其他」不再吞掉 86% 的占用：把更新缓存（`updates/`）与登录凭据（`credentials/`）单独分类，其余零散文件才计入「其他」。
- 占用分布只展示实际存在大小的分类并按大小排序，减少同时读 4～6 个数字的负担。
- 说明更新缓存到底是什么，并提供显式清理动作释放安装过程留下的辅助程序副本。

## Background

- 现状：`storage.MeasureUsage` 只区分数据库（`omniproxy.db` 及 `-wal`／`-shm`／`-journal`）、`logs/`、`cache/`，其余全部累加到 `Other`。用户实机数据目录里 `updates/` 有 102 MB（占 86%），其中 5 个 `runner-<32hex>.exe` 是每次安装尝试由 `appinstall.Launch` 复制的辅助程序（各约 21 MB），安装结束后从不删除；1 个 `package-<32hex>.exe` 是当前下载缓存。
- 因此「其他 102 MB 86%」实际是 5 份安装残留 + 1 份安装包缓存，用户无法从界面判断，也无法释放。
- `updates/` 的状态与清理规则由 `internal/application/updater` 持有（`status.json` + 包文件），安装辅助副本由 `internal/infrastructure/appinstall` 生成，两边都不负责回收。

## Scope

本任务预计包含：

- `internal/infrastructure/storage/usage.go`：新增 `Updates`、`Credentials` 分类与子目录常量，其余文件仍计入 `Other`。
- 新增 `internal/application/updater/clean.go`：`CleanCache` 删除 `runner-*.exe` 辅助副本与 `*.tmp` 残留，不动安装包缓存与 `status.json`。
- `internal/transport/wails/storage.go`：`StorageInfo` 增加 `updates`／`credentials` 两项，新增清理目标 `updates`。
- 前端：占用分布新增分类、按大小降序只显示非零项、更新缓存说明与「清理更新残留」按钮。
- 中英文案、Go／前端回归测试、重新生成绑定。
- 文档：`docs/product/UPDATE.md`、`docs/plans/CURRENT.md`。

## Out of Scope

- 不改 `logs/`、数据库的清理语义与 `CleanStorage` 既有目标行为。
- 不改更新下载、校验、安装流程本身，也不自动删除任何文件（清理仍由用户显式触发）。
- 不动 `cache/` 的实际用途（当前为空，保留分类）。
- 不删除已下载的 `package-*.exe`：它仍可能被重试安装复用，删除会强制重新下载。

## Existing Behavior

- 数据存储卡片固定显示数据库／运行日志／本地缓存／其他四项，`updates/`、`credentials/` 与其它零散文件都计入「其他」。
- 没有任何入口清理更新目录，安装残留会一直堆积。

## Expected Behavior

- 占用分布显示：数据库、运行日志、本地缓存、更新缓存、登录凭据、其他；只显示实际有内容的分类，并按字节数从大到小排列，占比与色块顺序一致。
- 「更新缓存」有于更新目录存在时的一行说明，并出现「清理更新残留」按钮；点击后删除安装辅助副本与临时文件，反馈释放空间并刷新占用。
- 没有内容时显示「暂无可统计的文件占用。」而不是六行 0 B。

---

# Analysis

## Related Modules

- 存储统计：`internal/infrastructure/storage`（`MeasureUsage`）、`internal/transport/wails/storage.go`（`StorageInfo`／`CleanStorage`）。
- 更新目录写入方（只读确认）：`internal/infrastructure/appinstall/launch_windows.go`（`runner-*.exe`）、`internal/application/updater/{transfer,persistence}.go`（`package-*.exe`、`status.json`）。
- 前端：`features/settings/{storage-usage-chart,storage-overview,use-storage}.tsx`、`services/omniproxy.ts`、`i18n/locales/{zh,en}.ts`。

## Dependencies

无新增依赖；绑定由 `wails3 generate bindings` 重新生成。

## Risks

- `runner-*.exe` 正在被辅助进程运行时删除会失败（Windows 文件占用）。此时必须返回错误而不是假装成功，且不影响其它文件已释放的空间。
- 更新目录的文件名规则必须与写入方一致（`package-[0-9a-f]{32}.exe`、`runner-<32hex>.exe`），否则可能误删 `status.json` 指向的包。
- 分类顺序依赖 `MeasureUsage` 的 `switch` 分支顺序：`logs` 必须先于其它子目录判断，避免自定义日志目录名与固定分类冲突。
- 既有前端测试断言「0 占用时出现 4 行 0 B」，隐藏零值分类后需同步改写。

---

# Implementation Plan

1. `storage/usage.go`：新增子目录常量 `DirCache`／`DirUpdates`／`DirCredentials`，`Usage` 增加 `Updates`、`Credentials`，`switch` 分类。
2. 新增 `updater/clean.go`：`CleanCache(directory)` 扫描目录，删除 `runner-<32hex>.exe` 与 `*.tmp`，返回释放字节；目录不存在视为 0。
3. `transport/wails/storage.go`：`Items` 增加 `updates`、`credentials`；`cleanTargetUpdates = "updates"` 调用 `updater.CleanCache`。
4. 重新生成绑定。
5. 前端图表：分类表增加两项与配色；按字节降序只渲染非零项；全部为 0 时显示空态文案；更新缓存存在时展示一行说明。
6. 前端卡片：新增「清理更新残留」按钮（无更新缓存时隐藏），复用现有清理反馈。
7. 补 Go 与前端回归测试，`task verify`，更新文档。

---

# Progress

- 2026-10-23 22:10 建立任务记录；实测数据目录：`updates/` 102.01 MB（5×runner + 1×package）、数据库 12.5 MB、日志 3.7 MB、`credentials/` 0.17 MB，确认「其他」几乎全部是更新残留。

---

# Implementation Result

## 后端

- `internal/infrastructure/storage/usage.go`：新增常量 `DirCache = "cache"`、`DirUpdates = "updates"`、`DirCredentials = "credentials"`；`Usage` 由 `{Database, Logs, Cache, Other}` 扩为 `{Database, Logs, Cache, Updates, Credentials, Other}`。单次遍历按目录名分类，`switch` 顺序为 `logs`（含自定义日志目录）→ `credentials` → `cache` → `updates` → 数据库文件（`omniproxy.db` 及 `-wal`／`-shm`／`-journal`）→ 其余计入 `Other`，`Total` 仍为实测总和。
- `internal/application/updater/clean.go`（新增）：`CleanCache(directory) (int64, error)` 删除 `runner-[0-9a-f]{32}.exe` 辅助程序副本与 `*.tmp` 临时文件，返回累计释放字节；目录不存在返回 0；被占用的文件返回错误，不谎报成功。
- `internal/transport/wails/storage.go`：`StorageInfo.Items` 由 4 项扩为 6 项（`database`／`logs`／`cache`／`updates`／`credentials`／`other`），路径与文案使用 `storage.Dir*` 常量；`CleanStorage` 新增目标 `updates` → `updater.CleanCache(<DataDir>/updates)`，其余目标行为不变。
- `frontend/bindings/…/appservice.ts` 由 `wails3 generate bindings -f "-tags production -trimpath -buildvcs=false -ldflags='-w -s -H windowsgui'" -clean=true -ts -i` 重新生成，仅 `CleanStorage` 文档注释变化。

## 前端

- `features/settings/storage-usage-chart.tsx`：分类增至 6 项（更新缓存 `bg-violet-500`、登录凭据 `bg-rose-400`）；过滤 `bytes > 0` 并按字节降序渲染，色块与列表顺序一致；全部为 0 时显示 `settings.storage.empty`；存在更新缓存时追加一行说明。
- `features/settings/storage-overview.tsx`：仅当更新缓存大于 0 时渲染「清理更新残留」，复用既有清理反馈与占用刷新。
- `features/settings/use-storage.ts`、`services/omniproxy.ts`：`CleanTarget`／`cleanStorage` 增加 `"updates"`。
- 中英文案：`settings.storage.updates`／`.credentials`／`.empty`／`.updatesNote`／`.cleanUpdates`。
- 回归测试：`storage-usage-chart`／`storage-overview` 覆盖非零排序、空态与清理按钮可见性；Go 侧覆盖分类字节与 `CleanCache` 的保留／删除规则。

## 实测

- 用户实机数据目录 `C:\Users\mimanchi\AppData\Roaming\OmniProxy\v2`：`updates\` 102.01 MB（5 × `runner-<32hex>.exe` ≈ 20.9 MB + 1 × `package-<32hex>.exe` 18.41 MB + `status.json`／`install-result.json`）、`omniproxy.db` 8.5 MB + `-wal` 3.94 MB、`logs\` 3.72 MB、`credentials\` 0.17 MB。界面旧口径「其他 102 MB 86%」即为这批安装残留。
- 预览渲染复核：分布为 更新缓存 102 MB 86% → 数据库 12 MB 11% → 运行日志 3.7 MB 3% → 登录凭据 174 KB <1%，无 0 B 行；说明文字与「清理更新残留」按钮就位。

---

# Verification

## Tests

- [x] Unit Tests
- [x] Integration Tests
- [ ] E2E Tests

## Quality Checks

- [x] Format
- [x] Lint
- [x] Type Check
- [x] Test
- [x] Build
- [x] Verify

## Verification Result

`task verify` exit 0：

- `gofmt`／`prettier --check` 通过（首次运行发现 `storage-overview.test.tsx` 未格式化，`prettier --write` 后复跑通过）。
- `go vet . ./internal/...`、`staticcheck . ./internal/...`、`eslint .`、`tsc --noEmit` 无问题。
- `go test . ./internal/...` 全部包通过（含新增 `updater.CleanCache`、`storage.MeasureUsage`、`transport/wails` 存储用例）。
- `vitest run`：126 文件 / 750 项测试全部通过。
- Windows production 构建：绑定重新生成、`vite build`、`go build -tags production … -o bin/omniproxy.exe` 成功。

说明：在 `$ErrorActionPreference = 'Stop'` 下 `task verify` 会因任务脚本的 `uname`／`tail` 探测向 stderr 输出而被 PowerShell 判为失败，实际构建与测试全部成功；以 `$ErrorActionPreference = 'Continue'` 复跑得到 `EXIT=0` 并保留日志。

原生窗口视觉尚待用户确认（本次通过 Vite 预览页在 960×900 宽度下核对布局，预览文件已在提交前删除）。

---

# Documentation

- [ ] PRD
- [ ] Architecture
- [ ] Tech Stack
- [ ] Coding Rules
- [ ] Testing
- [ ] Build
- [ ] ADR
- [x] CURRENT

- `docs/product/UPDATE.md`：新增「更新目录与残留清理」，说明 `updates/` 内容（状态、结果、安装包缓存、辅助程序副本）、数据存储的六类占用与非零降序展示、以及「清理更新残留」的删除范围与失败语义。
- `docs/plans/CURRENT.md`：In Progress 增加本任务摘要。
- `docs/product/PRD.md` 未改：数据存储属于既有设置能力，本次只细化分类与清理入口，不新增产品边界。ADR 无变化（存储目录布局与后台更新策略未变）。

---

# Deviations

- 「其他」没有降到 0：`credentials/production/` 之外的零散文件仍归「其他」，只是不再包含更新目录；实机上「其他」现在为 0，因此界面上不显示该行。
- 未自动清理安装残留：按需显式触发，避免在用户不知情时删除文件；`updates/` 的自动回收不在本任务范围。
- 保留已下载的 `package-*.exe`（可能被重试安装复用）与 `status.json`／`install-result.json`（可诊断依据），因此清理后更新缓存通常仍显示约 18 MB。
- 保留 `cache/` 分类但当前无写入方，实际显示为 0 并被隐藏。
- 预览用的 `frontend/storage-preview.html` 与 `frontend/src/preview/storage-preview.tsx` 为临时验证文件，未进入提交。

---

# Git

## Commit

- `59f4674` feat(settings): 数据存储拆出更新缓存与登录凭据并支持清理残留

---

# Result

「其他」不再是一个黑盒：占用分布拆成更新缓存、登录凭据等六类，只显示有内容的分类并按大小降序排列；更新缓存附一行说明，配合「清理更新残留」可释放安装过程留下的辅助程序副本，同时保留已下载安装包避免重复下载。实机 118 MB 中原来记为「其他」的 102 MB 现在明确显示为更新缓存。

---

# Completed

（待完成）
