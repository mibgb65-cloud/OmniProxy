package codex

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestEncryptedAuthWaiterCanCancelWithoutStartingAnotherKDF(t *testing.T) {
	started, release := make(chan struct{}), make(chan struct{})
	cache := newEncryptedAuthCache(func([]byte, []byte) ([]byte, error) {
		close(started)
		<-release
		return nil, ErrUnavailable
	})
	owner := make(chan error, 1)
	go func() {
		_, err := cache.readContext(context.Background(), "test-path", []byte("test-cipher"), []byte("test-passphrase"))
		owner <- err
	}()
	<-started
	defer func() {
		close(release)
		<-owner
	}()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()
	start := time.Now()
	data, err := cache.readContext(ctx, "other-path", []byte("other-cipher"), []byte("other-passphrase"))
	if !errors.Is(err, context.DeadlineExceeded) || len(data) != 0 || time.Since(start) > time.Second {
		t.Fatal("排队等待 KDF 时无法取消", err)
	}
}
