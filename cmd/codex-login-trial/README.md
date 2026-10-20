# 旧版 Codex 文件登录试验

此工具只用于用户明确要求的旧版兼容试验，不改变应用内一键接入的默认行为。
仅支持 Windows，保留 `model_provider=openai`、网关地址、模型、聊天及索引文件。

## 行为

- 只读打开所选 OmniProxy 环境的账号库，通过官方组件查询额度，不调用模型。
- 只选择启用、已认证、所需额度窗口完整且均未耗尽的 ChatGPT 账号；未知额度不推算可用。
- 使用隔离官方组件刷新并核验账号，在后端内存读取完整授权，不通过参数、日志或 UI 传递令牌。
- `-apply` 才写指定目录的 `auth.json`，并设置 ChatGPT/file 登录。不会回退到 API 登录，
  不切换自定义提供商，不修改系统加密登录命名空间。
- 写入前将原 `config.toml` 和原 `auth.json`（包括不存在状态）存入当前 profile 的 DPAPI
  凭据存储；目标目录只留下不含凭据的 `omniproxy-login-trial.json` 恢复引用。
- 配置必须已指向所选环境的本机网关。其他提供商、活跃 profile、强制工作区不匹配、
  不安全路径、并发试验和已存在试验均拒绝写入。
- 配置写入失败会尝试回滚本次授权写入；外部文件变化保留，不强制覆盖，恢复引用也保留。

## 使用

在仓库根目录编译：

```powershell
go build -o bin/codex-login-trial.exe ./cmd/codex-login-trial
```

先只读检查，`-home` 必须是已存在的绝对路径；如使用自定义 `CODEX_HOME`，替换示例路径：

```powershell
.\bin\codex-login-trial.exe -profile production -home "$env:USERPROFILE\.codex"
```

明确应用试验：

```powershell
.\bin\codex-login-trial.exe -profile production -home "$env:USERPROFILE\.codex" -apply
```

先完成当前工作，再完全退出并重新打开 Desktop。没有自动重启，也不将“配置写入成功”
或 CLI 登录状态识别当作 Desktop 额度门禁已解决。

恢复原文件：

```powershell
.\bin\codex-login-trial.exe -profile production -home "$env:USERPROFILE\.codex" -restore
```

恢复会核对当前文件是否仍是试验写入版本或试验前版本，拒绝覆盖客户端刷新／用户修改。
发生冲突时不要删除恢复引用或加密备份，应先核对并处理新增变更。

## 验证和限制

`-apply -verify` 只允许非默认的隔离目录，并通过本机 app-server 的 account/read
核对选中身份。隔离目录需要预先准备指向所选本机网关的 config.toml；验证完必须对同一
home/profile 执行 `-restore`，清除临时明文授权和试验备份。

试验期间，原一键接入的配置版本可能显示外部变更；先使用本工具恢复，不要同时执行普通
一键配置／恢复。复制授权后，两份凭据的刷新及 Desktop 的缓存重载兼容性仍需实际验收；
本工具不是后台自动换登录，也不伪造、禁用或代理官方的剩余额度判断。
