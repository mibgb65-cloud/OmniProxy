package workbuddyai

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
)

const catalogFixture = `{"code":0,"msg":"ok","data":{"agents":[{"name":"cli","models":["deepseek-v4.1-flash"]}],"models":[
{"id":"deepseek-v4.1-flash","name":"Deepseek-V4.1-Flash","maxInputTokens":1000000,"maxOutputTokens":128000,"supportsImages":true,"supportsReasoning":true,"reasoning":{"supportedEfforts":["none","low","high"],"defaultEffort":"high","canDisableThinking":false}},
{"id":"glm-5.3","name":"GLM-5.3","maxInputTokens":1000000,"supportsImages":true,"reasoning":{"supportedEfforts":["low","high","max"],"defaultEffort":"xhigh"}},
{"id":"kimi-k3"},
{"id":"custom:team|model","name":"Team Model","maxInputTokens":"200000","supportsImages":false},
{"id":"hy3","maxInputTokens":"192000","maxOutputTokens":64000,"supportsImages":true,"supportsReasoning":false,"reasoning":{"supportedEfforts":["low","high"],"defaultEffort":"low"}},
{"id":"broken-limits","maxInputTokens":-1,"maxOutputTokens":"abc"},
{"id":"dedupe-efforts","maxInputTokens":8000,"reasoning":{"supportedEfforts":["low","low","none","high"],"effort":"high"}},
{"id":"effort-only","maxInputTokens":8000,"supportsReasoning":true,"onlyReasoning":true,"reasoning":{"effort":"high","summary":"auto"}},
{"id":"effort-only-medium","maxInputTokens":8000,"supportsReasoning":true,"onlyReasoning":true,"reasoning":{"effort":"medium","summary":"auto"}}
]}}`

func TestDiscoverModelsParsesAccountCatalog(t *testing.T) {
	var path string
	var header http.Header
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		path, header = r.URL.Path, r.Header.Clone()
		io.WriteString(w, catalogFixture)
	})
	models, err := client.DiscoverModels(context.Background(), testCredentials())
	if err != nil {
		t.Fatal(err)
	}
	if path != "/v3/config" {
		t.Fatalf("path = %q", path)
	}
	if header.Get("Authorization") != "Bearer access-token-1" || header.Get("X-User-Id") != "uid-1" || header.Get("X-Domain") != "www.workbuddy.ai" {
		t.Fatalf("catalog headers = %v", header)
	}
	if len(models) != 9 {
		t.Fatalf("models = %d", len(models))
	}
	deepseek := models[0]
	if deepseek.ID != "deepseek-v4.1-flash" || deepseek.Name != "Deepseek-V4.1-Flash" || !deepseek.Images || deepseek.Free {
		t.Fatalf("deepseek = %+v", deepseek)
	}
	if deepseek.Context != 1000000 || deepseek.Output != 128000 {
		t.Fatalf("deepseek limits = %d / %d", deepseek.Context, deepseek.Output)
	}
	if len(deepseek.Efforts) != 2 || deepseek.Efforts[0] != "low" || deepseek.Efforts[1] != "high" || deepseek.DefaultEffort != "high" {
		t.Fatalf("deepseek reasoning = %+v", deepseek)
	}
	glm := models[1]
	if glm.Images {
		t.Fatal("glm-* must not declare image input")
	}
	if len(glm.Efforts) != 3 || glm.DefaultEffort != "" {
		t.Fatalf("glm reasoning = %+v", glm)
	}
	if glm.Output != 0 {
		t.Fatalf("glm output = %d, want unknown", glm.Output)
	}
	kimi := models[2]
	if kimi.Name != "kimi-k3" || kimi.Context != 0 || kimi.Output != 0 || len(kimi.Efforts) != 0 || kimi.Images {
		t.Fatalf("minimal model = %+v", kimi)
	}
	custom := models[3]
	if custom.ID != "custom:team|model" || custom.Name != "Team Model" || custom.Context != 200000 || custom.Images {
		t.Fatalf("enterprise model = %+v", custom)
	}
	hy3 := models[4]
	if len(hy3.Efforts) != 0 || hy3.DefaultEffort != "" || hy3.Output != 64000 {
		t.Fatalf("reasoning-disabled model = %+v", hy3)
	}
	broken := models[5]
	if broken.Context != 0 || broken.Output != 0 {
		t.Fatalf("invalid limits = %+v", broken)
	}
	dedupe := models[6]
	if len(dedupe.Efforts) != 2 || dedupe.Efforts[0] != "low" || dedupe.Efforts[1] != "high" || dedupe.DefaultEffort != "high" {
		t.Fatalf("deduped efforts = %+v", dedupe)
	}
	// 上游只给 effort 的模型（真实目录里的 max-model／gemini-4-*／kimi-k3）补通用档位。
	effortOnly := models[7]
	if len(effortOnly.Efforts) != 3 || effortOnly.Efforts[0] != "low" || effortOnly.Efforts[1] != "high" || effortOnly.Efforts[2] != "max" || effortOnly.DefaultEffort != "high" {
		t.Fatalf("effort-only model = %+v", effortOnly)
	}
	effortOnlyMedium := models[8]
	if len(effortOnlyMedium.Efforts) != 4 || effortOnlyMedium.Efforts[1] != "medium" || effortOnlyMedium.DefaultEffort != "medium" {
		t.Fatalf("effort-only medium model = %+v", effortOnlyMedium)
	}
}

