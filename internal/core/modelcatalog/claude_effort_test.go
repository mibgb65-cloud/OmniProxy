package modelcatalog

import (
	"strings"
	"testing"
)

// 目标：Claude 菜单每个模型只有一行，推理强度交给客户端原生 effort 控件，
// 网关不再用派生模型 ID 表达档位。
func TestClaudeDirectoryKeepsOneEntryPerModel(t *testing.T) {
	window := int64(1048576)
	source := Entry{ID: "DeepSeek-V4.1-Flash-1017", Name: "DeepSeek-V4.1-Flash-1017", ProviderKind: "river",
		Metadata: Metadata{ContextWindow: &window, ReasoningEfforts: []Effort{{Value: "none"}, {Value: "high"}, {Value: "max"}}, DefaultEffort: "high", IsDefault: true}}
	entries, err := ClaudeEntries([]Entry{source})
	if err != nil {
		t.Fatal(err)
	}
	// 普通 + [1m]，不再按档位展开。
	if len(entries) != 2 {
		t.Fatal("claude directory must not fan out per effort", len(entries))
	}
	for _, entry := range entries {
		if strings.HasPrefix(entry.ID, ClaudeEffortPrefix) {
			t.Fatal("derived effort entry still published", entry.ID)
		}
		// 目录保留能力枚举，供网关校验客户端原生 effort 选择。
		if len(entry.Metadata.ReasoningEfforts) == 0 {
			t.Fatal("capability enumeration was dropped together with the fan-out", entry.ID)
		}
	}

	// 客户端原生 effort 作用于网关模型：模型 ID 不变，只解析基础模型与 [1m]。
	for _, model := range []string{source.ID, source.ID + ClaudeLongContextSuffix, DesktopAlias(source.ID)} {
		got, found := ResolveClaudeModel(entries, model)
		if !found || !strings.HasPrefix(got, source.ID) {
			t.Fatal("native effort model no longer routes", model, got, found)
		}
	}
}

// 归档旧档位选择：历史 omniproxy-effort-* ID 必须继续可路由回基础模型，不能变成 unsupported_model。
func TestArchivedEffortChoiceStillResolvesToBaseModel(t *testing.T) {
	window := int64(1048576)
	source := Entry{ID: "DeepSeek-V4.1-Flash-1017", Name: "DeepSeek-V4.1-Flash-1017", ProviderKind: "river",
		Metadata: Metadata{ContextWindow: &window, ReasoningEfforts: []Effort{{Value: "none"}, {Value: "high"}, {Value: "max"}}, DefaultEffort: "high"}}
	entries, err := ClaudeEntries([]Entry{source})
	if err != nil {
		t.Fatal(err)
	}
	for _, effort := range []string{"none", "high", "max"} {
		id := claudeEffortID(source.ID, effort)
		for _, variant := range []string{id, id + ClaudeLongContextSuffix, DesktopAlias(id)} {
			model, gotEffort, found := ResolveClaudeEffort(entries, variant)
			if !found || model != source.ID || gotEffort != effort {
				t.Fatal("archived effort choice lost", variant, model, gotEffort, found)
			}
		}
	}
	// 未确认的能力仍然不解析，不能凭摘要长度猜测档位。
	if _, _, ok := ResolveClaudeEffort(entries, claudeEffortID("other-model", "high")); ok {
		t.Fatal("unknown model effort choice resolved")
	}
	if _, _, ok := ResolveClaudeEffort(entries, claudeEffortID(source.ID, "medium")); ok {
		t.Fatal("unverified effort resolved")
	}
}

// 不伪造能力：没有已确认档位的模型不产生波次条目，也不继承他人档位。
func TestEffortCapabilitiesComeOnlyFromMetadata(t *testing.T) {
	unknown := Entry{ID: "unknown-route", ProviderKind: "river"}
	entries, err := ClaudeEntries([]Entry{unknown})
	if err != nil || len(entries) != 1 {
		t.Fatal("unknown model gained entries", len(entries), err)
	}
	if id := claudeEffortID(unknown.ID, "high"); id != claudeEffortID(unknown.ID, "high") {
		t.Fatal("effort id is not stable")
	}
	if _, _, ok := ResolveClaudeEffort(entries, claudeEffortID(unknown.ID, "high")); ok {
		t.Fatal("effort invented for a model without metadata")
	}
}
