# TASK-20261022-005: 补齐推理档位与剩余问题并构建 Beta.20

## Status

Completed

## Created

2026-10-22（Asia/Singapore）

## Request

用户要求修复此前尚未完成的问题，特别是第三方模型思考强度，并在完成后构建新的 Beta 生产版本供本地更新；不自动安装或公开发布。

## Goal / Scope

- 确认第三方模型各型号／线路的实际推理契约，修复能力目录、客户端配置与请求参数链路。
- 续办 TASK-20261022-004 的上下文压缩／超时排查及 River 原生验收，不伪造未知能力或验证结果。
- 完整质量门、必要原生契约和界面验证；准备 Beta.20 版本、发行说明及 production NSIS 安装器。
- 核对源码来源、原生版本、运行时版本、SHA256，并保护既有发布资产、用户配置／数据／凭据和正在运行的程序。

## Out of Scope

不绕过客户端或模型限制，不用其他厂商身份伪装模型；不改无关模块、不引入大依赖；不自动安装、推送或公开发布。

## Analysis

- Codex 目录中的多款第三方模型虽有合法档位，但默认档位为 null；同步快照可能继续保留旧的 River 能力。需要提供有效的客户端默认选择并刷新已确认静态能力，不伪造未知型号的枚举。
- River Messages 转 Chat 时当前会丢弃 `output_config.effort`；需要显式校验和映射。官方控制表仍未单列 1017，不直接照搬原版，先用有界合成请求核对接口契约。
- 当前 Claude Desktop 的 `inferenceModels` 配置不支持任意档位能力注入；其自定义模型目录拒绝回环地址，不能绕过该限制或伪装厂商身份。考虑以模型菜单中的显式强度选项提供已确认 River 档位，网关受控还原真实型号与参数。
- 当前 HTTP／WebSocket 全部请求共用 120 秒上限，V2 压缩只保留头部却未独立识别用于时限与记账。拟仅对明确的原生压缩使用有界专用时限，普通推理保持原值；500k 触发根因仍不作已修复结论。
- 已保存 37 个保护文件及生产进程基线；不复用上次的默认 WebView 数据目录，验收采用独立 WebView／数据／配置目录。

## Implementation Plan

1. 阅读规范及续办记录，保存保护基线，确认推理参数缺失位置及压缩请求时限。
2. 先覆盖失败回归，再最小修复；确认本机客户端配置接口，必要时明确不支持的边界。
3. 解决隔离原生窗口验收环境，核对 River 与推理档位；执行定向测试和完整质量门。
4. 更新 Beta.20 及发行说明并冻结源码，使用标准发布脚本构建隔离生产安装器。
5. 核对版本、哈希、来源和保护基线，提交并归档实际结果；不得将仍未复现的问题宣称为已修复。

## Implementation Result

