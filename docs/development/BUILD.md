# Build

## Prerequisites

- Go 1.26+, Node.js 22, pnpm, Task, Wails 3 CLI (see root `README.md`).
- Windows: the WebView2 runtime (preinstalled on Windows 11 and up-to-date
  Windows 10).

当前 Wails Go 模块与 CLI 使用 `v3.0.0-beta.30`，前端运行时使用 `3.0.0-beta.30`。
本机安装 CLI：`go install github.com/wailsapp/wails/v3/cmd/wails3@v3.0.0-beta.30`。
升级时同步 Go 依赖、前端运行时和 CLI，避免生成绑定或构建使用不同版本。

## One-time setup

```bash
task setup
```

Installs frontend dependencies, downloads Go modules and installs
`staticcheck` (used by `task lint`).

## Development

```bash
task dev
```

What happens:

1. `wails3 build DEV=true` builds the frontend (Vite) and the Go binary in
   development mode.
2. The Vite dev server runs in the background (HMR on port 9245 by default).
3. 开发版应用在创建窗口前等待 Vite 首页返回 HTTP 200，最长 120 秒；
   单次请求最多 2 秒，每 500 毫秒重试。就绪后进入 Wails 原有启动流程并加载 UI。

等待使用 Wails 提供的 `FRONTEND_DEVSERVER_URL`，只接受本机 HTTP/HTTPS 地址，
不跟随重定向或使用系统代理。超时会返回明确错误并退出，检查同一终端的 Vite
错误输出后再启动。未配置开发地址时使用内嵌前端；生产版不执行此等待。

Frontend-only work can also use `cd frontend && pnpm dev`, but the app itself
needs `task dev`.

恢复开发窗口时使用 `task dev`，并确认启动日志包含连接前端开发服务的信息。`task run`／`wails3 task run` 仅运行现有二进制；Wails 开发代理还需要 `FRONTEND_DEVSERVER_URL`，单独启动 Vite 不会保证窗口加载最新源码。临时目录中的质量验证构建也不会替换正在运行的应用。完成界面修改后，必须在项目路径对应的原生窗口核对实际内容；存在多个同名窗口时按进程路径识别，避免只验证浏览器预览或选错应用。

## Production build

干净检出执行 `task verify` 时先安装前端依赖；`task lint`／`task test` 在 Go 分析前准备真实的 `frontend/dist`，避免 go:embed 因目录不存在而失败。准备阶段使用已提交的绑定，正式 build 仍重新生成绑定并构建前端。CI 与 Release 都运行同一个质量门，不依赖开发机残留产物。

```bash
task build
```

- Builds the frontend for production (`frontend/dist`, embedded into the
  binary via `//go:embed all:frontend/dist`).
- Builds `bin/omniproxy.exe` with the `production` build tag (which selects the
  production runtime profile).
- Generates Windows resource data (icon, manifest, version info) from
  `build/windows/`.

## 应用图标

NSIS 同时安装自有 `app-icon-v2.ico` 和 `OmniProxy-LICENSE.txt`（保留原 MIT 版权与许可文本）。
桌面／开始菜单快捷方式显式引用自有 ICO，避免复用旧 EXE 路径的 Shell 图标缓存。
卸载只删除这些明确归属程序的文件。更换下一代图标设计时应同时考虑新资源路径，
不能仅覆盖图像后假设 Explorer 已刷新。

当前采用 C「钴蓝汇聚」，源设计位于 `design/app-icon/c-cobalt-hub.png`。
`build/appicon.png` 是构建源图，`task common:generate:icons` 生成 Windows ICO，
保留 16／20／24／32／40／48／64／96／128／256 像素资源；Windows 窗口和任务栏从可执行文件读取该图标。

`internal/transport/wails/assets/app-icon.png` 与 `frontend/public/app-icon.png`
使用相同的 256 像素透明 PNG，分别供内嵌托盘／窗口回退和标题栏／侧栏／favicon 使用。
更新设计时同步上述资源，再重新构建与启动；运行中的进程仍使用旧的内嵌资源。

