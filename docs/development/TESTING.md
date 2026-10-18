# Testing

## Frameworks

| Layer   | Framework                      | Location                        |
| ------- | ------------------------------ | ------------------------------- |
| Go      | standard `testing` + `httptest`| co-located `*_test.go`          |
| Frontend| Vitest + Testing Library       | co-located `*.test.ts(x)`       |
| E2E     | Playwright (planned)           | `frontend/e2e/` when introduced |

## Where tests live

- Go tests live next to the code: `internal/infrastructure/logger/logger_test.go`.
- Frontend tests live next to the code: `src/lib/utils.test.ts`.
- There is no separate top-level `tests/` directory; the framework conventions
  above are the project convention.

## Running tests

```bash
task test            # Go + frontend
go test . ./internal/...   # Go only
cd frontend && pnpm test   # frontend only
```

干净检出直接执行包含根包的 Go 测试前，先运行 `task prepare:assets`，因为 main 的 go:embed 需要真实前端资源。`task test`／`task lint` 已自动处理；仅测试 `./internal/...` 无需桌面资源。

Windows 的配置及凭据测试使用规范绝对路径作为夹具，8.3 短路径别名会触发现有安全判断。GitHub CI／Release 将验证步骤的 TEMP／TMP 指向 `runner.temp`，避免默认 `RUNNER~1` 别名；应用路径安全规则保持一致。

`go test -race` is not used on Windows because the race detector requires CGO
there; CI runs the same commands as `task test`.

## What must be tested

Every new feature ships with tests for:

- its external behavior (bindings/HTTP contract),
- core business logic,
- boundary conditions and error paths,
- a regression test for every bug fix.

Priority order: external behavior > core business logic > boundaries >
regressions.

## What must not be done

- Deleting reasonable tests to make CI pass.
- Skipping a failing test without an explanation in the same commit.
- Turning a real failure into an always-passing mock.
- Writing large volumes of brittle tests for implementation details.

## Frontend specifics

- Component tests render with Testing Library and assert user-visible
  behavior.
- Hooks are tested with `renderHook` inside the providers they need
  (e.g. `QueryClientProvider`).
- Wails bindings are mocked at the `src/services/omniproxy` boundary; tests
  never talk to Go.
- The Vitest setup (`src/test/setup.ts`) registers jest-dom matchers and
  Testing Library cleanup.

## Go specifics

- Use table-driven tests where they improve clarity.
- Use `httptest` for HTTP handlers and the Control API.
- Database tests run against a temporary SQLite file (`t.TempDir()`), never a
  real user data directory.
- Migration tests use `testing/fstest.MapFS` with small synthetic migrations.

## E2E (planned)

Playwright will cover the core user flows (start app → configure provider →
complete a request through the gateway). It is added together with the first
complete user flow, not before.
