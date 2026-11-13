# TASK-20261113-020: 百度搭子 DuMate 个人版原生接入

## Status

Completed

## Created

2026-11-13 15:59（Asia/Singapore）

## Request

用户希望将 `E:\githubresource\magpie-dumate` 的逻辑接入 OmniProxy，先评估实现方式。

用户随后明确限定：**「你先做 task 记录，不改代码」**。
本轮只建立任务记录，不修改业务代码、配置、依赖或数据库，不构建、不启动、不安装插件、
不读取真实账号凭据，也不执行登录、签到或消耗积分的请求。后续实现等待用户明确指示。

## Goal

在不依赖 Magpie、Node/Bun 插件宿主或独立桥接服务的前提下，以 Go 原生适配接入
DuMate 个人版账号：受保护地管理登录态，通过现有 OmniProxy 网关调用模型，并提供
积分与签到能力。具体交付范围和默认行为在开始实现前确认。

## Background

参考项目是独立的 Magpie / OpenCode Provider 插件，并非百度千帆 Token Plan 或按量 API。
已只读检查本地 `index.mjs`、`README.md` 和 `LICENSE`；以下协议是参考实现的行为，
**尚未使用真实账号验证其当前可用性**。

- 参考项目标识：`dumate`，展示名称「百度搭子 DuMate」。
- 登录来源：只读复用搭子桌面端登录文件，或用户手动导入浏览器请求 Cookie。
- 凭据链路：Cookie → 百度控制台换取 gateway token → Bearer 调用模型网关。
- 参考项目为 MIT 许可；实际移植代码时保留适用的版权与许可声明。

## Scope

拟纳入后续实现，尚未执行：

- 独立 DuMate 个人版 Provider 及账号类型，不与千帆 API Key 混用。
- 用户主动触发的本机登录态导入、手动 Cookie 导入及失效提示。
- gateway token 的有界内存缓存、并发刷新合并和失效处理。
- Chat Completions 普通与 SSE 转发，以及工具调用、推理用量兼容性验证。
- 模型目录、路由、账号选择及受管客户端目录接入。
- 积分余额、已用／总量、到期信息及签到状态的公开 DTO 与界面展示。
- 手动签到；自动签到作为独立、默认关闭的可选能力，启用方式待确认。
- 必要的脱敏、安全边界、回归测试和产品／架构文档。

## Out of Scope

- 本轮任何业务实现、构建、启动、安装、提交或真实账号操作。
- DuMate 企业版、百度千帆 API Key／Token Plan 接入。
- 嵌入或运行 `index.mjs`，引入通用 JavaScript 插件系统。
- 自动化破解验证码、绕过登录验证，或读取与当前操作无关的本机账号。
- 把智能路由宣传成可指定的 DeepSeek／GLM 等底层模型。
- 无关重构，或改写现有 OpenAI／Google 订阅实现。

## Existing Behavior

- OmniProxy 已有 Provider 目录、账号管理、DPAPI 凭据存储、网关、模型目录及余额查询。
- `internal/provider` 尚无 DuMate 适配目录；本轮检索未发现 DuMate 业务实现。
- 现有余额服务主要按 API Key 和已支持的 Provider 分派，不能直接承载 DuMate Cookie 账号。
- 订阅账号存在 Provider 及凭据格式校验；不能把 Cookie 假装成 API Key 以绕过现有约束。
- 工作区已有无关未跟踪文件
  `docs/tasks/completed/2026-10/TASK-20261029-010-subq-provider.md`，保留不动。

## Expected Behavior

- 用户主动添加 DuMate 账号后，可通过现有本地网关使用获准暴露的模型别名。
- 调用消耗 DuMate 个人积分，登录态和 token 不进入 SQLite 明文、前端返回值或日志。
- 积分与签到信息区分实时结果、历史快照和查询失败，不把失败展示为零余额。
- Cookie 失效要求重新导入／登录；仅 gateway token 失效不直接等同于账号登录失效。
- 新 Provider 不改变其他 Provider 的认证、路由和重试语义。

---

# Analysis

## Confirmed Reference Behavior

