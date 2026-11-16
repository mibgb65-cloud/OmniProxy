package gateway

import (
	"encoding/json"
	"testing"
)

// 工具结果必须带非空字符串 call_id；缺失、null、空串与非法类型都算无法表达的孤儿结果。
func TestWithoutOrphanToolOutputsDropsOnlyOrphanResults(t *testing.T) {
	raw := json.RawMessage(`[
		{"role":"user","content":"Hi"},
		{"type":"function_call","call_id":"call_1","name":"exec_command","arguments":"{}"},
		{"type":"function_call_output","call_id":"call_1","output":"ok"},
		{"type":"function_call_output","name":"codex_apps_open_page","output":"{\"page_id\":null}"},
		{"type":"function_call_output","call_id":null,"output":"ok"},
		{"type":"function_call_output","call_id":"","output":"ok"},
		{"type":"function_call_output","call_id":"   ","output":"ok"},
		{"type":"function_call_output","call_id":7,"output":"ok"}
	]`)
	kept, dropped := withoutOrphanToolOutputs(raw)
	if dropped != 5 {
		t.Fatalf("dropped = %d, want 5", dropped)
	}
	if len(kept) != 3 {
		t.Fatalf("kept = %d, want 3: %s", len(kept), rawOf(kept))
	}
	if string(kept[0]) != `{"role":"user","content":"Hi"}` {
		t.Fatalf("non-tool item changed: %s", kept[0])
	}
	var call struct {
		Type   string `json:"type"`
		CallID string `json:"call_id"`
	}
	if json.Unmarshal(kept[1], &call) != nil || call.Type != "function_call" || call.CallID != "call_1" {
		t.Fatalf("function_call changed: %s", kept[1])
	}
	var output struct {
		Type   string `json:"type"`
		CallID string `json:"call_id"`
		Output string `json:"output"`
	}
	if json.Unmarshal(kept[2], &output) != nil || output.Type != "function_call_output" || output.CallID != "call_1" || output.Output != "ok" {
		t.Fatalf("legitimate tool output changed: %s", kept[2])
	}
}

// 缺少 input、input 不是数组时不做改动，避免把无法解析的正文改坏。
func TestWithoutOrphanToolOutputsKeepsUnexpectedInput(t *testing.T) {
	for _, raw := range []json.RawMessage{nil, json.RawMessage(`null`), json.RawMessage(`"text"`), json.RawMessage(`{}`)} {
		kept, dropped := withoutOrphanToolOutputs(raw)
		if dropped != 0 || kept != nil {
			t.Fatalf("input %s changed: kept=%v dropped=%d", raw, kept, dropped)
		}
	}
}

// 命名空间分组仍然被移除，函数工具与其它形状原样保留。
func TestWithoutNamespaceToolsDropsOnlyGroups(t *testing.T) {
	raw := json.RawMessage(`[
		{"type":"function","name":"exec_command"},
		{"type":"namespace","name":"mcp__codex_apps__pets","tools":[]},
		{"type":"function","name":"view_image"}
	]`)
	kept, dropped := withoutNamespaceTools(raw)
	if dropped != 1 || len(kept) != 2 {
		t.Fatalf("kept=%s dropped=%d", rawOf(kept), dropped)
	}
	for _, tool := range kept {
		var shape struct {
			Type string `json:"type"`
		}
		if err := json.Unmarshal(tool, &shape); err != nil || shape.Type != "function" {
			t.Fatalf("unexpected tool kept: %s", tool)
		}
	}
	if kept, dropped := withoutNamespaceTools(nil); kept != nil || dropped != 0 {
		t.Fatalf("missing tools changed: %v %d", kept, dropped)
	}
}

func rawOf(items []json.RawMessage) string {
	encoded, err := json.Marshal(items)
	if err != nil {
		return err.Error()
	}
	return string(encoded)
}
