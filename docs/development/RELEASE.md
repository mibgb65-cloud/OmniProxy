# Windows 发布与安装验收

## 版本和资产

1. 在正常开发提交中同步 `internal/version/version.go` 的完整版本和 `frontend/package.json`，例如 `2.0.0-beta.1`，并添加 `docs/releases/v2.0.0-beta.1.md`。
2. 本地运行 `./scripts/build-release.ps1 -Version v2.0.0-beta.1`。脚本先确认标签与源码／前端一致，再映射 Windows 数字资源，执行 `task --force verify` 和 `task --force package`。
3. 使用 Windows 原生版本 API 校验应用及安装器，最终产物复制到空的 `release-assets/`，生成匹配 SHA256 并运行资产验证。拒绝覆盖非空发布目录。
4. 验收后创建并推送对应版本标签。此操作才会触发发布，本任务不自动创建标签或推送。

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

## 安装生命周期验收

必须使用独立产品名、卸载登记键、快捷方式名、安装目录、数据／配置目录及端口，不在真实安装上演练。机器级与用户级分别记录，不能混淆。

- 首次安装到含空格的自定义目录，检查程序、卸载器、登记及快捷方式。
- 正在运行的 v2 Beta 从真实浮窗下载、取消、重试、校验并执行静默更新；确认旧进程退出、正确新版本重新打开、数据保留。
- 删除桌面图标，再覆盖升级，确认不重建；移除 `InstallLocation`，仅保留 `DisplayIcon`，确认沿用旧路径。
- 运行旧版源码的隔离副本，使用模拟账号生成 v1 DPAPI 数据，再覆盖至 v2，确认数据和旧目录保留。
- 目标应用运行中执行卸载，检查其退出、登记及受管快捷方式清理、数据和未知文件保留，并确认其他目录的同名应用仍运行。
- 用户级安装不要求管理员；机器级需实际 Windows 提权确认。权限被取消时明确记录未执行，不能记为通过。

自动化回环发布源仅存在于 `updateqa` 编译标签，并且只接受 test profile 和回环 HTTP。正式发布只使用 `production` 标签，不包含该入口。
