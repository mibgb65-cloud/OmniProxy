# TASK-20261017-007: 接入 Kilo、Qwen Code 与 MiMo Code

## Status

Completed

## Created

2026-10-17（Asia/Singapore）

## Request

用户要求继续接入 Kilo Code、Qwen Code、MiMo Code 的 CLI 和 Desktop，提供了官方文档及仓库。
后续明确 MiMo Desktop 同时适配国内版、国际版；两版同一卡片，分别检测、打开并提供
对应官方入口，核对共享引擎配置范围。

## Goal

核实各产品真实配置与共享范围后，纳入本地 Agent 接入，支持可用模型、独立 Key、配置
预览／应用、备份恢复、检测及固定打开入口；没有官方依据的桌面端不虚构支持。

# Analysis

已核实三个官方产品及桌面形态。Kilo／MiMo Code 的本机引擎使用兼容 OpenCode 的
JSON(C) 提供商配置，分别按自身配置目录与覆盖环境选择文件，不读写 OpenCode 文件。
Qwen Desktop 通过普通 qwen serve 运行，沿用 QWEN_HOME/settings.json；新增独立模型
目录／环境引用配置，保留其他协议模型、权限、OAuth 文件和会话。
桌面宿主可能覆盖引擎目录或内联配置；MiMo 国内／国际版已核对下述发行版规则，不能把
相同引擎表述成任何桌面版本都必定共用一个默认目录。

补充本机只读依据：已安装国内版 `Xiaomi MiMo 26.922.222056`，卸载注册表缺少 Publisher
和 InstallLocation，但 DisplayIcon 指向固定命名的现存主程序。安装包的公开程序代码
包含 domestic/overseas 的 `Xiaomi MiMo`／`Xiaomi MiMo AI` 两个产品名，二者共用 Mr 的
全局引擎路径规则，分别使用 Electron 用户数据目录。配置不写入登录或桌面私有数据库。

真实 Kilo 7.8.12 发送 `content: [{type:"text",text:...}, ...]`，既有 Chat
校验仅接受字符串，导致配置正确但首个请求 `invalid_messages`。需扩展有界纯文本
数组归一化，保留原请求工具与其他字段，继续拒绝图片／未知类型；这是接入调用必需
的协议兼容扩展，既有字符串请求保持不变。

实际运行还发现三个客户端会首次补写 schema／Qwen `$version`，使正常启动后即发生
版本冲突。新配置补齐准确格式标识，Qwen 已有历史配置先要求官方迁移，避免自己
标记新版本而跳过权限迁移。真实运行后进一步执行 Inspect 与 Restore 复验该闭环。

# Implementation Plan

1. 核对三个产品的 CLI／Desktop 形态、配置格式、凭据位置及覆盖优先级。
2. 复用既有配置生命周期和保护机制扩展适配，保留用户设置与会话。
3. 增加按产品归组的入口、检测、固定官方说明和打开功能。
4. 完成回归、真实客户端隔离验证、原生窗口检查及 task verify。
5. 同步文档，审查差异，提交并归档。

# Implementation Result

- 新增 `kilocode`、`qwen-code`、`mimo-code` 三个配置 ID，分别使用独立本地 Key；复用
  预览、CAS、DPAPI 备份、恢复及受管目录刷新，不改数据库结构或引入依赖。
- Kilo／MiMo 复用 JSONC 提供商增量合并；Qwen 维护 OpenAI Chat 模型列表、专用 env
  引用和认证选择，保留其他协议、权限及 OAuth／会话。新文件补齐格式标识，历史 Qwen
  设置需先官方迁移。实际启动后检查与恢复通过。
- 三张产品卡片归组 CLI／Desktop。MiMo 国内／国际版各用固定软件目标和官方链接，
  分别识别 `Xiaomi MiMo`／`Xiaomi MiMo AI`；兼容本机缺少发布者／安装目录、名称带
  版本的 NSIS 项，拒绝错误地区、脚本参数、网络地址或不匹配的主程序。
- Chat 增加有界纯文本内容数组归一化以兼容真实 Kilo 请求，保留工具和扩展字段，
  拒绝图片／未知部分。其他现有字符串请求不变。
