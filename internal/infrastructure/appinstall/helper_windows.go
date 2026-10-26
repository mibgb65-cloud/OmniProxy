//go:build windows

package appinstall

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"unsafe"

	"golang.org/x/sys/windows"
	"omniproxy/internal/platform/profile"
	"omniproxy/internal/version"
)

// HandleMode 必须在 profile／数据库初始化前调用；参数不经过 shell。
func HandleMode(args []string) (bool, int) {
	if len(args) == 2 && args[0] == "--omniproxy-installer-launch" {
		if startInstalled(args[1]) != nil {
			return true, 3
		}
		return true, 0
	}
	if len(args) == 2 && args[0] == "--omniproxy-installer-stop" {
		if stopInstalled(args[1]) != nil {
			return true, 2
		}
		return true, 0
	}
	if len(args) == 2 && args[0] == "--omniproxy-installer-cleanup" {
		cleanupAutostart(args[1])
		return true, 0
	}
	// 原安装程序路径必须显式传入；临时助手所在目录和管理员环境都不是用户数据来源。
	if len(args) == 2 && (args[0] == "--omniproxy-installer-data-dir-read" || args[0] == "--omniproxy-installer-data-dir-read-file") {
		var dir string
		err := withInstallerUser(args[1], func(location profile.InstallerLocation) error {
			var err error
			dir, err = location.DataDir()
			return err
		})
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return true, 2
		}
		if args[0] == "--omniproxy-installer-data-dir-read-file" {
			executable, err := os.Executable()
			if err == nil {
				err = writeInstallerDirectoryResult(executable, dir)
			}
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				return true, 2
			}
		} else {
			fmt.Println(dir)
		}
		return true, 0
	}
	if len(args) == 3 && args[0] == "--omniproxy-installer-data-dir-write" {
		err := withInstallerUser(args[1], func(location profile.InstallerLocation) error {
			return location.SetDataDir(args[2])
		})
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			if errors.Is(err, profile.ErrDataDirInUse) {
				return true, 3 // 原目录仍有数据：安装器不迁移，提示用户装完后在应用内改
			}
			return true, 2
		}
		return true, 0
	}
	if len(args) > 0 && args[0] == "--omniproxy-update-helper" {
		if len(args) != 8 {
			return true, 2
		}
		return true, runUpdate(args[1:])
	}
	if len(args) > 0 && (args[0] == "--omniproxy-installer-stop" || args[0] == "--omniproxy-installer-cleanup" || args[0] == "--omniproxy-installer-launch" || args[0] == "--omniproxy-installer-data-dir-read" || args[0] == "--omniproxy-installer-data-dir-read-file" || args[0] == "--omniproxy-installer-data-dir-write") {
		return true, 2
	}
	return false, 0
}

func runUpdate(args []string) int {
	size, err := strconv.ParseInt(args[2], 10, 64)
	digest, hashErr := hex.DecodeString(args[1])
	request := Request{Package: args[0], Digest: args[1], Size: size, Receipt: args[4], Version: args[5]}
	target, scope := args[3], args[6]
	if err != nil || hashErr != nil || len(digest) != 32 || size < 1 || size > 1<<30 || !version.Valid(request.Version) || !filepath.IsAbs(request.Package) || filepath.Dir(request.Package) != filepath.Dir(request.Receipt) || filepath.Base(request.Receipt) != "install-result.json" {
		return announceFailure("update: invalid_package")
	}
	actualScope, err := installation(target)
	if err != nil || scope != actualScope {
		return announceFailure("update: install_unavailable")
	}
	name, err := windows.UTF16PtrFromString(request.Package)
	if err != nil {
		return announceFailure("update: invalid_package")
	}
	// 校验到安装结束始终禁止其他进程写入／替换已验证文件。
	handle, err := windows.CreateFile(name, windows.GENERIC_READ, windows.FILE_SHARE_READ, nil, windows.OPEN_EXISTING, windows.FILE_ATTRIBUTE_NORMAL, 0)
	if err != nil {
		return announceFailure("update: package_unavailable")
	}
	file := os.NewFile(uintptr(handle), request.Package)
	defer file.Close()
	info, err := file.Stat()
	if err != nil || !info.Mode().IsRegular() || info.Size() != size {
		return announceFailure("update: package_unavailable")
	}
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil || hex.EncodeToString(hash.Sum(nil)) != strings.ToLower(request.Digest) {
		return announceFailure("update: checksum_mismatch")
	}
	process, err := startInstaller(request.Package, scope)
	if err != nil {
		return announceFailure(err.Error())
	}
	defer windows.CloseHandle(process)
	json.NewEncoder(os.Stdout).Encode(handshake{Ready: true})
	waitStatus, waitErr := windows.WaitForSingleObject(process, windows.INFINITE)
	var exit uint32
	exitErr := windows.GetExitCodeProcess(process, &exit)
	success := waitErr == nil && exitErr == nil && waitStatus == windows.WAIT_OBJECT_0 && exit == 0
	receipt := Receipt{Version: request.Version, Success: success}
	if !success {
		receipt.Code = "update: install_failed"
	}
	data, _ := json.Marshal(receipt)
	if os.WriteFile(request.Receipt+".tmp", data, 0600) == nil {
		os.Rename(request.Receipt+".tmp", request.Receipt)
	}
	// 此辅助进程保留启动用户的令牌，避免从提权安装器启动用户界面。
	cmd := exec.Command(target)
	cmd.Dir = filepath.Dir(target)
	if err := cmd.Start(); err != nil {
		return 4
	}
	cmd.Process.Release()
	if !success {
		return 3
	}
	return 0
}

func announceFailure(message string) int {
	json.NewEncoder(os.Stdout).Encode(handshake{Error: message})
	return 2
}

type shellExecuteInfo struct {
	Size, Mask                        uint32
	Window                            windows.Handle
	Verb, File, Parameters, Directory *uint16
	Show                              int32
	Instance                          windows.Handle
	IDList                            unsafe.Pointer
	Class                             *uint16
	ClassKey                          windows.Handle
	HotKey                            uint32
	Icon, Process                     windows.Handle
}

func startInstaller(path, scope string) (windows.Handle, error) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	ole := windows.NewLazySystemDLL("ole32.dll")
	resultCOM, _, _ := ole.NewProc("CoInitializeEx").Call(0, 6)
	if resultCOM == 0 || resultCOM == 1 {
		defer ole.NewProc("CoUninitialize").Call()
	}
	verb := "open"
	if scope == "machine" {
		verb = "runas"
	}
	v, _ := windows.UTF16PtrFromString(verb)
	file, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return 0, errors.New("update: invalid_package")
	}
	params, _ := windows.UTF16PtrFromString("/S /OMNIPROXY_AUTOUPDATE=1 /OMNIPROXY_NORESTART=1")
	dir, _ := windows.UTF16PtrFromString(filepath.Dir(path))
	info := shellExecuteInfo{Mask: 0x40 | 0x100, Verb: v, File: file, Parameters: params, Directory: dir, Show: 0}
	info.Size = uint32(unsafe.Sizeof(info))
	result, _, callErr := windows.NewLazySystemDLL("shell32.dll").NewProc("ShellExecuteExW").Call(uintptr(unsafe.Pointer(&info)))
	if result == 0 || info.Process == 0 {
		if callErr == windows.ERROR_CANCELLED {
			return 0, errors.New("update: elevation_cancelled")
		}
		return 0, errors.New("update: install_start_failed")
	}
	return info.Process, nil
}
