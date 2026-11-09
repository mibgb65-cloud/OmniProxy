package modelcatalog

import "context"

// GoogleAutoSync 保留既有入口，内部复用通用后台目录同步。
type GoogleAutoSync struct{ *AutoSync }

// StartGoogleAutoSync 在后台发现 Google 订阅目录；失败保留旧快照，由下一次授权或手动操作重试。
func StartGoogleAutoSync(ctx context.Context, source *Service, refresh func(context.Context) error, report func(error)) *GoogleAutoSync {
	return &GoogleAutoSync{StartAutoSync(ctx, source, "google", false, refresh, report)}
}

// OpenAIAutoSync 只补齐当前版本没有目录的 OpenAI 订阅账号。
type OpenAIAutoSync struct{ *AutoSync }

// StartOpenAIAutoSync 在启动、授权变化或账号更新后补齐过期目录。
func StartOpenAIAutoSync(ctx context.Context, source *Service, refresh func(context.Context) error, report func(error)) *OpenAIAutoSync {
	return &OpenAIAutoSync{StartAutoSync(ctx, source, "openai", true, refresh, report)}
}
