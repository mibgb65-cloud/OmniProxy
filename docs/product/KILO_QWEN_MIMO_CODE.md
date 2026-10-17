# Kilo、Qwen Code 与 MiMo Code 接入

“网关 → 客户端接入”按三个产品各展示一张 CLI／Desktop 卡片，支持本机模型配置、
独立接入 Key、预览、应用、重新读取、加密备份／恢复和受管目录刷新。MiMo 的同一张
卡片包含 CLI、Desktop 国内版、Desktop 国际版三个软件入口。

## 配置文件与范围

| 产品／接入 ID | 默认配置文件 | 格式 |
| --- | --- | --- |
| Kilo Code／`kilocode` | `~/.config/kilo/kilo.jsonc` | OpenCode 兼容 JSONC 提供商 |
| Qwen Code／`qwen-code` | `~/.qwen/settings.json` | `modelProviders.openai` 模型列表 |
| MiMo Code／`mimo-code` | `~/.config/mimocode/mimocode.jsonc` | OpenCode 兼容 JSONC 提供商 |

路径使用当前 OmniProxy 进程的环境：

- Kilo 优先 `KILO_CONFIG`，其次 `KILO_CONFIG_DIR`，再用 `XDG_CONFIG_HOME/kilo` 或默认
  目录。同目录按照引擎合并顺序选择最后一个已有文件：`config.json` → `kilo.json` →
  `kilo.jsonc` → `opencode.json` → `opencode.jsonc`，避免已有兼容文件覆盖本次更新。
- Qwen 使用 `QWEN_HOME/settings.json`，未设置时使用默认目录。官方 Desktop 启动普通
  `qwen serve`，继承用户设置；`QWEN_RUNTIME_DIR` 只控制运行输出，不改变设置路径。
- MiMo 优先 `MIMOCODE_CONFIG`，其次 `MIMOCODE_CONFIG_DIR`；再用 `MIMOCODE_HOME/config`，
  或 `XDG_CONFIG_HOME/mimocode`／默认目录。同目录选择最后已有的 `config.json` →
  `mimocode.json` → `mimocode.jsonc`。这是公开引擎源码的实际规则，不能把 README 的
  Windows 简写当作所有宿主的固定目录。

MiMo 国内版是 `Xiaomi MiMo`，国际版是 `Xiaomi MiMo AI`。本机国内版 26.922.222056 的
程序代码同时定义这两个发行版，二者的全局模型配置均使用上述 MiMo 引擎路径，分别保留
自己的 Electron 数据目录、登录和会话。一次配置使使用该全局目录的 CLI／两版 Desktop
均可发现 OmniProxy，仍需在各客户端选择模型。

Kilo Desktop 启动本地 Kilo 引擎；MiMo Desktop 内嵌该引擎，宿主可覆盖数据根目录、注入
内联配置。**本功能配置预览显示的当前本机引擎，不保证任意桌面版本都读取同一目录。**
独立桌面数据根目录、项目配置、组织策略、内联与启动参数可能优先。应核对实际配置
路径并重新加载客户端，选择 OmniProxy 模型。未得到用户桌面版本依据时，不猜测和写入
桌面私有目录；不能将写入全局配置成功表述成桌面调用验收通过。

## 模型与凭据

只同步网关可用的 Chat 文本／函数工具模型和路由，Responses 独占模型不进入目录。
默认保留仍可用的原选择，否则使用首个可用型号；只传递已确认的容量元数据。

Kilo／MiMo 合并 `provider.omniproxy`，使用本地 `/v1` 和 `@ai-sdk/openai-compatible`，
保留其他提供商、权限、插件及 JSONC 注释。MiMo 设置 `only_configured_models=true`，
避免引擎自动混入不属于该提供商的型号。同名提供商已有其他地址且不受管时拒绝接管。
新文件补齐对应客户端的 schema，避免客户端首次启动补写导致受管版本变化。

Qwen 在 `modelProviders.openai` 中加入显示名为 `OmniProxy · 模型名` 的条目，使用
`wireApi=chat-completions`、本地地址及专用 `envKey=OMNIPROXY_QWEN_CODE_KEY`。本地 Key
位于用户设置的同名 `env` 项，选择 `security.auth.selectedType=openai`、本地 `baseUrl`
与默认 `model.name`。其他模型（包括其他地址下的同 ID）、协议、环境项和设置保留；
专用引用已被非受管内容占用时拒绝覆盖。Qwen 的解析器要求标准 JSON，应用会重新格式化
设置并移除注释；加密备份保留完整原字节，恢复时包括原注释。

