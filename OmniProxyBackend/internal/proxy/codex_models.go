package proxy

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"omniproxy/internal/logs"
)

// The ChatGPT desktop app and Codex CLI fill their model picker from the model
// catalog returned by GET {openai_base_url}/v1/models. Upstream only lists
// official models, so models routed through OmniProxy (Atria, DeepSeek, ...)
// never show up even though requests for them are proxied fine. The entries
// below mirror the subset of the catalog schema that the client requires, with
// base_instructions standing in for the full GPT-5 instructions template.

const codexInjectedBaseInstructions = "You are Codex, a coding agent."

// Injected entries sort after the official flagship models (priority 0-3) but
// ahead of the legacy/review entries (priority 12+).
const codexInjectedModelPriority = 8

const maxCodexModelsBodyBytes = 4 * 1024 * 1024

type codexModelProfile struct {
	Model         string
	ContextWindow int
}

type codexReasoningLevel struct {
	Effort      string `json:"effort"`
	Description string `json:"description"`
}

type codexTruncationPolicy struct {
	Mode  string `json:"mode"`
	Limit int    `json:"limit"`
}

type codexCatalogEntry struct {
	Slug                       string                `json:"slug"`
	DisplayName                string                `json:"display_name"`
	Description                string                `json:"description"`
	SupportedReasoningLevels   []codexReasoningLevel `json:"supported_reasoning_levels"`
	ShellType                  string                `json:"shell_type"`
	Visibility                 string                `json:"visibility"`
	SupportedInAPI             bool                  `json:"supported_in_api"`
	Priority                   int                   `json:"priority"`
	SupportVerbosity           bool                  `json:"support_verbosity"`
	TruncationPolicy           codexTruncationPolicy `json:"truncation_policy"`
	ExperimentalSupportedTools []string              `json:"experimental_supported_tools"`
	ContextWindow              int                   `json:"context_window,omitempty"`
	MaxContextWindow           int                   `json:"max_context_window,omitempty"`
	BaseInstructions           string                `json:"base_instructions"`
}

var codexInjectedReasoningLevels = []codexReasoningLevel{
	{Effort: "low", Description: "Fast responses with lighter reasoning"},
	{Effort: "medium", Description: "Balances speed and reasoning depth for everyday tasks"},
	{Effort: "high", Description: "Greater reasoning depth for complex problems"},
	{Effort: "xhigh", Description: "Extra high reasoning depth for complex problems"},
	{Effort: "max", Description: "Maximum reasoning depth for the hardest problems"},
}

func isCodexModelsListRequest(r *http.Request) bool {
	if r == nil || r.URL == nil || r.Method != http.MethodGet {
		return false
	}
	if isWebSocketUpgrade(r) {
		return false
	}
	path := stripPathPrefix(r.URL.Path, "/backend-api/codex")
	path = stripPathPrefix(path, "/codex")
	path = stripPathPrefix(path, "/v1")
	return path == "/models"
}

func (s *Service) rewriteCodexModelsResponse(r *http.Request, resp *http.Response) {
	if !isCodexModelsListRequest(r) || resp.StatusCode != http.StatusOK {
		return
	}
	profiles, err := loadCodexModelProfiles()
	if err != nil || len(profiles) == 0 {
		return
	}
	if added := applyCodexModelsInjection(resp, profiles); added > 0 && s.logs != nil {
		s.logs.Add(logs.Entry{
			Level:   logs.LevelInfo,
			Method:  r.Method,
			Path:    r.URL.RequestURI(),
			Message: fmt.Sprintf("injected %d OmniProxy model(s) into Codex model catalog", added),
		})
	}
}

// applyCodexModelsInjection rewrites resp.Body in place, appending the given
// profiles when their models are not already part of the catalog. It returns
// the number of entries added; the response is left untouched on any error.
func applyCodexModelsInjection(resp *http.Response, profiles []codexModelProfile) int {
	if resp.ContentLength <= 0 || resp.ContentLength > maxCodexModelsBodyBytes {
		return 0
	}
	raw, err := io.ReadAll(resp.Body)
	closeBody(resp.Body)
	if err != nil {
		resp.Body = io.NopCloser(bytes.NewReader(raw))
		return 0
	}

	injected, added := injectCodexModels(raw, profiles)
	if added == 0 {
		resp.Body = io.NopCloser(bytes.NewReader(raw))
		return 0
	}

	resp.Body = io.NopCloser(bytes.NewReader(injected))
	resp.ContentLength = int64(len(injected))
	resp.Header.Del("Content-Encoding")
	resp.Header.Set("Content-Length", strconv.Itoa(len(injected)))
	return added
}

