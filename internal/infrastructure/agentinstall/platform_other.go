//go:build !windows

package agentinstall

import (
	"context"
	"os/exec"
)

func hideWindow(*exec.Cmd) {}
func (b *Backend) inspectDesktop(context.Context, Product) Detection {
	return unknown("unsupported_platform")
}
