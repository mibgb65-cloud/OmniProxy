package proxy

import (
	"log/slog"
	"net/http"
)

type trackedWriter struct {
	http.ResponseWriter
	status int
	code   string
}

func (w *trackedWriter) WriteHeader(status int) {
	if w.status == 0 {
		w.status = status
	}
	w.ResponseWriter.WriteHeader(status)
}
func (w *trackedWriter) Write(body []byte) (int, error) {
	if w.status == 0 {
		w.status = 200
	}
	return w.ResponseWriter.Write(body)
}
func (w *trackedWriter) Unwrap() http.ResponseWriter { return w.ResponseWriter }
func (t *Trace) completeLogged(status int, code string, log *slog.Logger) {
	if status == 0 {
		status = 200
	}
	err := t.finish(status, code)
	if log != nil {
		if err != nil {
			log.Error("request accounting failed", "module", "gateway", "request_id", t.ID(), "code", "accounting_unavailable")
		}
		usageState := "unknown"
		if t.event.Usage.InputKnown || t.event.Usage.OutputKnown {
			usageState = "partial"
		}
		if t.event.Usage.Complete {
			usageState = "complete"
		}
		log.Info("gateway request finished", "module", "gateway", "request_id", t.ID(), "model", t.event.Model, "status_code", status, "code", code, "latency_ms", t.event.LatencyMS, "input_tokens", t.event.Usage.Input, "output_tokens", t.event.Usage.Output, "usage_state", usageState)
	}
}
