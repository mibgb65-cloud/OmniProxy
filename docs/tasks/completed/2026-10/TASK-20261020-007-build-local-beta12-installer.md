# TASK-20261020-007: 构建 Beta.12 本地生产安装包

## Status

Completed

## Created

2026-10-20

## Request

用户要求修复完成后构建新的 Beta 生产版本，供其自行本地更新。

## Goal / Scope

- 将版本同步为 `2.0.0-beta.12`，编写准确的随包发行说明。
- 包含 Beta.11 之后已提交的额度快照持久化、River 模型／分页优化、
  订阅状态刷新误报修复。
- 冻结本地构建源，以既有发布脚本隔离执行完整质量门、production 构建和 NSIS 打包。
- 校验 Windows 原生版本、来源、安装器与 SHA256，交付独立 Beta.12 资产目录。

## Out of Scope

不自动安装、停止或替换生产程序，不修改真实账号／凭据／客户端配置；
不创建发布标签、推送或公开发布；不纳入无关未跟踪文件。

---

# Analysis

当前完整版本及最近本地安装包为 Beta.11。构建必须使用生产标签和既有安装身份，
不能将开发运行包作为更新安装器交付。Beta.11 之后的实际变更需逐项写入发行说明。

已确认包含额度留存 `f5de7ca`、River／分页 `3dd7426`、订阅刷新 `ac9d5a7`，
沿用原有 Windows 数字版本 `2.0.0.0`、NSIS 版本 `2.0.0` 与机器级产品身份。
NSIS 可用，目标 Beta.12 资产目录不存在或为空；已有资产及已安装程序共 15 个文件、
1 个生产进程的摘要／启动时刻基线已保存在本任务忽略目录。

---

# Implementation Plan

1. 核对发布规范、脚本、版本读取测试和旧资产／生产程序基线。
2. 更新版本与发行说明，运行定向检查，提交并冻结构建源。
3. 只导出冻结提交的受跟踪文件，隔离运行发布脚本的完整质量门及打包。
4. 独立核对生产标记、Windows 版本、SHA256 和来源一致性，确认旧资产不变。
5. 更新 CURRENT 和实际验证记录，归档任务并交付安装包。

---

# Progress

- 已创建任务记录，尚未修改版本、配置或发布文件。
- 已同步 Go／前端／Windows 完整版本为 Beta.12，新增随包发行说明和三种版本读取测试。
- 旧资产与生产程序只读保存基线，不读取真实配置或凭据。
- 版本映射脚本与发行说明定向 Prettier／29 项回归已通过，准备冻结构建源。
- 构建源已冻结为 `adb1e5f7f731c520688f6f5cfa2bf08a1ad2b359`，
  仅导出该提交的受跟踪文件到 `tmp/local-beta12-20261020/source/`。
  已确认三项功能提交均为该构建源的祖先。
- 已启动原有 `build-release.ps1`，依次执行强制完整门禁、production 构建及 NSIS 打包。
- 初次来源核对 1282 项源码／测试／依赖／发行说明与冻结 Git blob 一致；
  Windows 资源 JSON 仅有 CRLF／LF 差异，不影响内容。
- 发布脚本退出码 0，完整门禁、Windows production 构建、NSIS 与脚本内资产校验通过。
- 构建后独立来源、原生版本、生产标签、签名、SHA256 和保护基线校验全部通过。

---

# Implementation Result

- Go、前端及 Windows 完整版本已同步为 Beta.12；Windows／NSIS 数字基础版本
  和机器级安装产品身份保持原样。
- 新增 Beta.12 发行说明，覆盖刷新误报修复、额度持久留存、River 与分页优化，
  明确未签名和真实验收边界；保留所有历史版本发行说明。
- 冻结构建源与版本定向验证已提交；从该提交隔离运行既有发布流程，
  完成完整质量门、production 构建、NSIS 打包及独立核对。
