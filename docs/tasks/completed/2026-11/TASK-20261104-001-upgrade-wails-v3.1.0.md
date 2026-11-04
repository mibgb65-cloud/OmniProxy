# TASK-20261104-001: 升级 Wails 至 v3.1.0

## Status

Completed

## Created

2026-11-04

## Request

Wails 已发布最新 v3 正式版本，将项目 Wails 版本升级到 `v3.1.0`。

## Goal

将项目后端依赖、前端运行时及构建工具中的 Wails 版本统一到官方 `v3.1.0`，处理必要兼容调整并完成质量验证。

## Scope

- 核实官方发布及升级要求，检查项目当前版本与相关配置。
- 升级直接依赖、必要传递依赖和版本锁定。
- 检查生成绑定、开发构建和相关文档。

## Out of Scope

- 无关业务重构、应用版本发布、生产安装包制作和应用启动。
- 修改或提交已有的非本任务工作区变更。

# Analysis

## Related Modules

- `go.mod` / `go.sum`：当前 Wails `v3.0.0-beta.30`。
- `frontend/package.json` / `frontend/pnpm-lock.yaml`：当前 runtime `3.0.0-beta.30`。
- `.github/workflows/ci.yml` / `release.yml`：固定安装 beta.30 CLI。
- 生成绑定、关于页测试夹具、README、技术栈和构建说明。

## Dependencies

官方 `github.com/wailsapp/wails/v3@v3.1.0` 与 npm `@wailsio/runtime@3.1.0` 均已发布。
官方模块要求 `go 1.26.0`，项目当前已满足；CLI 当前为 beta.30。

## Risks

预发布版本至正式版本可能包含绑定生成、运行时和窗口 API 兼容变化；升级时保留项目现有构建流程，不引入无关迁移。

# Implementation Plan

1. 阅读相关规范和现有实现，确认官方升级要求并记录影响范围。
2. 最小化升级依赖与工具版本，按需调整兼容点和生成文件。
3. 执行针对性验证及 `task verify`，更新任务记录和受影响文档，审查差异。

验证使用 `task --force verify DEV=true BIN_DIR=tmp/wails-v3.1.0-20261104/verify-bin`，
保留完整质量门但构建隔离的开发 EXE，不制作生产安装包、不启动应用。

# Progress

- 已创建任务记录；工作区已有一个与本任务无关的未跟踪历史任务文件，将保持原样。
- 官方 GitHub Release API 确认 `v3.1.0` 非 draft、非 prerelease；npm 确认 `3.1.0`。
- 保留现有 Taskfile，不迁移到 v3.1.0 的实验性 Wake / `wails.yaml`，不修改业务和应用版本。
- Go 模块与前端 runtime 已升级，本机 `wails3 version` 确认为 `v3.1.0`。
- `pnpm typecheck` 与桌面运行信息／更新、日志、账号用量和额度相关的 11 文件／106 项前端测试通过。
- 首轮质量门的格式、lint、类型和 51 个 Go 测试包通过；前端默认并发下 886／890 项通过，3 项触发 5 秒超时，1 项未及时找到按钮。沿用仓库上一轮完整验证的 `VITEST_MAX_WORKERS=4` 重跑，不修改断言、超时或测试范围。
- 4 worker 重跑完整质量门退出码 0：146 文件／890 项前端测试、51 个 Go 测试包、绑定生成和 Windows amd64 开发 EXE 构建全部通过。
- 新 CLI 生成事件常量入口，并将 Windows ICO 的 10 个目录项 planes 字段从 0 规范化为 1；逐尺寸验证 16–256 像素的全部 RGBA 数据完全相同，保留生成结果。
- 提交前完整检查发现新生成的事件入口保留末尾空行，触发 Git `blank-at-eof`。遵循生成代码不手改、现有格式检查已排除生成目录的规则，使用 `.gitattributes` 仅豁免生成目录的末尾空行；其他空白规则和手写代码检查不变。

# Implementation Result

## Changes

- Go 模块、前端 runtime、本机 CLI、CI／Release 安装命令固定为 `3.1.0`。
- `go mod tidy` 更新必要依赖：`x/sys v0.48.0`、`x/crypto v0.57.0`，加入上游所需 `purego v0.11.1`、`x/term v0.46.0`，移除不再使用的旧间接依赖；不批量升级其他业务依赖。
- 同步关于页测试夹具与展示断言、README、技术栈、构建和发布说明。
- 原生窗口、托盘和事件适配保留既有实现；Windows 文件对话框取消错误仍为 `cancelled by user`，无需改写兼容逻辑。
- 使用 `v3.1.0` CLI 重新生成绑定，新增 `frontend/bindings/events/index.ts`（7 个已有业务事件的命名常量）；既有服务和模型绑定无差异。
- 同步新版 CLI 生成的 `build/windows/icon.ico` 目录项元数据，无图标外观变化。
- `.gitattributes` 为 `frontend/bindings/**` 定向设置 `whitespace=-blank-at-eof`，完整保留 Wails 的原始生成输出，避免手工修改或额外格式化导致后续构建反复出现无意义差异。

