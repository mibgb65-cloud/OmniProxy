# TASK-20261026-006: 移除未获批的 SignPath 发布依赖

## Status

Completed

## Created

2026-10-26（Asia/Singapore）

## Request

“该怎么填写？signpath没有通过，算了吧，把那部分给去掉。”
“可以自己签名吗？”
“自签名证书吧，不想买。”

## Goal

按维护者明确决定移除 SignPath，使用项目专属自签证书为发布安装包签名，准确说明其信任边界。

## Scope

- 移除 v2 Release 的 SignPath 凭据、审批任务和专用签名校验。
- 保留 GitHub Actions 构建、质量门、版本、SHA256、草稿及历史保护。
- 修正当前及默认分支中“已获 SignPath 服务”的说明。
- 明确签名现状、私钥保护与自签信任边界。
- 生成专用代码签名证书：私钥留在当前 Windows 用户证书库，供 Actions 使用的加密 PFX／密码
  只传入本项目 GitHub encrypted Secrets；Git 仅保存公开证书。
- 在隔离 Actions Runner 中临时信任精确匹配的项目公开证书以验证完整性，结束时清理；
  安装器不得向终端用户的信任库导入证书。

## Out of Scope

- 不提交私钥、不输出 PFX／密码、不购买证书；除本项目 GitHub encrypted Secrets 外不传输私钥。
- 不将自签描述为系统受信任签名，不关闭 Windows 安全检查。
- 先以不发布的 Actions 验证签名；通过前不创建正式标签、不公开未经验证的产物。
- 不改写历史或覆盖既有 Release，不改动 v1 业务实现。

## Analysis

原签名策略是上一任务的发布阻塞；维护者现已明确说明 SignPath 未获批并要求移除，
该明确要求优先于旧策略。自签可以验证文件完整性及内部身份，但不自动获得 Windows 信任或
SmartScreen 信誉；公开发行若需要受信任发布者，应使用正规 CA 的代码签名证书。
维护者随后明确选择自签，允许准备该专用证书及 Actions 签名所需的加密凭据。公开签名的目标是
发布安装包，应用程序本体暂不单独签名；不得宣称系统受信任或没有 SmartScreen 警告。

## Implementation Plan

1. 核对 v2 工作流及原默认 master 分支中的 SignPath 引用。
2. 准备项目自签证书、公开证书与加密 Secrets，替换签名任务并修正文档。
3. 运行本地检查及不发布的 Actions 签名验证；校验证书 pin、有效签名、摘要和篡改拒绝。
4. 验证通过后依照此前授权从正式标签运行发布，核对公开资产与实际签名状态。
5. 记录结果、证书边界及用户侧信任提示。

## Implementation Result

- 已生成 `CN=OmniProxy Self-Signed` 专用代码签名证书（RSA 3072／SHA256），私钥保存在
  Windows 当前用户 My 证书库。Git 只允许公开 `build/windows/codesign.cer`，没有 PFX／私钥文件。
- 公开证书 SHA256：
  `bfd485e92115356595b0a366490443f6e4c99ae33297f12a30049dcac5bec054`。
- 加密 PFX 与随机强密码经标准输入传给 gh，配置为本项目 encrypted Secrets
  `OMNIPROXY_SIGNING_PFX`、`OMNIPROXY_SIGNING_PASSWORD`；没有出现在命令行参数或日志，
  没有将本机 Root／TrustedPublisher 信任库加入该证书。
- 移除未获批服务的 Action、五项配置和 Foundation 身份依赖，改为项目公开证书 SHA256 pin。
  只有隔离 Actions Runner 临时信任该精确证书用于原生验签，清理后结束；客户端不自动信任。
- Actions 验证真实自签、版本、pin、摘要，并翻转被哈希代码区一字节验证篡改拒绝。
  签名后重算 SHA256；仍保留 GitHub 构建、草稿、版本及不覆盖历史的保护。
- 增加 `publish=false` 验证模式，构建所选 workflow 的固定 SHA，跳过公开发布作业，
  先验证云端签名流程，再创建正式标签。