新建 Qwen 设置使用已验证的 `$version:4`；已有配置未迁移、没有版本或版本不受支持时，
明确提示先启动当前版官方客户端完成迁移，再重新读取，不能跳过官方迁移后覆盖旧权限
字段。已有 v4 设置保留其其他内容，避免首次启动自动迁移让恢复版本立刻失效。

三个产品分别使用独立的本地 `opk_` Key，不包含任何上游账号凭据。OmniProxy 数据库
仅保存 DPAPI 引用；客户端按自己的官方格式保存本地 Key。只写上述模型配置，不读写
OAuth 凭据、官方登录或会话。预览不返回 Key 或完整原文件。版本变化后拒绝自动刷新
或恢复覆盖，需重新读取并预览；端口改变时仅刷新已受管配置。

Kilo 7.8.12 会把文本拆为多个 `text` 内容部分。网关在校验前将其合并为字符串，再交给
文本上游；每条最多 128 部分、合并文本最多 1 MiB，图片和未知类型明确拒绝，原有工具
与扩展参数保留。此兼容不代表新增图片输入能力。

## 安装与打开

CLI 检测固定 `kilo`、`qwen`、`mimo` 命令；桌面检测固定产品名与发布者的安装元数据。
已安装时提供固定打开入口，不接收前端命令片段或任意路径，不附带任务与凭据。
Portable、非标准名称和宿主专用安装可能需要从原入口打开；配置不依赖检测成功。

七个新增软件目标只提供官方安装说明，未虚构 WinGet 包或执行远程脚本。MiMo Desktop
两个目标分别指向[国内版](https://mimo.xiaomimimo.com/desktop/invite/)和[国际版](https://mimo-ai.xiaomimimo.com/desktop/invite/)，不能把一版的安装状态当作另一版。

MiMo 的 NSIS 安装信息可能缺少 Publisher／InstallLocation，名称附带版本。适配会校验
精确产品名、合法版本及 DisplayIcon 指向的固定现存主程序（国内 `Xiaomi MiMo.exe`、
国际 `Xiaomi MiMo AI.exe`），从该文件确定安装目录；不执行图标参数或卸载命令。
这属于本地安装检测，不是代码签名验证。安装任一桌面版和登录厂商账号不是自动执行的操作。

## 验证

2026-10-17：使用 Kilo CLI 7.8.12、Qwen Code 0.25.2、MiMo CLI 0.1.15 的实际运行时，
从隔离目录读取本实现生成的配置，通过实际 OmniProxy 网关访问本机模拟上游。三者
均完成 SSE、读取标记文件、工具续接、思考内容回传及最终回答；运行后检查受管版本
未被客户端改写，并成功恢复原配置。没有使用真实上游账号或修改真实客户端文件。

Qwen 0.25.2 的后台 managed auto-memory 另行请求会丢弃历史思考内容，严格模拟上游会
拒绝该请求；主对话已完成且进程正常退出。上述主调用隔离验收仅在测试原设置中关闭
自动记忆／自动整理，接入实现保留用户这些设置，不宣称后台记忆兼容性也已验证。

完整测试和质量门结果见 TASK-20261017-007。实际 Desktop UI 调用、桌面宿主的
私有目录覆盖、真实上游权限与所有高级参数，需对应安装和版本进一步验收。

## 依据

- [Kilo Desktop](https://kilo.ai/docs/desktop/overview)、[AI 设置](https://kilo.ai/docs/desktop/settings/ai)、[CLI](https://kilo.ai/docs/code-with-ai/platforms/cli)。
- [Qwen 模型配置](https://github.com/QwenLM/qwen-code/blob/main/docs/users/configuration/model-providers.md)、[设置路径](https://github.com/QwenLM/qwen-code/blob/main/packages/core/src/config/storage.ts)、[官方 Desktop](https://github.com/QwenLM/qwen-code/blob/main/packages/desktop/README.md)。
- [MiMo Code 产品说明](https://github.com/XiaomiMiMo/MiMo-Code/blob/main/README.md)、[中文及国内版入口](https://github.com/XiaomiMiMo/MiMo-Code/blob/main/README.zh.md)、[引擎全局目录](https://github.com/XiaomiMiMo/MiMo-Code/blob/main/packages/shared/src/global.ts)、[配置合并](https://github.com/XiaomiMiMo/MiMo-Code/blob/main/packages/cli/src/config/config.ts)。
