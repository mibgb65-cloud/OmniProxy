//go:build !windows

package agentconfig

import "os"

func replaceConfigFile(temporary, path string, _ []byte, _ string) error {
	if os.Rename(temporary, path) != nil {
		return failure("write_failed")
	}
	return nil
}
