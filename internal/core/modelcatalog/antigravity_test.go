package modelcatalog

import (
	"encoding/json"
	"testing"
)

func TestAntigravityJSONPublishesGoogleGeminiOnly(t *testing.T) {
	window, output := int64(1048576), int64(65536)
	body, err := AntigravityJSON([]Entry{
		{ID: "gemini-3.8-flash", Name: "Gemini 3.8 Flash", ProviderKind: "google", Metadata: Metadata{
			ContextWindow: &window, OutputTokenLimit: &output,
			InputModalities: []string{"text", "image"}, ReasoningEfforts: []Effort{{Value: "high"}},
		}},
		{ID: "claude-opus-4-6-thinking", Name: "Claude", ProviderKind: "google"},
		{ID: "gemini-alias", TargetModel: "gemini-3.8-pro", Name: "Alias", ProviderKind: "google"},
		{ID: "gemini-other", Name: "Other", ProviderKind: "openai"},
		{ID: "gpt-6.1-sol", Name: "GPT", ProviderKind: "openai"},
	})
	if err != nil {
		t.Fatal(err)
	}
	var value struct {
		List []AntigravityModel `json:"list"`
	}
	if json.Unmarshal(body, &value) != nil {
		t.Fatal("invalid catalog JSON")
	}
	if len(value.List) != 2 {
		t.Fatalf("expected two Gemini models, got %+v", value.List)
	}
	first := value.List[0]
	if first.ModelID != "gemini-3.8-flash" || first.MaxTokens != window || first.MaxOutputTokens != output || !first.SupportsImages || !first.SupportsThinking {
		t.Fatalf("metadata not published: %+v", first)
	}
	if value.List[1].ModelID != "gemini-alias" || value.List[1].MaxTokens != 0 {
		t.Fatalf("alias or unknown window incorrect: %+v", value.List[1])
	}
}

func TestAntigravityJSONEmptyList(t *testing.T) {
	body, err := AntigravityJSON(nil)
	if err != nil || string(body) != `{"list":[]}` {
		t.Fatalf("unexpected empty catalog: %s %v", body, err)
	}
}
