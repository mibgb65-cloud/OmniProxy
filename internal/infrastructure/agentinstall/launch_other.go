//go:build !windows

package agentinstall

import "context"

// Launch 在非目标平台明确拒绝启动。
func Launch(context.Context, string) error { return ErrUnavailable }
