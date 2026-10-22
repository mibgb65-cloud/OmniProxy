package gateway

import (
	"context"
	"io"
	"net/http"
	"omniproxy/internal/application/activity"
	"omniproxy/internal/core/proxy"
	"os"
	"strings"
	"testing"
)

func TestActualUpstreamUsageAcrossConversionAndStreaming(t *testing.T) {
	for _, stream := range []bool{false, true} {
		service, db, _ := setupService(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if stream {
				raw, _ := io.ReadAll(r.Body)
				if !strings.Contains(string(raw), `"include_usage":true`) {
					t.Error("upstream usage not requested")
				}
				w.Header().Set("Content-Type", "text/event-stream")
				io.WriteString(w, "data: {\"choices\":[{\"delta\":{\"content\":\"OK\"}}]}\n\ndata: {\"choices\":[],\"usage\":{\"prompt_tokens\":12,\"completion_tokens\":4}}\n\ndata: [DONE]\n\n")
			} else {
				w.Header().Set("Content-Type", "application/json")
				io.WriteString(w, `{"id":"test","model":"deepseek-flash","choices":[{"finish_reason":"stop","message":{"content":"OK"}}],"usage":{"prompt_tokens":12,"completion_tokens":4}}`)
			}
		}))
		sql, err := os.ReadFile("../../../migrations/008_request_accounting.sql")
		if err != nil {
			t.Fatal(err)
		}
		if _, err = db.Exec(string(sql)); err != nil {
			t.Fatal(err)
		}
		providerSQL, err := os.ReadFile("../../../migrations/009_provider_model_usage.sql")
		if err != nil {
			t.Fatal(err)
		}
		if _, err = db.Exec(string(providerSQL)); err != nil {
			t.Fatal(err)
		}
		dashboardSQL, err := os.ReadFile("../../../migrations/010_dashboard_buckets.sql")
		if err != nil {
			t.Fatal(err)
		}
		if _, err = db.Exec(string(dashboardSQL)); err != nil {
			t.Fatal(err)
		}
		service.Stop(context.Background())
		stepHistory, err := os.ReadFile("../../../migrations/022_stepfun_request_scope.sql")
		if err != nil {
			t.Fatal(err)
		}
		if _, err = db.Exec(string(stepHistory)); err != nil {
			t.Fatal(err)
		}
		firstTokenSQL, err := os.ReadFile("../../../migrations/032_request_first_token.sql")
		if err != nil {
			t.Fatal(err)
		}
		if _, err = db.Exec(string(firstTokenSQL)); err != nil {
			t.Fatal(err)
		}
		observer := activity.New(db)
		failoverSQL, err := os.ReadFile("../../../migrations/033_account_failover.sql")
		if err != nil {
			t.Fatal(err)
		}
		if _, err = db.Exec(string(failoverSQL)); err != nil {
			t.Fatal(err)
		}
		estimateSQL, err := os.ReadFile("../../../migrations/037_quota_cost_estimates.sql")
		if err != nil {
			t.Fatal(err)
		}
		if _, err = db.Exec(string(estimateSQL)); err != nil {
			t.Fatal(err)
		}
		clientSQL, err := os.ReadFile("../../../migrations/040_request_client.sql")
		if err != nil {
			t.Fatal(err)
		}
		if _, err = db.Exec(string(clientSQL)); err != nil {
			t.Fatal(err)
		}
		server := proxy.New(proxy.Options{Port: 0, Upstream: service, Models: []string{"deepseek-flash"}, Observer: observer})
		if _, err = server.Start(); err != nil {
			t.Fatal(err)
		}
		defer server.Stop(context.Background())
		protocol, body := "responses", `{"model":"deepseek-flash","input":"hello"}`
		if stream {
			protocol = "chat/completions"
			body = `{"model":"deepseek-flash","stream":true,"messages":[{"role":"user","content":"hello"}]}`
		}
		response, err := http.Post(server.Status().Address+"/v1/"+protocol, "application/json", strings.NewReader(body))
		if err != nil {
			t.Fatal(err)
		}
		io.Copy(io.Discard, response.Body)
		response.Body.Close()
		if response.StatusCode != 200 {
			t.Fatal("proxy failed")
		}
		id := response.Header.Get("X-Request-ID")
		item, err := observer.Request(context.Background(), id)
		if err != nil || item.InputTokens != "12" || item.OutputTokens != "4" || !item.UsageComplete {
			t.Fatal("upstream usage lost", item, err)
		}
		if (item.FirstTokenMS != nil) != stream {
			t.Fatal("first-token timing lost or fabricated across conversion", item)
		}
		totals, err := observer.Usage(context.Background())
		if err != nil || totals.AllTime.TotalTokens != "16" {
			t.Fatal("wrong totals", err)
		}
	}
}