## Packaging

正式 Windows 资产、版本一致性、SHA256 与发布流程见 [RELEASE](RELEASE.md)。机器级实际安装需要管理员权限，编译安装器不等于完成安装验收。

```bash
task package
```

Creates an NSIS installer (`build/windows/nsis/`) using the binary produced by
`task build`. Code signing (SignPath) is configured separately and is not part
of the default build.

NSIS 打包需要 `makensis` 在 PATH 中。打包命令显式使用 `-INPUTCHARSET UTF8`，
与仓库 UTF-8 无 BOM 的脚本编码保持一致，避免中文 Windows 按系统代码页读取时报错。

## 安装向导（语言、安装目录、数据目录）

向导顺序：语言（中文/English，记录在 `HKCU\Software\OmniProxy\InstallerLanguage`）→
欢迎 → 安装目录 → **数据存储位置** → 安装 → 完成。自定义文案集中在
`build/windows/nsis/strings.nsh`，页面逻辑在 `data_dir_page.nsh`。

数据存储位置页的语义：

- 页面打开时通过安装辅助程序（`omniproxy.exe --omniproxy-installer-data-dir-read <原安装程序绝对路径>`）
  读取实际使用者的 `app.json`，然后兼容**原安装目录**的 `data-dir.txt`，均不存在才使用默认位置。
  原安装目录在向导初始化时保留，不受后续程序目录选择影响；助手自己的临时目录不能作为来源。
  配置／标记损坏、权限不足或身份无法确认时停止安装，不猜测默认值。
- 用户改动时先弹出「不会迁移已有数据」的确认框，取消则留在本页。
- 确认后由辅助程序写入（`--omniproxy-installer-data-dir-write <原安装程序绝对路径> <数据目录绝对路径>`）：
  路径必须绝对、可写、不与当前位置互相包含；**当前位置仍含数据库时直接拒绝**（退出码 3），
  因为安装器不做迁移，真实搬迁只能由应用在退出时完成并可回滚。
- 数据目录里放全部数据（数据库、DPAPI 凭据、日志、更新缓存），C 盘只留 `app.json` 指针。
- 保持旧标记位置时也会安全固化到 `app.json`，避免改变程序安装目录后失去数据指针；
  已有正常 `app.json` 的重复选择保持原配置，不触发迁移或清除已有回滚信息。
- 机器级提权助手从交互桌面取得普通用户令牌，通过该令牌的 KnownFolder 查询定位配置，
  提权时在专属锁定线程中以该用户身份验证权限与读写文件，普通用户进程直接使用自身身份；
  不继承管理员的 APPDATA／OMNIPROXY 配置，
  不通过修改进程环境模拟身份。模拟失败不回退为管理员写入；没有普通交互桌面时明确失败。
- 原生向导使用 `--omniproxy-installer-data-dir-read-file <原安装程序绝对路径>`，
  从助手旁固定的独占 UTF-16LE 结果文件读取目录，使用后删除；不经 nsExec 系统代码页传递中文路径。
  普通命令行 `-read` 模式继续返回 UTF-8 标准输出，不能用它替代原生向导的结果协议。
- 静默升级（`/OMNIPROXY_AUTOUPDATE=1`）不显示该页、也不写任何配置；帮助程序在 `.onInit`
  解压一次，供页面调用。

兼容：Beta.33 安装目录里的 `data-dir.txt` 仍被运行时和安装向导兼容读取，
卸载时删除；新安装不再写它。静默更新继续沿用原安装目录，不迁移配置或数据。

编译安全性：向导页只影响安装器产物，`makensis` 编译通过不等于完成真实安装验收。

## Unified commands

| Command            | Purpose                                            |
| ------------------ | -------------------------------------------------- |
| `task setup`       | Install dependencies and tools                     |
| `task dev`         | Development run with HMR                           |
| `task build`       | Production desktop build                           |
| `task package`     | NSIS installer                                     |
| `task test`        | Go + frontend tests                                |
| `task lint`        | go vet + staticcheck + ESLint                      |
| `task format`      | gofmt + Prettier (write)                           |
| `task typecheck`   | `tsc --noEmit` for the frontend                    |
| `task verify`      | format check → lint → typecheck → test → build     |
| `task clean`       | Remove `bin/` and `frontend/dist`                  |

