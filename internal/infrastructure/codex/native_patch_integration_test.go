package codex

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	catalog "omniproxy/internal/core/modelcatalog"
)

// 仅访问 loopback 假模型，指令与补丁完全固定，写入仅限 t.TempDir；不继承真实授权或用户规则。
// 测试子进程用独立权限配置避免 Windows 未配置沙箱回退为只读，不更改用户权限。
func TestInstalledCodexNativePatchEvents(t *testing.T) {
	executable := os.Getenv("OMNIPROXY_CODEX_TEST_EXECUTABLE")
	if os.Getenv("OMNIPROXY_CODEX_PATCH_QA") != "1" || executable == "" {
		t.Skip("显式启用本机离线补丁契约验收")
	}
	if !filepath.IsAbs(executable) {
		t.Fatal("executable must be absolute")
	}
	for _, native := range []bool{false, true} {
		t.Run(fmt.Sprintf("native_%t", native), func(t *testing.T) {
			home, work := t.TempDir(), t.TempDir()
			for _, name := range []string{"existing.txt", "deleted.txt"} {
				if os.WriteFile(filepath.Join(work, name), []byte("before\n"), 0600) != nil {
					t.Fatal("fixture unavailable")
				}
			}
			body, err := catalog.CodexJSON([]catalog.Entry{{ID: "gpt-qa", Name: "QA", ProviderKind: "openai"}})
			if err != nil {
				t.Fatal(err)
			}
			if !native {
				var payload map[string]any
				json.Unmarshal(body, &payload)
				for _, model := range payload["models"].([]any) {
					model.(map[string]any)["apply_patch_tool_type"] = nil
				}
				body, _ = json.Marshal(payload)
			}
			catalogPath := filepath.Join(home, "models.json")
			if os.WriteFile(catalogPath, body, 0600) != nil {
				t.Fatal("catalog unavailable")
			}
			var calls atomic.Int32
			var patchOffered atomic.Bool
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "POST" || r.URL.Path != "/v1/responses" {
					http.Error(w, "qa route only", 404)
					return
				}
				raw, err := io.ReadAll(io.LimitReader(r.Body, 4<<20))
				if err != nil {
					t.Error("request read failed")
					return
				}
				var payload map[string]any
				if json.Unmarshal(raw, &payload) != nil {
					t.Error("invalid QA request")
					return
				}
				offered := hasNativePatchTool(payload["tools"])
				patchOffered.Store(offered)
				count := calls.Add(1)
				item := map[string]any{"type": "message", "id": "msg_qa", "role": "assistant", "status": "completed", "content": []any{map[string]any{"type": "output_text", "text": "QA done"}}}
				if native && offered && count == 1 {
					item = map[string]any{"type": "custom_tool_call", "id": "tool_qa", "call_id": "call_qa", "name": "apply_patch", "namespace": "functions", "input": "*** Begin Patch\n*** Add File: added.txt\n+added\n*** Update File: existing.txt\n@@\n-before\n+after\n*** Delete File: deleted.txt\n*** End Patch"}
				}
				w.Header().Set("Content-Type", "text/event-stream")
				events := []map[string]any{{"type": "response.created", "response": map[string]any{"id": "resp_qa", "status": "in_progress", "output": []any{}}}, {"type": "response.output_item.done", "output_index": 0, "item": item}, {"type": "response.completed", "response": map[string]any{"id": "resp_qa", "status": "completed", "output": []any{item}, "usage": map[string]int{"input_tokens": 10, "output_tokens": 1, "total_tokens": 11}}}}
				for _, event := range events {
					raw, _ := json.Marshal(event)
					fmt.Fprintf(w, "data: %s\n\n", raw)
				}
			}))
			defer server.Close()
			ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
			defer cancel()
			provider := fmt.Sprintf(`model_providers.mock={name="Offline QA",base_url="%s/v1",wire_api="responses",requires_openai_auth=false,env_key="OMNIPROXY_TEST_KEY",supports_websockets=false,request_max_retries=0,stream_max_retries=0}`, server.URL)
			command := exec.CommandContext(ctx, executable, "exec", "--ignore-user-config", "--ignore-rules", "--ephemeral", "--skip-git-repo-check", "--sandbox", "danger-full-access", "--json", "-C", work, "-c", `model_provider="mock"`, "-m", "gpt-qa", "-c", "model_catalog_json="+strconv.Quote(filepath.ToSlash(catalogPath)), "-c", provider, "Offline QA only. Use the provided tool if requested.")
			command.Dir = work
			hideWindow(command)
			for _, name := range []string{"PATH", "SystemRoot", "WINDIR", "COMSPEC", "PATHEXT"} {
				if value := os.Getenv(name); value != "" {
					command.Env = append(command.Env, name+"="+value)
				}
			}
			command.Env = append(command.Env, "CODEX_HOME="+home, "USERPROFILE="+home, "HOME="+home, "APPDATA="+home, "LOCALAPPDATA="+home, "TEMP="+home, "TMP="+home, "OMNIPROXY_TEST_KEY=isolated-placeholder", "NO_PROXY=127.0.0.1,localhost", "HTTP_PROXY="+server.URL, "HTTPS_PROXY="+server.URL, "ALL_PROXY="+server.URL)
			output, runErr := command.CombinedOutput()
			changes := 0
			turnDone := false
			scanner := bufio.NewScanner(strings.NewReader(string(output)))
			scanner.Buffer(make([]byte, 4096), 2<<20)
			for scanner.Scan() {
				var event struct {
					Type string `json:"type"`
					Item struct {
						Type    string            `json:"type"`
						Changes []json.RawMessage `json:"changes"`
					} `json:"item"`
				}
				if json.Unmarshal(scanner.Bytes(), &event) == nil {
					if event.Type == "turn.completed" {
						turnDone = true
					}
					if event.Type == "item.completed" && event.Item.Type == "file_change" {
						changes += len(event.Item.Changes)
					}
				}
			}
			var eventTypes []string
			for _, line := range strings.Split(string(output), "\n") {
				var event map[string]any
				if json.Unmarshal([]byte(line), &event) == nil {
					if item, ok := event["item"].(map[string]any); ok {
						if kind, ok := item["type"].(string); ok {
							eventTypes = append(eventTypes, kind)
						}
					}
				}
			}
			t.Logf("offered_native_patch=%t file_changes=%d turn_completed=%t requests=%d item_types=%v", patchOffered.Load(), changes, turnDone, calls.Load(), eventTypes)
			if runErr != nil || !turnDone {
				t.Fatal("isolated native core did not complete QA turn")
			}
			if patchOffered.Load() != native {
				t.Fatal("catalog patch capability does not reach native tool registry")
			}
			if native {
				added, e1 := os.ReadFile(filepath.Join(work, "added.txt"))
				updated, e2 := os.ReadFile(filepath.Join(work, "existing.txt"))
				_, e3 := os.Stat(filepath.Join(work, "deleted.txt"))
				if changes != 3 || e1 != nil || e2 != nil || !os.IsNotExist(e3) || string(added) != "added\n" || string(updated) != "after\n" {
					t.Fatal("native patch did not emit and apply all three file operations")
				}
			}
		})
	}
}

func hasNativePatchTool(value any) bool {
	switch node := value.(type) {
	case map[string]any:
		if node["name"] == "apply_patch" {
			return true
		}
		for _, child := range node {
			if hasNativePatchTool(child) {
				return true
			}
		}
	case []any:
		for _, child := range node {
			if hasNativePatchTool(child) {
				return true
			}
		}
	}
	return false
}
