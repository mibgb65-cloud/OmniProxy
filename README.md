# OmniProxy

OmniProxy is a local-first desktop application: an AI API gateway and
multi-account management tool. It proxies requests to multiple AI providers
(OpenAI, Anthropic, Google, DeepSeek, Kimi, MiMo, OpenAI-compatible), manages
multiple accounts per provider, routes models, schedules accounts, monitors
quota and usage, and manages the local configuration of AI clients such as
Codex, Claude Code and Gemini CLI.

Architecture in one line: **Go Core + Wails 3 Desktop Shell + React Frontend**.
The core is independent of the desktop shell and can run headless behind a
local HTTP Control API.

## Requirements

- Go 1.26+ (pinned via `go.mod`)
- Node.js 22 (pinned via `.nvmrc` and `frontend/package.json` engines)
- pnpm 10+ (the only supported package manager)
- [Task](https://taskfile.dev) (`go install github.com/go-task/task/v3/cmd/task@latest`)
- Wails 3 CLI (`go install github.com/wailsapp/wails/v3/cmd/wails3@v3.0.0-beta.30`)
- Windows 10/11 with the WebView2 runtime (other platforms are future work)

## Development

One-time setup (installs frontend dependencies, Go modules and lint tools):

```bash
task setup
```

Start the app in development mode (Go rebuild + Vite HMR + live reload):

```bash
task dev
```

Optional: copy `.env.example` to `.env` to override the runtime profile, log
level or ports. `.env` is gitignored.

## Test

```bash
task test
```

Runs the Go test suite (`go test`) and the frontend test suite (Vitest).

## Build

```bash
task build      # production desktop build -> bin/omniproxy.exe
task package    # NSIS installer -> build/windows/nsis/
task verify     # format check + lint + typecheck + test + build
```

See `docs/development/BUILD.md` for details, runtime profiles and artifacts.

## Documentation

See [`docs/README.md`](docs/README.md).

Key documents:

- `docs/product/PRD.md` — what the product is (and is not)
- `docs/architecture/OVERVIEW.md` — components, data flow, boundaries
- `docs/architecture/TECH_STACK.md` — the technology baseline
- `docs/development/` — coding, Git, testing and build rules
- `docs/plans/CURRENT.md` — current development status

## AI Development

AI agents must read [`AGENTS.md`](AGENTS.md) before starting any task.

## License

TBD — no license has been selected yet.
