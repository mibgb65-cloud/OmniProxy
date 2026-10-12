package logger

import (
	"strings"
	"sync"
)

// UIBuffer is a bounded, concurrency-safe ring buffer of formatted log lines.
// The desktop UI streams these lines to the logs page through Wails events.
type UIBuffer struct {
	mu          sync.Mutex
	entries     []Entry
	latest      uint64
	subscribers map[chan struct{}]struct{}
	next        int
	count       int
}

// NewUIBuffer creates a buffer retaining at most capacity lines.
func NewUIBuffer(capacity int) *UIBuffer {
	if capacity <= 0 {
		capacity = 1
	}
	if capacity > MaxUIEntries {
		capacity = MaxUIEntries
	}
	return &UIBuffer{entries: make([]Entry, capacity), subscribers: make(map[chan struct{}]struct{})}
}

// Add appends a line, evicting the oldest line when the buffer is full.
func (b *UIBuffer) Add(line string) {
	b.addEntry(Entry{Message: line, Fields: map[string]string{}})
}

func (b *UIBuffer) addEntry(entry Entry) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.latest++
	entry.ID = b.latest
	b.entries[b.next] = copyEntry(entry)
	b.next = (b.next + 1) % len(b.entries)
	if b.count < len(b.entries) {
		b.count++
	}
	for subscriber := range b.subscribers {
		select {
		case subscriber <- struct{}{}:
		default:
		}
	}
}

// Lines returns a copy of the buffered lines, oldest first.
func (b *UIBuffer) Lines() []string {
	b.mu.Lock()
	defer b.mu.Unlock()
	out := make([]string, 0, b.count)
	start := (b.next - b.count + len(b.entries)) % len(b.entries)
	for i := 0; i < b.count; i++ {
		out = append(out, b.entries[(start+i)%len(b.entries)].Message)
	}
	return out
}

// Latest 返回当前序号，用于合并通知与清空显示的边界。
func (b *UIBuffer) Latest() uint64 { b.mu.Lock(); defer b.mu.Unlock(); return b.latest }

// Subscribe 的有界信号队列合并重复通知，不阻塞日志写入。
func (b *UIBuffer) Subscribe() (<-chan struct{}, func()) {
	channel := make(chan struct{}, 1)
	b.mu.Lock()
	b.subscribers[channel] = struct{}{}
	b.mu.Unlock()
	var once sync.Once
	return channel, func() {
		once.Do(func() {
			b.mu.Lock()
			defer b.mu.Unlock()
			delete(b.subscribers, channel)
			close(channel)
		})
	}
}

// Len returns the number of buffered lines.
func (b *UIBuffer) Len() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.count
}

// lineWriter adapts a UIBuffer to io.Writer, splitting input on newlines and
// dropping the trailing partial line until it is terminated.
type lineWriter struct {
	buf     *UIBuffer
	partial strings.Builder
}

func (w *lineWriter) Write(p []byte) (int, error) {
	w.partial.Write(p)
	for {
		s := w.partial.String()
		i := strings.IndexByte(s, '\n')
		if i < 0 {
			break
		}
		line := strings.TrimRight(s[:i], "\r")
		if line != "" {
			w.buf.Add(line)
		}
		w.partial.Reset()
		w.partial.WriteString(s[i+1:])
	}
	return len(p), nil
}
