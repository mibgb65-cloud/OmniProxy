//go:build windows

package appinstall

import (
	"errors"
	"path/filepath"
	"runtime"
	"testing"
	"unsafe"

	"golang.org/x/sys/windows"

	"omniproxy/internal/platform/profile"
)

func TestInstallerLocationUsesTokenInsteadOfInheritedEnvironment(t *testing.T) {
	token, err := installerUserToken()
	if err != nil {
		t.Fatal(err)
	}
	defer token.Close()
	root, err := token.KnownFolderPath(windows.FOLDERID_RoamingAppData, 0)
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("APPDATA", t.TempDir())
	t.Setenv("OMNIPROXY_CONFIG_DIR", t.TempDir())
	t.Setenv("OMNIPROXY_DATA_DIR", t.TempDir())
	target := filepath.Join(t.TempDir(), "OmniProxy.exe")
	want := filepath.Join(root, installerAppName)
	visited := false
	expected := errors.New("synthetic callback failure")
	err = withInstallerUser(target, func(location profile.InstallerLocation) error {
		visited = true
		if location.ConfigDir != want || location.DefaultDataDir != want || location.InstallDir != filepath.Dir(target) {
			t.Errorf("wrong installer location: %+v", location)
		}
		// 此回调只检查路径，不读写实际使用者的任何配置或数据。
		return expected
	})
	if !visited || !errors.Is(err, expected) {
		t.Fatalf("callback failure was lost: visited=%v err=%v", visited, err)
	}
}

func TestInstallerUserRunsOnDedicatedImpersonatedThread(t *testing.T) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	var primary windows.Token
	if err := windows.OpenProcessToken(windows.CurrentProcess(), windows.TOKEN_QUERY|windows.TOKEN_DUPLICATE, &primary); err != nil {
		t.Fatal(err)
	}
	defer primary.Close()
	want, err := primary.GetTokenUser()
	if err != nil {
		t.Fatal(err)
	}
	expected := errors.New("callback failure")
	err = runAsInstallerUser(primary, func() error {
		var thread windows.Token
		if err := windows.OpenThreadToken(windows.CurrentThread(), windows.TOKEN_QUERY, true, &thread); err != nil {
			return err
		}
		defer thread.Close()
		got, err := thread.GetTokenUser()
		if err != nil {
			return err
		}
		if !got.User.Sid.Equals(want.User.Sid) {
			return errors.New("wrong impersonation identity")
		}
		return expected
	})
	if !errors.Is(err, expected) {
		t.Fatal(err)
	}
	var thread windows.Token
	err = windows.OpenThreadToken(windows.CurrentThread(), windows.TOKEN_QUERY, true, &thread)
	if err == nil {
		thread.Close()
		t.Fatal("impersonation leaked into the caller thread")
	}
	if !errors.Is(err, windows.ERROR_NO_TOKEN) {
		t.Fatal(err)
	}
}

func TestInstallerUserFailsClosedWithoutValidTokenOrTarget(t *testing.T) {
	visited := false
	if err := runAsInstallerUser(0, func() error { visited = true; return nil }); err == nil || visited {
		t.Fatal("invalid identity was allowed to run")
	}
	for _, target := range []string{"", "OmniProxy.exe", `C:\OmniProxy.exe`, `C:\Tools\notepad.exe`} {
		err := withInstallerUser(target, func(profile.InstallerLocation) error { visited = true; return nil })
		if err == nil || visited {
			t.Fatalf("invalid target was allowed: %q", target)
		}
	}
}

func TestInstallerUserPersistsConfigWithRealFilePermissions(t *testing.T) {
	var token windows.Token
	if err := windows.OpenProcessToken(windows.CurrentProcess(), windows.TOKEN_QUERY|windows.TOKEN_DUPLICATE, &token); err != nil {
		t.Fatal(err)
	}
	defer token.Close()
	location := profile.InstallerLocation{
		ConfigDir: t.TempDir(), DefaultDataDir: filepath.Join(t.TempDir(), "default"),
		InstallDir: t.TempDir(),
	}
	target := filepath.Join(t.TempDir(), "data")
	if err := runAsInstallerUser(token, func() error { return location.SetDataDir(target) }); err != nil {
		t.Fatal(err)
	}
	if got, err := location.DataDir(); err != nil || got != target {
		t.Fatalf("user config was not persisted: %q %v", got, err)
	}
}

func TestInstallerUserPersistsConfigInEncryptedDirectory(t *testing.T) {
	directory := t.TempDir()
	path, err := windows.UTF16PtrFromString(directory)
	if err != nil {
		t.Fatal(err)
	}
	ok, _, encryptErr := windows.NewLazySystemDLL("advapi32.dll").NewProc("EncryptFileW").Call(uintptr(unsafe.Pointer(path)))
	if ok == 0 {
		// EFS 是可选的 Windows／文件系统能力，不支持时不把普通权限测试伪装成 EFS 验收。
		t.Skipf("EFS is unavailable for this temporary directory: %v", encryptErr)
	}
	var token windows.Token
	if err := windows.OpenProcessToken(windows.CurrentProcess(), windows.TOKEN_QUERY|windows.TOKEN_DUPLICATE, &token); err != nil {
		t.Fatal(err)
	}
	defer token.Close()
	location := profile.InstallerLocation{
		ConfigDir: directory, DefaultDataDir: filepath.Join(t.TempDir(), "default"),
		InstallDir: t.TempDir(),
	}
	for range 2 {
		target := t.TempDir()
		if err := runAsInstallerUser(token, func() error { return location.SetDataDir(target) }); err != nil {
			t.Fatal(err)
		}
		if got, err := location.DataDir(); err != nil || got != target {
			t.Fatalf("encrypted user config was not atomically saved: %q %v", got, err)
		}
	}
}
