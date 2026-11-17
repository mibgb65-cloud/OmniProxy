package modelcatalog

import (
	"context"

	"omniproxy/internal/provider/workbuddyai"
)

// WorkBuddyAutoSync 保留独立入口，内部复用通用后台目录同步。
type WorkBuddyAutoSync struct{ *AutoSync }

// StartWorkBuddyAutoSync 在后台发现 WorkBuddy 订阅目录（国际版与国内版共用一次扫描）；
// 失败保留旧快照，由下一次授权或手动操作重试。
func StartWorkBuddyAutoSync(ctx context.Context, source *Service, refresh func(context.Context) error, report func(error)) *WorkBuddyAutoSync {
	return &WorkBuddyAutoSync{StartKindsAutoSync(ctx, source, []string{workbuddyai.Kind, workbuddyai.ChinaKind}, false, refresh, report)}
}
