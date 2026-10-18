//go:build !windows

package appinstall

import "errors"

// HandleMode 在非 Windows 不执行安装模式。
func HandleMode([]string) (bool, int) { return false, 0 }

// Installed 目前只支持 Windows 安装包更新。
func Installed() bool { return false }

// Launch 不将 Windows 安装器交给其他系统执行。
func Launch(Request) error { return errors.New("update: unsupported_platform") }

// ListenForQuit 非 Windows 没有安装器退出事件。
func ListenForQuit(func()) (func() error, error) { return func() error { return nil }, nil }
