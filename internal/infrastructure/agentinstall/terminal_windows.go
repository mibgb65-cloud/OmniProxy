//go:build windows

package agentinstall

import (
	"context"
	"errors"
	"io"
	"os"
	"strings"
	"unsafe"

	"golang.org/x/sys/windows"
)

var errTerminalUnavailable = errors.New("agentinstall: terminal_unavailable")
var updateConsoleAttribute = windows.NewLazySystemDLL("kernel32.dll").NewProc("UpdateProcThreadAttribute")

// runInstallCommand 为 WinGet 提供后台控制台，使新版 WinGet 不隐藏进度输出。
func runInstallCommand(ctx context.Context, path string, args []string, report func(Progress)) error {
	writer := &progressWriter{report: report}
	err := runConsoleCommand(ctx, path, args, writer)
	if errors.Is(err, errTerminalUnavailable) {
		// 只有子进程尚未启动时才降级，避免重试造成重复安装。
		err = runObservedCommand(ctx, path, args, writer)
	}
	writer.flush()
	return err
}

func runConsoleCommand(ctx context.Context, path string, args []string, output io.Writer) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	inRead, inWrite, err := os.Pipe()
	if err != nil {
		return errTerminalUnavailable
	}
	defer inRead.Close()
	defer inWrite.Close()
	outRead, outWrite, err := os.Pipe()
	if err != nil {
		return errTerminalUnavailable
	}
	defer outRead.Close()
	defer outWrite.Close()
	var console windows.Handle
	if err := windows.CreatePseudoConsole(windows.Coord{X: 120, Y: 32}, windows.Handle(inRead.Fd()), windows.Handle(outWrite.Fd()), 0, &console); err != nil {
		return errTerminalUnavailable
	}
	// 输出在独立 goroutine 持续排空，关闭控制台时也能消费最后一帧，防止死锁。
	readDone := make(chan error, 1)
	go func() { _, err := io.Copy(output, outRead); readDone <- err }()
	defer func() { windows.ClosePseudoConsole(console); outWrite.Close(); <-readDone }()
	attributes, err := windows.NewProcThreadAttributeList(1)
	if err != nil {
		return errTerminalUnavailable
	}
	defer attributes.Delete()
	// HPCON 是句柄值，Win32 要求 lpValue 直接为 HPCON，不是 &HPCON。
	result, _, _ := updateConsoleAttribute.Call(uintptr(unsafe.Pointer(attributes.List())), 0, windows.PROC_THREAD_ATTRIBUTE_PSEUDOCONSOLE, uintptr(console), unsafe.Sizeof(console), 0, 0)
	if result == 0 {
		return errTerminalUnavailable
	}
	startup := windows.StartupInfoEx{ProcThreadAttributeList: attributes.List()}
	startup.Cb = uint32(unsafe.Sizeof(startup))
	// 避免继承父进程的重定向标准句柄，让 ConPTY 为子进程建立控制台句柄。
	startup.Flags = windows.STARTF_USESTDHANDLES
	command := []string{windows.EscapeArg(path)}
	for _, arg := range args {
		command = append(command, windows.EscapeArg(arg))
	}
	commandLine, err := windows.UTF16PtrFromString(strings.Join(command, " "))
	if err != nil {
		return errTerminalUnavailable
	}
	application, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return errTerminalUnavailable
	}
	var process windows.ProcessInformation
	if err := windows.CreateProcess(application, commandLine, nil, nil, false, windows.EXTENDED_STARTUPINFO_PRESENT, nil, nil, &startup.StartupInfo, &process); err != nil {
		return errTerminalUnavailable
	}
	defer windows.CloseHandle(process.Process)
	defer windows.CloseHandle(process.Thread)
	// ConPTY 已持有端点；释放本进程副本，才能在结束时收到 EOF。
	inRead.Close()
	outWrite.Close()
	waitDone := make(chan error, 1)
	go func() { _, err := windows.WaitForSingleObject(process.Process, windows.INFINITE); waitDone <- err }()
	select {
	case err := <-waitDone:
		if err != nil {
			return ErrInstallFailed
		}
	case <-ctx.Done():
		if err := windows.TerminateProcess(process.Process, 1); err != nil {
			return ErrInstallFailed
		}
		<-waitDone
		return ctx.Err()
	}
	var exitCode uint32
	if err := windows.GetExitCodeProcess(process.Process, &exitCode); err != nil || exitCode != 0 {
		return ErrInstallFailed
	}
	return nil
}
