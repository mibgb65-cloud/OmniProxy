//go:build !windows

package codex

func UseToken(string, string, func([]byte, string) error) error { return ErrUnavailable }
