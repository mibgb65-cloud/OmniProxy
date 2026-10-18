//go:build windows

package appinstall

import (
	"errors"
	"path/filepath"
	"strings"
	"syscall"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

var user32 = windows.NewLazySystemDLL("user32.dll")

func matchingProcesses(target string) []uint32 {
	snapshot, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPPROCESS, 0)
	if err != nil {
		return nil
	}
	defer windows.CloseHandle(snapshot)
	entry := windows.ProcessEntry32{Size: uint32(unsafe.Sizeof(windows.ProcessEntry32{}))}
	var result []uint32
	for err := windows.Process32First(snapshot, &entry); err == nil; err = windows.Process32Next(snapshot, &entry) {
		if !strings.EqualFold(windows.UTF16ToString(entry.ExeFile[:]), filepath.Base(target)) {
			continue
		}
		if processMatches(entry.ProcessID, target) {
			result = append(result, entry.ProcessID)
		}
	}
	return result
}

func processMatches(pid uint32, target string) bool {
	process, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, pid)
	if err != nil {
		return false
	}
	defer windows.CloseHandle(process)
	var buffer [32768]uint16
	size := uint32(len(buffer))
	return windows.QueryFullProcessImageName(process, 0, &buffer[0], &size) == nil && strings.EqualFold(filepath.Clean(windows.UTF16ToString(buffer[:size])), filepath.Clean(target))
}

func closeWindows(pids []uint32) {
	callback := syscall.NewCallback(func(hwnd, lparam uintptr) uintptr {
		var pid uint32
		user32.NewProc("GetWindowThreadProcessId").Call(hwnd, uintptr(unsafe.Pointer(&pid)))
		for _, target := range pids {
			if pid == target {
				user32.NewProc("PostMessageW").Call(hwnd, 0x10, 0, 0)
			}
		}
		return 1
	})
	user32.NewProc("EnumWindows").Call(callback, 0)
}

func stopInstalled(target string) error {
	if !validTarget(target) {
		return errors.New("installer: invalid_target")
	}
	signalQuit(target)
	closeWindows(matchingProcesses(target))
	deadline := time.Now().Add(20 * time.Second)
	for time.Now().Before(deadline) {
		if len(matchingProcesses(target)) == 0 {
			return nil
		}
		time.Sleep(150 * time.Millisecond)
	}
	for _, pid := range matchingProcesses(target) {
		if !processMatches(pid, target) {
			continue
		}
		process, err := windows.OpenProcess(windows.PROCESS_TERMINATE|windows.SYNCHRONIZE|windows.PROCESS_QUERY_LIMITED_INFORMATION, false, pid)
		if err != nil {
			return errors.New("installer: process_busy")
		}
		if !processMatches(pid, target) {
			windows.CloseHandle(process)
			continue
		}
		err = windows.TerminateProcess(process, 1)
		if err == nil {
			_, err = windows.WaitForSingleObject(process, 5000)
		}
		windows.CloseHandle(process)
		if err != nil {
			return errors.New("installer: process_busy")
		}
	}
	if len(matchingProcesses(target)) != 0 {
		return errors.New("installer: process_busy")
	}
	return nil
}

func cleanupAutostart(target string) {
	if !validTarget(target) {
		return
	}
	key, err := registry.OpenKey(registry.CURRENT_USER, `Software\Microsoft\Windows\CurrentVersion\Run`, registry.QUERY_VALUE|registry.SET_VALUE)
	if err != nil {
		return
	}
	defer key.Close()
	for _, name := range []string{"OmniProxy", "com.omniproxy.app"} {
		value, _, err := key.GetStringValue(name)
		if err != nil {
			continue
		}
		args, err := windows.DecomposeCommandLine(value)
		if err == nil && len(args) > 0 && strings.EqualFold(filepath.Clean(args[0]), filepath.Clean(target)) {
			key.DeleteValue(name)
		}
	}
}
