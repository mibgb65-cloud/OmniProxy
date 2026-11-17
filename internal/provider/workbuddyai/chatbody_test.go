package workbuddyai

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"

	"omniproxy/internal/core/proxy"
)

// 上游要求首条消息是 system：只有 user 消息时必须补上，否则真实账号返回 11128。
func TestPrepareChatBodyAddsLeadingSystem(t *testing.T) {
	parsed := decodeMessages(t, prepareChatBody([]byte(`{"model":"glm-5.3-flash","messages":[{"role":"user","content":"hi"}]}`)))
	if len(parsed) != 2 || parsed[0]["role"] != "system" || parsed[0]["content"] != defaultSystemPrompt {
		t.Fatalf("messages = %+v", parsed)
	}
	if parsed[1]["role"] != "user" || parsed[1]["content"] != "hi" {
		t.Fatalf("user message changed: %+v", parsed[1])
	}
}

// developer 角色与 max_completion_tokens 是 Chat 的新形态，上游只认 system 与 max_tokens。
func TestPrepareChatBodyNormalizesRoleAndTokenField(t *testing.T) {
	body := []byte(`{"model":"m","max_completion_tokens":64,"messages":[{"role":"developer","content":"dev"},{"role":"user","content":"hi"}]}`)
	prepared := prepareChatBody(body)
	var top map[string]json.RawMessage
	if err := json.Unmarshal(prepared, &top); err != nil {
		t.Fatal(err)
	}
	if _, found := top["max_completion_tokens"]; found {
		t.Fatal("max_completion_tokens kept")
	}
	if string(top["max_tokens"]) != "64" {
		t.Fatalf("max_tokens = %s", top["max_tokens"])
	}
	parsed := decodeMessages(t, prepared)
	if parsed[0]["role"] != "system" || parsed[0]["content"] != "dev" {
		t.Fatalf("developer not normalized: %+v", parsed[0])
	}
}

// 已经合规的正文必须原样返回，避免无谓改写未知字段。
func TestPrepareChatBodyKeepsCompliantBody(t *testing.T) {
	body := []byte(`{"model":"m","messages":[{"role":"system","content":"s"},{"role":"user","content":"hi"}],"temperature":0.3}`)
	if prepared := prepareChatBody(body); string(prepared) != string(body) {
		t.Fatalf("compliant body rewritten: %s", prepared)
	}
	if prepared := prepareChatBody([]byte(`not json`)); string(prepared) != "not json" {
		t.Fatalf("invalid body rewritten: %s", prepared)
	}
}

// 11128 重试形态：系统提示词读作首条用户消息（<instructions> 包裹），开头换默认 system。
func TestSystemAsUserBodyMovesSystemIntoUser(t *testing.T) {
	body := []byte(`{"model":"m","messages":[{"role":"system","content":"You are Codex, a coding agent."},{"role":"user","content":"fix the bug"}]}`)
	moved := systemAsUserBody(body)
	parsed := decodeMessages(t, moved)
	if len(parsed) != 2 {
		t.Fatalf("messages = %+v", parsed)
	}
	if parsed[0]["role"] != "system" || parsed[0]["content"] != defaultSystemPrompt {
		t.Fatalf("leading system = %+v", parsed[0])
	}
	if parsed[1]["role"] != "user" {
		t.Fatalf("second message role = %v", parsed[1]["role"])
	}
	want := "<instructions>\nYou are Codex, a coding agent.\n</instructions>\n\nfix the bug"
	if parsed[1]["content"] != want {
		t.Fatalf("user content = %q", parsed[1]["content"])
	}
}

func TestSystemAsUserBodyReturnsNilWithoutSystem(t *testing.T) {
	if moved := systemAsUserBody([]byte(`{"messages":[{"role":"user","content":"hi"}]}`)); moved != nil {
		t.Fatalf("moved = %s", moved)
	}
}

// 真实账号上被 11128 拒绝时，适配器要用「系统提示词下移」形态重试一次并以重试结果回包。
func TestForwardRetriesOnceWhenSystemPromptIsRefused(t *testing.T) {
	var bodies []string
	var requests int32
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&requests, 1)
		raw, _ := io.ReadAll(r.Body)
		bodies = append(bodies, string(gunzipBytes(t, raw)))
		if len(bodies) == 1 {
			w.WriteHeader(http.StatusBadRequest)
			io.WriteString(w, `{"code":11128,"msg":"first message is not system prompt"}`)
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		io.WriteString(w, "data: {\"choices\":[{\"delta\":{\"content\":\"OK\"}}]}\n\ndata: [DONE]\n\n")
	})
	body := []byte(`{"model":"m","stream":true,"messages":[{"role":"system","content":"You are Codex, a coding agent."},{"role":"user","content":"hi"}]}`)
	request := proxy.Request{Protocol: ChatProtocol, Stream: true, Model: "m", Body: body}
	var status int
	if err := client.Forward(context.Background(), testCredentials(), request, func(response proxy.Response) error {
		status = response.StatusCode
		_, _ = io.Copy(io.Discard, response.Body)
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if status != http.StatusOK || atomic.LoadInt32(&requests) != 2 {
		t.Fatalf("status = %d requests = %d", status, requests)
	}
	second := decodeMessages(t, []byte(bodies[1]))
	if second[0]["role"] != "system" || second[0]["content"] != defaultSystemPrompt {
		t.Fatalf("retry leading system = %+v", second[0])
	}
	if !strings.Contains(second[1]["content"].(string), "<instructions>\nYou are Codex") {
		t.Fatalf("retry user content = %q", second[1]["content"])
	}
}

// 重试仍被 11128 拒绝时，按固定码返回，不无限重试。
func TestForwardReportsChannelRejectedAfterRetry(t *testing.T) {
	var requests int32
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&requests, 1)
		w.WriteHeader(http.StatusBadRequest)
		io.WriteString(w, `{"code":11128,"msg":"first message is not system prompt"}`)
	})
	body := []byte(`{"model":"m","stream":true,"messages":[{"role":"system","content":"You are Claude Code."},{"role":"user","content":"hi"}]}`)
	request := proxy.Request{Protocol: ChatProtocol, Stream: true, Model: "m", Body: body}
	err := client.Forward(context.Background(), testCredentials(), request, func(proxy.Response) error { return nil })
	if ErrorCode(err) != CodeChannelRejected {
		t.Fatalf("error = %v", err)
	}
	if atomic.LoadInt32(&requests) != 2 {
		t.Fatalf("requests = %d, want 2 (one retry)", requests)
	}
}

// decodeMessages 解出正文里的 messages 为可读的 role/content 对。
func decodeMessages(t *testing.T, body []byte) []map[string]any {
	t.Helper()
	var payload struct {
		Messages []map[string]any `json:"messages"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		t.Fatalf("body = %s: %v", body, err)
	}
	return payload.Messages
}
