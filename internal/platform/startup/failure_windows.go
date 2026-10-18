//go:build windows

// Package startup 在桌面框架尚未就绪时展示可见的启动失败提示。
package startup

import "golang.org/x/sys/windows"

// NotifyFailure 只展示固定阶段说明和日志位置，不把数据库／凭据错误原文放入弹窗。
func NotifyFailure(stage, logDir string) {
	message := "OmniProxy 无法启动：" + stage + "。\n现有数据已保留，请勿删除数据库。"
	if logDir != "" {
		message += "\n请查看日志目录：\n" + logDir
	}
	text, err := windows.UTF16PtrFromString(message)
	if err != nil {
		return
	}
	title, _ := windows.UTF16PtrFromString("OmniProxy 启动失败")
	_, _ = windows.MessageBox(0, text, title, windows.MB_OK|windows.MB_ICONERROR)
}
