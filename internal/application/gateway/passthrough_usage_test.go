package gateway

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"omniproxy/internal/application/activity"
)

// waitPassthroughRequest 等待持久化完成通知，不把客户端 EOF 当作数据库事务完成屏障。
func waitPassthroughRequest(t *testing.T, observer *activity.Service, id string) activity.RequestItem {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	for {
		item, err := observer.Request(ctx, id)
		if err != nil {
			t.Fatal(err)
		}
		if item.State != "pending" {
			return item
		}
		select {
		case <-observer.Updates():
		case <-ctx.Done():
			t.Fatal("等待透明请求持久化超时")
		}
	}
}

func TestPassthroughPersistsReportedUsage(t *testing.T) {
	for _, media := range []string{"missing", "text/event-stream", "application/json"} {
		for _, zero := range []bool{false, true} {
			name := media
			input, output, cached, total := "12", "4", "8", "16"
			if zero {
				name += "/zero"
				input, output, cached, total = "0", "0", "0", "0"
			}
			t.Run(name, func(t *testing.T) {
				s, db, observer, _ := subscriptionQuotaFixture(t)
				expected := `{"id":"resp_test","usage":{"input_tokens":` + input +
					`,"output_tokens":` + output + `,"input_tokens_details":{"cached_tokens":` + cached + `}}}`
				if media != "application/json" {
					expected = "data: {\"type\":\"response.completed\",\"response\":" + expected + "}\n\n"
				}
				upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
					if media == "missing" {
						w.Header()["Content-Type"] = nil
					} else {
						w.Header().Set("Content-Type", media)
					}
					_, _ = io.WriteString(w, expected)
				}))
				defer upstream.Close()
				if _, err := db.Exec(`UPDATE codex_settings SET upstream_base_url=? WHERE id=1`, upstream.URL+"/backend-api/codex"); err != nil {
					t.Fatal(err)
				}
				s.SetCodexUpstreamBase(NewCodexSettingsService(db).UpstreamBaseURL)
				status, body, id := postCodexPassthrough(t, s)
				item := waitPassthroughRequest(t, observer, id)
				if status != http.StatusOK || body != expected || item.State != "success" {
					t.Fatalf("透明响应改变: status=%d state=%s body=%q", status, item.State, body)
				}
				if !item.InputKnown || !item.OutputKnown || !item.UsageComplete ||
					item.InputTokens != input || item.OutputTokens != output ||
					!item.CachedInputKnown || item.CachedInputTokens != cached {
					t.Fatalf("请求用量未保存: %+v", item)
				}
				if len(item.Attempts) != 1 || !item.Attempts[0].UsageComplete ||
					item.Attempts[0].State != "success" || item.Attempts[0].InputTokens != input ||
					item.Attempts[0].OutputTokens != output {
					t.Fatalf("尝试用量未保存: %+v", item.Attempts)
				}
				usage, err := observer.Usage(context.Background())
				if err != nil || usage.AllTime.Requests != 1 || usage.AllTime.CompleteUsage != 1 ||
					usage.AllTime.TotalTokens != total || len(usage.Accounts) != 1 ||
					usage.Accounts[0].Summary.TotalTokens != total || len(usage.ProviderModels) != 1 ||
					usage.ProviderModels[0].Summary.TotalTokens != total {
					t.Fatalf("用量汇总不一致: %+v err=%v", usage, err)
				}
			})
		}
	}
}

func TestPassthroughFailoverPersistsOnlyAcceptedUsage(t *testing.T) {
	s, db, observer, _ := subscriptionQuotaFixture(t)
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.Header.Get("Authorization"), "-a") {
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = io.WriteString(w, `{"error":{"code":"rate_limit"}}`)
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = io.WriteString(w, "data: {\"type\":\"response.completed\",\"response\":{\"usage\":{\"input_tokens\":12,\"output_tokens\":4}}}\n\n")
	}))
	defer upstream.Close()
	if _, err := db.Exec(`UPDATE codex_settings SET upstream_base_url=? WHERE id=1`, upstream.URL+"/backend-api/codex"); err != nil {
		t.Fatal(err)
	}
	s.SetCodexUpstreamBase(NewCodexSettingsService(db).UpstreamBaseURL)
	status, _, id := postCodexPassthrough(t, s)
	item := waitPassthroughRequest(t, observer, id)
	if status != http.StatusOK || item.AccountID != "b" || !item.UsageComplete ||
		item.InputTokens != "12" || item.OutputTokens != "4" || len(item.Attempts) != 2 {
		t.Fatalf("换号后用量丢失: %+v", item)
	}
	if item.Attempts[0].State != "failed" || item.Attempts[0].InputKnown ||
		item.Attempts[0].OutputKnown || item.Attempts[1].State != "success" ||
		!item.Attempts[1].UsageComplete {
		t.Fatalf("拒绝与成功尝试混记: %+v", item.Attempts)
	}
	usage, err := observer.Usage(context.Background())
	if err != nil || usage.AllTime.Requests != 1 || usage.AllTime.TotalTokens != "16" ||
		len(usage.Accounts) != 1 || usage.Accounts[0].Summary.TotalTokens != "16" {
		t.Fatalf("换号重复累计或归属错误: %+v err=%v", usage, err)
	}
}
