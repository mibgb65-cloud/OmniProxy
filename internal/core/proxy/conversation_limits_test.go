package proxy

import (
	"encoding/json"
	"strings"
	"testing"
)

func conversationBody(count int) []byte {
	messages := strings.Repeat(`{"role":"user","content":""},`, count-1) + `{"role":"user","content":""}`
	return []byte(`{"model":"test-model","max_tokens":16,"messages":[` + messages + `]}`)
}

func TestConversationCountUsesBoundedProtocolScale(t *testing.T) {
	for _, count := range []int{512, 513, MaxConversationMessages, MaxConversationMessages + 1} {
		body := conversationBody(count)
		_, messagesErr := ParseProtocolRequest(body, "/v1/messages")
		_, chatErr := ParseConvertedChatRequest(body)
		want := count <= MaxConversationMessages
		if (messagesErr == nil) != want || (chatErr == nil) != want {
			t.Fatalf("count=%d messages=%v converted=%v", count, messagesErr, chatErr)
		}
	}
	for _, raw := range []string{"null", "[]", "{}", `"not-an-array"`} {
		if _, err := ParseProtocolRequest([]byte(`{"model":"test-model","max_tokens":16,"messages":`+raw+`}`), "/v1/messages"); err == nil {
			t.Fatal("empty or malformed Messages array accepted")
		}
	}
}

func TestConvertedChatKeepsMessagesCapacityWithoutRelaxingExternalChat(t *testing.T) {
	body := []byte(`{"model":"test-model","messages":[{"role":"user","content":"` + strings.Repeat("x", MaxRequestBytes) + `"}]}`)
	request, err := ParseConvertedChatRequest(body)
	if err != nil || string(request.Body) != string(body) {
		t.Fatal("trusted converted text changed or retained the small Chat budget", err)
	}
	if _, err := ParseRequest(body); err == nil {
		t.Fatal("ordinary Chat acquired the larger Messages budget")
	}
	single := []byte(`{"model":"test-model","messages":[{"role":"user","content":"` + strings.Repeat("x", 1<<20+1) + `"}]}`)
	if _, err := ParseRequest(single); err == nil {
		t.Fatal("ordinary Chat lost its per-message text bound")
	}
	forged := append([]byte(`{"convertedFromMessages":true,`), body[1:]...)
	if _, err := ParseProtocolRequest(forged, "/v1/chat/completions"); err == nil {
		t.Fatal("client JSON forged the internal conversion marker")
	}
	if _, err := ParseConvertedChatRequest([]byte(`{"model":"test-model","messages":[{"role":"user","content":"` + strings.Repeat("x", MaxMessagesRequestBytes) + `"}]}`)); err == nil {
		t.Fatal("trusted conversion lost its byte bound")
	}
	for _, body := range []string{
		`{"model":"test-model","messages":[{"role":"user","content":"one","content":"two"}]}`,
		`{"model":"test-model","messages":[{"role":"tool","content":"missing id"}]}`,
		`{"model":"test-model","messages":[{"role":"unknown","content":"text"}]}`,
	} {
		if _, err := ParseConvertedChatRequest([]byte(body)); err == nil {
			t.Fatal("converted request bypassed structural validation")
		}
	}
	plain, err := ParseProtocolRequest([]byte(`{"model":"test-model","convertedFromMessages":true,"messages":[{"role":"user","content":"small"}]}`), "/v1/chat/completions")
	if err != nil || plain.ConvertedFromMessages {
		t.Fatal("client fields assigned a trusted server flag", err)
	}
}

func TestConvertedChatPreservesTextArrayAndRejectsDeepJSON(t *testing.T) {
	body := []byte(`{"model":"test-model","messages":[{"role":"user","content":[{"type":"text","text":"` + strings.Repeat("x", 1<<20+1) + `"}]}]}`)
	request, err := ParseConvertedChatRequest(body)
	if err != nil {
		t.Fatal(err)
	}
	var parsed struct{ Messages []struct{ Content string } }
	if json.Unmarshal(request.Body, &parsed) != nil || len(parsed.Messages[0].Content) != 1<<20+1 {
		t.Fatal("text normalization truncated long content")
	}
	deep := `{"model":"test-model","messages":[{"role":"user","content":"ok"}],"extra":` + strings.Repeat("[", 40) + "0" + strings.Repeat("]", 40) + "}"
	if _, err := ParseConvertedChatRequest([]byte(deep)); err == nil {
		t.Fatal("trusted conversion bypassed JSON depth validation")
	}
}
