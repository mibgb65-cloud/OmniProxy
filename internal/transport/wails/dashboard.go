package wails

import (
	"context"
	"omniproxy/internal/application/activity"
	"time"
)

// DashboardRead 读取本地运行统计，不查询官方配额或调用模型。
func (s *AppService) DashboardRead(query activity.DashboardQuery) (activity.DashboardInfo, error) {
	if s.activity == nil {
		return activity.DashboardInfo{}, activity.ErrUnavailable
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return s.activity.Dashboard(ctx, query)
}
