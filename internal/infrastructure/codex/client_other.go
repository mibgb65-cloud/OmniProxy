//go:build !windows

package codex

import "os/exec"

func hideWindow(_ *exec.Cmd) {}
