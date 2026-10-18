package wails

import (
	"context"
	"errors"
	"sync"

	"omniproxy/internal/application/legacyimport"
	"omniproxy/internal/infrastructure/credential"
	"omniproxy/internal/infrastructure/legacy"
	"omniproxy/internal/platform/profile"
)

// 全部导入入口串行，防止重复点击或多窗口并行创建同一旧账号。
var legacyImportMu sync.Mutex

func (s *AppService) legacyDirectory(directory string) (string, error) {
	if directory == "" {
		if s.prof.Kind == profile.Test {
			return "", errors.New("legacy: explicit_test_directory_required")
		}
		return legacy.DefaultDirectory(s.prof.Kind == profile.Development)
	}
	return directory, nil
}

// PreviewLegacyImport 仅按用户点击读取旧版固定文件，返回无凭据的预览。
func (s *AppService) PreviewLegacyImport(directory string) (legacyimport.Preview, error) {
	legacyImportMu.Lock()
	defer legacyImportMu.Unlock()
	directory, err := s.legacyDirectory(directory)
	if err != nil {
		return legacyimport.Preview{}, err
	}
	if samePath(directory, s.prof.DataDir) {
		return legacyimport.Preview{}, errors.New("legacy: source_is_target")
	}
	store, err := credential.New(s.prof)
	if err != nil {
		return legacyimport.Preview{}, errors.New("legacy: target_unavailable")
	}
	return legacyimport.New(s.db, store).Preview(context.Background(), directory)
}

// ImportLegacyData 必须携带刚预览的源快照，避免导入用户没有确认的变化。
func (s *AppService) ImportLegacyData(directory, snapshot string) (legacyimport.Result, error) {
	legacyImportMu.Lock()
	defer legacyImportMu.Unlock()
	directory, err := s.legacyDirectory(directory)
	if err != nil {
		return legacyimport.Result{}, err
	}
	if samePath(directory, s.prof.DataDir) {
		return legacyimport.Result{}, errors.New("legacy: source_is_target")
	}
	store, err := credential.New(s.prof)
	if err != nil {
		return legacyimport.Result{}, errors.New("legacy: target_unavailable")
	}
	// 导入只新增账号；不触发目录同步，避免同时改写正在使用的客户端配置。
	return legacyimport.New(s.db, store).Import(context.Background(), directory, snapshot)
}
