# ADR 005: 保留本地 Agent 的原提供商与会话身份

Status: Accepted

Date: 2026-10-09

## 背景

用户要求 Codex／Claude Code 一键接入，并保留官方会话。改变 Codex 的 `model_provider` 会改变会话筛选条件。实机 Codex 0.155.1 已禁止覆盖 `[model_providers.openai]`；桌面 app-server 不读取 CLI 的 CODEX_API_KEY 环境覆盖。

## 决策

- 不修改 Codex `model_provider`、CODEX_HOME、官方登录文件、会话数据库或索引。
- 内置 `openai` 使用 `openai_base_url` 指向 `/agent/codex/<本地接入Key>/v1`。保留官方认证头，但网关只校验独立接入 Key，并丢弃客户端原认证头。提供商表不会被重定义。
- 已有自定义提供商保留其标识，只合并地址、协议和接入认证字段；认证助手或活跃 profile 存在冲突时明确拒绝。
- 网关仅监听 loopback，禁止浏览器跨来源请求和任意地址转发。接入 Key、原配置备份存于 DPAPI；数据库只存随机凭据引用。配置文件含本地接入 Key，不含本应用的上游凭据。所有日志出口额外屏蔽 `opk_` 格式；前端预览不展示 Key 或原配置中的秘密。
- Claude CLI 合并用户设置的 env，保留其余设置、官方登录和会话根目录。Claude Desktop 添加独立 3P 配置库条目；用户明确应用一键配置时，同步启用 `deploymentMode=3p` 并设置 `appliedId`。保留原模式与配置选择用于恢复，只合并受管字段，不改写官方登录、原有配置条目或会话文件。
- 应用前校验文件版本，受管文件若被其他工具修改则拒绝覆盖和恢复。备份加密，恢复后停用对应接入 Key。
- ChatGPT 授权由官方隔离进程刷新；后端在内存中兼容旧 keyring 与新 `codex_auth.age` 读取。只读取 CodexAuth 命名空间，age 标准库负责解密；KDF 上限为官方当前使用的 20。Windows 以 GetFinalPathNameByHandle 匹配 Rust canonicalize，正确处理 junction。

## 验证和限制

2026-10-12 更新：按用户要求对齐旧项目的一键启用行为，替代原先要求手动选择配置的决定。启动模式与当前配置的切换来自用户明确应用操作；原选择和模式可恢复，原会话文件保留。见 TASK-20261012-006。迁移 014 只保存这些非敏感控制字段，客户端密钥及原配置仍使用 DPAPI 备份。

同次实机验证确认 Claude 客户端过滤非 Claude 模型 ID。配置适配使用稳定的兼容路由名和实际显示名称；应用层先解析当前目录，再复用现有精确路由，不持久化另一份路由表或覆盖用户规则。目录移除、账号停用和路由错误继续阻止调用。

测试目录实机 app-server 使用 ChatGPT 测试身份时，原认证文件保持原字节，turn 成功，按 `openai` 筛选仍能找到原会话。CLI 原标识亦保持不变。没有改写用户真实默认客户端配置。

本地专用路径不可作为公网接口。第一期跨协议提供文本／工具调用，ChatGPT Responses 原生流式；跨协议先收完整响应再输出 SSE。DeepSeek 关闭跨协议思考，避免无法恢复原生 reasoning_content 的工具续接。多模态、托管搜索等不可转换能力明确报错。官方私有订阅接口及授权存储变化需要持续维护。
