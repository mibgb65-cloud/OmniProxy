package antigravity

import (
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// backfillBody 构造一次「assistant 调用 read_file + tool 结果」的 Chat 请求。
func backfillBody(t *testing.T, arguments, result string, withCall bool) string {
	t.Helper()
	messages := []any{map[string]any{"role": "user", "content": "read the png and describe it"}}
	if withCall {
		messages = append(messages, map[string]any{
			"role": "assistant", "content": nil,
			"tool_calls": []any{map[string]any{
				"id": "call_1", "type": "function",
				"function": map[string]any{"name": "read_file", "arguments": arguments},
			}},
		})
	}
	messages = append(messages, map[string]any{"role": "tool", "tool_call_id": "call_1", "content": result})
	messages = append(messages, map[string]any{"role": "user", "content": "what colour is it?"})
	body, err := json.Marshal(map[string]any{"messages": messages})
	if err != nil {
		t.Fatalf("marshal body: %v", err)
	}
	return string(body)
}

// antigravityArguments 复刻桌面端的双引号参数：{"Path":"\"C:\\\\…png\""}。
func antigravityArguments(t *testing.T, path string) string {
	t.Helper()
	quoted := `"` + path + `"`
	raw, err := json.Marshal(map[string]string{"Path": quoted})
	if err != nil {
		t.Fatalf("marshal arguments: %v", err)
	}
	return string(raw)
}

func writeTempPNG(t *testing.T) (string, []byte) {
	t.Helper()
	data, err := base64.StdEncoding.DecodeString(tinyPNG)
	if err != nil {
		t.Fatalf("decode tiny png: %v", err)
	}
	path := filepath.Join(t.TempDir(), "blind.png")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("write png: %v", err)
	}
	return path, data
}

// 客户端只留占位文本时，网关要按同一轮调用的路径把图片补回 inlineData。
func TestBackfillRestoresImageDroppedByClient(t *testing.T) {
	path, data := writeTempPNG(t)
	arguments := antigravityArguments(t, path)
	body := backfillBody(t, arguments, "Created At: 2026-11-15T12:22:43+08:00\nShowing binary file of type image/png", true)

	native := envelopeOf(t, chatRequest("gemini-4-hydrogen-xhigh", body), false)
	parts := toolResultParts(t, native)
	if len(parts) != 2 {
		t.Fatalf("parts = %v, want functionResponse + inlineData", parts)
	}
	response, _ := parts[0].(map[string]any)["functionResponse"].(map[string]any)
	if response["name"] != "read_file" {
		t.Fatalf("functionResponse = %v", response)
	}
	inline, ok := parts[1].(map[string]any)["inlineData"].(map[string]any)
	if !ok {
		t.Fatalf("image part = %v", parts[1])
	}
	if inline["mimeType"] != "image/png" {
		t.Fatalf("mimeType = %v", inline["mimeType"])
	}
	if inline["data"] != base64.StdEncoding.EncodeToString(data) {
		t.Fatal("inline data is not the file on disk")
	}
}

// 单层转义的参数同样要能取到路径。
func TestBackfillAcceptsPlainPathArgument(t *testing.T) {
	path, _ := writeTempPNG(t)
	raw, err := json.Marshal(map[string]string{"Path": path})
	if err != nil {
		t.Fatalf("marshal arguments: %v", err)
	}
	body := backfillBody(t, string(raw), "Showing binary file of type image/png", true)
	native := envelopeOf(t, chatRequest("gemini-4-hydrogen-xhigh", body), false)
	if parts := toolResultParts(t, native); len(parts) != 2 {
		t.Fatalf("parts = %v, want image backfilled", parts)
	}
}

// 只对声明了图片输入的 Gemini 族回填；Claude／GPT-OSS 通道保持纯文本。
func TestBackfillStaysOffForNonGeminiModels(t *testing.T) {
	path, _ := writeTempPNG(t)
	body := backfillBody(t, antigravityArguments(t, path), "Showing binary file of type image/png", true)
	for _, model := range []string{"claude-opus-4-6-thinking", "gpt-oss-120b-medium"} {
		native := envelopeOf(t, chatRequest(model, body), false)
		if parts := toolResultParts(t, native); len(parts) != 1 {
			t.Fatalf("%s got an image backfilled: %v", model, parts)
		}
	}
}

// 不是图片的文件绝不回填，也不会因为读文件让请求失败。
func TestBackfillSkipsNonImageFiles(t *testing.T) {
	path := filepath.Join(t.TempDir(), "notes.txt")
	if err := os.WriteFile(path, []byte("not an image"), 0o600); err != nil {
		t.Fatalf("write file: %v", err)
	}
	body := backfillBody(t, antigravityArguments(t, path), "Showing binary file of type image/png", true)
	native := envelopeOf(t, chatRequest("gemini-4-hydrogen-xhigh", body), false)
	if parts := toolResultParts(t, native); len(parts) != 1 {
		t.Fatalf("parts = %v, want text-only functionResponse", parts)
	}
}

// 文件不存在、没有调用参数、或结果不是二进制占位文本时都保持原样。
func TestBackfillLeavesOtherToolResultsAlone(t *testing.T) {
	path, _ := writeTempPNG(t)
	cases := []struct {
		name       string
		arguments  string
		result     string
		withCall   bool
		wantImages int
	}{
		{name: "missing file", arguments: antigravityArguments(t, filepath.Join(t.TempDir(), "gone.png")), result: "Showing binary file of type image/png", withCall: true},
		{name: "relative path", arguments: antigravityArguments(t, "blind.png"), result: "Showing binary file of type image/png", withCall: true},
		{name: "no path argument", arguments: `{"StartLine":1}`, result: "Showing binary file of type image/png", withCall: true},
		{name: "text result", arguments: antigravityArguments(t, path), result: "file contents", withCall: true},
		{name: "other binary", arguments: antigravityArguments(t, path), result: "Showing binary file of type application/pdf", withCall: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			body := backfillBody(t, tc.arguments, tc.result, tc.withCall)
			native := envelopeOf(t, chatRequest("gemini-4-hydrogen-xhigh", body), false)
			parts := toolResultParts(t, native)
			if len(parts) != 1 {
				t.Fatalf("parts = %v, want text-only", parts)
			}
			if response, _ := parts[0].(map[string]any)["functionResponse"].(map[string]any); response["id"] != "call_1" {
				t.Fatalf("functionResponse = %v", response)
			}
		})
	}
}
