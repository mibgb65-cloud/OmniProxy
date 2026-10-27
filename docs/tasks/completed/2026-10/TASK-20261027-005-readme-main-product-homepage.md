# TASK-20261027-005: 参考 master 完善 main 项目首页并推送

## Status

Completed

## Created

2026-10-27（Asia/Singapore）

## Request

参考 master 分支 README 的完整产品首页结构，修改 main 的 README，提交并推送更新；用户计划后续将默认分支设置为 main。

## Goal

让 main 的首页承接同一项目 v1 → v2 的演进，保留产品价值、架构、能力与接入说明、发布和贡献入口，以及 LINUX DO 致谢；全部内容以 v2 实现为准。

## Scope

- 更新中文 README，补充匹配的英文 README 与互链。
- 核对 v2 提供商、客户端、入口、Control API、开发命令、发布和平台范围。
- 检查格式、链接、关键事实，提交并正常推送至 origin/main。

## Out of Scope

- 不切换或修改默认分支，不合并 master，不改写提交历史。
- 不修改业务代码、版本号、许可证、生产数据或 Release 资产。
- 不提交前序任务已有的未提交文档。

## Analysis

master 的产品首页组织较完整，但 Wails 2／Vue、macOS、旧目录与脚本、旧 API 和提供商能力不能直接迁入 v2。main 使用 Wails 3／React，当前公开维护 Windows x64，HTTP Control API 仅实现 `/healthz`。默认分支仍为 master；本任务只准备 main 首页。
已核对 `agentconfig.Clients()` 的十三项白名单与 `agentendpoint` 前缀，Gemini CLI 不在 v2 托管名单；上游 Gemini 模型能力不能等同于该客户端的一键配置。
提供商范围对照网关实际分派与产品文档，普通 OpenAI API Key、Anthropic、智谱、自定义预设不写成已完成的原生推理适配。
中英文首页包含较多能力矩阵，允许超过 200 行以保持单页产品介绍；详细限制仍链接到现有产品文档，不复制完整手册或编造路线图。

## Implementation Plan

1. 对照代码与现有产品文档，冻结可公开描述的 v2 能力和入口。
2. 按 master 的产品导向重组中英文首页，保留社区致谢、MIT 和同一项目延续关系。
3. 校验文档、命令及差异，记录提交并推送，核对远端提交及默认分支。

## Implementation Result

- 参考 master 的产品首页组织，完善中文标题区、真实技术徽章、使用价值、架构图、提供商与客户端矩阵、本地请求示例、发行渠道、项目结构和贡献说明。
- 新增 `README_EN.md` 并建立中英文互链；产品范围、命令、请求示例及本地文档目标保持一致。
- 主动纠正旧概览中的 Gemini CLI 接入暗示，明确当前托管白名单、仅 `/healthz` 的 HTTP Control API、未完成的原生推理预设及 Windows x64 维护边界。
- 明确同一项目 v1 → v2 的版本演进，不使用容易被误解为 Fork 的“感谢原 OmniProxy 项目”表述；保留现有 MIT 与版权文件不变。
- 两份首页均保留 LINUX DO 社区致谢与官方站点链接，不宣称官方背书。
- 只修改中英文 README 及本任务记录，不合并旧版代码、不改默认分支或 Release、不纳入已有未提交文档。

## Verification

- `pnpm --dir frontend exec prettier --check ../README.md ../README_EN.md`：PASS。
- UTF-8 无 BOM、本地链接／锚点、HTTPS 格式检查：PASS。两份首页各有 35 个不同本地文档目标，互链之外的本地目标集合一致。
- 中英文 Bash 命令序列一致，6 个 Task 命令与真实定义核对，NSIS 安装器路径核对：PASS。
- 请求示例按 `agentendpoint` 前缀、网关协议路径与 `/healthz` 注册核对：PASS；未向生产网关发送测试请求。
- Mermaid 的 LR 结构、八个节点、引用与引号基本语法人工及静态检查：PASS；没有把静态检查声称为 GitHub 实际渲染验收。
- `go test ./internal/core/agentendpoint ./internal/core/proxy ./internal/transport/httpapi -count=1`：PASS（0.121s／2.804s／0.174s），覆盖入口与 HTTP 健康检查相关实现。
- `git diff --check`：PASS。
- 未运行 `task verify`：本次仅修改 Markdown，未变更业务代码、依赖、配置或构建流程；命令和接口示例已经做针对性验证。
- fetch 与推送后的首次远端核对曾遭遇 TLS 握手错误；保留证书校验，仅按命令使用 HTTP/1.1 重试成功。推送本身成功后没有盲目重推，而是重新读取远端引用；未修改全局 Git 网络设置或关闭 SSL 验证。
- 正常快进推送 `origin/main` 成功，随后远端 `main` 核对为 `f7abca0927b2d0362eeed9f34919c95ae616a0ee`，与实现提交一致。
- 远端默认分支仍为 `refs/heads/master`，master 仍为 `4bcee95bf1831e2f70de29281ef23a207de6e146`；没有修改默认分支、旧版分支或公开资产。

## Git

```text
f7abca0927b2d0362eeed9f34919c95ae616a0ee docs(readme): 参考旧版完善 v2 中英文项目首页
```

已正常推送到 `origin/main`。本任务归档随后作为独立文档提交推送；未纳入既有 CURRENT 排查段落和 TASK-20261027-002 未提交记录。

## Result

main 的中英文 README 已按完整产品首页结构更新并推送，补齐架构、矩阵、接入和贡献说明，保留 LINUX DO 社区致谢，明确同一项目 v1 → v2 的延续。
内容遵循 v2 实现，不继承旧版过时技术栈、平台、接口或脚本。默认分支设置维持不变，后续由维护者决定何时切换为 main。

## Completed

2026-10-27（Asia/Singapore）