// injectCodexModels returns the catalog bytes with profiles appended, plus how
// many entries were added. Catalog documents that fail to parse, or profiles
// whose model is already listed, are passed through unchanged.
func injectCodexModels(raw []byte, profiles []codexModelProfile) ([]byte, int) {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	var doc map[string]any
	if err := decoder.Decode(&doc); err != nil {
		return raw, 0
	}
	existing, ok := doc["models"].([]any)
	if !ok {
		return raw, 0
	}

	seen := map[string]bool{}
	for _, item := range existing {
		if model, ok := item.(map[string]any); ok {
			if slug, ok := model["slug"].(string); ok {
				seen[strings.ToLower(strings.TrimSpace(slug))] = true
			}
		}
	}

	updated := append(make([]any, 0, len(existing)+len(profiles)), existing...)
	added := 0
	for _, profile := range profiles {
		model := strings.TrimSpace(profile.Model)
		if model == "" || seen[strings.ToLower(model)] {
			continue
		}
		seen[strings.ToLower(model)] = true
		updated = append(updated, codexCatalogEntry{
			Slug:                       model,
			DisplayName:                codexModelDisplayName(model),
			Description:                fmt.Sprintf("%s routed through OmniProxy", codexModelDisplayName(model)),
			SupportedReasoningLevels:   codexInjectedReasoningLevels,
			ShellType:                  "unified_exec",
			Visibility:                 "list",
			SupportedInAPI:             true,
			Priority:                   codexInjectedModelPriority,
			SupportVerbosity:           true,
			TruncationPolicy:           codexTruncationPolicy{Mode: "tokens", Limit: 10000},
			ExperimentalSupportedTools: []string{},
			ContextWindow:              profile.ContextWindow,
			MaxContextWindow:           profile.ContextWindow,
			BaseInstructions:           codexInjectedBaseInstructions,
		})
		added++
	}
	if added == 0 {
		return raw, 0
	}

	doc["models"] = updated
	out, err := json.Marshal(doc)
	if err != nil {
		return raw, 0
	}
	return out, added
}

func codexModelDisplayName(model string) string {
	switch strings.ToLower(strings.TrimSpace(model)) {
	case "atria-dawn-preview":
		return "Atria Dawn Preview"
	case "deepseek-v4-pro", "deepseek-v4-pro[1m]":
		return "DeepSeek V4 Pro"
	case "deepseek-v4-flash":
		return "DeepSeek V4 Flash"
	case "mimo-v2.5-pro", "mimo-v2.5-pro[1m]", "mimo-v2.5":
		return "MiMo V2.5"
	case "kimi-for-coding":
		return "Kimi for Coding"
	case "glm-5.1":
		return "GLM 5.1"
	case "minimax-m2.7":
		return "MiniMax M2.7"
	default:
		var builder strings.Builder
		for _, word := range strings.Fields(strings.ReplaceAll(model, "-", " ")) {
			if builder.Len() > 0 {
				builder.WriteByte(' ')
			}
			builder.WriteString(strings.ToUpper(word[:1]))
			builder.WriteString(word[1:])
		}
		return builder.String()
	}
}

func loadCodexModelProfiles() ([]codexModelProfile, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	return codexModelProfilesInDir(filepath.Join(home, ".codex"))
}

func codexModelProfilesInDir(dir string) ([]codexModelProfile, error) {
	matches, err := filepath.Glob(filepath.Join(dir, "omniproxy-*.config.toml"))
	if err != nil {
		return nil, err
	}
	profiles := make([]codexModelProfile, 0, len(matches))
	for _, path := range matches {
		raw, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		if profile := parseCodexModelProfile(string(raw)); profile.Model != "" {
			profiles = append(profiles, profile)
		}
	}
	return profiles, nil
}

func parseCodexModelProfile(raw string) codexModelProfile {
	var profile codexModelProfile
	for _, line := range strings.Split(raw, "\n") {
		key, value, ok := splitCodexProfileLine(line)
		if !ok {
			continue
		}
		switch key {
		case "model":
			profile.Model = value
		case "model_context_window":
			if window, err := strconv.Atoi(value); err == nil {
				profile.ContextWindow = window
			}
		}
	}
	return profile
}

func splitCodexProfileLine(line string) (string, string, bool) {
	line = strings.TrimSpace(line)
	if strings.HasPrefix(line, "#") || !strings.Contains(line, "=") {
		return "", "", false
	}
	key, value, ok := strings.Cut(line, "=")
	if !ok {
		return "", "", false
	}
	key = strings.ToLower(strings.TrimSpace(key))
	value = strings.TrimSpace(value)
	if len(value) >= 2 && strings.HasPrefix(value, `"`) && strings.HasSuffix(value, `"`) {
		value = value[1 : len(value)-1]
	}
	return key, value, true
}
