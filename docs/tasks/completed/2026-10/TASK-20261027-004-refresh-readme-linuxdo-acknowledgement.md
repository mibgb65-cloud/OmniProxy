# TASK-20261027-004: 更新 README 并致谢 LINUX DO 社区

## Status

Completed

## Created

2026-10-27（Asia/Singapore）

## Request

项目 README 没有更新，需要认可 Linux DO 社区，请修改。

## Goal

让 README 面向实际使用者反映当前项目能力、Windows 安装与开发入口，并明确致谢 LINUX DO 社区，不虚构官方背书、合作或具体贡献。

## Scope

- 根据现有代码、产品文档及发布记录更新 README。
- 增加社区致谢与官方站点链接，保留原项目 MIT 许可和来源说明。
- 核对文档链接、命令定义、构建产物及文字边界。

## Out of Scope

- 不修改业务代码、版本号、构建流程、生产数据或发布资产。
- 不推送或发布，不改动前序任务的未提交文档。

## Analysis

现有 README 仍是英文开发骨架，缺少安装入口、实际功能概览、能力边界及社区致谢。
已核对产品文档、提供商目录、开发入口和 NSIS 配置：安装器实际输出到 `bin/OmniProxy-amd64-installer.exe`，不是旧说明中的 `build/windows/nsis/`。
README 使用简体中文，区分源码与公开安装包，不承诺所有提供商具备相同协议能力，也不把 Windows 支持扩展为 Linux／macOS 已支持。
社区链接采用 `https://linux.do/`；致谢仅表达认可与感谢，不声称官方合作、资助或背书。

## Implementation Plan

1. 核对当前版本、功能文档、运行环境和构建路径。
2. 更新为简体中文项目首页，补充 LINUX DO 致谢。
3. 检查链接、命令与 UTF-8 编码，审查差异并记录提交。

## Implementation Result

- 将英文开发骨架更新为简体中文项目首页，增加当前能力概览、提供商预设、下载入口、快速开始、使用边界、安全说明及文档导航。
- 保留开发依赖与命令，依据真实 NSIS 配置修正安装器路径为 `bin/OmniProxy-amd64-installer.exe`。
- 明确 Windows x64 的维护范围、项目自签证书与源码／公开安装包区别，不夸大跨平台或协议能力。
- 独立增加「社区致谢」，链接 `https://linux.do/`，感谢社区的技术交流、知识分享与开源协作；不虚构官方合作、资助或背书。
- 保留原 OmniProxy 项目及开源依赖致谢、原有 MIT 许可与版权说明。
- 无需更改产品规格、架构或 CURRENT 状态；前序未提交文档保持原状。

## Verification

- Prettier 格式检查通过。
- README 的 30 处本地链接／锚点存在性与 UTF-8 无 BOM 检查通过。
- 6 个 Task 命令已与 Taskfile 定义核对；Go／Node／Wails 版本、默认端口和安装包路径已对照实现人工核对。
- GitHub Releases 外部链接 HEAD 返回 HTTP 200；LINUX DO 站点 HEAD 返回 HTTP 403，记录为访问防护限制，不声称已通过站点可达性验证。
- `git diff --check` 通过。
- 未执行 `task verify`：本次仅修改 Markdown，不涉及业务代码、配置、依赖或发布；命令与产物规则已经做针对性检查。

## Git

```text
8b1bee8e61aa4ef619f9be18a3b578e12cf39857 docs(readme): 更新项目说明并致谢 LINUX DO 社区
```

仅提交本任务 README，未纳入前序任务的未提交文档；未推送或发布。

## Result

README 已反映当前主要能力、安装／开发入口与维护边界，并明确认可、致谢 LINUX DO 社区。
本次不改变业务功能、版本号或许可证；格式、本地链接、命令定义和构建路径已核对。

## Completed

2026-10-27（Asia/Singapore）
