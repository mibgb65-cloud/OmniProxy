# Windows 发布与安装验收

## 版本和资产

1. 在正常开发提交中同步 `internal/version/version.go` 的完整版本和 `frontend/package.json`，例如 `2.0.0-beta.1`，并添加 `docs/releases/v2.0.0-beta.1.md`。
2. 本地运行 `./scripts/build-release.ps1 -Version v2.0.0-beta.1`。脚本先确认标签与源码／前端一致，再映射 Windows 数字资源，执行 `task --force verify` 和 `task --force package`。
3. 使用 Windows 原生版本 API 校验应用及安装器，最终产物复制到空的 `release-assets/`，生成匹配 SHA256 并运行资产验证。拒绝覆盖非空发布目录。
4. 验收后在对应提交创建并推送版本标签。该推送会触发公开发布，应作为明确的版本发布操作执行。

示例：

```powershell
./scripts/build-release.ps1 -Version v2.0.0-beta.1
./scripts/verify-release-assets.ps1 -Version v2.0.0-beta.1 -AssetDirectory ./release-assets
```

所需工具延用 BUILD：Go 1.26、Node 22、pnpm、Task、Wails beta.30、staticcheck、NSIS。Release 工作流固定 pnpm 12.6.0、Task 3.53.1、staticcheck v0.8.1 及 Wails beta.30，不安装新的业务依赖。机器级实际安装需要管理员确认，打包本身不需要安装程序到本机。

## GitHub Actions

`.github/workflows/release.yml` 响应版本标签或手动指定已存在标签。构建作业只有读取权限；发布作业才有 `contents: write`，使用 Actions 临时令牌，令牌不写入文件。

构建和校验成功后上传安装器及同名 `.sha256`。发布作业再次校验下载的 artifact，先创建 draft、上传全部资产并检查数量和大小，最后取消 draft。Beta 不取代 stable latest；正式版设为 latest。不会删除历史 Release、重写标签或覆盖已经公开的 Release。失败重试只复用同标签的 draft。

工作流本地用 `actionlint .github/workflows/release.yml` 校验。真正的 Actions 调度、仓库权限和远端发布只能在连接目标仓库后验证；本地脚本和静态检查不冒充远端运行成功。

首发 v2.0.0-beta.1 的既有标签保留不变。该标签的真实时钟测试在慢运行器上触发 5 秒期限；从 main 手动恢复此标签发布时，工作流只补入工作流提交中的提供商虚拟时钟测试，核对其余应用源码与标签无差异后运行完整质量门及打包。正常标签推送流程不补入文件，后续版本直接包含修复后的测试；已经公开的同标签 Release 仍禁止覆盖。

## 安装生命周期验收

必须使用独立产品名、卸载登记键、快捷方式名、安装目录、数据／配置目录及端口，不在真实安装上演练。机器级与用户级分别记录，不能混淆。

- 首次安装到含空格的自定义目录，检查程序、卸载器、登记及快捷方式。
- 检查完成页的「立即启动」及旧版更新器兼容启动函数，确认界面成功打开且进程 TokenElevation 为普通用户；界面不能直接继承安装器的管理员令牌。
- 正在运行的 v2 Beta 从真实浮窗下载、取消、重试、校验并执行静默更新；确认旧进程退出、正确新版本重新打开、数据保留。
- 删除桌面图标，再覆盖升级，确认不重建；移除 `InstallLocation`，仅保留 `DisplayIcon`，确认沿用旧路径。
- 运行旧版源码的隔离副本，使用模拟账号生成 v1 DPAPI 数据，再覆盖至 v2，确认数据和旧目录保留。
- 额外构造默认目录下的历史 `omniproxy.db`，包含旧 `request_records(client,client_model,provider_model,success)` 和已落盘的前七个 v2 迁移；确认迁移前选择独立 v2 目录、完成全量迁移并能重启，旧统计及迁移记录完全保留。仅测试 tokens.json 导入不能替代此场景。已有正常 v2 账号库必须保留，混有新版账号／后期迁移或未知布局必须明确失败。
- 构造只有旧安装目录 `data-dir.txt` 的自定义数据位置，分别放置根数据库与 `v2/omniproxy.db`；
  向导必须显示旧位置，改选空目录必须被拒绝。同位置保存后改变程序安装目录仍须指向旧数据。
- 验证标准用户使用另一管理员凭据提权时，配置仍写给实际使用者，目录可写性也按该用户校验。
  不能用改 APPDATA 的测试代替真实跨用户 UAC；自动化测试仅验证令牌上下文与环境隔离。
- 快捷方式实际 `IconLocation` 必须指向安装目录的 `app-icon-v2.ico`，检查 Shell 读取的新图标；删除桌面快捷方式后升级仍不重建，卸载删除自有 ICO、保留未知文件。
- 目标应用运行中执行卸载，检查其退出、登记及受管快捷方式清理、数据和未知文件保留，并确认其他目录的同名应用仍运行。
- 用户级安装不要求管理员；机器级需实际 Windows 提权确认。权限被取消时明确记录未执行，不能记为通过。

自动化回环发布源仅存在于 `updateqa` 编译标签，并且只接受 test profile 和回环 HTTP。正式发布只使用 `production` 标签，不包含该入口。
