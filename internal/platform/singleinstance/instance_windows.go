//go:build windows

package singleinstance

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"golang.org/x/sys/windows"
)

const (
	// activationWait bounds how long a second launch keeps retrying to reach an
	// instance that is still publishing its activation event during startup.
	activationWait = 2 * time.Second
	// activationRetry is the pause between those attempts.
	activationRetry = 25 * time.Millisecond
)

// procAllowSetForegroundWindow hands the foreground right to the running
// instance. The second process was started by the user and therefore owns that
// right; without transferring it SetForegroundWindow only flashes the taskbar
// button instead of activating the window.
var procAllowSetForegroundWindow = windows.NewLazySystemDLL("user32.dll").NewProc("AllowSetForegroundWindow")

// guard holds the named mutex plus the events used to wake the running window.
type guard struct {
	mutex windows.Handle
	event windows.Handle
	stop  windows.Handle

	mu       sync.Mutex
	activate func()
	pending  bool
	closed   bool
}

// mutexName is the per-profile lock name. It lives in the Local namespace so a
// second Windows session (fast user switching) keeps its own instance, matching
// the per-user data directory.
func mutexName(uniqueID string) string { return `Local\OmniProxy.Instance.` + uniqueID }

// eventName is the per-profile activation event created by the lock holder.
func eventName(uniqueID string) string { return `Local\OmniProxy.Activate.` + uniqueID }

func acquire(uniqueID string) (Guard, error) {
	if uniqueID == "" {
		return nil, errors.New("singleinstance: empty application ID")
	}
	activateName := eventName(uniqueID)
	// 先发布激活事件再抢互斥量：任何看到互斥量已存在的进程都能立刻通知正在
	// 启动的实例，不会因为「事件还没建好」丢掉这次激活请求。
	event, err := windows.CreateEvent(nil, 0, 0, windows.StringToUTF16Ptr(activateName))
	if err != nil && !errors.Is(err, windows.ERROR_ALREADY_EXISTS) {
		return nil, fmt.Errorf("singleinstance: create activation event: %w", err)
	}
	mutex, err := windows.CreateMutex(nil, false, windows.StringToUTF16Ptr(mutexName(uniqueID)))
	if err != nil {
		_ = windows.CloseHandle(event)
		if errors.Is(err, windows.ERROR_ALREADY_EXISTS) {
			requestActivation(activateName)
			return nil, ErrAlreadyRunning
		}
		return nil, fmt.Errorf("singleinstance: create instance mutex: %w", err)
	}
	// 停止事件让等待协程在退出时立即醒来，不必靠关闭句柄去打断一次挂起的等待。
	stop, err := windows.CreateEvent(nil, 1, 0, nil)
	if err != nil {
		_ = windows.CloseHandle(event)
		_ = windows.CloseHandle(mutex)
		return nil, fmt.Errorf("singleinstance: create stop event: %w", err)
	}
	g := &guard{mutex: mutex, event: event, stop: stop}
	go g.watch()
	return g, nil
}

// watch delivers every activation request until the guard is closed. The
// activation event is auto-reset, so one successful wait means one request.
func (g *guard) watch() {
	handles := []windows.Handle{g.event, g.stop}
	for {
		index, err := windows.WaitForMultipleObjects(handles, false, windows.INFINITE)
		if err != nil || index != windows.WAIT_OBJECT_0 {
			return
		}
		g.fire()
	}
}

func (g *guard) fire() {
	g.mu.Lock()
	if g.closed {
		g.mu.Unlock()
		return
	}
	activate := g.activate
	if activate == nil {
		// 窗口还没创建：先记住这次请求，注册回调时立即补发。
		g.pending = true
	}
	g.mu.Unlock()
	if activate != nil {
		activate()
	}
}

func (g *guard) OnActivate(activate func()) {
	g.mu.Lock()
	g.activate = activate
	pending := g.pending
	g.pending = false
	g.mu.Unlock()
	if pending && activate != nil {
		activate()
	}
}

func (g *guard) Close() error {
	g.mu.Lock()
	if g.closed {
		g.mu.Unlock()
		return nil
	}
	g.closed = true
	g.activate = nil
	g.pending = false
	event, stop, mutex := g.event, g.stop, g.mutex
	g.event, g.stop, g.mutex = 0, 0, 0
	g.mu.Unlock()
	// 先置位停止事件让等待协程退出，再关闭句柄。
	var err error
	if stop != 0 {
		if setErr := windows.SetEvent(stop); setErr != nil {
			err = setErr
		}
	}
	for _, handle := range []windows.Handle{stop, event, mutex} {
		if handle == 0 {
			continue
		}
		if closeErr := windows.CloseHandle(handle); closeErr != nil && err == nil {
			err = closeErr
		}
	}
	return err
}

// requestActivation asks the running instance to show its window. Missing the
// request only means the holder is starting up or already exiting, so the
// failure is not reported: the second process exits either way.
func requestActivation(activateName string) {
	const asfwAny = 0xFFFFFFFF // ASFW_ANY: allow the running instance to take the foreground.
	if procAllowSetForegroundWindow.Find() == nil {
		_, _, _ = procAllowSetForegroundWindow.Call(asfwAny)
	}
	deadline := time.Now().Add(activationWait)
	for {
		event, err := windows.OpenEvent(windows.EVENT_MODIFY_STATE, false, windows.StringToUTF16Ptr(activateName))
		if err == nil {
			_ = windows.SetEvent(event)
			_ = windows.CloseHandle(event)
			return
		}
		if time.Now().After(deadline) {
			return
		}
		time.Sleep(activationRetry)
	}
}