| 能力 | 本地参考实现 |
| --- | --- |
| 桌面登录态 | 默认读取 `%APPDATA%\qianfan-desktop-app\auth.json` 和同目录 `.cookie-key`，支持 `DUMATE_USER_DATA_DIR` 覆盖 |
| Cookie 解密 | AES-256-GCM；32 字节密钥，密文布局为 `IV(12) + Tag(16) + Ciphertext` |
| 登录校验 | `GET https://console.bce.baidu.com/api/dumate/user/info` |
| 换取 token | `GET /api/dumate/app/token/get`，读取 `token`／`expired_at`，并合并并发请求 |
| 模型调用 | `POST https://dumate-svc.baidu.com/gateway/apis/v1/chat/completions`，使用 Bearer token |
| 模型目录 | 固定回退别名 `model-text`、`dm-auto-model/text.L0`；参考实现说明上游无 `/models` |
| 积分总览 | `GET /api/dumate/points/quota_overview`，包含时区、客户端及忽略签到奖励参数 |
| 签到状态 | `GET /api/dumate/points/loginBonusInfo` |
| 领取签到 | `POST /api/dumate/points/loginBonus`，需要域名过滤 Cookie、CSRF 及来源头 |

控制台相对路径均位于 `https://console.bce.baidu.com`。
参考实现对 HTML 登录页、接口业务码和 HTTP 错误有分别处理；
签到先查是否已领取，提交后再核对到账，不应仅凭 POST 成功就显示已领取。

## Proposed Approach

采用**原生 Go Provider + 独立账号应用服务**，复用已有网关、协议转换和 DPAPI。
不将整个插件复制进 OmniProxy，也不为一个 Provider 引入插件运行时。

拟定调用链：

```text
本地 API 客户端
  → OmniProxy 现有网关／路由／账号选择
  → 受控 DuMate 凭据回调
  → 按账号与凭据版本获取／刷新 gateway token
  → DuMate Chat Completions 网关
  → 现有响应／SSE／请求用量链路
```

桌面登录文件解析、控制台 HTTP、模型传输和签到职责分开；Core 不依赖 Wails／React。

## Related Modules

以下为预计接入点，不表示已完成全部影响分析：

- 新增 `internal/provider/dumate/`：固定端点、控制台协议、模型请求、积分及签到 DTO。
- `internal/infrastructure/credential/`：沿用现有受保护凭据存储。
- 拟新增基础设施适配：有界、只读解析桌面登录文件；具体目录待实现前确定。
- `internal/application/account/`：账号创建、版本／身份校验和凭据生命周期。
- 拟新增 DuMate 应用服务：导入编排、token 刷新、积分查询及签到。
- `internal/application/provider/`、`modelcatalog/`、`routing/`、`gateway/`：
  Provider 注册、模型能力与转发接线。
- `internal/application/balance/`：评估复用查询入口或提供独立积分查询入口，
  不将积分塞进货币余额字段。
- `internal/transport/wails/` 与前端账号／用量相关模块：仅暴露脱敏结果与受控操作。
- `migrations/`：是否需要新增 Provider、凭据类型或快照字段，待正式影响分析后确定。

## Dependencies

- 优先 Go 标准库 `net/http`、`crypto/aes`、`crypto/cipher`、`encoding/json`。
- 复用现有凭据存储、代理设置、HTTP／SSE、查询与前端组件，不计划新增大型依赖。
- 保留 MIT 来源声明；移植范围及声明位置在实施时记录。
- 涉及账号安全边界或凭据生命周期的长期决策时，再评估是否需要 ADR。

## Risks and Required Safeguards

- **凭据泄漏**：Cookie 等同账号凭据；只通过受控回调使用，持久化交给 DPAPI。
  对控制台错误、URL、请求头和异常做固定错误码映射与集中脱敏，不回显原始敏感数据。
- **输入与外发边界**：限制文件／JSON／Cookie 大小、结构及字符，拒绝头注入。
  只访问固定 HTTPS 主机与路径；控制重定向，禁止携带凭据跳转到未知域名。
- **身份错绑**：参考实现部分身份字段取首个 profile，不能直接照搬。
  导入时核对实际远端身份；若桌面端切换到另一账号，不得静默覆盖已有账号。
- **刷新与多账号**：缓存按账号与凭据版本隔离、设置上限并合并刷新，
  注销／更新后失效；失败不无限重试，不在开始输出 SSE 后自动重放请求。
- **协议差异**：上游仅确认 Chat 路径；Responses／Messages 拟经现有转换链路接入，
  必须逐项验证，不宣称原生支持。取消、断流、工具参数与 reasoning 用量需要回归测试。
- **模型与额度语义**：回退表中的上下文／输出容量不是已验证契约；
  不猜测底层模型。到期积分不能直接把积分包原始总额当作确定的剩余可用额。
- **真实副作用**：模型测试消耗积分，签到会变更账号状态；真实验证必须获用户授权。
- **上游变动与条款**：非官方接口可能变更，使用受百度账号及服务条款约束，
  不能承诺长期可用或账号无风险。

## Needs Confirmation

