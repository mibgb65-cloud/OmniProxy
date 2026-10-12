package proxy

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	catalog "omniproxy/internal/core/modelcatalog"
	"omniproxy/internal/core/routing"
)

// 只允许目录自身使用的参数；推理请求仍拒绝未知查询字符串。
func validModelQuery(path, raw string) bool {
	if raw == "" {
		return true
	}
	if len(raw) > 512 || path != "/v1/models" && path != "/v1/codex-models" {
		return false
	}
	values, err := url.ParseQuery(raw)
	if err != nil || len(values) > 3 {
		return false
	}
	for key, values := range values {
		if len(values) != 1 || values[0] == "" {
			return false
		}
		value := values[0]
		switch {
		case path == "/v1/codex-models" && key == "client_version":
			if len(value) > 64 || strings.ContainsFunc(value, func(r rune) bool {
				return !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || strings.ContainsRune("._+-", r))
			}) {
				return false
			}
		case path == "/v1/models" && key == "limit":
			limit, err := strconv.Atoi(value)
			if err != nil || limit < 1 || limit > catalog.MaxModels {
				return false
			}
		case path == "/v1/models" && (key == "after_id" || key == "before_id"):
			if !routing.ValidModel(value) {
				return false
			}
		default:
			return false
		}
	}
	return values.Get("after_id") == "" || values.Get("before_id") == ""
}

func (s *Server) catalogEntries(ctx context.Context) ([]catalog.Entry, error) {
	if source, ok := s.opts.Upstream.(interface {
		Catalog(context.Context) ([]catalog.Entry, error)
	}); ok {
		entries, err := source.Catalog(ctx)
		if err != nil {
			return nil, err
		}
		return catalog.Normalize(entries)
	}
	ids := s.opts.Models
	if source, ok := s.opts.Upstream.(ModelCatalog); ok {
		var err error
		ids, err = source.AvailableModels(ctx)
		if err != nil {
			return nil, err
		}
	}
	entries := make([]catalog.Entry, 0, len(ids))
	for _, id := range ids {
		entries = append(entries, catalog.Entry{ID: id, Name: id})
	}
	return catalog.Normalize(entries)
}

func (s *Server) writeModelCatalog(w http.ResponseWriter, r *http.Request, codex bool) {
	entries, err := s.catalogEntries(r.Context())
	if err != nil {
		writeFailure(w, safeFailure(r.Context(), err))
		return
	}
	var body []byte
	if codex {
		body, err = catalog.CodexJSON(entries)
	} else {
		body, err = s.modelListJSON(entries, r.URL.Query())
	}
	if err != nil {
		writeFailure(w, Failure("model_catalog_unavailable", 503))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if _, err := w.Write(body); err != nil && s.opts.Logger != nil {
		s.opts.Logger.Warn("gateway model response failed", "code", "client_write_failed")
	}
}

type listedModel struct {
	ID          string `json:"id"`
	Object      string `json:"object"`
	Owner       string `json:"owned_by"`
	Name        string `json:"display_name"`
	Description string `json:"description,omitempty"`
}

func (s *Server) modelListJSON(entries []catalog.Entry, query url.Values) ([]byte, error) {
	start, end := 0, len(entries)
	afterFound, beforeFound := query.Get("after_id") == "", query.Get("before_id") == ""
	for index, entry := range entries {
		if entry.ID == query.Get("after_id") {
			start = index + 1
			afterFound = true
		}
		if entry.ID == query.Get("before_id") {
			end = index
			beforeFound = true
		}
	}
	if !afterFound || !beforeFound {
		start, end = 0, 0
	}
	if start > end {
		start = end
	}
	limit := catalog.MaxModels
	if query.Get("limit") != "" {
		limit, _ = strconv.Atoi(query.Get("limit"))
	}
	hasMore := end-start > limit
	if hasMore {
		end = start + limit
	}
	models := make([]listedModel, 0, end-start)
	for _, entry := range entries[start:end] {
		owner := entry.ProviderKind
		if owner == "" {
			owner = "omniproxy"
			if s.supportsModel(entry.ID) {
				owner = "deepseek"
			}
		}
		models = append(models, listedModel{entry.ID, "model", owner, entry.Name, entry.Metadata.Description})
	}
	first, last := "", ""
	if len(models) > 0 {
		first, last = models[0].ID, models[len(models)-1].ID
	}
	return json.Marshal(struct {
		Object  string        `json:"object"`
		Data    []listedModel `json:"data"`
		HasMore bool          `json:"has_more"`
		FirstID string        `json:"first_id"`
		LastID  string        `json:"last_id"`
	}{"list", models, hasMore, first, last})
}
