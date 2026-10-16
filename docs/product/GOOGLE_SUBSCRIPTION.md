# Google AI / Antigravity 订阅

Google 账号支持普通 Gemini API Key 和独立 Antigravity 订阅登录。两条通道的
凭据、接口与额度互相独立；Google AI Plus／Pro／Ultra 的名称不代表实际模型权限。

## 使用

1. 首次登录需在本机安装兼容的官方 Antigravity CLI，确保 PATH 中可找到 `agy`。
   当前本机识别验证版本为 `agy 1.0.0`；不自动安装或升级。
2. 在“账号 → Google”点击“登录 Google”，名称选填，在 Google 官方页面授权。
3. 关闭弹窗保留后台授权，重新打开可恢复；“取消”终止回调。授权最长等待 10 分钟。
4. 成功后可刷新授权、退出或重新登录。重新登录只能替换为同一 Google 身份，
   新身份需要新增账号。退出只清除 OmniProxy 的本机授权。
5. 在路由页选择该订阅账号同步模型。目录使用当前账号上游实际返回的 Gemini
   文本生成模型，不按套餐名补充模型。重新授权使旧目录失效，普通 token 刷新保留目录。
6. 通过网关 Chat Completions 调用真实模型 ID 或路由别名。
   若普通 API Key 和订阅存在同名模型，可创建固定该订阅账号的路由规则来明确使用
   订阅通道；新增账号不会改变既有的默认账号选择顺序。

## 当前能力与边界

- 支持文本、函数工具、工具结果、JSON 与增量 SSE；工具不透明思考签名通过
  `tool_calls[].extra_content.google.thought_signature` 往返保存。
- 首版只开放 Gemini 文本模型。Claude、图片／语音等多模态、Responses 与 Messages
  尚未开放；不支持的内容或选项明确返回错误，不静默改变语义。
- 工具调用在流结束时输出完整参数与签名；普通文本逐段输出。缺少终态的流返回失败。
- 思考正文暂不公开，思考 Token 纳入输出用量；本机用量沿用现有记录与统计。
- 官方订阅剩余额度暂未知，不展示 Codex 的额度窗口或重置卡。
- 不请求付费积分回退，不切换普通 Gemini API Key 重试；上游限流返回 429。
- 本实现使用 Antigravity 客户端内部协议，接口兼容性与真实账号权限需持续验证。

## 安全

首次登录有界读取本机官方 agy 内的公开桌面 OAuth 配置，校验已知客户端 ID 与
配置指纹，不执行程序，不读取已有用户登录。仓库只保存公开标识和指纹，不含 secret。
无法识别组件时提示缺失或不兼容，不能降级到其他客户端配置。

授权使用随机 state、PKCE、一次性 loopback 回调和有限超时。账号 token、刷新 token
及所需客户端配置全部通过既有 DPAPI 存储；数据库只保存引用和白名单身份元数据。
授权信息不进入日志、请求 URL、前端凭证接口或客户端配置。网络只访问固定 Google
HTTPS 端点且拒绝重定向；授权 URL 仅允许进行中的固定 Google 登录地址。

内部协议兼容版本标识沿用参考实现的 `2.23.0` 下限，并在 User-Agent 保留 OmniProxy
身份；此值不是已安装 agy 的版本或账号权限证明。参考源码采用 MIT 许可证，
本次独立实现沿用标准库及已有模块，未复制其翻译器或令牌存储代码。

设计决策见 [ADR-013](../decisions/ADR-013-google-antigravity-subscription.md)。
参考项目固定版本：
https://github.com/router-for-me/CLIProxyAPI/tree/e7003d2a3e2cee2d45f9950c067bf8e02a2ac0f8

## 验证记录

2026-10-16：实际独立浏览器授权与开发 profile 的 DPAPI 存储成功，发现 14 个支持模型；
固定新账号通过本机网关调用 `gemini-2.5-flash` 返回 HTTP 200，输入 6／输出 1 Token，
本机用量记录一致。高级模型与真实多轮工具／长窗口仍需按实际账号验证，不由目录推断。
OAuth、并发刷新、身份变化、目录版本、工具签名两轮、SSE 中断与错误脱敏由契约测试覆盖。
