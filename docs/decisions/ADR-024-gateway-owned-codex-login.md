# ADR 024: Codex 订阅登录改由网关拥有（v1 池账号形态）

Status: Accepted

Date: 2026-10-24

Supersedes: ADR-003 的「令牌只由官方组件持有、网关只读」一条

## 背景

v2 原来把登录交给官方 Codex 组件：每个订阅账号是一个隔离 `CODEX_HOME`，
令牌躺在官方 keyring / age 密文里，网关只能通过官方 `account/read` 借出访问
令牌，账号表只存定位信息（`codex-keyring` + profile）。用户在客户端里仍然是
"用自己的官方登录发请求"，网关只是替换身份。

v1 的做法相反：网关拥有登录材料。它把 auth.json（访问令牌＋续期令牌＋账号
标识）存进自己的池，导入本机 `~/.codex/auth.json`，自己用 `refresh_token`
续期并回写池记录，还能把池账号写回客户端 `~/.codex/auth.json`。

用户要求把"谁拥有登录"这一点也换回 v1。

## 决策

- 新增账号凭据类型 `codex_auth_json`（`account.AuthCodexFile`）：完整 auth.json
  存在本应用的受保护凭据存储（DPAPI 保护文件）里，SQLite 只存 `credential_ref`
  与可展示身份（邮箱／套餐／官方账号标识）。
- 导入（v1 的 `configureCodex` 导入语义）：`AppService.ImportCodexAuthFromClient`
  读取本机 `~/.codex/auth.json`，校验后收进池；同一官方账号重复导入只更新凭据，
  不新增账号。
- 续期（v1 的 `UpdateTokenValue` 语义）：借出令牌前若访问令牌临近过期，网关用
  池里的 `refresh_token` 走 `auth.openai.com/oauth/token` 自行换新，并把新
  auth.json 写回凭据存储；写回只换凭据引用，不改账号版本与身份，避免把进行中
  的请求判为"路由已变"。
- 客户端登录（v1 的 `writeCodexAccountAuth` 语义）：Codex 一键配置在写入
  `config.toml` 的同时，把池账号写成客户端 `~/.codex/auth.json`；首次覆盖前保留
  `auth.json.omniproxy.bak`，恢复配置时还原用户原来的登录并删除备份。
- 没有池账号时不接管登录（`SetCodexLogin` 返回空）：保留 ADR-003 的隔离登录
  形态可用，两种凭据来源在同一套路由、冷却与换号逻辑里并存。
- 网关与路由把 `subscription_login` 与 `codex_auth_json` 视为同一类"Codex 登录
  凭据"，同池判定、候选排序、模型资格与故障切换都共用一套规则。

## 后果

- 官方隔离登录不再是唯一通道：用户可以直接把本机已有的 Codex 登录收进池，
  也可以把池账号下发给客户端；登录材料的所有权在网关。
- 会话令牌会进入 `http.Header` 参与上游请求（与 v1 相同），Core 在尝试结束后
  清零；续期结果只写受保护存储，不写日志、不返回前端。
- 客户端 `auth.json` 的覆盖是可逆操作：备份文件与 `config.toml` 备份一样属于
  恢复流程的一部分，恢复后不留残留。
- 官方组件刷新（`account/read`）仍保留给隔离登录账号使用；两条来源互不依赖。
- 2026-10-24 握手修复：隔离登录在受保护材料仍有效且成员／工作区匹配时直接借出，
  只在需要续期时启动官方组件并由其保存新授权。不能由网关自行旋转隔离登录的
  refresh token 却只缓存访问令牌，否则缓存到期或进程重启后会失去可用续期材料。
  网关自管账号仍自行续期，但使用版本、成员及原凭据引用的比较交换保护回写，
  不允许旧结果覆盖用户重新导入的登录。
