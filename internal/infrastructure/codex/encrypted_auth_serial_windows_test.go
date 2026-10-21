//go:build windows

package codex

import (
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestEncryptedAuthSerializesHighCostMissesAcrossHomes(t *testing.T) {
	var active, peak atomic.Int32
	cache, _, path := cacheFixture(t)
	cache.decrypt = func([]byte, []byte) ([]byte, error) {
		current := active.Add(1)
		for old := peak.Load(); current > old && !peak.CompareAndSwap(old, current); old = peak.Load() {
		}
		time.Sleep(5 * time.Millisecond)
		active.Add(-1)
		return []byte("synthetic-auth"), nil
	}
	start := make(chan struct{})
	var group sync.WaitGroup
	for index := range 8 {
		group.Go(func() {
			<-start
			value, err := cache.read(filepath.Join(path, string(rune('a'+index))), []byte("cipher"), []byte("pass"))
			clear(value)
			if err != nil {
				t.Error(err)
			}
		})
	}
	close(start)
	group.Wait()
	if peak.Load() != 1 || len(cache.entries) != 8 {
		t.Fatal("different authorization paths ran simultaneous KDF workspaces")
	}
}