- 首版是否包含积分界面与手动签到，还是先完成模型调用闭环。
- 是否需要自动签到；建议默认关闭，仅在应用运行且用户启用时执行，不建系统定时任务。
- 桌面登录导入后是否跟随同一远端账号的重新登录；建议先快照导入、失效后显式重新导入。
- 模型公开命名、嵌套模型 ID 的路由兼容性，以及能确认的容量与多模态边界。

---

## This Round: Scope Decisions

用户要求按本记录开始开发。确认不了的项按记录中的建议取值，并明确写入实现边界：

| 待确认项 | 本轮取值 | 依据 |
| --- | --- | --- |
| 首版是否包含积分与手动签到 | 包含积分展示与手动签到 | Scope 已列入；两者都只读公开 DTO，不改变转发语义 |
| 自动签到 | **不实现**，等用户确认启用方式 | 记录建议默认关闭；不建系统定时任务，不擅自加入后台循环 |
| 桌面登录导入 | 快照导入；失效后显式重新导入；不跟随桌面端切换账号 | 记录建议；按远端身份核对，不静默覆盖其他账号 |
| 模型命名 | 只公布参考实现的两条回退别名 | 上游无 `/models`，容量未经真机验证 |
| 协议 | Chat 原生；Responses／Messages 经既有转换链路 | 不宣称上游原生支持 |
| 账号类型 | 新 `auth_method = dumate_cookie`，Provider `kind = dumate` | 不把 Cookie 伪装成 API Key；避免与 Codex／Google 订阅凭据生命周期混用 |
| 积分查询入口 | 新增 `application/dumate` 服务与独立 Wails 绑定 | 不塞进货币余额字段 |

# Implementation Plan

以下为开发计划（本轮按此实施）：

1. 完整梳理账号认证枚举、Provider 注册、凭据回调、路由、目录及协议转换；
   确认首版范围、迁移必要性及 MIT 声明位置。
2. 先补脱敏模拟夹具与单测，再实现登录文件解析、Cookie 校验和控制台接口；
   不使用仓库外真实凭据作测试夹具。
3. 接入 DPAPI 账号管理、身份核对及有界 token 缓存／刷新。
4. 接通 Chat／SSE、模型别名和路由，验证现有 Responses／Messages 转换、
   工具调用、reasoning、取消及错误语义。
5. 按确认范围接入积分查询、手动签到及可选自动签到；
   Wails 绑定经生成流程更新，不手改生成文件。
6. 增加前后端回归，执行 `task verify`，同步必要文档；
   获授权后再执行真实账号最小验证并记录实际消耗与结果。

# Progress

- 2026-11-13：只读检查参考实现与 OmniProxy 部分接入点，建立本记录；
  用户明确要求先记录、不改代码，状态保持 `Planned`。
- 2026-11-13：用户要求按记录开始开发。梳理 Provider 注册、账号认证类型、凭据回调、
  路由资格、模型目录与协议转换，按本记录的建议确定首版范围（见 Scope Decisions）。
- 2026-11-13：实现 `provider/dumate`、`application/account` 的 `dumate_cookie` 账号类型、
  `application/dumate` 导入／积分／签到服务、网关转发与路由接入、迁移 048、
  Wails 绑定和前端导入／积分界面；补齐单元与网关回归测试。
- 2026-11-13：`task verify` exit 0（format check、go vet、staticcheck、eslint、
  tsc、全部 Go 包、154 文件／935 项前端测试、Windows production 构建）。

# Implementation Result

- 新增 `internal/provider/dumate/`：
  - Cookie 解析、域名过滤头序列化、引号规则与 `csrftoken` 提取，输入有界并拒绝控制字符；
  - 桌面端 `auth.json` + `.cookie-key` 只读解析（AES-256-GCM，`IV(12)+Tag(16)+CT`），
    支持 `OMNIPROXY_DUMATE_USER_DATA_DIR`／`DUMATE_USER_DATA_DIR` 覆盖；
  - 固定 HTTPS 控制台客户端（user/info、token/get、quota_overview、loginBonusInfo、
    loginBonus），禁止重定向，HTML 登录页／3xx／业务码统一映射固定错误码；
  - 有界 token 缓存（并发合并、提前 5 分钟刷新、401 失效、容量淘汰）；
  - Chat 客户端（固定网关、Bearer、流式补 `include_usage`、非 Chat 协议拒绝）；
  - 积分 DTO 与到期计算（只取最近 `active` 包，到期积分取剩余下界）。
