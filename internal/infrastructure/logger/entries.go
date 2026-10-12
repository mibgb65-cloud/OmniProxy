package logger

import (
	"errors"
	"sort"
	"strings"
	"unicode/utf8"
)

const (
	MaxUIEntries   = 5000
	MaxReadEntries = 200
	maxSafeCursor  = 1<<53 - 1
	maxLogText     = 2048
)

// Entry 是已脱敏的运行记录；字段只保留诊断元数据。
type Entry struct {
	ID      uint64            `json:"id"`
	Time    string            `json:"time"`
	Level   string            `json:"level"`
	Module  string            `json:"module"`
	Message string            `json:"message"`
	Fields  map[string]string `json:"fields"`
}

// Query 使用固定上界分批读取，避免高频日志使一次读取永远无法结束。
type Query struct {
	After uint64 `json:"after"`
	Until uint64 `json:"until"`
	Limit int    `json:"limit"`
}

// Snapshot 返回游标、缓存边界及本批记录，不读取磁盘历史。
type Snapshot struct {
	Entries       []Entry  `json:"entries"`
	NextCursor    uint64   `json:"nextCursor"`
	LatestID      uint64   `json:"latestId"`
	OldestID      uint64   `json:"oldestId"`
	Until         uint64   `json:"until"`
	Capacity      int      `json:"capacity"`
	Modules       []string `json:"modules"`
	HasMore       bool     `json:"hasMore"`
	Dropped       bool     `json:"dropped"`
	Reset         bool     `json:"reset"`
	RecordedLevel string   `json:"recordedLevel"`
}

func copyEntry(entry Entry) Entry {
	entry.Fields = cloneFields(entry.Fields)
	return entry
}

func cloneFields(fields map[string]string) map[string]string {
	out := make(map[string]string, len(fields))
	for key, value := range fields {
		out[key] = value
	}
	return out
}

func boundedText(text string) string {
	text = strings.ToValidUTF8(text, "�")
	if len(text) <= maxLogText {
		return text
	}
	text = text[:maxLogText]
	for !utf8.ValidString(text) {
		text = text[:len(text)-1]
	}
	return text + "…"
}

// Read 返回独立副本；缓存覆盖时明确报告间隙，重启后旧游标会复位。
func (b *UIBuffer) Read(query Query) (Snapshot, error) {
	if query.Limit < 1 || query.Limit > MaxReadEntries || query.After > maxSafeCursor || query.Until > maxSafeCursor {
		return Snapshot{}, errors.New("logs: invalid_query")
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	out := Snapshot{Entries: []Entry{}, Modules: []string{}, Capacity: len(b.entries), LatestID: b.latest}
	if query.After > b.latest {
		query.After, query.Until, out.Reset = 0, 0, true
	}
	if query.Until == 0 {
		query.Until = b.latest
	}
	if query.Until > b.latest || query.After > query.Until {
		return Snapshot{}, errors.New("logs: invalid_query")
	}
	out.Until, out.NextCursor = query.Until, query.After
	modules := make(map[string]bool)
	start := (b.next - b.count + len(b.entries)) % len(b.entries)
	for i := 0; i < b.count; i++ {
		entry := b.entries[(start+i)%len(b.entries)]
		if i == 0 {
			out.OldestID = entry.ID
		}
		modules[entry.Module] = true
		if entry.ID <= query.After || entry.ID > query.Until {
			continue
		}
		if len(out.Entries) == query.Limit {
			out.HasMore = true
			continue
		}
		out.Entries = append(out.Entries, copyEntry(entry))
		out.NextCursor = entry.ID
	}
	// 上界之前的记录全部被覆盖时，也要推进游标，避免永久重试同一批。
	if !out.HasMore {
		out.NextCursor = query.Until
	}
	out.Dropped = out.OldestID > query.After+1
	for module := range modules {
		out.Modules = append(out.Modules, module)
	}
	sort.Strings(out.Modules)
	return out, nil
}

// Select 获取导出点击时仍保留的记录；失效条目数量必须向界面报告。
func (b *UIBuffer) Select(ids []uint64) ([]Entry, int, error) {
	if len(ids) == 0 || len(ids) > MaxUIEntries {
		return nil, 0, errors.New("logs: invalid_selection")
	}
	selected := make(map[uint64]bool, len(ids))
	for _, id := range ids {
		if id == 0 || id > maxSafeCursor || selected[id] {
			return nil, 0, errors.New("logs: invalid_selection")
		}
		selected[id] = true
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	out := make([]Entry, 0, len(ids))
	start := (b.next - b.count + len(b.entries)) % len(b.entries)
	for i := 0; i < b.count; i++ {
		entry := b.entries[(start+i)%len(b.entries)]
		if selected[entry.ID] {
			out = append(out, copyEntry(entry))
		}
	}
	return out, len(ids) - len(out), nil
}
