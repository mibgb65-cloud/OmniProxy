package river

import (
	"reflect"
	"testing"
)

func TestCatalogKeepsSixDistinctModelsAndExactIdentity(t *testing.T) {
	upstream := []string{
		"Qwen/Qwen3.8-27B-FP8",
		"deepseek-ai/DeepSeek-V4.1-Flash",
		"deepseek-ai/DeepSeek-V4.1-Flash-1017",
		"moonshotai/Kimi-K3",
		"nvidia/NVIDIA-Nemotron-3.5-Super-VL-120B-A12B-NVFP4",
		"zai-org/GLM-5.3-Flash",
	}
	public := []string{"Qwen3.8-27B-FP8", "deepseek-flash", "DeepSeek-V4.1-Flash-1017", "Kimi-K3", "NVIDIA-Nemotron-3.5", "GLM-5.3-Flash"}
	names := []string{"Qwen3.8-27B-FP8", "DeepSeek-V4.1-Flash", "DeepSeek-V4.1-Flash-1017", "Kimi-K3", "NVIDIA-Nemotron-3.5", "GLM-5.3-Flash"}
	windows := []int64{262144, 1048576, 1048576, 1048576, 1048576, 1048576}
	outputs := []int64{262143, 384000, 384000, 32768, 131072, 131072}
	if !reflect.DeepEqual(Models(), upstream) || !reflect.DeepEqual(DisplayModels(), names) {
		t.Fatalf("catalog/display mismatch: %v / %v", Models(), DisplayModels())
	}
	for i, id := range upstream {
		if !SupportsModel(id) || PublicModel(id) != public[i] || UpstreamModel(public[i]) != id || UpstreamModel(id) != id {
			t.Fatalf("model identity changed: %s", id)
		}
		if ModelName(id) != names[i] || ModelName(public[i]) != names[i] {
			t.Fatalf("model display changed: %s", id)
		}
		metadata := ModelMetadata(id)
		if metadata.ContextWindow == nil || *metadata.ContextWindow != windows[i] || metadata.OutputTokenLimit == nil || *metadata.OutputTokenLimit != outputs[i] {
			t.Fatalf("incorrect model limits: %s", id)
		}
	}
	for _, id := range []string{"deepseek-ai/DeepSeek-V4.1-Flash-1018", "DeepSeek-V4.1-Flash-1017 ", "deepseek-v4.1-flash-1017", "Qwen/Qwen3.8-27B", "moonshotai/kimi-k3"} {
		if SupportsModel(id) || UpstreamModel(id) != id || PublicModel(id) != id {
			t.Fatalf("unverified model alias accepted: %q", id)
		}
	}
}

func TestCatalogPublishesOnlyDocumentedEfforts(t *testing.T) {
	cases := map[string][]string{
		"Qwen/Qwen3.8-27B-FP8":                                {"none", "low", "medium", "xhigh"},
		"deepseek-ai/DeepSeek-V4.1-Flash":                     {"none", "high", "max"},
		"deepseek-ai/DeepSeek-V4.1-Flash-1017":                {"none", "high", "max"},
		"moonshotai/Kimi-K3":                                  {"none", "max"},
		"nvidia/NVIDIA-Nemotron-3.5-Super-VL-120B-A12B-NVFP4": nil,
		"zai-org/GLM-5.3-Flash":                               {"low", "high", "max"},
	}
	defaults := map[string]string{"deepseek-ai/DeepSeek-V4.1-Flash": "high", "deepseek-ai/DeepSeek-V4.1-Flash-1017": "high", "moonshotai/Kimi-K3": "max", "zai-org/GLM-5.3-Flash": "high"}
	for model, values := range cases {
		metadata := ModelMetadata(model)
		if len(metadata.ReasoningEfforts) != len(values) {
			t.Fatalf("effort enumeration changed: %s %v", model, metadata.ReasoningEfforts)
		}
		for i, value := range values {
			if metadata.ReasoningEfforts[i].Value != value {
				t.Fatalf("effort enumeration changed: %s %v", model, metadata.ReasoningEfforts)
			}
		}
		if metadata.DefaultEffort != defaults[model] {
			t.Fatalf("default effort changed: %s %q", model, metadata.DefaultEffort)
		}
	}
}
