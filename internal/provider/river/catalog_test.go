package river

import (
	"reflect"
	"testing"
)

func TestCatalogKeepsFourDistinctModelsAndExactIdentity(t *testing.T) {
	upstream := []string{
		"deepseek-ai/DeepSeek-V4.1-Flash",
		"deepseek-ai/DeepSeek-V4.1-Flash-1017",
		"nvidia/NVIDIA-Nemotron-3.5-Super-VL-120B-A12B-NVFP4",
		"zai-org/GLM-5.3-Flash",
	}
	public := []string{"deepseek-flash", "DeepSeek-V4.1-Flash-1017", "NVIDIA-Nemotron-3.5", "GLM-5.3-Flash"}
	names := []string{"DeepSeek-V4.1-Flash", "DeepSeek-V4.1-Flash-1017", "NVIDIA-Nemotron-3.5", "GLM-5.3-Flash"}
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
		output := int64(131072)
		if i < 2 {
			output = 384000
		}
		if metadata.ContextWindow == nil || *metadata.ContextWindow != 1048576 || metadata.OutputTokenLimit == nil || *metadata.OutputTokenLimit != output {
			t.Fatalf("incorrect model limits: %s", id)
		}
	}
	for _, id := range []string{"deepseek-ai/DeepSeek-V4.1-Flash-1018", "DeepSeek-V4.1-Flash-1017 ", "deepseek-v4.1-flash-1017"} {
		if SupportsModel(id) || UpstreamModel(id) != id || PublicModel(id) != id {
			t.Fatalf("unverified model alias accepted: %q", id)
		}
	}
}
