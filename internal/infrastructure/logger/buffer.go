package logger

import (
	"strings"
	"sync"
)

// UIBuffer is a bounded, concurrency-safe ring buffer of formatted log lines.
// The desktop UI streams these lines to the logs page through Wails events.
type UIBuffer struct {
	mu    sync.Mutex
	lines []string
	next  int
	count int
}

// NewUIBuffer creates a buffer retaining at most capacity lines.
func NewUIBuffer(capacity int) *UIBuffer {
	if capacity <= 0 {
		capacity = 1
	}
	return &UIBuffer{lines: make([]string, capacity)}
}

// Add appends a line, evicting the oldest line when the buffer is full.
func (b *UIBuffer) Add(line string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.lines[b.next] = line
	b.next = (b.next + 1) % len(b.lines)
	if b.count < len(b.lines) {
		b.count++
	}
}

// Lines returns a copy of the buffered lines, oldest first.
func (b *UIBuffer) Lines() []string {
	b.mu.Lock()
	defer b.mu.Unlock()
	out := make([]string, 0, b.count)
	start := (b.next - b.count + len(b.lines)) % len(b.lines)
	for i := 0; i < b.count; i++ {
		out = append(out, b.lines[(start+i)%len(b.lines)])
	}
	return out
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
