//go:build !windows

// Package startup 在桌面框架尚未就绪时展示可见的启动失败提示。
package startup

// NotifyFailure 在非 Windows 保留已有控制台日志报告方式。
func NotifyFailure(string, string) {}
