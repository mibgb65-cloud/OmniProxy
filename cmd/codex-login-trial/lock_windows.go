package main

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"path/filepath"
	"runtime"
	"strings"

	"golang.org/x/sys/windows"
)

func lockTrial(home string) (func(), error) {
	canonical, err := filepath.EvalSymlinks(home)
	if err != nil {
		return nil, errors.New("试验目标必须是已存在的目录")
	}
	sum := sha256.Sum256([]byte(strings.ToLower(filepath.Clean(canonical))))
	name, err := windows.UTF16PtrFromString(`Local\OmniProxyLegacyLogin_` + hex.EncodeToString(sum[:]))
	if err != nil {
		return nil, errors.New("创建试验锁失败")
	}
	// Windows Mutex 归属线程而非 goroutine，整个受控试验期间必须保留同一线程。
	runtime.LockOSThread()
	handle, err := windows.CreateMutex(nil, false, name)
	if err != nil && !errors.Is(err, windows.ERROR_ALREADY_EXISTS) {
		runtime.UnlockOSThread()
		return nil, errors.New("创建试验锁失败")
	}
	result, err := windows.WaitForSingleObject(handle, 0)
	if err != nil || result != windows.WAIT_OBJECT_0 && result != windows.WAIT_ABANDONED {
		windows.CloseHandle(handle)
		runtime.UnlockOSThread()
		return nil, errors.New("该客户端目录已有登录试验正在进行")
	}
	return func() {
		windows.ReleaseMutex(handle)
		windows.CloseHandle(handle)
		runtime.UnlockOSThread()
	}, nil
}
