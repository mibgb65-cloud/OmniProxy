package gateway

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"

	"omniproxy/internal/core/agentprotocol"
)

func riverLongMessages(t *testing.T, reminder bool, resultBytes int) []byte {
	t.Helper()
	messages := []any{agentprotocol.Object{"role": "user", "content": "synthetic request"}}
	pairs := 170
	if resultBytes > 0 {
		pairs = 1
	}
	for i := range pairs {
		id := fmt.Sprintf("call-%d", i)
		result := fmt.Sprintf("result-%d", i)
		if resultBytes > 0 {
			result = strings.Repeat("x", resultBytes)
		}
		messages = append(messages,
			agentprotocol.Object{"role": "assistant", "content": []any{agentprotocol.Object{"type": "tool_use", "id": id, "name": "run", "input": agentprotocol.Object{}}}},
			agentprotocol.Object{"role": "user", "content": []any{
				agentprotocol.Object{"type": "tool_result", "tool_use_id": id, "content": result},
				agentprotocol.Object{"type": "text", "text": "synthetic reminder"},
			}})
	}
	if reminder {
		messages = append(messages, agentprotocol.Object{"role": "system", "content": "extra reminder"})
	}
	body, err := json.Marshal(agentprotocol.Object{
		"model": "river-test", "max_tokens": 16, "system": "synthetic instructions", "messages": messages,
		"tools": []any{agentprotocol.Object{"name": "run", "input_schema": agentprotocol.Object{"type": "object"}}},
	})
	if err != nil {
		t.Fatal(err)
	}
	return body
}

func TestRiverMessagesConversionPreservesExpandedHistory(t *testing.T) {
	for _, test := range []struct {
		name        string
		reminder    bool
		resultBytes int
		wantCount   int
	}{
		{"512_chat_messages", false, 0, 512},
		{"513_chat_messages", true, 0, 513},
		{"single_tool_result_over_1MiB", false, 1<<20 + 1, 5},
		{"large_tool_result", false, 2 << 20, 5},
	} {
		t.Run(test.name, func(t *testing.T) {
			var calls atomic.Int32
			s, _, _ := riverPoolFixture(t, roundTripFunc(func(r *http.Request) (*http.Response, error) {
				calls.Add(1)
				var body struct {
					Messages []struct {
						Role, Content string
						ToolCallID    string            `json:"tool_call_id"`
						ToolCalls     []json.RawMessage `json:"tool_calls"`
					}
				}
				if json.NewDecoder(r.Body).Decode(&body) != nil || len(body.Messages) != test.wantCount {
					t.Error("converted history count changed")
				}
				results, toolCalls := 0, 0
				known := map[string]bool{}
				for _, message := range body.Messages {
					toolCalls += len(message.ToolCalls)
					for _, raw := range message.ToolCalls {
						var call struct{ ID string }
						if json.Unmarshal(raw, &call) != nil || known[call.ID] {
							t.Error("invalid or duplicated tool call")
						}
						known[call.ID] = true
					}
					if message.Role == "tool" {
						results++
						if !known[message.ToolCallID] {
							t.Error("tool result lost its original call")
						}
						if test.resultBytes > 0 && len(message.Content) != test.resultBytes {
							t.Error("tool result was truncated")
						}
					}
				}
				if results != toolCalls {
					t.Error("tool calls and results lost association")
				}
				return upstreamReply(200, atriaJSON("chat"), false), nil
			}))
			response, err := http.Post(s.Info().Runtime.Address+"/v1/messages", "application/json",
				strings.NewReader(string(riverLongMessages(t, test.reminder, test.resultBytes))))
			if err != nil {
				t.Fatal(err)
			}
			defer response.Body.Close()
			if _, err := io.Copy(io.Discard, response.Body); err != nil {
				t.Fatal(err)
			}
			if response.StatusCode != 200 || calls.Load() != 1 {
				t.Fatalf("valid Messages history rejected before upstream: status=%d calls=%d", response.StatusCode, calls.Load())
			}
		})
	}
}
