package wails

import (
	"context"
	"omniproxy/internal/application/activity"
)

// RiverAccountTokenUsage 仅读取账号元数据和本地用量，不读取或查询 API Key。
func (s *AppService) RiverAccountTokenUsage(id string) (*activity.AccountTokenUsage, error) {
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
	return s.activity.RiverTokenUsage(ctx, item)
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
