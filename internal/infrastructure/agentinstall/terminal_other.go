//go:build !windows

package agentinstall

import "context"

func runInstallCommand(ctx context.Context, path string, args []string, report func(Progress)) error {
	writer := &progressWriter{report: report}
	err := runObservedCommand(ctx, path, args, writer)
	writer.flush()
	return err
}
