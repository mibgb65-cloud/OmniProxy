# TASK-20261008-006: Codex 浏览器登录兼容性排查

## Status

Completed

## Created

2026-10-08 17:57（Asia/Singapore）

## Request

用户反馈浏览器选择 ChatGPT 账号后出现 Operation timed out，并要求参考 `E:\go\new\OmniProxy` 的登录实现。

## Goal

对照参考项目核对授权流程，修正已确认的环境隔离缺口，并继续定位真实浏览器授权超时，保持凭证安全。

## Analysis

- 原项目在 `OmniProxyBackend/internal/proxy/codex_refresh.go` 固定授权端点、scope、PKCE、简化流程与 originator=codex_vscode，并自管回调和令牌交换。
- 当前官方 CLI 实际生成的授权端点、client_id、redirect_uri、scope、PKCE 方法和简化流程与参考项目一致；无需自行复制 OAuth 或更改回调协议。
- 启动开发环境的 Codex 桌面进程设置 CODEX_INTERNAL_ORIGINATOR_OVERRIDE=Codex Desktop。当前子进程继承此变量，initialize 中的 OmniProxy 客户端身份被覆盖，实际授权链接 originator=Codex Desktop。
- 真实网页登录仍停在官方 choose-an-account，回调端口正常。该环境隔离缺口可复现，但是否是网页超时的全部原因尚未确认，需用户实际授权验证。
- 用户确认旧项目同电脑、同账号浏览器登录成功，而当前版本仍超时。剩余已测得的浏览器参数差异是旧版 originator=codex_vscode，当前官方组件按 clientInfo 生成 originator=omniproxy。将仅对旧版授权端点的浏览器来源参数进行兼容映射，保持官方组件真实客户端身份和授权凭证生命周期。

## Implementation Plan

1. 将授权子进程环境隔离独立为文件，过滤启动器的来源、登录客户端和签发方覆盖变量，保留网络代理及证书配置。
2. 增加继承环境回归验证，用已安装官方组件检查生成链接的客户端身份与授权参数，不输出 state、code 或令牌。
3. 运行 task verify，恢复开发窗口；更新订阅说明、任务记录和状态看板。
4. 用户真实对照仍失败后，增加旧版浏览器流程兼容映射，仅改变 originator，验证所有安全参数保留，再请用户确认实际授权。
5. 浏览器实际账号选择返回 200 并到达 localhost:1455/success，用户确认网页授权成功，但应用未保存账号；继续定位回调后的协议／元数据保存失败，新增仅含状态和稳定错误码的诊断。
6. 用户截图显示旧会话 callback 连接拒绝，当前重启后的应用无新会话／1455 监听。停止在授权进行时重启开发环境；关闭弹窗改为保留后台授权，提供当前会话恢复与后台完成后的列表刷新，显式取消／应用退出／超时才停止回调。
7. 新会话实际网页成功，日志明确 state=failed / code=account_unavailable，SQLite 仍无订阅账号。修正完成通知之后的官方账号刷新／缓存同步，并保留成功授权供元数据读取重试，不能直接注销刚取得的有效授权。

## Implementation Result

- 独立 environment.go，新增过滤 CODEX_INTERNAL_ORIGINATOR_OVERRIDE、CODEX_APP_SERVER_LOGIN_CLIENT_ID、CODEX_APP_SERVER_LOGIN_ISSUER；保留既有凭证变量隔离以及 HTTPS_PROXY、CODEX_CA_CERTIFICATE、SSL_CERT_FILE。
- 官方组件的 initialize 仍真实标识客户端为 OmniProxy；没有复制自管令牌交换或改变 keyring / DPAPI 存储。
- 增加启动器身份／签发方不继承、网络配置保留的回归测试，扩展本机官方 CLI 探针，核对实际授权链接的来源与 PKCE / 简化流程参数。
- 浏览器兼容适配只在 auth.openai.com/oauth/authorize 映射 originator=codex_vscode，对齐已成功登录的参考实现；保留全部其他字段及重复查询值，不改变 SDK 真实请求归属、设备码或其他端点。

## Root Cause

实际验证明确了两处生命周期问题：开发重启／弹窗关闭会结束待完成授权，使旧浏览器返回时 1455 无监听；官方完成通知之后读取账号未同步到身份时，原实现立即 Logout 并删除隔离配置，导致网页成功但账号无法保存。最终保留后台授权，强制同步官方缓存，并在新增账号元数据暂不可用时保存有效授权引用、明确标记待确认且允许刷新恢复。来源环境继承及旧入口兼容一并修正，但没有证据单独将官方网页 Operation timed out 归因于 originator。

## Verification

