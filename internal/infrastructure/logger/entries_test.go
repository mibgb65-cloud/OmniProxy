package logger

import (
	"fmt"
	"sync"
	"testing"
)

func TestReadPagedAndBounded(t *testing.T) {
	buffer := NewUIBuffer(4)
	for i := 1; i <= 6; i++ {
		buffer.addEntry(Entry{Message: fmt.Sprint(i), Module: fmt.Sprintf("module%d", i), Fields: map[string]string{"code": "ok"}})
	}
	first, err := buffer.Read(Query{Limit: 2})
	if err != nil || !first.Dropped || !first.HasMore || first.OldestID != 3 || first.NextCursor != 4 || len(first.Modules) != 4 {
		t.Fatalf("first = %+v, %v", first, err)
	}
	buffer.Add("7")
	second, err := buffer.Read(Query{After: first.NextCursor, Until: first.Until, Limit: 2})
	if err != nil || second.HasMore || second.NextCursor != 6 || len(second.Entries) != 2 {
		t.Fatalf("second = %+v, %v", second, err)
	}
	second.Entries[0].Fields["code"] = "mutated"
	entries, missing, err := buffer.Select([]uint64{3, 5, 6})
	if err != nil || missing != 1 || len(entries) != 2 || entries[0].Fields["code"] != "ok" {
		t.Fatalf("Select = %+v,%d,%v", entries, missing, err)
	}
}

func TestReadReportsCompleteEvictionAndRestart(t *testing.T) {
	buffer := NewUIBuffer(2)
	for i := 0; i < 5; i++ {
		buffer.Add("event")
	}
	result, err := buffer.Read(Query{After: 1, Until: 2, Limit: 2})
	if err != nil || !result.Dropped || result.NextCursor != 2 || len(result.Entries) != 0 || result.HasMore {
		t.Fatalf("evicted = %+v,%v", result, err)
	}
	result, err = buffer.Read(Query{After: 100, Limit: 2})
	if err != nil || !result.Reset || result.NextCursor != 5 || len(result.Entries) != 2 {
		t.Fatalf("restart = %+v,%v", result, err)
	}
}

func TestReadRejectsInvalidInput(t *testing.T) {
	buffer := NewUIBuffer(2)
	buffer.Add("event")
	for _, query := range []Query{{Limit: 0}, {Limit: 201}, {After: maxSafeCursor + 1, Limit: 1}, {Until: 2, Limit: 1}} {
		if _, err := buffer.Read(query); err == nil {
			t.Fatalf("accepted %+v", query)
		}
	}
	for _, ids := range [][]uint64{{}, {0}, {1, 1}, {maxSafeCursor + 1}} {
		if _, _, err := buffer.Select(ids); err == nil {
			t.Fatalf("accepted ids %+v", ids)
		}
	}
}

func TestBufferSubscriberIsNonBlockingAndCancelable(t *testing.T) {
	buffer := NewUIBuffer(8)
	notices, cancel := buffer.Subscribe()
	var wait sync.WaitGroup
	for i := 0; i < 4; i++ {
		wait.Add(1)
		go func() {
			defer wait.Done()
			for n := 0; n < 1000; n++ {
				buffer.Add("event")
			}
		}()
	}
	wait.Wait()
	if buffer.Latest() != 4000 || len(notices) != 1 {
		t.Fatal("notification was not coalesced")
	}
	cancel()
	cancel()
	if _, open := <-notices; !open {
		t.Fatal("queued notice unexpectedly missing")
	}
	if _, open := <-notices; open {
		t.Fatal("subscriber not closed")
	}
	buffer.Add("after cancellation")
}
