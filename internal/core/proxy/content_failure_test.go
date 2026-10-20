package proxy

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestImageFailurePreservesCodeAndExplainsSafeRecovery(t *testing.T) {
	recorder := httptest.NewRecorder()
	writeFailure(recorder, UnsupportedImage())
	var result struct {
		Error struct{ Code, Message string }
	}
	if json.Unmarshal(recorder.Body.Bytes(), &result) != nil || recorder.Code != 400 || result.Error.Code != "unsupported_content" {
		t.Fatal("stable HTTP error contract changed")
	}
	for _, phrase := range []string{"earlier messages", "tool results", "vision-capable", "text-only", "No image content was removed"} {
		if !strings.Contains(result.Error.Message, phrase) {
			t.Fatal("image failure lacks actionable diagnosis", phrase)
		}
	}
	if message := (&Error{Code: "unsupported_content", StatusCode: 400}).clientMessage(); message != "unsupported_content" {
		t.Fatal("unrelated unsupported content incorrectly reported as images")
	}
}
