//go:build !windows

package codex

func DeleteCredentials(string) error { return ErrUnavailable }
