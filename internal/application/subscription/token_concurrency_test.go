package subscription

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"
)

// 模拟已经取到凭据、但上游仍在输出的长请求，不访问真实系统凭据或网络。
func TestStreamingTokenConsumerDoesNotBlockQuotaOrAnotherRequest(t *testing.T) {
	f := newFixture(t)
	view, client, home := f.begin(t, "")
	client.authorize("stream@example.test")
	item := awaitState(t, f.manager, view.ID, "completed").Account
	otherView, otherClient, _ := f.begin(t, "")
	otherClient.authorize("other@example.test")
	other := awaitState(t, f.manager, otherView.ID, "completed").Account
	f.manager.readToken = func(_ context.Context, path, expectedID, _ string, consume func([]byte, string) error) error {
		if filepath.Clean(path) != filepath.Clean(home) || expectedID != item.AuthAccountID {
			return failure("identity_changed")
		}
		key := []byte("test-only-token")
		defer clear(key)
		return consume(key, expectedID)
	}
	started, release := make(chan struct{}), make(chan struct{})
	done := make(chan error, 1)
	results := make(chan error, 3)
	received := 0
	go func() {
		done <- f.manager.UseTokenAt(context.Background(), item.ID, item.UpdatedAt, func([]byte, string) error {
			close(started)
			<-release
			return nil
		})
	}()
	<-started
	defer func() {
		close(release)
		if err := <-done; err != nil {
			t.Error("first token consumer failed")
		}
		for received < 3 {
			<-results
			received++
		}
	}()
	go func() {
		_, err := f.manager.ReadQuota(context.Background(), item.ID)
		results <- err
	}()
	go func() {
		_, err := f.manager.ReadQuota(context.Background(), other.ID)
		results <- err
	}()
	go func() {
		results <- f.manager.UseTokenAt(context.Background(), item.ID, item.UpdatedAt, func([]byte, string) error { return nil })
	}()
	for received < 3 {
		select {
		case err := <-results:
			received++
			if err != nil {
				t.Fatal("independent operation failed during streaming")
			}
		case <-time.After(time.Second):
			t.Fatal("streaming token consumer blocked quota or another request")
		}
	}
}

func TestPreparedTokenIsIndependentAndClearedAfterConsumption(t *testing.T) {
	for _, consumerFails := range []bool{false, true} {
		t.Run(map[bool]string{false: "success", true: "failure"}[consumerFails], func(t *testing.T) {
			f := newFixture(t)
			view, client, _ := f.begin(t, "")
			client.authorize("token@example.test")
			item := awaitState(t, f.manager, view.ID, "completed").Account
			var borrowed, consumed []byte
			readerReturned := false
			f.manager.readToken = func(_ context.Context, _ string, expectedID, _ string, callback func([]byte, string) error) error {
				borrowed = []byte("test-only-token")
				defer func() { clear(borrowed); readerReturned = true }()
				return callback(borrowed, expectedID)
			}
			want := errors.New("test-consumer-error")
			err := f.manager.UseToken(context.Background(), item.ID, func(key []byte, officialID string) error {
				consumed = key
				if !readerReturned || string(key) != "test-only-token" || officialID != item.AuthAccountID {
					return errors.New("token consumer ran inside credential preparation")
				}
				if consumerFails {
					return want
				}
				return nil
			})
			if consumerFails && !errors.Is(err, want) || !consumerFails && err != nil {
				t.Fatal("consumer result was not preserved")
			}
			for _, buffer := range [][]byte{borrowed, consumed} {
				if len(buffer) == 0 {
					t.Fatal("credential path was not exercised")
				}
				for _, value := range buffer {
					if value != 0 {
						t.Fatal("temporary token buffer was not cleared")
					}
				}
			}
		})
	}
}
