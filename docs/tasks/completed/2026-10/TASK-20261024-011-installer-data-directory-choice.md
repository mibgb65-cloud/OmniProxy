# TASK-20261024-011-installer-data-directory-choice

## Status

Completed

## Created

2026-10-24 14:10

## Request

用户看到「数据存储」显示当前数据在 `C:\Users\<用户>\AppData\Roaming\OmniProxy\v2`，提出：安装的时候应该让用户自己选择数据存储位置，直接放在 C 盘不合适。

## Goal

Windows 安装向导增加可选的「数据存储位置」步骤：留空沿用当前用户的应用数据目录（现状），选择其他目录后程序首次启动即在该目录建库；应用内「数据存储」的迁移能力保持不变。

## Background

- 数据目录解析顺序（`internal/platform/profile/profile.go`）：`OMNIPROXY_DATA_DIR` > `app.json` 的 `dataDir` > 各 profile 默认（`%APPDATA%\OmniProxy`）。数据库实际落在 `dataDir\v2\omniproxy.db`。
- 安装器 `build/windows/nsis/project.nsi` 默认 `INSTALL_SCOPE=machine`（`$PROGRAMFILES64` + HKLM，需要 UAC），升级与自动更新走 `installer_runtime.nsh` 的 `/OMNIPROXY_AUTOUPDATE=1` 静默路径。
- 数据目录是**每用户**概念（`%APPDATA%` + DPAPI 凭据），而安装器是**机器级**的；因此安装页只能记录「建议位置」，不能直接写某个用户的 `app.json`（提权进程的 `$APPDATA` 未必属于实际使用该程序的账号）。

## Scope

- `internal/platform/profile`：读取安装目录下的 `data-dir.txt`，作为 `app.json` 之下的默认值来源。
- `build/windows/nsis`：新增自定义安装页（可留空）、写入／保留／删除 `data-dir.txt`、卸载清理。
- `docs/development/BUILD.md`、`docs/architecture/OVERVIEW.md`：记录解析顺序与安装页行为。

## Out of Scope

- 不改 `app.json` 结构、不改数据库位置规则（`dataDir\v2\omniproxy.db` 不变）。
- 不做多用户共享数据目录，也不在安装时创建／授权目标目录（由应用首次启动按当前用户权限创建）。
- 不改安装范围（继续 machine scope），不改更新与卸载流程，不改应用内「数据存储 → 数据主目录」的搬迁行为。

## Existing Behavior

安装完成首次启动，数据写入 `%APPDATA%\OmniProxy`（历史库迁移过的用户为 `%APPDATA%\OmniProxy\v2`）；用户只能在应用内「数据存储 → 数据主目录」改位置并重启完成搬迁。

## Expected Behavior

- 安装向导在「选择安装目录」之后新增「数据存储位置」页：留空 = 使用默认（当前用户应用数据目录），可选其他目录。
- 选择自定义目录后，安装目录写入 `data-dir.txt`；应用启动时在该目录建库并使用。
- 应用内改过位置（`app.json.dataDir`）的用户不受影响：`app.json` 优先于安装选择。
- 静默升级不触碰该文件；卸载随安装目录删除，数据本身保留。

---

# Analysis

## Related Modules

- `internal/platform/profile/profile.go`（`Detect`）、`config.go`（`app.json`）
- `build/windows/nsis/project.nsi`、`installer_runtime.nsh`
- `docs/development/BUILD.md`、`docs/architecture/OVERVIEW.md`

## Root Cause

安装器从未收集数据位置，`profile.New` 直接使用 `defaultDataDir()`；应用内虽然有迁移入口，但用户要在装完后主动去找，首启数据已经落在系统盘。

## Fix

安装页把选择写成安装目录下的 `data-dir.txt`（单行绝对路径），`profile.Detect` 在 `OMNIPROXY_DATA_DIR` 与 `app.json` 之后、内置默认之前读取它；读取失败或目录不可创建时回落默认并写 warn 日志。

## Regression Prevention

- profile 单测覆盖解析规则（空、相对路径、带引号、BOM、UTF-16LE、多行）与解析顺序（env > app.json > 安装文件 > 默认）以及不可用目录的回落。
- 安装页写入只发生在非静默路径，`${IfNot} ${Silent}` 保证静默升级不改写既有选择。

## Dependencies

无新增 Go 依赖；`makensis` 已在既有打包环境使用。

## Risks

- 机器级安装下该选择对「尚未自行选择的用户」生效；多用户共享同一台机器时属于已知限制，任何人都可在应用内改回自己的位置。
- 自定义目录被删除或权限变化时，启动回落默认位置并记录原因，界面上显示的是实际生效目录。
- 安装向导页只能人工验收；本轮以 `makensis` 编译通过（5 个页面）为界，未在真实安装流程中点击验证。

---

# Implementation Plan

1. 创建 Task Record。
2. `profile` 增加 `data-dir.txt` 解析并接入 `Detect`，补单测。
3. NSIS 增加数据位置页与文件写入／保留／删除逻辑。
4. `makensis` 编译安装器脚本验证语法，运行 `task verify`。
5. 更新 BUILD／OVERVIEW 文档与看板，提交。

