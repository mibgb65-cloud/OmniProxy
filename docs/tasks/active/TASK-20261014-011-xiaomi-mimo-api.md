# TASK-20261014-011: 接入小米 MiMo 普通 API Key 模型调用

## Status

In Progress

## Created

2026-10-14（Asia/Singapore）

## Request

接下来接入小米 mimo 的模型、API 调用，看看怎么实现。用户明确优先普通 API Key（按量计费）；开发环境使用 main。

## Goal

将现有 MiMo 提供商从展示预设扩展为可调用的网关适配器，支持官方模型目录、路由、客户端目录、原生 Chat Completions／Responses／Messages、流式与工具调用，复用现有凭证保护和用量统计。

## Analysis

已阅读产品、架构、技术栈、编码、测试及开发规范。main 为 `3038bfa`，现有 MiMo 预设只提供展示名称，网关没有 MiMo 分支。官方文档确认按量入口 `https://api.xiaomimimo.com`，Chat `/v1/chat/completions`、Responses `/v1/responses`、Messages `/anthropic/v1/messages`，使用 `api-key`；原生协议可避免思考／工具上下文在跨协议转换中丢失。复用现有标准库 HTTP 与代理协议，不引入 SDK。

## Scope

- 普通 API Key，官方对话模型与有界模型发现。
- 现有路由／客户端目录及原生 HTTP／SSE 调用、用量观察、安全输入校验。
- 最小充分回归测试、完整质量门、文档及实际开发构建核对。

Token Plan、余额／配额、ASR／TTS 独立接口不在本次范围。未确认能力标为未知，不伪造账号调用或模型权限验证。

## Implementation Plan

1. 基于 main 在独立工作区实施，保留正在运行的 main 与既有任务记录。
2. 新增 MiMo 原生适配器、官方模型元数据和模型发现，接入既有目录／路由／网关。
3. 验证三个协议的 JSON／SSE、思考及工具回传、认证／错误／取消／输入边界。
4. 运行 task verify，更新相关文档，审查差异并提交本任务。
5. 将已验证实现同步至 main，从正常桌面启动 main 开发环境并核对实际窗口与目录。

## Sources

- https://mimo.mi.com/docs/zh-CN/api/chat/openai-api
- https://mimo.mi.com/docs/zh-CN/api/chat/responses
- https://mimo.mi.com/docs/zh-CN/api/chat/anthropic-api
- https://mimo.mi.com/docs/zh-CN/api/model/list-models
- https://mimo.mi.com/docs/zh-CN/api/guidance/model-hyperparameters

## Implementation Result

- 新增 `provider/mimo`，普通 `api-key` 认证、官方按量域名和独立 Messages 前缀；三种原生 HTTP／SSE 在现有凭证回调内转发，响应关闭后清理，禁止重定向，复用用量观察器。
- 五个已核对的 V2.6／V2.5 对话模型、官方 1M／131072 输出元数据；仅公布文本和函数工具能力，不伪造长输入验证。思考只提供关闭／开启，Claude adaptive 归一为 enabled。
- 官方模型同步有界、去重和对话筛选；未同步时提供明确标注的预设，成功快照优先，失败保留，更换 Key 后旧快照失效。
- 接入默认映射、别名／固定账号、启停保护、客户端统一目录、前端同步账号选择和网关接入检查；现有 MiMo 图标和账号存储复用，无新增依赖或迁移。
- 新增认证／地址／取消／正文／模型发现契约测试，以及三协议 JSON／SSE 首包、用量、凭证清理、目录和路由集成测试。相关前端同步及接入检查回归通过。
- 实现位于基于 main 的 `codex/xiaomi-mimo-api`，待同步主分支并核对实际正常桌面开发窗口。

## Verification

- PASS：MiMo 原生认证／地址／错误／取消／输入与目录边界契约测试。
- PASS：网关三协议 JSON／SSE 首包、别名与 Claude 1M 解析、用量记录和凭证清理集成测试。
- PASS：账号目录预设／同步／失败保留／换 Key，以及路由目录和账号启停边界。
- PASS：三项相关前端测试文件共 23 项测试，含 MiMo 官方目录同步与接入检查。
- 首次 `task verify` 在新 worktree 的 Go vet 阶段因 `frontend/dist` 尚不存在而中止；先按现有命令构建前端内嵌资源，再重跑完整质量门。不是实现错误，未修改验证门。
- 静态检查指出测试中的 HTTP 重定向数字常量，已替换为标准库常量后重跑。
- PASS：`VITEST_MAX_WORKERS=4 task verify BIN_DIR=E:/go/new/wails3-v/OmniProxy-mimo-api/tmp/mimo-api/quality-bin`，退出码 0；格式、Go vet／staticcheck、ESLint、TypeScript、全部 Go 测试、70 个前端文件／386 项测试和 Windows 生产构建通过。日志位于忽略文件 `tmp-mimo-verify.log`。
- PASS：`git diff --check`；Wails 绑定、Go 模块和前端锁文件没有额外差异。

## Documentation

新增 `docs/product/MIMO.md`，同步 PRD、网关、路由、客户端接入、架构与技术栈说明；CURRENT 记录本任务。基于既有适配器边界，无需新 ADR。

## Deviations

新 worktree 需先生成前端内嵌产物后才能运行现有质量门；未修改构建或验证脚本。官方多模态、Token Plan 和余额不在已确认范围内，仅接入普通 Key 的文本／函数调用。

## Git

待提交。

## Result

待完成。
