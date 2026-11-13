# 百度搭子 DuMate 个人版

OmniProxy 以 Go 原生适配接入百度搭子（DuMate）个人版账号，复用现有本地网关、
协议转换、账号选择与 DPAPI 凭据存储。没有引入 Magpie、Node／Bun 插件宿主或独立桥接服务。

## 账号与凭据

- 账号类型为 `dumate_cookie`，只能挂在内置提供商 `builtin_dumate` 下；
  不会与千帆 API Key、Token Plan 或 Google／Codex 订阅凭据混用。
- 登录态有两种来源，都由用户主动触发：
  1. **本机桌面端登录态**：只读解析 `%APPDATA%\qianfan-desktop-app\auth.json`
     与同目录 `.cookie-key`。密钥必须是 32 字节 AES-256 密钥，密文布局为
     `IV(12) + Tag(16) + Ciphertext`。可用环境变量覆盖目录：
     `OMNIPROXY_DUMATE_USER_DATA_DIR` 优先，其次是参考实现使用的
     `DUMATE_USER_DATA_DIR`。
  2. **粘贴浏览器 Cookie**：用户在 `console.bce.baidu.com` 的开发者工具网络面板
     复制请求头 Cookie 值。关键登录 Cookie（如 `bce-sessionid`）是 httpOnly，
     `document.cookie` 取不到。
- 导入时先用 `GET /api/dumate/user/info` 核对远端身份（`bceAccountId`）。
  身份相同才更新原账号；桌面端切换到另一个百度账号时新建账号，不会静默覆盖。
- Cookie 只写入 DPAPI 保护的凭据文件，SQLite 只保存随机引用；返回值、日志和
  前端缓存都不包含 Cookie 或 gateway token。账号预览不提供密钥片段。

## 网关调用

```text
客户端 → OmniProxy 网关 → 帐号选择 → 受保护凭据回调
       → /api/dumate/app/token/get 换取 gateway token
       → POST https://dumate-svc.baidu.com/gateway/apis/v1/chat/completions
```

- token 按「账号 + 凭据版本」缓存 1 小时、提前 5 分钟刷新，并发刷新合并；
  上游返回 401 只清除 token 缓存并让本次请求失败，不据此判定 Cookie 失效。
- 只访问固定 HTTPS 主机，禁止重定向，不透传客户端头部。
- 上游只确认 Chat Completions；Responses 与 Messages 由网关经既有转换链路
  在本进程缓冲后回包，**不是**上游原生协议。
- 流式请求补标准 `stream_options.include_usage`，用量进入现有请求统计。

## 模型

上游没有 `/models`，只公布参考实现使用的两条回退别名：

| 模型 ID | 展示名 |
| --- | --- |
| `model-text` | DuMate Auto（智能路由） |
| `dm-auto-model/text.L0` | DuMate Auto L0 |

上游会把实际底层模型隐藏起来（响应固定回显 `model-text`），因此界面不宣称
可指定 DeepSeek、GLM 等具体模型，也不发布上下文／输出容量——参考实现里的
192k／128k 未经验证，不写入目录。

## 积分与签到

- 积分查询走 `GET /api/dumate/points/quota_overview`（时区 Asia/Shanghai、
  desktop 客户端、忽略签到奖励），签到状态走 `GET /api/dumate/points/loginBonusInfo`。
- 查询结果区分三种状态：`live` 实时成功、`stale` 控制台暂时失败但有保留期内
  的成功快照、`unavailable` 无可用数据。失败不会显示成零余额。
- 快照只保存在进程内存（每账号一条、保留 1 小时、最多 64 条），不落库。
- 到期积分取「最近一个 `active` 积分包」的到期时间，并把可展示的到期积分
  限制为 `min(剩余积分, 该包总额)`，不把积分包原始总额当作确定剩余额。
- 手动签到：先读今日状态，已领取则返回 `done`；否则提交
  `POST /api/dumate/points/loginBonus`（需要域名过滤 Cookie、`csrftoken`
  与 XHR／来源头），提交后重新读取状态并核对到账：
  `claimed`（已到账，积分按实际差值计算）、`pending`（已提交但未核对到账，
  不宣称成功）、`failed`（带稳定错误码）。
- **自动签到本轮未实现**。启用方式仍待确认，不创建系统定时任务，也不在后台
  循环里消耗账号操作。

## 安全与边界

- Cookie 输入有界（≤64 KiB、≤200 条、单值 ≤4 KiB），拒绝控制字符与无效 UTF-8，
  防止请求头注入。
- 控制台与网关错误只映射为固定错误码（`sign_in_expired`、`console_unavailable`、
  `rate_limited`、`invalid_response` 等），不回显上游原文。
- 登录态失效要求重新导入；仅 gateway token 失效不等同于账号登录失效。
- 不自动破解验证码、不绕过登录校验、不读取与当前操作无关的本机账号。
- 非官方接口可能变更，使用受百度账号与服务条款约束；OmniProxy 不承诺长期可用
  或账号无风险。

## 已验证与未验证

- 已用隔离的假控制台／假网关覆盖：Cookie 解析与头序列化、桌面登录解密与
  错误分支、并发 token 刷新、401 后重新换取、Chat 转发与 Responses／Messages
  转换、积分状态机、签到四种结果、身份错绑与凭据不泄漏。
- **未使用真实百度账号验证**：真实接口可用性、模型别名、工具调用、推理用量、
  积分消耗与签到到账都尚未核对。真实验证需要用户明确授权，并会消耗积分、
  变更账号状态。