func TestDiscoverModelsAcceptsFlatCatalogAndEnterprise(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Enterprise-Id") != "ent-1" || r.Header.Get("X-Tenant-Id") != "ent-1" {
			t.Errorf("enterprise headers = %v", r.Header)
		}
		io.WriteString(w, `{"models":[{"id":"glm-5.2","name":"GLM-5.2"}]}`)
	})
	cred := testCredentials()
	cred.EnterpriseID = "ent-1"
	models, err := client.DiscoverModels(context.Background(), cred)
	if err != nil || len(models) != 1 || models[0].ID != "glm-5.2" || models[0].Images {
		t.Fatalf("models = %+v, err = %v", models, err)
	}
}

func TestDiscoverModelsRejectsBadCatalogs(t *testing.T) {
	cases := map[string]string{
		"empty":        `{"code":0,"data":{"models":[]}}`,
		"missing list": `{"code":0,"data":{}}`,
		"duplicate":    `{"code":0,"data":{"models":[{"id":"a"},{"id":"a"}]}}`,
		"invalid id":   `{"code":0,"data":{"models":[{"id":"bad model"}]}}`,
		"long name":    `{"code":0,"data":{"models":[{"id":"a","name":"` + strings.Repeat("n", maxModelNameBytes+1) + `"}]}}`,
		"controls":     `{"code":0,"data":{"models":[{"id":"a","name":"a\u0001b"}]}}`,
		"not json":     `nope`,
	}
	for name, body := range cases {
		client := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
			io.WriteString(w, body)
		})
		if _, err := client.DiscoverModels(context.Background(), testCredentials()); ErrorCode(err) != CodeInvalidResponse {
			t.Errorf("%s => %v", name, err)
		}
	}
	tooMany := strings.Builder{}
	tooMany.WriteString(`{"code":0,"data":{"models":[`)
	for index := 0; index <= maxCatalogModels; index++ {
		if index > 0 {
			tooMany.WriteByte(',')
		}
		fmt.Fprintf(&tooMany, `{"id":"model-%d"}`, index)
	}
	tooMany.WriteString(`]}}`)
	client := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		io.WriteString(w, tooMany.String())
	})
	if _, err := client.DiscoverModels(context.Background(), testCredentials()); ErrorCode(err) != CodeInvalidResponse {
		t.Errorf("too many => %v", err)
	}
	oversized := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		io.WriteString(w, strings.Repeat(" ", maxCatalogBytes+16))
	})
	if _, err := oversized.DiscoverModels(context.Background(), testCredentials()); ErrorCode(err) != CodeInvalidResponse {
		t.Errorf("oversized => %v", err)
	}
}

func TestDiscoverModelsMapsFailures(t *testing.T) {
	cases := []struct {
		status int
		body   string
		want   string
	}{
		{status: http.StatusUnauthorized, want: CodeUnauthorized},
		{status: http.StatusTooManyRequests, want: CodeRateLimited},
		{status: http.StatusInternalServerError, want: CodeUpstreamUnavailable},
		{status: http.StatusOK, body: `{"code":11158,"msg":"no entitlement"}`, want: CodeModelNotEntitled},
		{status: http.StatusOK, body: `{"code":10085}`, want: CodeUserAgentRefused},
		{status: http.StatusBadRequest, body: `{"code":11128}`, want: CodeChannelRejected},
	}
	for _, test := range cases {
		client := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(test.status)
			io.WriteString(w, test.body)
		})
		if _, err := client.DiscoverModels(context.Background(), testCredentials()); ErrorCode(err) != test.want {
			t.Fatalf("status %d body %s => %v, want %q", test.status, test.body, err, test.want)
		}
	}
	if _, err := New(nil).DiscoverModels(context.Background(), Credentials{}); ErrorCode(err) != CodeCredentialUnavailable {
		t.Fatalf("invalid credential => %v", err)
	}
}
