package wails

import (
	"testing"
	"time"
)

func TestActivityChangesAreCoalescedAndShutdownIsIdempotent(t *testing.T) {
	updates := make(chan struct{}, 4)
	emitted := make(chan struct{}, 4)
	stop := forwardActivityChanges(updates, func() { emitted <- struct{}{} })
	for i := 0; i < 4; i++ {
		updates <- struct{}{}
	}
	select {
	case <-emitted:
	case <-time.After(2 * time.Second):
		t.Fatal("no activity update")
	}
	stop()
	stop()
	select {
	case <-emitted:
		t.Fatal("burst emitted duplicate notices")
	default:
	}
}
