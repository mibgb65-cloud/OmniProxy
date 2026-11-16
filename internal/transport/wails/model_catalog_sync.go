package wails

import "omniproxy/internal/application/modelcatalog"

// requestModelSync 通知所有按账号发现目录的后台任务：账号、服务商或授权变化后
// 由各自实例判断是否真的需要访问上游。
func (s *AppService) requestModelSync() {
	s.requestOpenAIModelSync()
	s.requestHowToFixModelSync()
}

// ModelCatalogSyncStatus 返回后台目录同步的只读状态，不包含账号身份或上游正文。
// 多个服务商各自后台同步时合并成一份状态：任一在运行即显示同步中，
// 待同步数量求和，错误码取第一个，时间取最近一次。
func (s *AppService) ModelCatalogSyncStatus() modelcatalog.SyncStatus {
	return mergeSyncStatus(s.openAIStatus(), s.howToFixSyncStatus())
}

// mergeSyncStatus 合并后台同步状态；空状态不参与覆盖。
func mergeSyncStatus(values ...modelcatalog.SyncStatus) modelcatalog.SyncStatus {
	merged := modelcatalog.SyncStatus{}
	for _, value := range values {
		merged.Running = merged.Running || value.Running
		merged.Stale += value.Stale
		if merged.LastError == "" {
			merged.LastError = value.LastError
		}
		if value.UpdatedAt > merged.UpdatedAt {
			merged.UpdatedAt = value.UpdatedAt
		}
	}
	return merged
}
