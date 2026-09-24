# Build

## Prerequisites

- Go 1.26+, Node.js 22, pnpm, Task, Wails 3 CLI (see root `README.md`).
- Windows: the WebView2 runtime (preinstalled on Windows 11 and up-to-date
  Windows 10).

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
3. The app binary runs and loads the UI from the dev server.

Frontend-only work can also use `cd frontend && pnpm dev`, but the app itself
needs `task dev`.

## Production build

```bash
task build
```

- Builds the frontend for production (`frontend/dist`, embedded into the
  binary via `//go:embed all:frontend/dist`).
- Builds `bin/omniproxy.exe` with the `production` build tag (which selects the
  production runtime profile).
- Generates Windows resource data (icon, manifest, version info) from
  `build/windows/`.

## Packaging

```bash
task package
```

Creates an NSIS installer (`build/windows/nsis/`) using the binary produced by
`task build`. Code signing (SignPath) is configured separately and is not part
of the default build.

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
