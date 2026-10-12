//go:build !windows

package agentconfig

import "path/filepath"

func canonicalConfigPath(path string) (string, error) {
	return filepath.EvalSymlinks(path)
}
