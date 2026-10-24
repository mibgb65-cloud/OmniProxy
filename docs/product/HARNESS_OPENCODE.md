# DeepSeek Harness 与 OpenCode 接入

本地 Agent 页面新增 OpenCode、DeepSeek Harness 两张卡片。入口复用配置预览、独立
接入 Key、DPAPI 加密备份、重新读取、恢复和受管模型目录刷新；只配置本机客户端。

## 配置范围

| 客户端 | 接入 ID | 配置文件 | 范围 |
| --- | --- | --- | --- |
| OpenCode CLI／Desktop | `opencode` | `~/.config/opencode/opencode.json` 或已有 `opencode.jsonc` | 两个界面共用全局提供商配置 |
| Harness Web | `harness-web` | `$DSH_HOME/profiles/web/cordis.patch.yml` | `dsh web` 的 profile |
| Harness Desktop | `harness-desktop` | `$DSH_HOME/profiles/desktop/cordis.patch.yml` | 官方桌面应用的 profile |

`DSH_HOME` 默认 `~/.dsh`。OpenCode 优先使用显式 `OPENCODE_CONFIG`，其次
`OPENCODE_CONFIG_DIR`，再使用 `XDG_CONFIG_HOME/opencode` 或上述默认目录；同目录已有
JSONC 时优先修改该文件。拒绝相对路径、链接替换、超限文件及无法安全解析的内容。

OpenCode 仅合并 `provider.omniproxy` 并选择该提供商中仍可用的原默认或首个可用型号。
保留其他提供商、权限、插件和界面设置，JSONC 中的其他注释保留。同名提供商指向其他
服务时拒绝接管；已有受管记录的配置可随网关端口更新地址。项目配置、环境内联配置、
组织策略和远程后端可能覆盖本机全局设置。

Harness 仅修改目标 profile 中 `llm-pi-ai` 的 `config.providers.omniproxy`，保留同文件
其他提供商和插件。主目录补丁、启动覆盖及旧版用户设置可具有更高优先级。应用后在
Harness 模型选择器中选择 OmniProxy；不修改原默认选择或既有会话的模型。

## 使用

1. 添加提供商账号，准备可用模型／路由；账号同步会沿用已有的受管目录刷新。
2. 在“网关 → 客户端接入”打开 OpenCode，或 Harness 对应的 Desktop／Web 入口。
3. 点击“一键配置 → 配置预览”，核对文件路径、模型范围和配置范围后应用。
4. OpenCode 重新加载客户端；Harness 依据是否启用 HMR 在下一请求或重启后生效。
5. 启动 OmniProxy 网关，从客户端的 OmniProxy 提供商选择模型。

两款产品均通过本地 Chat API 接入，提供文本／函数工具模型，并按上游已确认能力声明图片输入。携带图片的路由在 Harness／Pi 写 `input: [text, image]`、OpenCode 写 `modalities.input`、Goose 写 `supports_vision: true`；未确认图片能力的模型保持纯文本，不因同目录其他模型支持图片而放宽。容量仅传递已有已确认元数据，未知容量使用客户端本身的回退值，不据此宣称长上下文权限。托管工具、全部高级参数和真实账号模型权限不由目录保证。

目录声明的图片能力只表示「允许客户端发送」，实际能否识别由上游模型决定；`system`／`assistant` 消息中的图片仍由上游按各自官方限制拒绝。

## 凭据、备份与恢复

OpenCode 配置只含独立本地接入 Key，不含任何上游账号 Key。Harness 模型配置仅保存
`apiKeyEnv` 引用，实际本地接入 Key 写入 `$DSH_HOME/.credentials.yaml` 的独立引用：
`OMNIPROXY_HARNESS_WEB_KEY`、`OMNIPROXY_HARNESS_DESKTOP_KEY`。

保留已有的 version-1 `refs`／`records` 布局；兼容 rc.7 的扁平引用布局，新建文件使用
新旧客户端都能接受或自动迁移的扁平布局。新版 Harness 的格式迁移不改变单个引用的
受管版本。未来不认识的版本明确拒绝，不把不可解析文件当作空文件覆盖。

OmniProxy 的本地 Key 和原配置备份继续使用 DPAPI，数据库只保存凭据引用。Harness
备份将本 profile 配置和本 profile 的一个凭据值作为整体，不保存其他凭据供恢复覆盖。
恢复 Web 不会移除 Desktop 或其他凭据；外部修改本 profile／本引用后，刷新和恢复均
拒绝覆盖，需重新读取并预览。配置和凭据文件分步写入，失败按已写版本回滚；数据库
提交失败也恢复本次文件内容。两个文件不具备操作系统级的跨文件崩溃原子性。

客户端自行保存的本地接入 Key 遵循其官方文件格式，不能把客户端文件当作上游密钥存储。
预览只返回受管摘要与公开模型元数据，不返回 Key、完整原配置或既有授权。

## 安装与打开

检测固定的 `opencode`／`dsh` 命令和官方桌面安装元数据；npm 启动脚本只确认存在，
不执行脚本探测版本。新产品目前提供固定官方安装说明，未配置未经核实的 WinGet 包名，
不会自动执行 npm／远程脚本安装。已安装时提供“打开”，只启动固定软件及 Harness Web
所需的 `web` 参数，不把任务、路径、凭据传入命令。Portable、非标准安装或远程客户端
可能需要用户从原入口打开，配置操作不以安装检测成功为前提。

Harness Desktop 的 Windows NSIS 安装记录可能使用带版本名称（如
`DeepSeek Harness 0.2.0-rc.2`）且发布者为空。检测兼容这种记录：校验精确产品名和
合法版本，并从 DisplayIcon 确认固定的现存 `DeepSeek Harness.exe`，据此定位安装目录。
支持注册于其他盘符／自定义目录的安装，不执行图标参数或卸载命令。已知发布者的原
安装记录继续识别；没有安装记录的 Portable 仍不进行全盘扫描。

## 验证记录

2026-10-17：本机 Harness `0.1.0-rc.7` 与隔离安装的 OpenCode CLI `1.19.3` 读取本实现
生成的配置，通过实际 OmniProxy 网关访问本机模拟上游。两者均完成 SSE 响应、读取
隔离工作区标记文件、工具结果续接和最终回答；增加 `reasoning_content` 必须完整回传
的检查后亦通过。使用真实客户端运行时，没有调用付费模型或读取／修改真实客户端会话。
Desktop 的共享配置／profile、原生 OmniProxy 入口和预览已验证，两个产品的实际 Desktop
界面调用及所有真实上游权限仍待使用对应安装验证。

## 依据

- [Harness 模型配置](https://github.com/deepseek-ai/deepseek-harness/blob/master/docs/user/guide/providers.md)
- [Harness 凭据布局](https://github.com/deepseek-ai/deepseek-harness/blob/master/packages/credentials/credentials-local/README.md)
- [Harness Desktop](https://github.com/deepseek-ai/deepseek-harness/blob/master/apps/desktop/README.md)
- [OpenCode 配置](https://opencode.ai/docs/config/)、[自定义提供商](https://opencode.ai/docs/providers/#custom-provider)
