package proxy

import (
	"context"
	"testing"
)

func TestUsageCacheCountsAreBoundedAndUnknownStaysUnknown(t *testing.T) {
	for _, tc := range []struct {
		protocol, body string
		want           int64
		known          bool
	}{
		{"responses", `{"usage":{"input_tokens":100,"output_tokens":20,"input_tokens_details":{"cached_tokens":80}}}`, 80, true},
		{"chat", `{"usage":{"prompt_tokens":100,"completion_tokens":20,"prompt_cache_hit_tokens":50}}`, 50, true},
		{"chat", `{"usage":{"prompt_tokens":100,"completion_tokens":20,"prompt_tokens_details":{"cached_tokens":0}}}`, 0, true},
		{"messages", `{"usage":{"input_tokens":10,"output_tokens":20,"cache_read_input_tokens":80}}`, 80, true},
		{"responses", `{"usage":{"input_tokens":100,"output_tokens":20}}`, 0, false},
		{"responses", `{"usage":{"input_tokens":100,"output_tokens":20,"input_tokens_details":{"cached_tokens":101}}}`, 0, false},
		{"responses", `{"usage":{"input_tokens":100,"output_tokens":20,"input_tokens_details":{"cached_tokens":-1}}}`, 0, false},
	} {
		value := ParseUsage([]byte(tc.body), tc.protocol, false)
		if (value.CachedInput != nil) != tc.known || tc.known && *value.CachedInput != tc.want {
			t.Fatal("cache count lost, fabricated or accepted beyond input", tc.protocol, tc.want)
		}
	}
}

func TestCacheMetadataSurvivesOutputOnlyAndConvertedUsage(t *testing.T) {
	cached := int64(80)
	trace := &Trace{}
	if err := trace.Report(context.Background(), UsageSnapshot{Input: 100, InputKnown: true, CachedInput: &cached}); err != nil {
		t.Fatal(err)
	}
	for _, value := range []UsageSnapshot{
		{Output: 20, OutputKnown: true},
		{Input: 100, Output: 20, InputKnown: true, OutputKnown: true, Complete: true},
	} {
		if err := trace.Report(context.Background(), value); err != nil {
			t.Fatal(err)
		}
	}
	if trace.event.Usage.CachedInput == nil || *trace.event.Usage.CachedInput != 80 {
		t.Fatal("converted downstream usage overwrote known upstream cache details")
	}
}
