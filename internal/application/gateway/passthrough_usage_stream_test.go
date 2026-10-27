package gateway

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// 断流不能提前把尝试算成成功；已交付终结事件则保留真实用量与成功结果。
func TestPassthroughStreamPersistsFinalAttemptState(t *testing.T) {
	for _, mode := range []string{"completed_disconnect", "upstream_disconnect", "client_cancelled"} {
		t.Run(mode, func(t *testing.T) {
			s, db, observer, _ := subscriptionQuotaFixture(t)
			first := "data: {\"type\":\"response.created\",\"response\":{\"id\":\"resp_test\"}}\n\n"
			completed := "data: {\"type\":\"response.completed\",\"response\":{\"usage\":{\"input_tokens\":12,\"output_tokens\":4}}}\n\n"
			upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "text/event-stream")
				_, _ = io.WriteString(w, first)
				w.(http.Flusher).Flush()
				if mode == "client_cancelled" {
					<-r.Context().Done()
					return
				}
				if mode == "completed_disconnect" {
					_, _ = io.WriteString(w, completed)
					w.(http.Flusher).Flush()
				}
				panic(http.ErrAbortHandler)
			}))
			defer upstream.Close()
			if _, err := db.Exec(`UPDATE codex_settings SET upstream_base_url=? WHERE id=1`, upstream.URL+"/backend-api/codex"); err != nil {
				t.Fatal(err)
			}
			s.SetCodexUpstreamBase(NewCodexSettingsService(db).UpstreamBaseURL)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			request, err := http.NewRequestWithContext(ctx, http.MethodPost, s.Info().Runtime.Address+"/codex/v1/responses",
				strings.NewReader(`{"model":"gpt-test","stream":true,"input":[]}`))
			if err != nil {
				t.Fatal(err)
			}
			request.Header.Set("Content-Type", "application/json")
			response, err := http.DefaultClient.Do(request)
			if err != nil {
				t.Fatal(err)
			}
			defer response.Body.Close()
			if response.StatusCode != http.StatusOK {
				t.Fatalf("status=%d", response.StatusCode)
			}
			if mode == "client_cancelled" {
				if _, err := io.ReadFull(response.Body, make([]byte, len(first))); err != nil {
					t.Fatal(err)
				}
				cancel()
			} else {
				body, err := io.ReadAll(response.Body)
				if err != nil || !strings.HasPrefix(string(body), first) {
					t.Fatalf("响应前缀丢失: %q err=%v", body, err)
				}
				if mode == "completed_disconnect" && string(body) != first+completed {
					t.Fatalf("终结事件未交付: %q", body)
				}
			}
			item := waitPassthroughRequest(t, observer, response.Header.Get("X-Request-ID"))
			state, code := "failed", "upstream_stream_failed"
			if mode == "completed_disconnect" {
				state, code = "success", ""
			} else if mode == "client_cancelled" {
				state, code = "cancelled", "request_cancelled"
			}
			if item.State != state || item.ErrorCode != code || len(item.Attempts) != 1 ||
				item.Attempts[0].State != state || item.Attempts[0].ErrorCode != code {
				t.Fatalf("请求与尝试收尾不一致: %+v", item)
			}
			known := mode == "completed_disconnect"
			if item.UsageComplete != known || item.Attempts[0].UsageComplete != known ||
				item.InputKnown != known || item.OutputKnown != known {
				t.Fatalf("未知用量被伪造或完整用量丢失: %+v", item)
			}
			usage, err := observer.Usage(context.Background())
			total := "0"
			if known {
				total = "16"
			}
			if err != nil || usage.AllTime.Requests != 1 || usage.AllTime.TotalTokens != total {
				t.Fatalf("断流用量累计错误: %+v err=%v", usage.AllTime, err)
			}
		})
	}
}
