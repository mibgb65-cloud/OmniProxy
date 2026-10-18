//go:build windows

package appinstall

import (
	"errors"
	"os/exec"
	"path/filepath"
	"unsafe"

	"golang.org/x/sys/windows"
)

// startInstalled 兼容首次安装和旧版更新器；桌面不能继承 NSIS 的管理员令牌。
func startInstalled(target string) error {
	if _, err := installation(target); err != nil {
		return err
	}
	if !windows.GetCurrentProcessToken().IsElevated() {
		cmd := exec.Command(target)
		cmd.Dir = filepath.Dir(target)
		if err := cmd.Start(); err != nil {
			return err
		}
		return cmd.Process.Release()
	}
	// 只使用当前交互桌面的 shell 身份；没有普通用户桌面时提示手动启动。
	hwnd, _, _ := user32.NewProc("GetShellWindow").Call()
	var pid uint32
	user32.NewProc("GetWindowThreadProcessId").Call(hwnd, uintptr(unsafe.Pointer(&pid)))
	if hwnd == 0 || pid == 0 {
		return errors.New("installer: desktop_unavailable")
	}
	process, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, pid)
	if err != nil {
		return err
	}
	defer windows.CloseHandle(process)
	var token windows.Token
	if err := windows.OpenProcessToken(process, windows.TOKEN_QUERY|windows.TOKEN_DUPLICATE|windows.TOKEN_ASSIGN_PRIMARY, &token); err != nil {
		return err
	}
	defer token.Close()
	if token.IsElevated() {
		return errors.New("installer: desktop_elevated")
	}
	// 使用当前进程持有的主令牌副本并加载用户配置，避免直接使用 shell
	// 进程令牌时 CreateProcessWithTokenW 返回拒绝访问。
	var primary windows.Token
	if err := windows.DuplicateTokenEx(token, windows.TOKEN_ALL_ACCESS, nil, windows.SecurityImpersonation, windows.TokenPrimary, &primary); err != nil {
		return err
	}
	defer primary.Close()
	var environment *uint16
	if err := windows.CreateEnvironmentBlock(&environment, token, true); err != nil {
		return err
	}
	defer windows.DestroyEnvironmentBlock(environment)
	application, err := windows.UTF16PtrFromString(target)
	if err != nil {
		return err
	}
	command, _ := windows.UTF16PtrFromString(windows.ComposeCommandLine([]string{target}))
	directory, _ := windows.UTF16PtrFromString(filepath.Dir(target))
	desktop, _ := windows.UTF16PtrFromString(`winsta0\default`)
	startup := windows.StartupInfo{Desktop: desktop}
	startup.Cb = uint32(unsafe.Sizeof(startup))
	var child windows.ProcessInformation
	const logonWithProfile = 1
	result, _, callErr := windows.NewLazySystemDLL("advapi32.dll").NewProc("CreateProcessWithTokenW").Call(
		uintptr(primary), logonWithProfile, uintptr(unsafe.Pointer(application)), uintptr(unsafe.Pointer(command)),
		windows.CREATE_UNICODE_ENVIRONMENT, uintptr(unsafe.Pointer(environment)), uintptr(unsafe.Pointer(directory)),
		uintptr(unsafe.Pointer(&startup)), uintptr(unsafe.Pointer(&child)),
	)
	if result == 0 {
		return callErr
	}
	windows.CloseHandle(child.Thread)
	return windows.CloseHandle(child.Process)
}
