package wails

import (
	"errors"
	"omniproxy/internal/infrastructure/logger"
)

// LogsRead 返回有界的已脱敏结构化记录。
func (s *AppService) LogsRead(query logger.Query) (logger.Snapshot, error) {
	buffer := s.logFiles.Buffer()
	if buffer == nil {
		return logger.Snapshot{}, errors.New("logs: unavailable")
	}
	snapshot, err := buffer.Read(query)
	snapshot.RecordedLevel = s.logFiles.RecordedLevel()
	return snapshot, err
}

// LogsHead 返回清空显示所需的当前位置，不删除内存或文件日志。
func (s *AppService) LogsHead() (uint64, error) {
	if buffer := s.logFiles.Buffer(); buffer != nil {
		return buffer.Latest(), nil
	}
	return 0, errors.New("logs: unavailable")
}

// LogExportResult 报告导出条数及缓存中已过期的选择，取消不是导出成功。
type LogExportResult struct {
	Cancelled bool `json:"cancelled"`
	Count     int  `json:"count"`
	Missing   int  `json:"missing"`
}

// LogsExport 只接受记录 ID，正文从后端已脱敏缓存获取，文件路径由原生对话框提供。
func (s *AppService) LogsExport(ids []uint64, language string) (LogExportResult, error) {
	if language != "zh" && language != "en" {
		return LogExportResult{}, errors.New("logs: invalid_language")
	}
	buffer := s.logFiles.Buffer()
	if buffer == nil || s.selectLogExport == nil {
		return LogExportResult{}, errors.New("logs: unavailable")
	}
	if !s.logActions.TryLock() {
		return LogExportResult{}, errors.New("logs: busy")
	}
	defer s.logActions.Unlock()
	entries, missing, err := buffer.Select(ids)
	if err != nil {
		return LogExportResult{}, err
	}
	if len(entries) == 0 {
		return LogExportResult{}, errors.New("logs: selection_expired")
	}
	path, err := s.selectLogExport(language)
	if err != nil {
		return LogExportResult{}, errors.New("logs: dialog_failed")
	}
	if path == "" {
		return LogExportResult{Cancelled: true}, nil
	}
	if err := logger.WriteExport(path, entries); err != nil {
		return LogExportResult{}, err
	}
	if s.log != nil {
		s.log.Info("runtime logs exported", "module", "logs", "operation", "export")
	}
	return LogExportResult{Count: len(entries), Missing: missing}, nil
}

// LogsOpenDirectory 仅打开当前环境的固定日志目录，不接受外部路径。
func (s *AppService) LogsOpenDirectory() error {
	if s.prof == nil || s.prof.LogDir == "" || s.openLogDirectory == nil {
		return errors.New("logs: unavailable")
	}
	if err := s.openLogDirectory(s.prof.LogDir); err != nil {
		return errors.New("logs: directory_open_failed")
	}
	if s.log != nil {
		s.log.Info("runtime log directory opened", "module", "logs", "operation", "open_directory")
	}
	return nil
}
