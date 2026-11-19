package wails

import (
	"context"
	"errors"
	"strconv"
	"time"

	"omniproxy/internal/application/activity"
	"omniproxy/internal/infrastructure/xlsx"
)

// RequestExportResult 报告导出条数与是否因行数上限被截断；取消不是导出成功。
type RequestExportResult struct {
	Count     int  `json:"count"`
	Limited   bool `json:"limited"`
	Cancelled bool `json:"cancelled"`
}

var errRequestsExport = errors.New("requests: export_failed")

// RequestsExport 按当前筛选导出全部匹配记录；文件路径只能来自原生保存对话框。
func (s *AppService) RequestsExport(query activity.RequestQuery, language string) (RequestExportResult, error) {
	if s.activity == nil {
		return RequestExportResult{}, activity.ErrUnavailable
	}
	if language != "zh" && language != "en" {
		return RequestExportResult{}, errors.New("requests: invalid_language")
	}
	if s.selectRequestsExport == nil {
		return RequestExportResult{}, errors.New("requests: unavailable")
	}
	if !s.requestsExportMu.TryLock() {
		return RequestExportResult{}, errors.New("requests: busy")
	}
	defer s.requestsExportMu.Unlock()
	path, err := s.selectRequestsExport(language)
	if err != nil {
		return RequestExportResult{}, errors.New("requests: dialog_failed")
	}
	if path == "" {
		return RequestExportResult{Cancelled: true}, nil
	}
	var limited bool
	ctx := context.Background()
	written, writeErr := xlsx.Write(path, sheetName(language), exportHeaders(language), func(yield func([]xlsx.Cell) error) error {
		var exportErr error
		_, limited, exportErr = s.activity.ExportRequests(ctx, query, func(row activity.ExportRequest) error {
			return yield(exportCells(row, language))
		})
		return exportErr
	})
	if writeErr != nil {
		return RequestExportResult{}, errRequestsExport
	}
	if s.log != nil {
		s.log.Info("requests exported", "module", "activity", "operation", "export", "count", written, "limited", limited)
	}
	return RequestExportResult{Count: written, Limited: limited}, nil
}

func sheetName(language string) string {
	if language == "zh" {
		return "请求记录"
	}
	return "Requests"
}

// exportHeaders 是导出列的中英文表头；顺序与 exportCells 一致。
func exportHeaders(language string) []string {
	if language == "zh" {
		return []string{
			"请求时间", "请求 ID", "结果", "协议 / 传输", "模型", "输入模型", "推理等级",
			"服务商", "账号", "邮箱", "工作区", "接入客户端", "输入 Token", "输出 Token",
			"缓存命中 Token", "用量状态", "HTTP 状态码", "错误代码", "耗时(ms)", "首字(ms)",
			"估算成本(USD)", "请求正文", "正文已记录", "正文截断",
		}
	}
	return []string{
		"Time", "Request ID", "Outcome", "Protocol / transport", "Model", "Input model", "Effort",
		"Provider", "Account", "Email", "Workspace ID", "Client", "Input tokens", "Output tokens",
		"Cached tokens", "Usage status", "HTTP status", "Error code", "Latency (ms)", "First token (ms)",
		"Estimated cost (USD)", "Request content", "Content recorded", "Content truncated",
	}
}

// exportCells 把一行请求记录转成单元格；正文按 Excel 单元格上限截断并标记。
func exportCells(row activity.ExportRequest, language string) []xlsx.Cell {
	item := row.Item
	content, excelTruncated := xlsx.FitCell(row.Content.Content)
	return []xlsx.Cell{
		{Text: exportTime(item.StartedAt)},
		{Text: item.ID},
		{Text: stateText(item.State, language)},
		{Text: item.Protocol + " / " + item.Transport},
		{Text: item.Model},
		{Text: item.InputModel},
		{Text: item.Effort},
		{Text: item.ProviderName},
		{Text: item.AccountName},
		{Text: item.Email},
		{Text: item.WorkspaceID},
		{Text: item.Client},
		numberCell(item.InputTokens, item.InputKnown),
		numberCell(item.OutputTokens, item.OutputKnown),
		numberCell(item.CachedInputTokens, item.CachedInputKnown),
		{Text: usageText(item, language)},
		numberCell(strconv.Itoa(item.StatusCode), item.StatusCode > 0),
		{Text: item.ErrorCode},
		numberCell(strconv.FormatInt(item.LatencyMS, 10), item.State != "pending"),
		firstTokenCell(item.FirstTokenMS),
		{Text: exportCost(item)},
		{Text: content},
		{Text: yesNo(row.Content.Available, language)},
		{Text: yesNo(row.Content.Truncated || excelTruncated, language)},
	}
}

func numberCell(value string, known bool) xlsx.Cell {
	if !known || value == "" {
		return xlsx.Cell{}
	}
	return xlsx.Cell{Number: value}
}

func firstTokenCell(value *int64) xlsx.Cell {
	if value == nil || *value < 0 {
		return xlsx.Cell{}
	}
	return xlsx.Cell{Number: strconv.FormatInt(*value, 10)}
}

func exportTime(value string) string {
	parsed, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		return value
	}
	return parsed.Local().Format("2006-01-02 15:04:05")
}

func exportCost(item activity.RequestItem) string {
	if !item.CostKnown {
		return ""
	}
	if item.CostMin == item.CostMax {
		return strconv.FormatFloat(item.CostMin, 'f', 6, 64)
	}
	return strconv.FormatFloat(item.CostMin, 'f', 6, 64) + "–" + strconv.FormatFloat(item.CostMax, 'f', 6, 64)
}

func stateText(state, language string) string {
	switch state {
	case "success":
		return pick(language, "成功", "Success")
	case "failed":
		return pick(language, "失败", "Failed")
	case "cancelled":
		return pick(language, "已取消", "Cancelled")
	case "pending":
		return pick(language, "进行中", "In progress")
	default:
		return state
	}
}

func usageText(item activity.RequestItem, language string) string {
	switch {
	case item.UsageComplete:
		return pick(language, "完整", "Complete")
	case item.InputKnown || item.OutputKnown:
		return pick(language, "部分", "Partial")
	default:
		return pick(language, "未知", "Unknown")
	}
}

func yesNo(value bool, language string) string {
	if value {
		return pick(language, "是", "Yes")
	}
	return pick(language, "否", "No")
}

func pick(language, zh, en string) string {
	if language == "zh" {
		return zh
	}
	return en
}
