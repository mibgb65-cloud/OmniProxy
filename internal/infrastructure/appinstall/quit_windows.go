//go:build windows

package appinstall

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"

	"golang.org/x/sys/windows"
)

func eventName(target string) *uint16 {
	digest := sha256.Sum256([]byte(strings.ToLower(filepath.Clean(target))))
	name, _ := windows.UTF16PtrFromString(`Local\OmniProxy.Quit.` + hex.EncodeToString(digest[:16]))
	return name
}

// ListenForQuit 让安装器按完整路径请求正常退出，而不是关闭其他安装的同名应用。
func ListenForQuit(quit func()) (func() error, error) {
	path, err := os.Executable()
	if err != nil {
		return nil, err
	}
	event, err := windows.CreateEvent(nil, 0, 0, eventName(path))
	if err != nil {
		return nil, err
	}
	var closing atomic.Bool
	done := make(chan struct{})
	go func() {
		defer close(done)
		status, err := windows.WaitForSingleObject(event, windows.INFINITE)
		if err == nil && status == windows.WAIT_OBJECT_0 && !closing.Load() {
			quit()
		}
	}()
	return func() error { closing.Store(true); windows.SetEvent(event); <-done; return windows.CloseHandle(event) }, nil
}

func signalQuit(target string) {
	event, err := windows.OpenEvent(windows.EVENT_MODIFY_STATE, false, eventName(target))
	if err != nil {
		return
	}
	windows.SetEvent(event)
	windows.CloseHandle(event)
}
