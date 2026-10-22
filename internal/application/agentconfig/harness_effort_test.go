package agentconfig

import (
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"

	catalog "omniproxy/internal/core/modelcatalog"
)

// Harness 只在模型声明 reasoning levels 时显示 Effort 菜单；
// 这里锁住「按上游已确认档位声明、未确认则写 false」的行为。
func TestHarnessReasoningEffortsFollowConfirmedMetadata(t *testing.T) {
	deepseek := catalog.Entry{ID: "deepseek-x", ProviderKind: "river", Metadata: catalog.Metadata{
		ReasoningEfforts: []catalog.Effort{{Value: "none"}, {Value: "high"}, {Value: "max"}}, DefaultEffort: "high"}}
	efforts := harnessReasoningEfforts(deepseek)
	if efforts == nil {
		t.Fatal("confirmed model declared no reasoning levels")
	}
	// off 必须写 none 而不是留空：留空等于不发参数，上游会继续思考。
	if efforts["off"] != "none" {
		t.Fatal("off must send none explicitly", efforts)
	}
	if efforts["high"] != "high" || efforts["max"] != "max" {
		t.Fatal("confirmed levels missing", efforts)
	}
	// 未确认的档位不能凭模型支持 max 就推断出 medium／low。
	if _, exists := efforts["medium"]; exists {
		t.Fatal("unconfirmed level advertised", efforts)
	}
	if _, exists := efforts["low"]; exists {
		t.Fatal("unconfirmed level advertised", efforts)
	}
	if harnessThinkingFormat(deepseek, efforts) != "deepseek" {
		t.Fatal("thinking format missing for a toggleable River model")
	}

	// GLM 不能关闭思考：没有 off，不得伪造成 deepseek 格式。
	glm := catalog.Entry{ID: "glm-x", ProviderKind: "river", Metadata: catalog.Metadata{
		ReasoningEfforts: []catalog.Effort{{Value: "low"}, {Value: "high"}, {Value: "max"}}, DefaultEffort: "high"}}
	glmEfforts := harnessReasoningEfforts(glm)
	if _, exists := glmEfforts["off"]; exists {
		t.Fatal("off invented for a model without a close option")
	}
	if harnessThinkingFormat(glm, glmEfforts) != "" {
		t.Fatal("thinking format granted without a close option")
	}

	// 未确认档位、非 River、单档位：一律不声明，菜单里不出现 Effort。
	for _, entry := range []catalog.Entry{
		{ID: "unknown", ProviderKind: "river"},
		{ID: "other", ProviderKind: "other", Metadata: deepseek.Metadata},
		{ID: "single", ProviderKind: "river", Metadata: catalog.Metadata{ReasoningEfforts: []catalog.Effort{{Value: "high"}}}},
	} {
		if got := harnessReasoningEfforts(entry); got != nil {
			t.Fatal("reasoning levels invented", entry.ID, got)
		}
		if got := harnessThinkingFormat(entry, harnessReasoningEfforts(entry)); got != "" {
			t.Fatal("thinking format granted without declared levels", entry.ID, got)
		}
	}
}

// 写进 Harness 配置的形状必须是 pi-ai 认的 reasoningEfforts 映射，而不是布尔值。
func TestHarnessConfigDeclaresReasoningEffortsMapping(t *testing.T) {
	entry := catalog.Entry{ID: "deepseek-x", Name: "DeepSeek X", ProviderKind: "river", Metadata: catalog.Metadata{
		ReasoningEfforts: []catalog.Effort{{Value: "none"}, {Value: "high"}, {Value: "max"}}, DefaultEffort: "high"}}
	plan := Plan{Models: []catalog.Entry{entry}, DefaultModel: entry.ID}
	input, err := json.Marshal(harnessBundle{Config: backup{[]byte("[]"), true}})
	if err != nil {
		t.Fatal(err)
	}
	body, _, err := buildHarness("harness-web", input, []byte("test-key"), 3000, plan)
	if err != nil {
		t.Fatal(err)
	}
	var bundle struct {
		Config struct {
			Body string `json:"body"`
		} `json:"config"`
	}
	if json.Unmarshal(body, &bundle) != nil {
		t.Fatal("bundle unreadable")
	}
	decoded, err := base64.StdEncoding.DecodeString(bundle.Config.Body)
	if err != nil {
		t.Fatal(err)
	}
	config := string(decoded)
	for _, want := range []string{"reasoningEfforts", "off\": none", "thinkingFormat: deepseek"} {
		if !strings.Contains(config, want) {
			t.Fatal("missing in harness config", want, config)
		}
	}
}