## Files Changed

- `go.mod`、`go.sum`、前端依赖清单和锁文件。
- CI／Release 的 CLI 安装命令、关于页测试与生成资源。
- README 中英文、TECH_STACK、BUILD、RELEASE、CURRENT 和本任务记录。

## Technical Notes

- [官方 Release](https://github.com/wailsapp/wails/releases/tag/v3.1.0)，源码 `4b454e443f6f11ccb64679c5c3d3f3bf5e6fb440`。
- [官方发布说明](https://v3.wails.io/blog/wails-v3-1-released/)：Wake 为实验性且迁移改变 Taskfile 路由，本次不启用。
- 保留项目现有 `go 1.26.0` 声明；本机验证使用 Go `1.26.2`。上游模块的建议工具链为 `go1.26.10`，本次不变更全局 Go 安装。

# Verification

- PASS：官方 GitHub API、Go 模块下载信息、npm 元数据确认正式发布。
- PASS：`wails3 version`、`go list -m github.com/wailsapp/wails/v3` 均为 `v3.1.0`。
- PASS：`pnpm typecheck`，11 文件／106 项针对性前端测试。
- PASS：`go mod verify`。
- PASS：`actionlint .github/workflows/ci.yml .github/workflows/release.yml`、`git diff --check`、已变更文本的 UTF-8 无 BOM 校验。
- PASS：暂存全部生成文件后再次运行 `git diff --cached --check`；`git check-attr whitespace` 确认末尾空行豁免仅作用于生成绑定，手写代码仍使用默认规则。
- 首轮：`task --force verify DEV=true BIN_DIR=tmp/wails-v3.1.0-20261104/verify-bin` 在前端测试阶段退出 201；`accounts-view`、`mimo`、`minimax` 三项超时，`agent-update-action` 一项异步按钮等待失败。保留日志 `tmp/wails-v3.1.0-20261104/verify.log`。
- PASS（退出码 0）：仅对子进程设置 `VITEST_MAX_WORKERS=4` 后重新运行同一完整质量门；格式、Go vet／staticcheck／ESLint、TypeScript、51 个 Go 测试包、146 文件／890 项前端测试、绑定生成和 Windows amd64 开发构建全部通过。日志 `tmp/wails-v3.1.0-20261104/verify-workers4.log`。
- PASS：构建产物 `tmp/wails-v3.1.0-20261104/verify-bin/omniproxy.exe` 的 `go version -m` 含 Wails `v3.1.0`、Go `1.26.2`、`CGO_ENABLED=0`、`GOOS=windows`、`GOARCH=amd64`，无 `production` 标签。
- PASS：复用 `scripts/test-windows-metadata.ps1` 的原生版本读取器，仅核对上述开发 EXE；应用版本仍为 `2.2.2`、数字资源 `2.2.2.0`，产品与公司均为 OmniProxy。未把开发 EXE 当作安装器验收。
- PASS：ICO 的 10 种尺寸全部保留且像素逐字节相同；`go mod tidy -diff` 无差异，构建后 `go mod verify` 通过。
- 非阻断提示：保留既有 react-refresh 两条警告、jsdom `scrollTo` 提示、Vite 大 chunk 提示及 Taskfile 平台探测的 `uname`／`tail` 缺失提示；新版 runtime 在无原生窗口的测试环境提示 browser preview，不代表桌面启动失败。
- 未执行：原生窗口／托盘人工验收、真实账号或客户端调用、应用启动、生产安装器、覆盖安装和发布。

# Documentation

- 已更新：README 中英文、TECH_STACK、BUILD、RELEASE、CURRENT。
- PRD、架构边界和 ADR 无变化，无需调整。

# Deviations

首轮默认前端测试并发下出现异步等待失败，验证改为仓库此前使用的 4 worker 后全部通过；不修改业务或测试实现，不更改全局环境或 CI 配置的测试策略。

# Git

## Commit

```text
b74dd234f2380b2b4680308e4fe3f5168b2e7f53 build(wails): 升级至 v3.1.0 正式版
```

本地提交，未推送远端。原有无关未跟踪历史任务文件未修改、未纳入提交。

# Result

已完成 Wails `v3.1.0` 升级、完整质量验证及本地实现提交，无需业务 API 兼容改写，保留现有构建系统与应用版本。运行中的应用和既有非本任务文件保持原样；原生窗口／托盘人工验收仍未执行，Go SDK 未扩展升级至上游建议的 `1.26.10`。

# Completed

2026-11-04（Asia/Singapore）
