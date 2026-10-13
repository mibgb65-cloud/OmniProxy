# TASK-20261008-005: ChatGPT 订阅账号登录

## Status

Completed

## Created

2026-10-08 16:06（Asia/Singapore）

## Request

OpenAI 账号需要支持登录 ChatGPT 订阅账号用于 Codex。用户确认实施：浏览器授权、设备码备选、多账号隔离、安全凭证存储、状态／套餐、刷新及退出清理。

## Goal

在本地 OmniProxy 中完成官方 Codex 授权的账号管理闭环，保持 API Key 功能兼容，并保护用户现有 Codex 登录。

## Analysis

- 本机已安装官方 codex-cli 0.155.1，官方 app-server 提供 ChatGPT 浏览器／设备码登录、账号读取、刷新和退出接口。
- 采用本机官方登录组件，每个账号使用独立 CODEX_HOME，强制 keyring 模式，凭证由系统凭证库管理；SQLite 只存元数据与引用，OmniProxy 的 DPAPI 引用保存隔离配置定位信息。
- 不读取或覆盖用户原来的 ~/.codex/config.toml 或 auth.json；不自行拼装 OAuth、抓取 Cookie 或将订阅凭证视作 API Key。
- 本次先实现账号登录管理；Codex 实际推理调用、Responses 网关和 Claude 仍按后续顺序实施。

## Implementation Plan

1. 检查当前版本官方协议和本地能力，创建隔离授权与凭证生命周期管理。
2. 新增兼容迁移及账号认证类型，阻止订阅凭证进入 API Key 访问／编辑路径。
3. 接入 Wails 授权会话与状态事件，增加 OpenAI 登录入口、浏览器／设备码说明、套餐与刷新／退出／重新登录。
4. 验证取消、超时、多账号隔离、删除清理、输入边界与隐私；运行 task verify，恢复开发环境。
5. 实际授权由用户在官方页面完成，记录实际验证与剩余边界。

## Implementation Result

- 新增迁移 004，旧账号默认为 API Key；元数据只暴露认证方式、身份、套餐与状态，不返回凭证引用或令牌。
- 本机官方 Codex app-server 的 stdio 适配支持浏览器／设备码登录、取消、读取、刷新与退出；固定参数强制 keyring，环境变量隔离已有 API Key／令牌，官方输出不进入应用日志。
- 每个账号使用缓存目录下的随机 CODEX_HOME，开发／生产隔离，不读取或修改用户默认 Codex 配置。DPAPI 仅保存当前及待清理旧配置定位。
- 授权超时、取消、关闭应用、身份不一致、重新授权原子切换、退出及删除具备清理路径；失败临时授权保留标记，下次授权前重试，旧配置清理失败保留引用。
- Wails 提供 6 个订阅方法与无敏感载荷的 subscription:updated 事件，浏览器打开只允许当前会话中的官方 HTTPS URL；删除入口按认证类型选择清理路径。
- OpenAI 账号新增“登录 ChatGPT”，提供浏览器／设备码选择和状态反馈；详情显示身份／套餐与刷新、重新登录、退出操作。订阅账号不显示密钥预览／复制／替换，现有 API Key 功能兼容。
- 更新产品说明、架构与技术说明，新增 ADR-003；原网关、路由和日志的改动保持原状。

## Verification

- `task verify BIN_DIR=C:/Users/mimanchi/AppData/Local/Temp/omniproxy-subscription-qa-81c24f00/quality-bin`：通过格式检查、go vet、staticcheck、ESLint、类型检查、全部 Go 测试、30 文件／187 项前端测试及生产构建。
- 新增 Go 测试覆盖多账号隔离、API Key 类型保护、取消保留原授权、身份不一致拒绝、重新授权旧配置清理、刷新／退出、删除、失败清理重试与输入边界。Windows CGO 未启用，按照 TESTING.md 不运行 race detector。
- 本机 codex-cli 0.155.1 真实隔离探针：initialize、account/read（未继承默认账号）、浏览器授权启动、取消、退出均通过，未生成 auth.json；探针不打开浏览器、不提交真实用户密码、不发送模型请求。
- 前端交互测试覆盖官方浏览器打开、设备码、完成后刷新账号列表、关闭取消、错误脱敏与订阅操作不读取 API Key。
- Chrome / Playwright 预览：中文亮色、英文暗色，960×600、1200×800、1920×1080、2560×1392 的详情／弹窗，加账号总览共 12 种布局；无页面横向溢出，事件完成、设备码及关闭取消通过。预览使用测试元数据与服务替身，未实际授权。
- Windows 桌面 UI Automation：真实开发窗口打开账号页及“登录 ChatGPT”弹窗，确认浏览器／设备码选项并关闭，通过。
- `git diff --check`：通过；已恢复标准 `task dev`，Vite 9245 / Control API 3891 正常监听。
- QA 脚本、截图和构建输出位于系统临时目录 `omniproxy-subscription-qa-81c24f00`，未加入仓库。启动本身没有新增任务记录。

## Git

未提交。当前工作树已有网关、路由、日志的大量未提交文件，与 AppService、生成绑定、语言和状态文档交叉；按 Git 安全规则保留既有修改，避免合并提交不属于本次任务的内容。无 push、reset 或历史改写。

## Result

已实现 ChatGPT / Codex 订阅授权的账号管理闭环，开发环境运行中。真实账号授权完成由用户在官方页面执行；本次没有代登录或宣称真实订阅推理成功。下一阶段接入 Codex 推理与 Responses 网关。
