package proxy

import "testing"

func TestUpstreamErrorEnvelopeDoesNotExposeNestedErrors(t *testing.T) {
	for _, tc := range []struct {
		body string
		want bool
	}{
		{`{"error":{"message":"private-error"}}`, true},
		{`{"type":"response.failed","response":{"error":{"message":"private-error"}}}`, true},
		{`{"type":"response.completed","response":{"error":null,"output":[]}}`, false},
		{`{"choices":[],"error":null,"response":"compatible-extension"}`, false},
		{`{"choices":[],"response":42}`, false},
		{`{"choices":[]}`, false},
		{`not-json`, true},
	} {
		if got := hasUpstreamError([]byte(tc.body)); got != tc.want {
			t.Fatalf("incorrect error envelope classification: got=%v want=%v", got, tc.want)
		}
	}
}