- 新增 `internal/application/account/dumate.go`：`auth_method = dumate_cookie`、
  DPAPI 密文、远端身份绑定、按身份查找／创建／重导入、仅允许改名，
  拒绝 API Key 读取与替换；删除走普通账号清理。
- 新增 `internal/application/dumate/`：导入编排（先 `user/info` 核对身份再保存）、
  积分查询（`live`／`stale`／`unavailable`、内存快照保留 1 小时）、
  手动签到（先读、领取、再核对到账，结果 `claimed`／`done`／`pending`／`failed`）。
- 网关：`forwardDuMate` 在凭据回调内换取 token 并转发；上游只确认 Chat，
  Responses／Messages 经既有 `agentprotocol` 转换并缓冲回包；401 只清 token 缓存。
- 路由与目录：`dumate` 预设模型（`model-text`、`dm-auto-model/text.L0`）、
  账号资格（登录态 + authenticated）、目录元数据（只声明文本）；
  迁移 048 追加 `builtin_dumate`。
- Wails 绑定：`DuMateImportDesktop`、`DuMateImportCookie`、`DuMatePoints`、
  `DuMateCheckin`；生成文件由 `wails3 generate bindings` 更新，未手改。
- 前端：DuMate 导入弹窗（桌面端／Cookie 两种来源）、积分与签到面板、
  账号卡／列表／编辑／密钥预览适配、提供商卡片与品牌图标、中英文文案、
  11 项针对导入／积分／签到／账号卡片的组件测试。
- 未实现：自动签到（启用方式待确认）、DuMate 企业版、千帆 API Key／Token Plan。

# Verification

- `task verify`：**exit 0**。包含 format check、`go vet`、`staticcheck`、
  ESLint（0 error）、`tsc --noEmit`、`go test . ./internal/...`（全部包通过）、
  前端 154 文件／935 项测试、Windows amd64 production 构建（含重新生成绑定）。
- 新增后端测试：
  - `provider/dumate`：Cookie 解析／头序列化／边界、桌面登录解密与错误分支、
    控制台成功与各类失败映射、token 并发合并与失效、Chat 转发／401、
    积分 DTO 与到期计算；
  - `application/account`：Cookie 不进入元数据与预览、身份绑定与重导入、
    拒绝非 DuMate 提供商、仅改名、删除清理；
  - `application/dumate`：桌面／Cookie 导入、身份核对与失败码、积分实时／快照／失败、
    签到四种结果与登录态失效；
  - `application/gateway`：端到端 Chat 转发、Responses／Messages 转换、
    401 后重新换取 token、失效账号拒绝。
- 前端测试：导入成功／失败／身份冲突、积分实时／历史／失败、签到按钮与结果、
  账号卡片与列表展示。
- **未验证**：真实百度账号的接口可用性、模型别名、工具调用、reasoning 用量、
  积分消耗、签到到账；真实客户端联调。这些需要用户明确授权并会产生真实副作用。

# Documentation

- 新增 `docs/product/DUMATE.md`（账号、凭据、网关、模型、积分／签到、安全边界与未验证项）。
- 更新 `docs/architecture/OVERVIEW.md`（新增 DuMate 段落、提供商数量与网关适配列表）。
- 更新 `docs/product/PRD.md`（多提供商列表与默认启用数量）。
- 更新 `docs/plans/CURRENT.md`（本轮概览）。
- 更新前端图标来源 `frontend/public/provider-icons/SOURCE.md`（MIT 来源声明）。

# Deviations

- **自动签到未实现**：记录中的「启用方式待确认」，本轮只交付手动签到；
  未创建系统定时任务或后台循环。
- 未做真实百度账号验证（未获授权，且会消耗积分、改变签到状态）。
- 未发布上下文／输出容量：参考实现的 192k／128k 未经真机确认。
- Responses／Messages 为网关转换，不是上游原生协议。
- 不实现 DuMate 企业版与千帆 API Key／Token Plan。

# Git

- 实现提交：`741a5d29`（接入 DuMate 登录态、网关转发、积分与签到、迁移 048、绑定与界面）。
- 任务记录归档与 `CURRENT.md` 概览在同一文档提交中补齐。
- 无强制推送、无历史改写、无标签或发布。
- 无关未跟踪文件 `docs/tasks/completed/2026-10/TASK-20261029-010-subq-provider.md` 保留不动，未纳入本任务。

# Result

已交付百度搭子 DuMate 个人版的原生 Go 接入：登录态导入与身份绑定、
受保护凭据、gateway token 合并刷新、Chat 转发与既有协议转换、
模型别名与路由、积分查询与手动签到，以及对应的 Wails 绑定和前端界面。
自动签到与真实账号验证仍待用户确认。