- 安装器只分发公开证书文件，卸载删除自有文件，不导入用户信任库；更新签名及隐私说明。
- 将以独立临时 Git index 为原 master 创建仅三份签名文档的正常快进提交，
  不切换工作区、不合并 v1／v2 产品历史、不改动 v1 代码。

## Verification

- 本地真实副本已加入项目签名，原生状态为 `UnknownError`（未信任根），内嵌 signer 匹配；
  没有通过向用户 Root 加证书伪造系统信任。正向信任／完整性待隔离 Actions 验证。
- 本地公开证书检查、未签名与其他发布者拒绝、禁止本机临时根信任：PASS。
- PowerShell 语法、actionlint、`git diff --check`：PASS。
- `task --force verify` 与 `task --force package`：PASS，132 文件／809 项前端测试及 Windows 构建通过。
- 隔离用户级原生向导／改程序目录／静默升级／卸载：PASS。
  公开证书和 MIT 原文正确分发／清理，旧数据与未知文件保留，用户 Root 信任未被修改。
- 未执行真实上游付费请求或跨用户 UAC；自签不能取代这些既有人工验收边界。
- 云端不发布验签与正式标签重新构建发布：PASS，实际公开下载核验通过。
- 首次完整预验证 `40359373376`：构建及完整质量门成功，签名阶段 15 分钟超时，
  没有发布。改为无 UI 的内存 PFX 导入，并仅在 GitHub 托管 Windows Runner 既有管理员身份下
  使用临时机器信任库，增加不含密钥的阶段日志。
- 隔离验签 `40365440314`：SUCCESS，重用同仓库上述成功构建的 unsigned Artifact，
  校验其 provenance 后实际完成自签、pin 验证、原生验签、代码区篡改拒绝与最终摘要检查；
  所有公开发布作业跳过，不能把该夹具模式当成新版本的构建或发布。
- 正式 `v2.0.0` 标签已正常创建于 `c9c48b4a31988a6cc3fa2f4b1d423e61bd10f71b`。
  Release `40366364069` 从标签重新运行完整构建、签名和发布，未复用夹具作为公开产物；SUCCESS。
- 公开 Release 非草稿、非预发布，stable Latest 为 v2.0.0；实际下载的安装器与 API digest／
  `.sha256` 均一致，内嵌 signer 为 `CN=OmniProxy Self-Signed`，证书 SHA256 与公开 pin 一致。
- 安装器 20,286,400 bytes，SHA256
  `d43abd28773a192b46139a68dac6411543f146781dedb22b0da3e3f285a0e6a1`。
  客户端原生状态 `UnknownError` 为未信任根的预期边界，没有向用户 Root 增加证书。
- 并发主分支 CI 40365457327 暴露了另一项测试完成信号竞态，非签名／业务改动，
  由 TASK-20261027-001 独立修复；没有把失败记录隐去或替换已公开 v2.0.0。

## Git

- `d262ebd92c997246336d7ba8739822bfacfb15f2`：项目自签方案及公开证书，已正常推送 main。
- `c9c48b4a31988a6cc3fa2f4b1d423e61bd10f71b`：非交互 Runner 修正及隔离验签模式，
  已正常推送 main；正式标签指向此修订。
- `4bcee95bf1831e2f70de29281ef23a207de6e146`：master 文档快进提交，仅
  CODE_SIGNING_POLICY.md、README.md、README_EN.md 三份文件。未切换工作区、未改 v1 业务代码。
- 未强推、未移动或删除共享标签，未覆盖历史 Release。

## Result

已移除未获批服务，完成项目自签证书与加密凭据配置，并通过 GitHub Actions 发布 v2.0.0：
https://github.com/mibgb65-cloud/OmniProxy/releases/tag/v2.0.0 。
公开安装包实际自签、证书 pin 与最终 SHA256 均核验，历史版本保留。
自签不等同于公开受信任签名，可能继续出现 SmartScreen／未知发布者提示；
客户端信任库未被修改，私钥不在 Git 或公开资产中。

## Completed

2026-10-27（Asia/Singapore）
