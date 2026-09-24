# Coding Rules

## General Principles

- One module carries one clear responsibility.
- Prefer readability over cleverness. Explicit over magic.
- No premature abstraction; no design for requirements that do not exist.
- If the project already has an established style, inherit it.
- New dependencies need a current, concrete use. Do not add a framework for
  code the standard library can handle.

## Project Structure

Go backend (`go.mod` module `omniproxy`):

```text
main.go                          Wails entry point (root package, per Wails convention)
migrations/                      SQL migrations (append-only)
internal/
  platform/profile/              Runtime profile (production/development/test)
  infrastructure/database/       SQLite open + migration runner
  infrastructure/logger/         slog setup, Redactor, rotating file, UI buffer
  transport/wails/               Wails app service and shell adapter
  transport/httpapi/             Local Control API (chi)
  version/                       Application version constant
cmd/, internal/application/, internal/core/, internal/provider/,
internal/integration/, internal/platform/{windows,darwin,linux}/
                                 Created when the features that need them land
```

Notes:

- `main.go` stays at the repository root because the Wails v3 build tooling
  builds the root main package. All logic beyond bootstrap lives in
  `internal/…`.
- Do not create empty directories "for later". Add a package when its first
  real code arrives.

Frontend (`frontend/`):

```text
src/app/          Router, providers, route modules (routes/, routeTree.gen.ts)
src/pages/        Page-level compositions (created with real pages)
src/features/     Business features (created with real features)
src/components/ui/    shadcn/ui components
src/components/common/  Cross-feature components
src/hooks/        React hooks
src/services/     Typed wrappers around the generated Wails bindings
src/stores/       Zustand UI state
src/lib/          Utilities
src/types/        Frontend-only TypeScript types
```

## File Organization

- Go: one concern per file; package name matches the directory name.
- Frontend: co-locate tests with the code they test
  (`foo.ts` → `foo.test.ts`).
- Generated code (`frontend/bindings/`, `src/app/routeTree.gen.ts`) is never
  edited by hand and is exempt from the size rules below.

### File size limits

From the technology baseline (技术架构选择.md §32):

- A single hand-written file should stay **≤ 200 lines**.
- Beyond roughly 200 lines, actively evaluate splitting the file.
- The following are explicitly forbidden:
  - 2000+ line React components
  - "God" Service / Store / Provider / Handler files

These are review criteria, not compiler errors: when a file must exceed the
guideline (e.g. a table-driven test file with many cases), say why in the PR
description.

## Naming

- Go: exported identifiers carry doc comments; unexported names are short but
  not cryptic. Errors read as `package: what failed`.
- TypeScript: `PascalCase` for components/types, `camelCase` for functions and
  variables, `SCREAMING_SNAKE_CASE` only for true constants.
- File names: `kebab-case.ts(x)` in the frontend, snake-case in Go where the
  ecosystem uses it (`greet_service.go` style for multi-word files).

## Functions

- Keep functions small and single-purpose. If a function needs a paragraph of
  explanation, split it.
- Guidance (not a hard limit): evaluate splitting functions beyond roughly 50
  lines; a function should fit on one screen.
- Avoid more than ~3 levels of nesting; prefer early returns.
- No magic numbers or strings: name them or move them to a constant block.

## Components

- Business logic never lives inside React components; it belongs in services,
  hooks, or (for data fetching) TanStack Query.
- Components render; they do not compute business rules.
- Server state goes through TanStack Query; only genuine UI state goes into
  Zustand. Never mix the two.
- User-facing strings go through the i18n layer (`src/i18n` + the
  `useTranslation` hook). Do not hardcode display text; add the key to
  `locales/zh.ts` (source of truth) and `locales/en.ts` in the same change.

## Error Handling

- Errors must keep enough context to locate the failure (`fmt.Errorf("db:
  open %s: %w", path, err)`).
- No empty `catch`, no ignored error returns (`_ = f.Close()` only where the
  error is genuinely irrelevant, e.g. cleanup paths).
- Never swallow an error to make a test pass.

## Logging

- All logging goes through the shared `log/slog` logger from
  `internal/infrastructure/logger`.
- Structured fields, not string concatenation:
  `log.Info("request completed", "provider", p, "status", 200)`.
- No `fmt.Println` / `console.log` for diagnostics.
- Debug detail and user-visible errors are separate concerns.
- Logs must never contain credentials; the Redactor is the safety net, not an
  excuse to log secrets.

## Dependencies

Before adding a dependency, check:

1. Does the project already have this capability?
2. Can the standard library do it?
3. Does the dependency clearly reduce complexity?
4. Is it actively maintained? Does it pull in heavy transitive dependencies?
5. Does it affect binary size or build time?
6. Is the license acceptable?

Adding a dependency is part of the change that first needs it — never
"just in case".

## Comments

- Comments explain **why**, not what.
- Doc comments on every exported Go identifier.
- `TODO(name): …` with an owner when a deferral is intentional.

## Testing

See `docs/development/TESTING.md`. In short: test external behavior, core
logic, boundaries and regressions — not implementation details.

## Refactoring

- Refactor in separate commits from behavior changes.
- Keep the public API stable; if it must change, evaluate compatibility first.

## Documentation

- Docs describe the current state. Update them in the same change as the code.
- See `AGENTS.md` §4 for the required flow.
