package logger

import (
	"fmt"
	"sync"
	"testing"
)

func TestUIBufferKeepsOldestFirstOrder(t *testing.T) {
	b := NewUIBuffer(3)
	for i := 1; i <= 3; i++ {
		b.Add(fmt.Sprintf("line-%d", i))
	}
	got := b.Lines()
	want := []string{"line-1", "line-2", "line-3"}
	if len(got) != len(want) {
		t.Fatalf("Lines() = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("Lines() = %v, want %v", got, want)
		}
	}
}

func TestUIBufferEvictsOldest(t *testing.T) {
	b := NewUIBuffer(2)
	b.Add("a")
	b.Add("b")
	b.Add("c")
	got := b.Lines()
	if len(got) != 2 || got[0] != "b" || got[1] != "c" {
		t.Fatalf("Lines() = %v, want [b c]", got)
	}
	if b.Len() != 2 {
		t.Fatalf("Len() = %d, want 2", b.Len())
	}
}

func TestUIBufferConcurrentAccess(t *testing.T) {
	b := NewUIBuffer(16)
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				b.Add(fmt.Sprintf("g%d-%d", n, j))
				_ = b.Lines()
			}
		}(i)
	}
	wg.Wait()
	if b.Len() != 16 {
		t.Fatalf("Len() = %d, want 16", b.Len())
	}
}

func TestLineWriterSplitsOnNewlines(t *testing.T) {
	b := NewUIBuffer(8)
	w := &lineWriter{buf: b}
	if _, err := w.Write([]byte("first\nsec")); err != nil {
		t.Fatal(err)
	}
	if b.Len() != 1 {
		t.Fatalf("Len() = %d after partial line, want 1", b.Len())
	}
	if _, err := w.Write([]byte("ond\n\nthird\n")); err != nil {
		t.Fatal(err)
	}
	got := b.Lines()
	want := []string{"first", "second", "third"}
	if len(got) != len(want) {
		t.Fatalf("Lines() = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("Lines() = %v, want %v", got, want)
		}
	}
}

func TestNewUIBufferRejectsNonPositiveCapacity(t *testing.T) {
	if got := NewUIBuffer(0); got == nil {
		t.Fatal("NewUIBuffer(0) = nil, want usable buffer")
	}
}
