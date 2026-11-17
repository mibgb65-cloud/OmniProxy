package workbuddyai

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

// 两个构建共用实现，但站点、登录平台标识、UA 与 Messages 能力必须按构建区分。
func TestBuildsDescribeSitesAndFingerprints(t *testing.T) {
	cases := []struct {
		build    Build
		kind     string
		base     string
		platform string
		ua       string
		messages bool
		glmImage bool
	}{
		{
			build:    International,
			kind:     "workbuddy_ai",
			base:     "https://www.workbuddy.ai",
			platform: "workbuddy-ai",
			ua:       "WorkBuddy/5.9.3 workbuddy-ai/5.9.3 CLI/2.165.0",
			messages: false,
			glmImage: false,
		},
		{
			build:    China,
			kind:     "workbuddy",
			base:     "https://copilot.tencent.com",
			platform: "workbuddy",
			ua:       "WorkBuddy/5.9.3 WorkBuddy/5.9.3 CLI/2.165.0",
			messages: true,
			glmImage: true,
		},
	}
	for _, test := range cases {
		if test.build.Kind != test.kind || test.build.BaseURL != test.base || test.build.Platform != test.platform {
			t.Errorf("build = %+v", test.build)
		}
		if test.build.UserAgent() != test.ua {
			t.Errorf("%s user agent = %q", test.kind, test.build.UserAgent())
		}
		if test.build.ShortUserAgent() != "WorkBuddy/5.9.3" {
			t.Errorf("%s short user agent = %q", test.kind, test.build.ShortUserAgent())
		}
		if test.build.Messages != test.messages {
			t.Errorf("%s messages = %v", test.kind, test.build.Messages)
		}
		if test.build.MessagesNative("claude-opus-5.5") != test.messages {
			t.Errorf("%s claude messages = %v", test.kind, test.build.MessagesNative("claude-opus-5.5"))
		}
		if test.build.MessagesNative("glm-5.3") {
			t.Errorf("%s routed glm-5.3 to messages", test.kind)
		}
		// 国际版的 glm-* 声明了图片但上游会丢掉；国内版能读图。
		if got := test.build.ImagesExceptGLM; got == test.glmImage {
			t.Errorf("%s ImagesExceptGLM = %v", test.kind, got)
		}
		if build, ok := BuildOf(test.kind); !ok || build.Kind != test.kind {
			t.Errorf("BuildOf(%q) = %+v, %v", test.kind, build, ok)
		}
	}
	if _, ok := BuildOf("openai"); ok {
		t.Fatal("unrelated kind resolved to a WorkBuddy build")
	}
}

// 国内版的登录 state 必须带 platform=workbuddy，授权页版本与随机会话 ID 仍然补齐。
func TestChinaBuildUsesItsOwnPlatformAndVersion(t *testing.T) {
	var query, userAgent string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		query, userAgent = r.URL.RawQuery, r.Header.Get("User-Agent")
		w.Write([]byte(`{"code":0,"data":{"state":"state-1","authUrl":"https://copilot.tencent.com/login?state=state-1"}}`))
	}))
	t.Cleanup(server.Close)
	client, err := NewClientAtFor(China, server.URL, server.Client().Transport)
	if err != nil {
		t.Fatal(err)
	}
	signIn, err := client.BeginSignIn(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if query != "platform=workbuddy" {
		t.Fatalf("state query = %q", query)
	}
	if userAgent != "WorkBuddy/5.9.3" {
		t.Fatalf("login user agent = %q", userAgent)
	}
	// 授权页按桌面端形态补版本与随机会话 ID，国内登录页对版本有分支。
	authURL, err := url.Parse(signIn.AuthURL)
	if err != nil {
		t.Fatal(err)
	}
	if authURL.Query().Get("version") != "5.9.3" || authURL.Query().Get("loginSessionId") == "" {
		t.Fatalf("auth url query = %q", authURL.RawQuery)
	}
}

// 国内版目录里 glm-* 的图片能力按上游实际行为保留；国际版仍按文本处理。
func TestChinaBuildKeepsGLMImages(t *testing.T) {
	body := `{"code":0,"data":{"models":[
		{"id":"glm-5.3-flash","name":"GLM-5.3-Flash","supportsImages":true},
		{"id":"claude-opus-5.5","name":"Claude-Opus-5.5","supportsImages":true}
	]}}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte(body))
	}))
	t.Cleanup(server.Close)
	client, err := NewClientAtFor(China, server.URL, server.Client().Transport)
	if err != nil {
		t.Fatal(err)
	}
	models, err := client.DiscoverModels(t.Context(), testCredentials())
	if err != nil {
		t.Fatal(err)
	}
	if len(models) != 2 || !models[0].Images || !models[1].Images {
		t.Fatalf("china catalog images = %+v", models)
	}
	intl, err := NewClientAt(server.URL, server.Client().Transport)
	if err != nil {
		t.Fatal(err)
	}
	models, err = intl.DiscoverModels(t.Context(), testCredentials())
	if err != nil {
		t.Fatal(err)
	}
	if len(models) != 2 || models[0].Images || !models[1].Images {
		t.Fatalf("international catalog images = %+v", models)
	}
}
