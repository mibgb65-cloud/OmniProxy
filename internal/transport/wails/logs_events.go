package wails

import (
	"sync"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
	"omniproxy/internal/infrastructure/logger"
)

const logNotifyInterval = 150 * time.Millisecond

// LogNotice 只通知最新序号，日志正文通过有界绑定接口读取。
type LogNotice struct {
	LatestID uint64 `json:"latestId"`
}

func init() { application.RegisterEvent[LogNotice]("logs:updated") }

func (s *AppService) bindLogEvents(app *application.App, window application.Window) {
	s.selectLogExport = func(language string) (string, error) {
		title := "Export filtered logs"
		if language == "zh" {
			title = "导出筛选后的日志"
		}
		path, err := app.Dialog.SaveFileWithOptions(&application.SaveFileDialogOptions{Title: title}).
			SetFilename("omniproxy-logs-"+time.Now().Format("20060102-150405")+".jsonl").
			AddFilter("JSON Lines", "*.jsonl").AllowsOtherFileTypes(false).
			CanCreateDirectories(true).AttachToWindow(window).PromptForSingleSelection()
		// 当前固定 Wails 版本的 Windows 文件对话框使用此稳定取消错误。
		if err != nil && err.Error() == "cancelled by user" {
			return "", nil
		}
		return path, err
	}
	s.openLogDirectory = app.Browser.OpenFile
	buffer := s.logFiles.Buffer()
	if buffer == nil {
		return
	}
	s.stopLogs = forwardLogNotices(buffer, func(notice LogNotice) { app.Event.Emit("logs:updated", notice) })
}

func forwardLogNotices(buffer *logger.UIBuffer, emit func(LogNotice)) func() {
	notices, unsubscribe := buffer.Subscribe()
	stop, finished := make(chan struct{}), make(chan struct{})
	var once sync.Once
	stopForwarding := func() { once.Do(func() { close(stop); <-finished }) }
	go func() {
		defer close(finished)
		defer unsubscribe()
		ticker := time.NewTicker(logNotifyInterval)
		defer ticker.Stop()
		dirty := false
		for {
			select {
			case <-stop:
				return
			case _, open := <-notices:
				if !open {
					return
				}
				dirty = true
			case <-ticker.C:
				if dirty {
					dirty = false
					emit(LogNotice{LatestID: buffer.Latest()})
				}
			}
		}
	}()
	return stopForwarding
}

func (s *AppService) shutdownLogs() {
	if s.stopLogs != nil {
		s.stopLogs()
	}
}
