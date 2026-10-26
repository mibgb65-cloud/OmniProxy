# TASK-20261026-005: 按原 GitHub 仓库要求发布 v2.0.0

## Status

In Progress

## Created

2026-10-26（Asia/Singapore）

## Request

“需要查看原有 github 仓库，按照仓库的要求进行发布。”

## Goal

核对原仓库规范、发布资产与既有渠道，在不覆盖历史版本或改写共享历史的前提下发布 v2.0.0。

## Scope

- 确认 Git remote、目标账号、远端分支、版本历史及发布流程。
- 比较原仓库规范与本地发布配置，必要时做与发布直接相关的修正。
- 按仓库约定推送正常提交／新标签，跟踪 CI／Release，核验最终公开资产。
- 记录实际发布结果、链接与尚未执行的人工验收。

## Out of Scope

- 不强推、不重写远端历史、不移动或删除已共享标签，不覆盖已公开的 Release。
- 不泄露令牌，不替用户选择开源许可证或声称安装器已经签名。
- 不将 TASK-20261026-004 中尚未执行的跨用户 UAC／真实客户端验收写成通过。

## Analysis

- origin 为原公开仓库，当前 CLI 账号有 ADMIN 权限。远端默认分支为 v1 的 master；
  v2 使用 main，本地 main 是 origin/main 的正常快进，领先 185 个提交，不与 master 共享历史。
- 远端最新 stable 为 v1.3.1，最新公开 v2 为 v2.0.0-beta.5；新 v2.0.0 标签及 Release 均不存在。
- 原仓库 LICENSE 为 MIT；AI_CONSTRAINTS 的发布规则明确要求 GitHub Actions 构建与发布、
  中英双语说明和真实 compare 链接，禁止把本地安装包手工上传为 Release。
- CODE_SIGNING_POLICY 要求 GitHub 托管构建、先上传 Artifact、签名请求人工批准、
  签名产物回收校验并重生成摘要后才能公开；本地 v2 工作流未接入签名。
- 仓库当前无 SignPath Repository Secret／Variable，不能伪造配置、跳过审批或发布未签名正式版。
- 将恢复原 LICENSE／SECURITY／签名策略，按 v2 真实行为补隐私说明和双语发行说明，
  接入失败关闭的 SignPath 签名门。原 master 和既有 Release／标签不变。
- Changelog 使用同一 v2 历史中最后公开的 beta.5 至 v2.0.0，不能把跨根历史的 v1.3.1
  比较链接冒充有效增量。公开 Beta.5 到正式版会执行已有 035–044 迁移，不能笼统写“无迁移”。

## Implementation Plan

1. 读取远端仓库和发布历史，确认发布分支、标签与资产契约。
2. 检查历史兼容性与差异，处理必要发布配置。
3. 验证并提交发布任务记录，正常推送并创建新的 v2.0.0 标签。
4. 跟踪真实发布流程，核对 stable／latest、安装器及校验文件。
5. 归档任务记录并报告最终结果与边界。

## Implementation Result

- 从原 master 原样恢复 LICENSE、SECURITY、CODE_SIGNING_POLICY，并核对 Git blob 完全一致；
  按 v2 真实目录／功能补充 PRIVACY，修正 README 的许可证 TBD。
- 安装器分发原 MIT 文本 `OmniProxy-LICENSE.txt`，卸载只删除该自有文件；
  隔离用户级向导、覆盖升级、卸载及未知文件／数据保留复核通过。
- Release 工作流改为 build → unsigned Artifact → SignPath 人工审批签名 → signed Artifact → publish；
  使用官方 v2 固定 SHA `c92b958760219087e01f8d67a1669ed57afe2627`，核对其实际 action.yml 输入契约。
- 签名配置缺失时失败，不回退为未签名发布。签名者、Authenticode 信任和产品版本验证后，
  才为签名文件生成 SHA256；发布作业重新验证，保留原有 draft 校验／不覆盖公开版本的保护。
- 增加签名门禁测试，包含真实未签名文件拒绝与隔离作用域内的 mock 分支测试。
  Mock 只验证分支逻辑，不冒充有效签名或审批结果。
- 发布说明采用原仓库中英双语四节模板，比较最后公开 v2 Beta.5 至 v2.0.0；
  明确 Beta.5 到正式版会执行已有 035–044 迁移，修正此前笼统“无迁移”的文案。
- 增补真实 SignPath 配置／MFA／审批要求及从 main 调度 v2 工作流的指引，不更改旧 master。

## Verification

- `actionlint .github/workflows/release.yml`：PASS（v1.7.7）。
- PowerShell 新脚本语法解析：PASS。
- `scripts/test-release-signature.ps1`：PASS。实际未签名文件被拒绝；
  错误发布者、签名后摘要生成和拒绝过期摘要通过 mock 分支测试，不证明真实证书通过。
- `task --force verify`：PASS，格式、Go vet／staticcheck、ESLint、类型、Go 测试、
  132 文件／809 项前端测试及 Windows production 构建通过。
- `task --force package`：PASS，含原 MIT 文本的 NSIS 安装器成功编译，仅用于本地验证。
- 隔离用户级原生安装／静默覆盖升级／卸载：PASS；MIT 文件字节哈希与原 LICENSE 相同，
  卸载删除该自有文件并保留未知文件、用户配置和旧数据库。
- 非公开提交对象的凭据文件名检查无命中；高置信令牌模式只命中已在 origin/main
  公开的 redactor 单测样例，没有新增匹配。检查未输出候选值，不等同于完整秘密审计。
- 远端确认：v2.0.0 标签／Release 不存在；已有 stable Latest 为 v1.3.1，v2 最新公开为 Beta.5。
- SignPath Secret／Variable 目前均未配置；未发起签名请求，不得视为审批或签名通过。
- 本地验证日志：
  `C:/Users/mimanchi/.codex/visualizations/2026/10/26/01a17824-8bec-7e20-8fee-78aedc6c123e/v2-github-release-policy-verify.log`。

## Git

待记录。

## Result

合规修正与本地验证已完成；准备正常推送 main 并核对远端 CI。
公开发布仍需真实 SignPath 组织／项目／策略与 Token 配置，以及授权审批人手工批准。
在这些前提未满足前，不创建新版本标签，不上传本机构建文件，不更改现有公开 Release。

## Deviations

原计划核对后直接发布；实际发现原仓库有严格签名要求而 v2 流程及配置缺失，
因此先补合规门禁，不能把本地未签名包直接上传。
不复制 v1 的旧技术栈约束或删除历史 Beta 的工作流行为，也不强行合并无共同历史的 master。
