# TASK-20261024-013: 安装向导语言选择与数据目录页重做

## Status

Completed（2026-10-27 随 v2.0.0 正式发布归档）

## Created

2026-10-24 16:30

## Request

用户要求：初次安装时先选择语言（中文/English），再选择安装目录，再选择保存数据的目录；数据目录要包含**所有**数据，C 盘只保留一个指向它的必要文件。追加要求：升级安装时该页要显示首次设置的位置，用户改动时要提示会导致账号数据丢失的风险。

## Goal

安装向导四步可达：语言 → 安装目录 → 数据目录 → 安装；数据目录页在升级时显示当前实际使用的位置，改动前给出不会迁移数据的警告，安装器自身不做迁移，并在原位置仍有数据时拒绝切换。

## Background

- Beta.33 的该页只把选择写进安装目录的 `data-dir.txt`，对已有 `app.json.dataDir` 的用户完全不生效，看起来「没用」。
- 数据目录已经是全部数据的容器（数据库、DPAPI 凭据、日志、更新缓存），C 盘 `%APPDATA%\OmniProxy\app.json` 本来就是指针文件。
- 安装器是机器级提权进程，`$APPDATA` 未必属于实际使用者，所以既不能盲写别的账号的 `app.json`，也不该由安装器迁移数据。

## Existing Behavior

页面预填 `$INSTDIR\data-dir.txt`（安装时的猜测值），无语言选择，无改动警告，写入的是建议文件。

## Expected Behavior

- 交互安装先弹语言选择，全部自定义文案双语；静默升级不显示。
- 页面打开即显示当前用户实际使用的数据目录（读 `app.json`），而不是猜测值。
- 改动时先警告「不会迁移已有数据」；确认后由辅助程序写入指针文件。
- 原位置仍含数据库时安装器拒绝切换（退出码 3），提示装完后在应用内搬迁。
- 数据目录承载全部数据，C 盘只剩 `app.json`。

---

# Implementation Result

## Changes

- `build/windows/nsis/strings.nsh`（新增）：全部向导文案的 LangString（简中／英文），含警告、拒绝、完成页、卸载提示。
- `build/windows/nsis/data_dir_page.nsh`：页面打开时调用 `--omniproxy-installer-data-dir-read` 读取当前位置并预填；路径归一（去引号、去尾反斜杠、拒绝相对路径与磁盘根）；改动弹 YESNO 警告，选否留页；写入改走 `--omniproxy-installer-data-dir-write`，退出码 3 显示「原位置仍有数据，请装完后在应用内搬迁」。
- `build/windows/nsis/project.nsi`：加入 `MUI_LANGDLL_DISPLAY`（语言对话框，记录在 `HKCU\Software\OmniProxy\InstallerLanguage`）、双 `MUI_LANGUAGE`、完成页与卸载文案改为 LangString；不再写 `data-dir.txt`，卸载仍清理 Beta.33 留下的该文件。
- `build/windows/nsis/installer_runtime.nsh`：`ExtractHelper` 提到 `.onInit` 并加 `IfFileExists` 去重，使向导阶段就能调用辅助程序；两处消息框改 LangString。
- `internal/platform/profile/installer.go`（新增）：`InstallerDataDir`（当前实际位置，供安装器显示）、`InstallerSetDataDir`（绝对路径／可写／互不包含校验；当前位置含 `omniproxy.db` 或 `v2\omniproxy.db` 时返回 `ErrDataDirInUse`；写入 `app.json` 并清除 `previousDataDir`）。
- `internal/infrastructure/appinstall/helper_windows.go`：新增 `--omniproxy-installer-data-dir-read` 与 `-write` 两个受限模式，写入遇 `ErrDataDirInUse` 返回退出码 3；非预期参数仍按原逻辑拒绝。

## Files Changed

- `build/windows/nsis/strings.nsh`、`data_dir_page.nsh`、`installer_runtime.nsh`、`project.nsi`
- `internal/platform/profile/installer.go`、`installer_test.go`
- `internal/infrastructure/appinstall/helper_windows.go`、`process_windows_test.go`
- `docs/development/BUILD.md`、`docs/architecture/OVERVIEW.md`

## Technical Notes

- 读位置用程序自身的解析逻辑（`profile.New` + `LoadConfig` + `effectiveDataDir`），安装器不再重复实现路径规则。
- 写入只碰当前用户的 `app.json`：`InstallerSetDataDir` 不写 `previousDataDir`，因此不会触发应用启动时的迁移逻辑；真实迁移仍只在应用退出时由 `relocateIfPending` 完成并带 `RevertDataDir` 回滚。
- Beta.33 的 `data-dir.txt` 读取保留为最低优先级默认值，避免已安装用户升级后位置跳变。

# Verification

- `go test ./internal/platform/profile/ ./internal/infrastructure/appinstall/` 通过；新增 5 项安装器用例（默认值、读取已存值、写入并落盘、拒绝放弃已有数据 ×2 布局、重复选择不改配置、非法输入）。
- 实机验证辅助模式：`go run . --omniproxy-installer-data-dir-read` 在空配置下输出 `%APPDATA%\OmniProxy`；配置里写入 `D:\OmniProxyData` 后输出该路径；对含 `omniproxy.db` 的当前位置执行 `-write` 被拒绝。
- `makensis` 编译通过：5 个页面、2 个语言表（`Install: 5 pages ... 2 language tables`）。
- 未执行：真实安装向导的点击验收（需要交互式提权安装）。

# Result

向导支持中英双语，数据目录页显示的是当前真正在用的位置；改动前有明确的数据风险提示，安装器本身不迁移数据，并在原位置仍有数据时拒绝切换。数据全部位于数据目录，C 盘只剩 `app.json` 指针。

---

# Archival (v2.0.0)

- 原状态：Completed
- 2026-10-27：v2.0.0 正式版已公开（标签源码 `c9c48b4a31988a6cc3fa2f4b1d423e61bd10f71b`，
  Actions 完整质量门与 Windows 构建通过），本任务交付内容已随该版本交付。
- 归档依据：归档时工作区无未提交代码改动，本记录引用的实现提交均为上述发布提交的祖先提交。
- 未执行项（实机客户端／原生窗口视觉／真实覆盖安装验收等）不伪造为已验收，转为发布后
  跟进事项，见 `docs/plans/CURRENT.md` 的 Post-release Follow-ups。
