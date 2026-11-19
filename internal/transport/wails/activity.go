package wails

import (
	"context"
	"omniproxy/internal/application/activity"
)

// AccountTokenUsage 仅读取账号元数据和本地用量，不读取或查询 API Key。
func (s *AppService) AccountTokenUsage(id string) (*activity.AccountTokenUsage, error) {
	if s.activity == nil {
		return nil, activity.ErrUnavailable
	}
	accounts, err := s.accountService()
	if err != nil {
		return nil, err
	}
	ctx := context.Background()
	item, err := accounts.Metadata(ctx, id)
	if err != nil {
		return nil, err
	}
	return s.activity.AccountUsage(ctx, item)
}

func (s *AppService) UsageRead() (activity.UsageInfo, error) {
	if s.activity == nil {
		return activity.UsageInfo{}, activity.ErrUnavailable
	}
	return s.activity.Usage(context.Background())
}
func (s *AppService) RequestsRead(query activity.RequestQuery) (activity.RequestPage, error) {
	if s.activity == nil {
		return activity.RequestPage{}, activity.ErrUnavailable
	}
	return s.activity.Requests(context.Background(), query)
}
func (s *AppService) RequestRead(id string) (activity.RequestItem, error) {
	if s.activity == nil {
		return activity.RequestItem{}, activity.ErrUnavailable
	}
	return s.activity.Request(context.Background(), id)
}
func (s *AppService) RequestRetention() (activity.Retention, error) {
	if s.activity == nil {
		return activity.Retention{}, activity.ErrUnavailable
	}
	return s.activity.Retention(context.Background())
}
func (s *AppService) RequestRetentionSet(limit int) (activity.Retention, error) {
	if s.activity == nil {
		return activity.Retention{}, activity.ErrUnavailable
	}
	return s.activity.SetRetention(context.Background(), limit)
}

// RequestContentRead 按需读取一条请求正文；未记录时返回 Available=false。
func (s *AppService) RequestContentRead(id string) (activity.RequestContent, error) {
	if s.activity == nil {
		return activity.RequestContent{}, activity.ErrUnavailable
	}
	return s.activity.RequestContent(context.Background(), id)
}

// RequestContentSet 切换请求正文记录开关；关闭后只停止记录新正文。
func (s *AppService) RequestContentSet(enabled bool) (activity.Retention, error) {
	if s.activity == nil {
		return activity.Retention{}, activity.ErrUnavailable
	}
	return s.activity.SetContentCapture(context.Background(), enabled)
}

// RequestContentClear 删除全部已保存正文，返回删除条数。
func (s *AppService) RequestContentClear() (int64, error) {
	if s.activity == nil {
		return 0, activity.ErrUnavailable
	}
	return s.activity.ClearRequestContents(context.Background())
}