- 对照原项目与本机官方 CLI 0.155.1：授权端点、client_id、redirect_uri、scope、PKCE 和简化流程一致；显式 codexStreamlinedLogin 不改变此版本生成链接，不添加无效修复参数。
- 真实组件探针模拟 Codex Desktop 来源及错误登录客户端／签发方覆盖，过滤后 initialize、account/read、浏览器授权启动／取消／退出通过；链接 originator=omniproxy，PKCE、state 与简化流程保持，未生成 auth.json。
- 新增环境回归测试：覆盖变量不继承，HTTPS_PROXY 与两种 CA 配置保留。
- task verify（临时 quality-bin）：格式、静态检查、类型检查、全部 Go 测试、30 文件／187 项前端测试及生产构建通过。
- git diff --check 通过；最终 map 仅调整排版，并已 gofmt。
- QA 输出在系统临时目录 omniproxy-auth-environment-qa-20261008；真实用户授权仍待重试结果。
- 标准 task dev 已恢复，开发应用 PID 45820，Vite 9245 / Control API 3891 正常监听；Windows UI Automation 验证登录入口、浏览器／设备码选择和关闭弹窗通过。已向用户请求用新会话重新授权验证网页超时，任务等待实际反馈。
- 用户用新会话重试后仍出现 Operation timed out：来源环境继承修复不能解决当前网页超时，不将其作为已确认的网页根因；继续比较参考项目与官方组件、浏览器及网络。
- 用户确认参考项目同账号能够登录后，补充兼容映射：单测验证仅 originator 改变，其他授权字段（含多值查询参数）严格保留；非旧版官方端点不修改，非法 URL / 查询编码拒绝。真实 CLI 探针生成兼容链接、取消和退出通过。
- 兼容映射后再次运行完整 task verify（临时 compatibility-bin）：全部 Go 测试、30 文件／187 项前端测试、静态与类型检查和生产构建通过；开发环境重新启动加载映射。
- 兼容版本开发应用 PID 24456，Vite 9245 / Control API 3891 正常；桌面登录入口和弹窗检查再次通过，已请求用户验证新的浏览器会话。
- 用户再次确认兼容映射后仍超时。检查 Chrome 实际授权导航记录，最新 URL 的 client_id、redirect_uri、scope、PKCE 方法、简化流程与 originator 均对齐参考；state / challenge 长度正常，未发现浏览器打开时截断。继续检查实际网页请求与成功／失败环境差异，不再凭来源参数推断根因。
- 浏览器准备网络诊断后的会话中，用户确认网页授权成功，账号选择接口返回 HTTP 200（约 3.36 秒），浏览器到达 localhost:1455/success。但桌面显示通用失败，SQLite 无新增订阅账号，账号管理闭环尚未成功。
- 新增仅含 State / Code 的 DiagnosticState，授权阶段变更通过中央 logger 记录 module=subscription，禁止输出身份、会话 ID、URL、用户码、令牌。隐私与最近状态回归测试通过。后续两个会话只观察到 waiting → login_expired，未取得新的保存失败错误码，仍需用户完成一个带诊断的实际授权。
- 诊断变更后完整 task verify 再次通过：全部 Go 测试、30 文件／187 项前端测试、静态／类型／格式检查及生产构建；标准开发环境重新启动。任务保留 In Progress，不宣称问题已解决。
- 新截图确认 localhost:1455/auth/callback 连接拒绝。重启后应用无新授权日志、无官方子进程及回调监听，截图为已结束会话的浏览器返回；开发重启使旧链接失效。修正弹窗关闭取消的生命周期风险：关闭／导航保留后台会话，新增 SubscriptionCurrent 恢复同一操作、显式取消仍清理，账号查询独立监听后台完成事件。
- 保持新会话后用户确实到达 success，日志确认 account_unavailable，问题位于完成后的账号读取。改为强制同步官方缓存；新增账号的元数据暂时不可读时保存成功授权引用与待确认账号、返回明确警告，避免 Logout 删除有效授权。已有账号重新授权仍坚持身份确认。
- 回归测试覆盖必须刷新才能读到新身份、元数据暂不可用仍保存引用且不 Logout、随后 Refresh 可恢复身份与已授权状态。
- 账号读取修正完整 task verify 再次通过（全部 Go、187 前端及构建）；随后补充独立保存 context，避免读取超时后复用已取消 context 导致授权仍被清理，针对该包 Go 测试／go vet 和 diff 检查通过。开发窗口恢复加载最终代码，实际账号保存待用户重新完成授权。
- 最终实际验证：用户确认“已保存账号并显示身份”；日志状态 completed（初次读取暂未就绪时为 account_metadata_pending），随后数据库只读统计确认 1 个订阅账号处于 authenticated，身份及套餐字段均已填充。只记录统计，不记录实际身份或凭证。登录、回调、授权保留、账号保存与身份恢复闭环通过。
- 回归验证：前端关闭设备码弹窗不会取消、重开恢复同一会话且不再次 Begin、显式取消生效；后端 Current 隔离不同账号操作，恢复后后台完成可保存账号。完整 task verify 通过（全部 Go，30 文件／187 前端，静态／类型／格式与生产构建）。恢复开发环境后不再于用户授权进行时重启。

## Git

未提交。授权适配器本身属于前一任务的未提交实现，相关产品／任务文档也未提交；按 Git 安全规则保留当前授权、网关、路由及日志改动，避免合并提交无关内容。未操作参考项目的代码和凭证。

## Result

已完成真实 ChatGPT / Codex 订阅账号登录验证，账号保存并显示身份与套餐。开发环境持续运行。首次身份读取可能暂时未就绪，应用保留授权并明确提示刷新；订阅请求转发仍属于后续范围。
