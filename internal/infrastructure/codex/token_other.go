//go:build !windows

package codex

import "context"

func UseToken(string, string, func([]byte, string) error) error { return ErrUnavailable }

func useStoredAuth(string, func([]byte) error) error { return ErrUnavailable }

func useStoredAuthContext(ctx context.Context, _ string, _ func([]byte) error) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}
	return ErrUnavailable
}
