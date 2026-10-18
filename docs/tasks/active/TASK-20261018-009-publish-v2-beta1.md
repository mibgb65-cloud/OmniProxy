# TASK-20261018-009: 发布首个 v2 Beta

## Status

In Progress

## Created

2026-10-18（Asia/Singapore）

## Request

用户确认「可以，你做吧」，授权将已验收的 v2 main 推送到 `mibgb65-cloud/OmniProxy`，创建 `v2.0.0-beta.1` 标签，通过自动流程公开发布首个预发布版本。

## Goal

GitHub Pre-release 包含 CI 构建的 Windows amd64 安装器及匹配 SHA256，能通过实际下载校验；保留旧版 master 和 v1.3.1 Latest 状态。

## Scope

- 配置当前仓库 origin，仅正常推送新 main／版本标签，不强推或改写历史。
- 跟进 main CI、标签 Release 工作流；必要时修复首次远端构建的问题，不绕过质量门。
- 核对版本、发布说明、两项资产和下载校验、Beta／stable 查询及旧版状态。
- 记录真实远端链接、提交、验证和边界并归档；五份既有未跟踪任务记录不纳入提交。

## Analysis

发布候选 `1950b22`，应用版本 `2.0.0-beta.1`。安装生命周期及最终代码质量门已由 TASK-20261018-008 验收。当前本地没有 remote；先前只读核对远端默认分支为 master，没有 main 或 v2 标签，当前正式 Latest 为 v1.3.1，账号具备 push 权限。本次用户授权明确包含实际远端发布，扩展了上个任务「不发布远端」的范围。

再次核对确认：master 为 `56448d20ccd797bc32aae1c02a5e2f2030b8b6c8`，stable Release ID 为 `392117295`（v1.3.1），main 与目标标签不存在。仅推送当前仓库 main，未发现其提交历史包含 `.env`、PFX／P12 或常见 SSH 私钥文件；不纳入未跟踪的既有用户记录。

## Implementation Plan

1. 再核对远端引用、权限、候选和资产；提交发布准备记录。
2. 推送 main，跟进实际 CI，成功后创建并正常推送 v2.0.0-beta.1 标签。
3. 跟进 Release 全流程，下载最终资产并核对 SHA256／原生版本，检查预发布及 Latest／master 保留。
4. 更新任务和状态文档，提交归档并同步 main。

## Root Cause

首次远端 CI `39108569384` 在 lint 的 go vet 失败：干净检出没有被忽略的 frontend/dist，而 main 的 go:embed 需要该目录。本地已有构建产物掩盖了顺序问题。Release 同样从干净环境执行 task verify，必须在格式检查前安装前端依赖、Go 分析／测试前准备真实前端资源。

## Fix

在共用 Task 入口补齐依赖和嵌入资源准备，CI 使用同一个 task verify；不提交编译产物，不使用空白占位文件或跳过任何检查。用无 dist／node_modules 的隔离源码完整运行发布脚本，再推送 main 重跑远端 CI。

第二次 CI `39109546758` 的 agentconfig／Codex 凭据／Claude 启动三组失败已在本机使用真实 Win32 8.3 临时目录别名复现，属于同一原因：测试夹具路径经解析后与短路径拼写不同，触发现有严格路径策略。保留应用路径安全规则，CI 与 Release 的验证步骤统一使用 GitHub runner.temp 作为 TEMP／TMP，让测试夹具位于规范目录；不跳过失败测试、不扩大应用可接受的路径范围。

Release 原提交重跑后仍只有同一项提供商提示测试超过 5 秒。实际提示保持 4 秒，测试将渲染／交互和真实等待挤在同一期限内，已证实需要稳定化修复。改为虚拟时钟并断言 3999ms 仍展示、4000ms 收起，保留全部业务断言和原测试期限。标签已经共享，保持原标签不可变；手动发布恢复时只从工作流提交补入这一份修复后的测试文件，校验工作区除该测试外与标签完全一致，再执行原标签的完整发布脚本。应用源码和发布说明均使用原标签，不修改版本或运行时行为。

## Implementation Result

已配置 origin 并正常推送独立 main。远端 CI 完整通过后，创建并推送注释标签 v2.0.0-beta.1，固定到 `18cd42f399ba05b3e4da8208b81f4049d03c2675`；没有改写旧分支或移动标签。

Release 工作流 `39111414581` 两次执行均被同一项提供商提示交互测试的 5 秒超时拦住（其他 574 项前端测试通过）。改用虚拟时钟验证精确 4 秒边界，保留全部业务断言和原测试期限。手动恢复发布只补入这份确定性测试，先校验应用源码与既有标签无差异，保持原标签及发布应用代码不变；公开发布／下载验证待完成。

## Verification

- 无 dist／node_modules 的隔离源码实际执行 `scripts/build-release.ps1 -Version v2.0.0-beta.1` 成功，退出码 0：完整 task verify、575 项前端测试／Go 测试、Windows production 构建、NSIS、原生版本资源及两项发布资产 SHA256 均通过。日志位于忽略目录 `tmp/release-beta1-20261018/clean-release-build.log`。
- CI／Release 的 actionlint、git diff --check 通过；远端 CI 修复后重跑及公开发布验证待完成。
- 通过真实 Win32 8.3 临时目录本机复现第二次 CI 的三组失败；改用规范 TEMP／TMP 后，agentconfig、Codex 凭据及 Wails 三个包完整测试（-count=1）全部通过。没有修改测试断言或应用安全策略。
- main 远端完整 CI `39110496547` 成功，源码 `18cd42f`，生产二进制 artifact 已下载留存；发布工作流每次仍独立执行完整质量门。
- 确定性计时测试修复后，提供商文件 8 项测试、完整 task verify（575 项前端测试／全部 Go／Windows 构建）通过，日志 `tmp/release-beta1-20261018/final-verify.log`。真实 Git 隔离夹具验证恢复守卫：允许测试文件变化、拒绝 main.go 应用变化。actionlint／差异检查通过。

## Git

- `8ba415e923d6de01ceecd380ed146634391188b4`：首发准备记录，首次推送 main。
- `14edf51e48b50af4fa0b6ab08d0ccd94b16b270d`：干净检出嵌入资源准备及统一质量门。
- `18cd42f399ba05b3e4da8208b81f4049d03c2675`：Windows 测试规范临时目录。
- `v2.0.0-beta.1` 注释标签对象 `a3b737552a7a83e35a7bfaefd0073eb98045b7a2`，指向 `18cd42f`。

## Result

待完成。