- **显式推理档位（Claude/Messages 路径）**：新增 `internal/core/modelcatalog/claude_effort.go`。对 `ProviderKind == river` 且元数据已确认档位的模型，在 Claude 目录中派生 `omniproxy-effort-<sha256 前 20 字节>` 形式的显式选项，名称形如 `模型 · 高强度推理`，并同步派生 `[1m]` 变体；选项本身不声明 `ReasoningEfforts`／`DefaultEffort`，不改动原模型的默认标记。只读路径 `ResolveClaudeEffort` 仅从当前可用目录解析，停用／删除目标后立即失效，因此不会为缺失能力伪造档位。
- **档位落地到请求**：`internal/application/gateway/reasoning.go` 在两个方向校验：客户端选显式选项时，`withClaudeEffort` 写入 `output_config.effort` 并保留其余字段（如 `format`），同时用用户显式选择覆盖客户端沿用的旧 `thinking=disabled`（`none` 写 `disabled`，其他写 `adaptive`）；River Messages→Chat 转换时，`riverMessagesEffort` 只在 `effort` 属于该模型实际枚举时透传为 `reasoning_effort`，非法值返回 `unsupported_reasoning_effort` 400 而不是静默丢弃。`routing.go` 先尝试档位解析、失败再回退到普通别名解析，其他服务商行为不变。
- **Codex 默认档位**：`CodexJSON` 在目录声明了档位但元数据未给出默认值时，选择 `high`（存在时）否则第一项作为网关侧默认，避免客户端因 `default_reasoning_level = null` 不展示档位；显式 `DefaultEffort` 一律优先，未知档位仍输出 `null`，不编造能力。
- **River 能力目录**：`provider/river/catalog.go` 为 1017 补入已合成验证的三档（`none`／`high`／`max`）与中文说明及默认 `high`，GLM 改为 `low`／`high`／`max` 并加默认；`application/modelcatalog` 与 `routing` 在读取 River 目录时统一改用 `river.ModelMetadata(river.UpstreamModel(...))`，使旧快照不再继续输出过时能力（路由查询新增 `JOIN providers` 取 `kind`）。
- **压缩时限与记账**：新增 `internal/core/proxy/compaction_deadline.go`。`Request.IsCompaction` 只认固定端点或经 `ValidCodexMetadata` 校验后的 `remote_compaction_v2` + `responses_compaction_v2` 元数据，不按正文或仅凭 beta 头猜测；HTTP 与 WebSocket 改用 `s.requestTimeout(request)`，明确压缩升到 10 分钟，普通推理仍为原 120 秒。`trace.go` 将压缩记为 `responses/compact`，使留存记录可区分，不去推断 500k 提前压缩的根因。
- **回归测试**：新增 `reasoning_test.go`（River 档位透传、非法档位、GLM 不可关闭、未确认模型拒绝）、`claude_effort_test.go`（派生数量、真实身份与 `[1m]` 保留、幂等、不伪造/不遮蔽、失效）、`modelcatalog/reasoning_test.go`（Codex 默认档位四分支）、`compaction_timeout_test.go`（普通／仅 beta 头／普通轮次不放宽，V2 与 legacy 端点放宽且仍被记账区分），并扩展 `river_test.go` 断言 Messages 的 `output_config.effort` 不得被丢弃。
- 未修改用户真实客户端配置、凭据、运行中的生产程序；不为未知型号或未验证厂商的档位做猜测。

## Verification

