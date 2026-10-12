package logger

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
)

// WriteExport 先写临时文件，再替换用户选择的 JSONL 文件，防止失败时截断原文件。
func WriteExport(path string, entries []Entry) error {
	if !filepath.IsAbs(path) || !strings.EqualFold(filepath.Ext(path), ".jsonl") || len(entries) == 0 || len(entries) > MaxUIEntries {
		return errors.New("logs: invalid_export")
	}
	file, err := os.CreateTemp(filepath.Dir(path), ".omniproxy-log-export-*.tmp")
	if err != nil {
		return errors.New("logs: export_failed")
	}
	temporary := file.Name()
	defer func() { _ = os.Remove(temporary) }()
	encoder := json.NewEncoder(file)
	for _, entry := range entries {
		if err := encoder.Encode(entry); err != nil {
			_ = file.Close()
			return errors.New("logs: export_failed")
		}
	}
	if err := file.Sync(); err != nil {
		_ = file.Close()
		return errors.New("logs: export_failed")
	}
	if err := file.Close(); err != nil {
		return errors.New("logs: export_failed")
	}
	if err := os.Rename(temporary, path); err != nil {
		return errors.New("logs: export_failed")
	}
	return nil
}