---

# Progress

- 确认安装器为 machine scope：安装页不能直接写用户 `app.json`，改为安装目录内的建议文件。
- `makensis` 编译通过（Welcome／安装目录／数据存储位置／安装／完成 共 5 页）。
- 用 `task package` 实际产出含新页面的安装器，随后把 Beta.32 安装包原样还原。

# Implementation Result

## Changes

- 新增 `internal/platform/profile/installdir.go`：读取可执行文件同目录的 `data-dir.txt`，支持 UTF-8（含 BOM）与 UTF-16LE／BE；只取第一行、去引号，非绝对路径视为无选择。
- `profile.Detect`：在环境变量与 `app.json` 之后应用安装选择；目录无法创建时回落到 profile 默认并 `slog.Warn`，不阻断启动。
- 新增 `build/windows/nsis/data_dir_page.nsh`：自定义安装页（说明、目录输入框、浏览按钮、留空提示），校验绝对路径与磁盘根目录，覆盖安装时预填既有选择，留空即保留原设置。
- `project.nsi`：在「选择安装目录」后插入该页；安装段写入 `data-dir.txt`（静默升级跳过）；卸载删除该文件。
- 新增 7 组 profile 单测：解析规则、UTF-16 解码、解析顺序、不可用目录回落。

## Files Changed

- `internal/platform/profile/installdir.go`、`installdir_test.go`、`profile.go`
- `build/windows/nsis/data_dir_page.nsh`、`project.nsi`
- `docs/development/BUILD.md`、`docs/architecture/OVERVIEW.md`

## Technical Notes

- `data-dir.txt` 只承载「安装时建议」：应用内（`app.json.dataDir`）的选择永远优先，所以重装或静默升级不会把用户搬过的数据目录打回系统盘。
- 安装器是提权进程，`$APPDATA` 不一定是使用者的目录，因此不采用「安装时直接写 app.json」的方案。
- 目标目录不在安装阶段创建，由应用首次启动按当前用户身份创建，避免出现管理员属主目录。
- 交付给用户试用的安装器：`bin/OmniProxy-amd64-installer-data-dir-page.exe`（19,793,667 bytes，SHA256 `B1B197F2629215E7D5136453C243615B3458C72EBCBE2BE2218E891F23523269`，未签名，版本号仍为 2.0.0-beta.32）。为避免覆盖既有资产，Beta.32 的 `bin/OmniProxy-amd64-installer.exe`（19,779,433 bytes）已原样还原。

---

# Verification

## Tests

- [x] Unit Tests：`go test ./internal/platform/profile/` 全部通过，新增 7 项用例覆盖解析与优先级。
- [x] Integration Tests：`task verify` 内 Go 全部包与前端 126 文件／751 项测试通过。
- [ ] E2E Tests：N/A。安装向导页未做真实点击验收（需交互式安装），静默升级路径按 `${IfNot} ${Silent}` 逻辑静态确认。

## Quality Checks

- [x] Format：`gofmt`／Prettier 通过（首次 `format:check` 报 `installdir_test.go` 未格式化，已修正）。
- [x] Lint：`go vet`／staticcheck／ESLint 通过。
- [x] Type Check：`tsc --noEmit` 通过。
- [x] Test：通过。
- [x] Build：Windows production 构建通过。
- [x] Verify：`task verify` PASS。

## Verification Result

```text
task verify
VERIFY_EXIT=0
Test Files  126 passed (126)
Tests  751 passed (751)
```

```text
makensis -INPUTCHARSET UTF8 -DARG_WAILS_AMD64_BINARY=...\bin\omniproxy.exe project.nsi
Install: 5 pages ...
PKG_EXIT=0
```

`task package` 实际产物 `bin/OmniProxy-amd64-installer-data-dir-page.exe`：19,793,667 bytes（比 Beta.32 的 19,779,433 bytes 大 14,234 bytes，主要为新页脚本与更长的辅助程序）。未执行真实安装、未签名。

---

# Documentation

- [x] `docs/development/BUILD.md`：新增「安装时的数据存储位置」，说明该页语义、静默升级不触碰、目录由应用创建、卸载行为。
- [x] `docs/architecture/OVERVIEW.md`：Runtime profile 一节补充数据目录解析顺序与回落规则。
- [x] `docs/plans/CURRENT.md`：追加本任务一行（该文件含其他在途改动，未随本任务提交）。
- PRD／Tech Stack 无需修改。

---

# Deviations

原计划「安装时选择的数据目录写进应用配置」。实际改为写安装目录下的 `data-dir.txt` 并由 profile 解析：安装器是机器级提权进程，直接写某个用户的 `app.json` 在「输入了其他管理员账号」时会写错人，也无法服务多用户机器。

---

# Git

## Commit

```text
6529fde feat(installer): 安装时可选数据存储位置
```

---

# Result

安装向导新增「数据存储位置」页：默认仍是当前用户的应用数据目录，可改到其他磁盘；选择只作为建议值，应用内已设置的位置优先，静默升级不会改动它。应用侧解析顺序、回落规则与文档同步更新，并有单测覆盖。