- 安装器与同名 SHA256 已导出到独立 Beta.12 资产目录；没有覆盖旧包，
  没有创建远端标签、推送、公开发布或运行安装器。
- 旧资产、已安装程序及生产进程只读记录基线，没有自动安装或停止生产操作。

---

# Verification

- PASS：`scripts/set-release-version.ps1 -Version v2.0.0-beta.12`。
- PASS：前端包与发行说明测试文件的定向 Prettier 检查。
- PASS：`pnpm exec vitest run src/features/about/current-release.test.ts`，29 项测试通过。
- PASS：从冻结源码执行
  `scripts/build-release.ps1 -Version v2.0.0-beta.12 -OutputDirectory <workspace>/release-assets/v2.0.0-beta.12`，退出码 0。
- PASS：脚本内 `task --force verify`，格式、Go vet／staticcheck、ESLint、
  TypeScript、全部 Go 测试、113 个文件／669 项前端测试、
  Wails 绑定、前端生产构建及 Windows amd64 production 构建通过。
- PASS：脚本内 `task --force package`，NSIS 以 UTF-8 和既有产品身份生成安装器。
- PASS：脚本内及独立 Windows 原生版本 API 校验，
  应用 ProductVersion 为 `2.0.0-beta.12`，安装器为 `2.0.0`，
  两者数字资源均为 `2.0.0.0`，ProductName／CompanyName 均为 OmniProxy。
- PASS：`go version -m` 确认 Go 1.26.2、production 标签、
  windows／amd64、CGO_ENABLED=0，不含 updateqa。
- PASS：构建前后共 1282 项源码／测试／依赖／发行说明与冻结 Git blob 一致；
  Windows 资源 JSON 仅为 CRLF／LF 差异。三项功能提交均在构建源中。
- PASS：资产目录仅含安装器与同名 SHA256 文件，独立摘要一致；
  签名状态为 NotSigned。
- PASS：15 项旧资产／已安装程序文件大小和摘要未变；
  生产进程 PID 18320 的路径与 UTC 启动 ticks 未变，未重启或替换。
- PASS：`git diff --check`，本次 7 个源提交文件为 UTF-8 无 BOM。
- 非阻断既有提示：Windows Task 环境缺少 uname／tail 等 Unix 命令、
  jsdom scrollTo 未实现、Vite 大 chunk 和 Tailwind 插件耗时。
  未隐藏、跳过检查或删除测试。

未执行：真实覆盖安装、管理员确认、安装后生产启动、真实 OAuth／网络恢复、
原生交互和安装版重启验收；构建和隔离回归不等同于这些场景已通过。

---

# Artifact

- 安装器：
  `E:/go/new/wails3-v/OmniProxy/release-assets/v2.0.0-beta.12/OmniProxy-Setup-v2.0.0-beta.12-windows-amd64.exe`
- SHA256 文件：安装器同名追加 `.sha256`。
- 大小：19,434,522 bytes。
- SHA256：`df50c4bed9b0f80960c5edc428b083c0181399fef166782273169400001af8ef`。
- 签名状态：NotSigned。
- 构建日志、冻结来源、1282 项来源核对和独立产物／保护基线验证摘要位于
  忽略的 `tmp/local-beta12-20261020/`，不提交产物或本地配置。

---

# Git

`adb1e5f7f731c520688f6f5cfa2bf08a1ad2b359` —
`chore(release): 准备 Beta.12 本地生产安装包`。

仅本地提交，不推送、不打发布标签；产物验证和任务归档使用后续文档提交。

---

# Deviations

暂无。

---

# Result

Beta.12 Windows amd64 production 本地安装包与 SHA256 已生成并验证，
包含订阅刷新误报、额度快照留存、River 型号和请求分页修复。
用户可自行覆盖更新；旧资产和正在运行的生产程序保持原样。
未签名、未公开发布、未自动安装。

# Completed

2026-10-20（Asia/Singapore）
