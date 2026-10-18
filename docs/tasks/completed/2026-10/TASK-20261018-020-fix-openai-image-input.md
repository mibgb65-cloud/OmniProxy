# TASK-20261018-020: 排查并修复 OpenAI 6.1 Sol 图片输入

## Status

Completed

## Created

2026-10-18 23:57

## Request

用户反馈刚才测试图片无法输入到 OpenAI gpt-6.1-sol，要求检查原因和修复方式。

## Goal

依据最新真实日志、模型目录能力及请求处理定位图片输入失败，最小修复并验证，不把 API 的模型能力等同于订阅账号已可用。

## Scope

- 只读检查当前生产版本、近期请求和安全诊断字段。
- 图片能力声明、Codex 目录生成、Responses／WebSocket 输入校验、请求大小及订阅转发链。
- 根因清楚后补最小修复及回归，必要时使用自建无隐私测试图片和应用隔离账号做有界真实验收。

## Out of Scope

- 不改当前 Codex 配置或登录，不重启生产软件／Agent，不上传用户私人图片。
- 不自动发布或覆盖已发布 Beta.4。

# Analysis

官方模型页声明 gpt-6.1-sol 支持图片输入；Responses 图片使用 input_image 的 URL／data URL 等形式。此结论不是当前订阅链路成功的证明，需结合日志和真实实现检查。

# Implementation Plan

1. 检查当前运行版与最新请求、模型元数据及已生成目录的无敏感能力字段。
2. 对照图片传输与请求大小限制，建立根因回归并最小修复。
3. 定向／完整质量门，明确是否完成真实图片请求；记录配置未改动与生产未重启。

# Progress

- 当前生产已升级 Beta.4；23:53 两模型各有成功的真实请求。现有日志不记录图片正文，不能把这些成功记录当作图片已送达的证明，也不能把此前网络／凭据错误直接归为图片问题。
- gpt-6.1-sol 与 gpt-5.6-luna 的持久上游元数据 inputModalities=[text,image]；当前 Codex 本地目录却都是 input_modalities=[text]。根因是 core/modelcatalog.CodexJSON 硬编码 text。
- 修复只对原生 OpenAI Responses 线路且上游已确认 image 的条目（含继承目标的别名）声明图片；其他跨协议适配器仍不具备图片转换能力，不能一律继承多模态标记或猜测未知能力。
- Responses HTTP／WebSocket 的现有限制为 2 MiB。新增独立 16 MiB JSON 上限以容纳 Base64 图片，与已有 Messages 容量量级一致；保持 Chat Completions 2 MiB、JSON 深度／重复字段、loopback、鉴权、超时和并发限制。WebSocket 合并历史后再次校验总量。
- 自动目录刷新坚持外部改动保护，不通过覆盖用户共享文件来使测试变绿。只读计算确认 Codex 配置与 agent_backups 的已应用哈希不匹配，而目录与目录备份哈希匹配；因此当前自动刷新会在 config_changed 分支拒绝覆盖。需升级后由用户预览确认重新应用目录，并重新加载客户端。

# Implementation Result

## Changes

- CodexJSON 按原生 OpenAI 路线及已确认元数据生成文本＋图片能力，其他线路保持保守声明，不猜测或广告音频。
- 新增 Responses 专用 16 MiB 容量；HTTP、WebSocket 帧、历史合并与路由改写后边界一致；超限不进入上游，普通 Chat 仍为 2 MiB。
- 新增模型目录、真实 Apply／刷新／外部改动保护、大图 JSON 透明转发、别名、预热历史、重复字段、边界 413／1009 及统计测试；实际两内核 model/list 契约增加图片能力断言。
- 更新 LOCAL_AGENTS 的原生图片／跨协议区别、容量及重新加载说明。没有新增依赖或迁移，没有修改生产配置或安装软件。

## Real Validation

- 以自建无隐私的彩色几何 PNG（2,371,469 bytes，含无隐私噪声边带）生成 3,162,297-byte JSON，明确大于旧 2 MiB 限制。请求不在文本中提供颜色和形状答案。
- 使用真实应用 gateway.Service／routing／chatgpt adapter／proxy／activity，独立临时库和随机 loopback 端口；只读定位应用隔离账号，直接 UseToken 回调，不刷新登录或读取默认 Codex 登录。
- HTTP 与 WebSocket 实际 gpt-6.1-sol 各一次，均完整成功且识别红色三角形、蓝色圆形；输入用量 365、输出 11，首字与完整用量均记录。
- 通过真实 agentconfig.Service.Apply 在临时目录生成图片目录，再使用安装的 CLI 0.163.0 与 Desktop 内核 0.163.0-alpha.10.2 各发一次附件请求；exit=0、turn.completed、颜色与形状识别均为真，完整用量与首字存在。共四次真实图片请求。
- 第一次原生夹具因 --image 的多值参数吞掉后续提示词，两个客户端在调用模型前报告 No prompt provided via stdin；改用明确参数分隔符，并分离临时进程目录后通过。该问题属于 QA 命令构造，不是产品缺陷；没有用失败结果冒充成功，也未输出图片／模型正文。
- model/list 的真实两内核隔离契约也分别通过，确认图片元数据被客户端消费，而非只检查生成 JSON。
- 用户真实 config.toml 与受管目录哈希均未变化；生产仍为 Beta.4，未安装或重启。日志不含 data:image／input_image 正文。

## Remaining Scope

本地修复尚未发布或替换当前 Beta.4。升级修复构建后需刷新受管目录；若检测到外部改动，需用户预览确认后重新应用。已有客户端／会话可能保留旧模型能力快照，需重新加载；不会通过删除共享配置或改登录来处理。

# Verification

- RED：旧实现模型／别名仅输出 text、3 MiB Responses 请求被拒，根因回归实际失败。
- PASS：修复后模型目录、代理、配置服务定向测试；关键目录／大小／预热／路由边界与网关图片透传重复五轮。
- PASS：安装的两种 Codex 内核读取图片目录；四次真实图片请求及语义识别、统计通过。
- PASS：首轮 task --force verify：Go、104 文件／575 前端测试、格式、lint、类型及 Windows 生产构建。
- PASS：补齐 WebSocket 路由改写后的最后容量检查后，最终 task --force verify 再次完整通过：全部 Go、104 文件／575 前端测试、格式、静态检查、类型及 Windows 生产构建。原有分包大小／jsdom 提示不影响通过。


# Git

b2eb0a187969663552eda944e396c5ef01a54a20 fix(codex): 修复原生图片能力与请求容量

仅本地提交，未推送／发布。


# Evidence Sources

- https://developers.openai.com/api/docs/models/gpt-6.1-sol ：模型支持图片输入（不等于特定订阅权限保证）。
- https://developers.openai.com/api/docs/guides/images-vision?api-mode=responses ：Responses input_image 的 URL／data URL 形式。

# Documentation

只更新原生图片接入、容量和验证边界说明，不改变跨协议多模态限制，不需要新增 ADR。


# Result

根因已确认并在本地修复：原生图片能力被错误降为仅文本，另有独立请求容量不足。四次真实图片请求、两内核目录读取、临时配置保护及完整门禁通过。当前用户共享目录仍保持原样，需后续升级修复构建、受控重新应用目录并重新加载客户端后生效；本任务没有发布或自动安装。

# Completed

2026-10-19 00:22
