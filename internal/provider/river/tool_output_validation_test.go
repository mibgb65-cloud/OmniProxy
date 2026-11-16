package river

import (
	"encoding/json"
	"testing"

	"omniproxy/internal/core/proxy"
)

// 工具结果必须带非空 call_id：缺少它时 River 会在响应头之后中断连接，
// 适配器要在本地就明确拒绝，而不是把请求发出去再让客户端看到断流。
func TestResponsesToolOutputRequiresCallID(t *testing.T) {
	cases := []struct {
		name  string
		input string
		code  string
	}{
		{
			name:  "desktop companion snapshot without call_id",
			input: `[{"type":"function_call_output","name":"codex_apps_open_page","namespace":"agent","output":"{\"page_id\":null}"}]`,
			code:  "unsupported_content",
		},
		{
			name:  "null call_id",
			input: `[{"type":"function_call_output","call_id":null,"output":"ok"}]`,
			code:  "unsupported_content",
		},
		{
			name:  "empty call_id",
			input: `[{"type":"function_call_output","call_id":"","output":"ok"}]`,
			code:  "unsupported_content",
		},
		{
			name:  "non-string call_id",
			input: `[{"type":"function_call_output","call_id":7,"output":"ok"}]`,
			code:  "unsupported_content",
		},
		{
			name:  "matched call and output",
			input: `[{"type":"function_call","call_id":"call_1","name":"exec_command","arguments":"{}"},{"type":"function_call_output","call_id":"call_1","output":"ok"}]`,
			code:  "",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var input json.RawMessage = []byte(tc.input)
			body, err := json.Marshal(map[string]any{
				"model":  DeepSeekFlash1017,
				"stream": true,
				"input":  input,
			})
			if err != nil {
				t.Fatalf("marshal request: %v", err)
			}
			request := proxy.Request{Protocol: "responses", Model: DeepSeekFlash1017, Stream: true, Body: body}
			_, err = validatedBody(request, "/responses")
			if got := failureCode(err); got != tc.code {
				t.Fatalf("code = %q, want %q (err=%v)", got, tc.code, err)
			}
		})
	}
}