- 增加本地品牌素材及来源，修正 MiMo 深色主题可见性；更新产品和架构文档。

# Verification

- 适配回归覆盖预览无 Key 写入、协议过滤、独立鉴权、保留其他配置、模型／端口刷新、
  精确原字节恢复、拒绝外部改动、占用冲突、历史设置版本和路径覆盖优先级。
- 安装测试覆盖两地区、NSIS 元数据、固定主程序、非法参数；前端定向 24 项通过，覆盖
  三个产品预览应用、CLI／桌面打开、MiMo 两地区状态与官方入口、Qwen 迁移错误提示。
- 实际 Kilo 7.8.12、Qwen 0.25.2、MiMo 0.1.15 从隔离配置通过真实 OmniProxy 网关访问
  本机模拟上游：均完成 SSE、一次读取标记文件的工具调用、工具结果／思考内容续接、
  最终回答；请求数分别 2／2／3。启动后 Inspect 无配置变化错误、Restore 成功。
  证据 `tmp/kilo-qwen-mimo-20261017/acceptance-lifecycle.log`，无真实上游付费调用。
- Qwen 默认后台记忆提取会剥离历史思考，在严格模拟上游下失败；主对话已完成。
  隔离主调用验收在原测试设置中关闭自动记忆／整理，实现保留用户原设置，不保证
  第三方高级后台工作流。实际 Desktop 界面模型调用和国际版安装仍待对应版本验收。
- 本机实际检测：国内版 26.922.222056 安装于 `E:/MimoDesktop/Xiaomi MiMo`，国际版未装。
  只读核对国内版发行名映射、全局配置函数与覆盖注入，未修改原客户端文件。
- 原生三个产品配置预览可应用（未点击应用实际用户文件）。最终开发版正确显示 MiMo
  CLI／国内版已装、国际版未装，两版独立入口、MiMo 74 模型预览、深浅主题和 960×600
  布局均验收通过；原窗口尺寸／暗色主题已恢复。
- 中间质量门曾因新素材来源未格式化、开发热重载与构建竞争资源文件、测试共用错误
  mock 固定返回 failed 而失败，分别修正格式、隔离构建运行、补充可控 mock 后重验；
  没有删除校验或放宽合理测试。
- 默认高并发全量运行出现原有 `providers-view.test.tsx` 一次 5 秒超时，单独 8 项复验
  全通过；经本地 Vitest 实现核对支持 `VITEST_MAX_WORKERS`，以值 4 重跑相同完整质量门，
  未改测试内容或超时阈值。
- 最终 `VITEST_MAX_WORKERS=4 task verify BIN_DIR=tmp/kilo-qwen-mimo-20261017/verify-bin`
  退出码 0：格式、Go vet／staticcheck、ESLint、TypeScript、全部 Go 测试、88 个前端文件／
  491 项测试、生产前端及 Windows 原生构建全部通过。证据 `verify-workers4.log`。
- 开发热重载与生产构建竞争时已停止旧开发进程，生产进程保留。正常桌面启动器补齐
  Git 工具路径，Wails 的构建包装命令停滞后改用同参数 `task build DEV=true`；采用
  `tmp/.../dev.yml` 临时启动配置恢复热重载，不改仓库构建文件。实际共享目录实体路径
  已核对，95 个方法的新开发版保持运行。

# Git

实现提交：`c58d931930cf54edc4bfe0f0b4822f4edcf68846`
（`feat(gateway): 接入 Kilo、Qwen 与 MiMo 双版本客户端`）。
工作分支：`codex/river-builtin-provider`；未推送。

# Result

完成 Kilo、Qwen Code、MiMo Code 本机模型接入，MiMo 同一卡片支持 CLI 和国内／国际
两个桌面软件目标。完整质量门、实际三个 CLI 调用及配置恢复、原生状态／预览／布局
验收通过。桌面实际模型对话和 Qwen 后台记忆边界见 Verification 与产品说明，不把
配置写入、安装检测或模拟上游结果等同于真实厂商模型权限。原真实客户端文件未应用
本任务配置，既有无关未跟踪任务记录保持原样。