## Runtime profiles

The active profile is selected by the build tag and can be overridden with
`OMNIPROXY_PROFILE` (or `.env`):

| Profile     | Proxy port | Control port | Data directory                  |
| ----------- | ---------- | ------------ | ------------------------------- |
| production  | 3000       | 3890         | `<config dir>/OmniProxy`        |
| development | 3001       | 3891         | `<config dir>/OmniProxy-dev`    |
| test        | 3002       | 3892         | `<config dir>/OmniProxy-test`   |

`<config dir>` is `%APPDATA%` on Windows. Dev and production never share a
data directory. Logs go to `<data dir>/logs/omniproxy.log` (rotating, 5 MiB ×
3 backups).

Windows login autostart is available only in the production profile. Development
and production builds can use the same `bin/omniproxy.exe` output path, so the
development profile must not register it as a startup command. The window-close
choice is stored in the profile's `app.json`; when set to tray, use the tray
menu's Quit action to run normal shutdown and any pending data relocation.

## Build artifacts

| Artifact                          | Produced by    |
| --------------------------------- | -------------- |
| `bin/omniproxy.exe`               | `task build`   |
| `frontend/dist/`                  | frontend build |
| `frontend/bindings/`              | `wails3 generate bindings` (committed) |
| `build/windows/nsis/*installer.exe` | `task package` |

`frontend/bindings/` is generated code but is committed on purpose so that
`tsc`, ESLint and Vitest work on a fresh clone without running the Wails
toolchain. It is regenerated automatically by `task build` / `task dev`; commit
the regenerated output when it changes.

## 原生客户端接入验收

在同一 Windows 用户的正常终端运行 `task dev`，再用实际桌面客户端核对地址、配置选择、模型列表和连接状态。部分工具执行环境对用户配置使用独立视图；出现工具写入成功、客户端仍读旧配置时，应先比较两个运行上下文中的受管字段，再继续排查。

本机 Codex 商店应用的工具及其直接启动的开发进程，对 `Local\Claude-3p` 的访问会落入 `Packages\OpenAI.Codex_2p2nqsd0c76g0\LocalCache\Local`；路径字符串、用户名和未打包进程身份都可能相同。使用文件句柄的 `GetFinalPathNameByHandle` 核对实际位置。发生此情况时，先通过托盘退出该开发版，再由资源管理器或正常桌面终端启动项目的普通 `task dev`，保留原开发数据库。仅通过工具调用 COM ShellExecute 启动仍可能继承虚拟视图；本机已验证资源管理器打开启动器能够进入真实共享目录。不要复制配置到多个包缓存，也不要删除缓存或凭据。

在正常窗口中重新读取、预览、应用后，需从正常桌面上下文核对共享文件，再让实际 Claude 重新加载并核对网关地址、模型和连接状态。接入 Key 请求成功或工具看到文件更新，均不足以替代客户端读取验收。

切换验收构建前退出已有开发版，释放 3001／3891；窗口设置为关闭到托盘时，使用托盘的退出菜单。开发版与生产版的数据目录及端口继续隔离。客户端实机结果、实际代理响应和 `task verify` 结果均写入 Task Record。

## Platform differences

- Windows is the only supported target today.
- The SQLite driver is pure Go (`modernc.org/sqlite`), so builds do not need
  CGO — matching the Wails Windows build defaults (`CGO_ENABLED=0`).
- `build/ios`, `build/android`, `build/darwin` and `build/linux` come from the
  Wails template and are not maintained yet. `go build ./...` intentionally
  skips them: use `go build .` / `go test . ./internal/...` (what the Taskfile
  does) because the iOS template files do not form a buildable main package on
  desktop platforms.
- Line endings are LF everywhere (`.gitattributes`); the repo must stay
  CRLF-free even when developed on Windows.
