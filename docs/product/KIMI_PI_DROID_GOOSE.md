# Kimi Code、Pi、Factory Droid 与 Goose

网关 → 客户端接入增加四个产品分组，复用现有模型选择、默认模型、独立接入 Key、加密备份、外部修改保护和恢复流程。Pi 指 [pi.dev 的 Pi coding agent](https://pi.dev/)，不指其他同名聊天产品。

| 产品 | 受管配置 | 接入协议 | 软件入口 |
| --- | --- | --- | --- |
| Kimi Code | `~/.kimi-code/config.toml`；已有旧 Python 版则选择 `~/.kimi/config.toml` | Chat Completions；当前版 `openai`，旧版 `openai_legacy` | CLI |
| Pi | `~/.pi/agent/models.json`、同目录 `settings.json` | `openai-completions`，显式 Bearer 鉴权 | CLI |
| Factory Droid | `~/.factory/settings.json` | `generic-chat-completion-api` | CLI |
| Goose | Windows：`%APPDATA%/Block/goose/config/config.yaml`，同目录 `custom_providers/omniproxy.json` | 自定义 `openai` provider，固定 Chat Completions URL | CLI／Desktop |

路径以详情和预览为准，未安装软件也可以先预览配置。安装仅提供固定官方说明／下载入口与只读检测，不执行远程安装脚本，也不自动安装或升级。Goose CLI 与 Desktop 共用配置及 Key，安装状态各自检测；Droid 的个人模型也可被其官方桌面版读取，但本次只提供 CLI 软件入口。

## 配置与模型

- 只管理专用 `omniproxy` 提供商或 Droid 的 `OmniProxy ` 模型项。同名未接管项会阻止应用，保留其他提供商、工具权限、登录和会话文件。
- 四者使用已有 Chat Completions 目录，按实际协议能力过滤；不写入仅支持 Responses 的型号。首版按现有网关的文本／函数工具范围接入，不声明图像能力。
- 完整目录随同步更新；选择部分模型后刷新只保留仍可用的所选模型。已接入列表读取实际受管文件，不将新增但尚未写入的网关模型冒充为已配置模型。
- Kimi Code 默认选择使用 `omniproxy/模型ID` 别名。窗口已知时使用目录元数据，未知时使用保守的 65,536 token 运行预算；该预算不表示厂商能力已核实。
- Pi 同时写入 `models.json` 的模型列表与 `settings.json` 的 `defaultProvider`／`defaultModel`。已有非空模型循环范围会补入所选 OmniProxy 模型，避免新默认模型被原范围隐藏，同时保留其他提供商范围；其余用户设置保持原值。`auth.json` 不参与读取或修改。
- Droid 追加自定义模型，保留已有模型的顺序；默认值采用官方 CLI 的 `custom:展示名称-数组下标` 形式。保留其他模型与设置，不修改组织策略；更高优先级的组织、项目或 `settings.local.json` 仍可能覆盖用户配置。
- Goose 使用独立 provider JSON，鉴权头仅包含本客户端的本地网关 Key，不覆盖系统 Keyring、`secrets.yaml` 或原 OpenAI 凭据。`requires_auth: false` 只关闭该 provider 内置的密钥查找，实际请求仍携带固定 Bearer 头并由 OmniProxy 鉴权。关闭动态目录发现，遵守用户的模型选择；Goose 模型条目只有 API 名称，已接入列表相应展示 ID。
- 保存后重新加载对应客户端；环境变量、启动参数及项目设置可能具有更高优先级。已有会话可继续采用它保存的旧模型选择。

## 路径与兼容性

Kimi Code 优先读取 `KIMI_CODE_HOME`；未设置时，兼容旧 Python 版显式 `KIMI_SHARE_DIR`。无显式覆盖时优先使用已有 `.kimi-code/config.toml`，其次已有 `.kimi/config.toml`，均不存在时采用当前版目录。旧 Python 版配置使用 `openai_legacy` 与其原有能力枚举，不混用新版字段。旧版只有 `config.json` 时应先由官方客户端完成 TOML 迁移；两套客户端同时使用不同数据目录时，接入仅作用于预览展示的一套配置。

Pi 支持 `PI_CODING_AGENT_DIR`；Goose 支持绝对路径 `GOOSE_PATH_ROOT`，配置位于其 `config/` 子目录。相对覆盖按 Goose 官方行为忽略。非 Windows 的默认路径仍按客户端平台约定解析，但本项目仅对 Windows 提供安装检测和原生验收。

预览版本绑定当前目录快照、客户端配置位置和 Kimi 协议变体，避免旧预览被应用到同字节的另一目录或另一版本。Pi、Goose 的两个文件共同生成版本摘要与加密备份；应用和恢复逐文件校验版本，失败回滚本次已提交的版本。任何关联文件被外部修改都会阻止静默刷新和恢复。失败回滚不等于文件系统事务，进程被强制终止或断电时仍可能留下部分写入，需重新读取并检查实际配置。原文件恢复按原字节和存在状态执行，不触碰其他文件。

## 官方依据

- [Kimi Code 配置](https://www.kimi.com/code/docs/en/kimi-code-cli/configuration/config-files)、[环境变量](https://www.kimi.com/code/docs/en/kimi-code-cli/configuration/env-vars)、[旧 Python 版路径源码](https://github.com/MoonshotAI/kimi-cli/blob/main/src/kimi_cli/share.py)、[旧版协议源码](https://github.com/MoonshotAI/kimi-cli/blob/main/src/kimi_cli/llm.py)。
- [Pi 配置](https://pi.dev/docs/latest/configuration)、[模型配置参考](https://pi.dev/docs/latest/models-configuration)。
- [Factory BYOK](https://docs.factory.com/model-independence/byok)、[CLI 自定义模型引用](https://docs.factory.com/droid-exec/overview)、[个人设置](https://docs.factory.com/droid-cli/settings)。
- [Goose 配置](https://goose-docs.ai/docs/guides/config-files/)、[自定义 provider 结构](https://github.com/aaif-goose/goose/blob/main/crates/goose-providers/src/declarative.rs)、[OpenAI 兼容实现](https://github.com/aaif-goose/goose/blob/main/crates/goose-providers/src/openai.rs)、[路径源码](https://github.com/aaif-goose/goose/blob/main/crates/goose/src/config/paths.rs)。
