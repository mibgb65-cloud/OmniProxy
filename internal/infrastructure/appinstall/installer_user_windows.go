//go:build windows

package appinstall

import (
	"errors"
	"path/filepath"
	"runtime"
	"unsafe"

	"golang.org/x/sys/windows"

	"omniproxy/internal/platform/profile"
)

// 与卸载登记身份一样，隔离验收仅允许链接时改名，不读取管理员继承的运行时环境变量。
var installerAppName = profile.AppName

func desktopUserToken() (windows.Token, error) {
	hwnd, _, _ := user32.NewProc("GetShellWindow").Call()
	var pid uint32
	user32.NewProc("GetWindowThreadProcessId").Call(hwnd, uintptr(unsafe.Pointer(&pid)))
	if hwnd == 0 || pid == 0 {
		return 0, errors.New("installer: desktop_unavailable")
	}
	process, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, pid)
	if err != nil {
		return 0, err
	}
	defer windows.CloseHandle(process)
	var token windows.Token
	if err := windows.OpenProcessToken(process, windows.TOKEN_QUERY|windows.TOKEN_DUPLICATE, &token); err != nil {
		return 0, err
	}
	if token.IsElevated() {
		token.Close()
		return 0, errors.New("installer: desktop_elevated")
	}
	return token, nil
}

func installerUserToken() (windows.Token, error) {
	if windows.GetCurrentProcessToken().IsElevated() {
		return desktopUserToken()
	}
	var token windows.Token
	err := windows.OpenProcessToken(windows.CurrentProcess(), windows.TOKEN_QUERY|windows.TOKEN_DUPLICATE, &token)
	return token, err
}

func installerLocation(token windows.Token, target string) (profile.InstallerLocation, error) {
	if !validTarget(target) || installerAppName == "" || filepath.Base(installerAppName) != installerAppName ||
		installerAppName == "." || installerAppName == ".." {
		return profile.InstallerLocation{}, errors.New("installer: invalid_target")
	}
	// KnownFolder 查询使用目标用户令牌，不受提权账号的 APPDATA 或 OMNIPROXY_* 环境污染。
	root, err := token.KnownFolderPath(windows.FOLDERID_RoamingAppData, 0)
	if err != nil {
		return profile.InstallerLocation{}, err
	}
	base := filepath.Join(root, installerAppName)
	return profile.InstallerLocation{ConfigDir: base, DefaultDataDir: base, InstallDir: filepath.Dir(target)}, nil
}

func withInstallerUser(target string, action func(profile.InstallerLocation) error) error {
	if !validTarget(target) {
		return errors.New("installer: invalid_target")
	}
	token, err := installerUserToken()
	if err != nil {
		return err
	}
	defer token.Close()
	run := func() error {
		location, err := installerLocation(token, target)
		if err != nil {
			return err
		}
		return action(location)
	}
	if !windows.GetCurrentProcessToken().IsElevated() {
		return run()
	}
	return runAsInstallerUser(token, run)
}

func runAsInstallerUser(token windows.Token, action func() error) error {
	var impersonation windows.Token
	if err := windows.DuplicateTokenEx(token, windows.TOKEN_QUERY|windows.TOKEN_IMPERSONATE, nil,
		windows.SecurityImpersonation, windows.TokenImpersonation, &impersonation); err != nil {
		return err
	}
	defer impersonation.Close()
	done := make(chan error)
	go func() {
		runtime.LockOSThread()
		if err := windows.SetThreadToken(nil, impersonation); err != nil {
			runtime.UnlockOSThread()
			done <- err
			return
		}
		var result error
		defer func() {
			err := windows.RevertToSelf()
			if err == nil {
				runtime.UnlockOSThread()
			}
			// 恢复失败时不解锁线程；协程退出后运行时销毁它，不能把模拟身份泄漏给其他任务。
			done <- errors.Join(result, err)
		}()
		result = action()
	}()
	return <-done
}
