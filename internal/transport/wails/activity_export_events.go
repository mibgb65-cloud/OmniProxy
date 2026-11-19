package wails

import (
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// bindActivityExport 安装请求导出用的原生保存对话框；路径始终由用户选择。
func (s *AppService) bindActivityExport(app *application.App, window application.Window) {
	s.selectRequestsExport = func(language string) (string, error) {
		title := "Export requests to XLSX"
		if language == "zh" {
			title = "导出请求记录为 XLSX"
		}
		path, err := app.Dialog.SaveFileWithOptions(&application.SaveFileDialogOptions{Title: title}).
			SetFilename("omniproxy-requests-"+time.Now().Format("20060102-150405")+".xlsx").
			AddFilter("Excel Workbook", "*.xlsx").AllowsOtherFileTypes(false).
			CanCreateDirectories(true).AttachToWindow(window).PromptForSingleSelection()
		// 与日志导出一致：当前 Wails 版本的 Windows 对话框用这个稳定取消错误。
		if err != nil && err.Error() == "cancelled by user" {
			return "", nil
		}
		return path, err
	}
}