- PASS：`gofmt -l main.go internal` 输出为空（首次质量门因两个新增测试文件未格式化而失败，已 `gofmt -w` 修复后重跑）。
- PASS：`go vet . ./internal/...`、`staticcheck . ./internal/...`、`pnpm format:check`、`pnpm lint` 与前端 `tsc` 类型检查。
- PASS：定向 `go test ./internal/core/modelcatalog ./internal/core/proxy ./internal/application/gateway ./internal/application/routing ./internal/application/modelcatalog -count=1`。
- PASS：专项用例逐项执行（`-run` 过滤）并列出 `=== RUN`，确认新回归真实运行而非跳过：`TestClaudeExplicitEffortsPreserveRealIdentityAndLongContext`、`TestClaudeEffortDoesNotInventCapabilitiesOrShadowExistingModels`、`TestCodexReasoningAlwaysHasSupportedDefault`（4 子例）、`TestCompactionUsesDedicatedDeadlineAndNormalRequestsStayBounded`（5 子例，含 normal／feature_only／ordinary_turn／v2／legacy）、`TestRiverMessagesEffortPreservedAndValidated`、`TestExplicitClaudeEffortKeepsOtherFieldsAndOverridesStaleThinking`、`TestRiverNativeGatewayStreamingRoutingAndUsage`（Flash 与 1017 各 6 个协议流式组合）。
- PASS：完整质量门 `task verify BIN_DIR=../verify-bin` 返回退出码 0：格式化检查 → lint → typecheck → Go 全套 → 121 文件／724 项前端测试 → Windows production `go build -tags production`，产物 `../verify-bin/omniproxy.exe` 22,334,976 bytes。
- 说明：同一条门禁第一次执行以 exit 1 结束，原因是另一次 verify 并发启动，`build/windows/Taskfile.yml` 的 `powershell Remove-item *.syso` 尾部清理与另一次 `generate:syso` 竞争；对空目录单独执行该命令返回 0，非串行重跑为 0。未修改质量门脚本、未跳过任何检查。
- 非阻塞既有告警：前端 1,405.97 kB 大包、jsdom 环境耗时及 `Window.scrollTo` 未实现提示，均为本任务之前既存在，未为隐藏告警改动无关代码。
- 未执行：原生窗口视觉／UIA 验收（本任务只改动后端目录、路由与代理时限，无界面改动）；真实上游档位调用与真实覆盖安装由用户在实际客户端验收，不以合成测试替代。
- PASS：版本冻结后 `internal/version` 与根包测试、前端 121 文件／724 项测试复跑通过。
- PASS：`scripts/set-release-version.ps1 -Version v2.0.0-beta.20` 校验通过，Windows 数字版本维持 `2.0.0.0`；`build/config.yml`、`build/windows/nsis/wails_tools.nsh` 与 Beta.19 一致（未被改写）。
- 说明：脚本以 `ConvertTo-Json` 重排 `build/windows/info.json`，已手工归一化回原缩进，最终 diff 仅 `ProductVersion` 一行，并重跑 `scripts/test-windows-metadata.ps1` 通过。
- PASS：`task --force verify`（在配置好工具链的 shell 中）退出码 0。
- PASS：`task --force package` 退出码 0，产出 `bin/omniproxy.exe` 22,338,560 bytes 与 `bin/OmniProxy-amd64-installer.exe` 19,562,228 bytes。
- PASS：`scripts/test-windows-metadata.ps1 -Version v2.0.0-beta.20` 输出「Windows 应用和安装器版本资源校验通过」。
- PASS：`release-assets/v2.0.0-beta.20/` 新建且非覆盖；安装器 19,562,228 bytes，SHA256 `53010de584cbff6b915d345b525a1e906afbdb31aa27c62b44929e3d87169da2`；`scripts/verify-release-assets.ps1` 校验通过。
- PASS：Beta.6–Beta.19 共 14 个既有发布目录仍各为 2 个文件，未被覆盖或删除。
- 未执行：签名、自动安装、主动停止／替换生产进程。生产应用仍为 PID 49000／`E:\codexproxy\OmniProxy\OmniProxy.exe`；未创建或推送标签，未公开发布。
- 说明：`scripts/build-release.ps1` 一次性执行时以 exit 1 结束，原因是该脚本调用 `task` 时 PATH 缺少 bash 类工具，`task` 把 `uname: executable file not found` 的 stderr 当作终止错误；随后在同一次构建所需的相同命令（`task --force verify`／`--force package`／元数据／资产校验）逐项执行，全部返回 0，未修改质量门与发布脚本。

## Git

实现与发布准备提交：`6c0a98b` — `feat(reasoning): 补齐第三方模型推理档位并准备 Beta.20`（24 个文件，+581/−17；River 用量提交 `d196d39` 属于 TASK-20261022-004，不在本任务范围内）。`docs/plans/CURRENT.md` 与 TASK-20261021-006 记录中的既有他人未提交改动保持原样，未纳入本次提交；未创建或推送任何标签。

## Result

第三方模型推理档位链路（Codex 默认档位、Claude 显式档位选项、River Messages→Chat 透传与校验、River 能力目录刷新）与压缩专用时限／独立记账已实现，并通过完整质量门、专项回归和 Windows production 打包。Beta.20 本地安装包及 SHA256 已生成并通过版本资源与资产校验，未签名、未自动安装、未推送或公开发布。Codex 约 500k 提前压缩的根因仍未确认，本任务不宣称已修复该项。

## Remaining Risks

- Codex 1.05M／900k 配置已生效，但在隔离核心中仍未能复现约 500k 提前压缩；同期存在无首 Token 的约 120 秒超时。压缩专用 10 分钟时限只是边界保护，不是根因修复。
- 模型「支持思考」不等于支持任意低／中／高档位；本任务只发布已用有界合成请求确认过的 River 档位，其他厂商保持未知。
- Claude 显式档位依赖网关生成的选项标识，真实 Claude Code／Desktop 菜单渲染与选择后的上游效果需用户在实际客户端验收。
- 真实 River 上游档位调用、真实覆盖安装、数据保留与模型表现均未在本任务中验证；安装包未签名，Windows 可能提示未知发布者。
