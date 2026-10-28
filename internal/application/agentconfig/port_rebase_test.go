package agentconfig

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func appliedCodexFixture(t *testing.T) *Service {
	t.Helper()
	s := fixture(t)
	original := []byte("model_provider='openai'\nmodel='old'\n")
	if err := os.MkdirAll(filepath.Dir(s.paths.Codex), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(s.paths.Codex, original, 0600); err != nil {
		t.Fatal(err)
	}
	input := Input{Client: "codex", Model: "gpt-test", Revision: digest(original, true)}
	if _, err := s.Apply(context.Background(), input); err != nil {
		t.Fatal(err)
	}
	return s
}

func readClientFile(t *testing.T, path string) string {
	t.Helper()
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(body)
}

func TestRebasePortRewritesManagedCodexAddress(t *testing.T) {
	s := appliedCodexFixture(t)
	ctx := context.Background()
	s.SetPort(4321)
	result, err := s.RebasePort(ctx, 3001, 4321)
	if err != nil {
		t.Fatalf("RebasePort error: %v", err)
	}
	if len(result.Updated) != 1 || result.Updated[0] != "codex" {
		t.Fatalf("updated = %v, want [codex]", result.Updated)
	}
	body := readClientFile(t, s.paths.Codex)
	if !strings.Contains(body, "http://127.0.0.1:4321/codex/v1") {
		t.Fatalf("address not rebased: %s", body)
	}
	if strings.Contains(body, ":3001") {
		t.Fatalf("old port still present: %s", body)
	}
	// 改写后版本与备份一致，重复执行不应再改动文件。
	again, err := s.RebasePort(ctx, 3001, 4321)
	if err != nil {
		t.Fatalf("second RebasePort error: %v", err)
	}
	if len(again.Updated) != 0 || len(again.Manual) != 0 {
		t.Fatalf("second rebase = %+v, want unchanged", again)
	}
	if got := readClientFile(t, s.paths.Codex); got != body {
		t.Fatal("second rebase modified the file")
	}
}

func TestRebasePortSkipsExternallyModifiedConfig(t *testing.T) {
	s := appliedCodexFixture(t)
	edited := readClientFile(t, s.paths.Codex) + "# edited outside OmniProxy\n"
	if err := os.WriteFile(s.paths.Codex, []byte(edited), 0600); err != nil {
		t.Fatal(err)
	}
	result, err := s.RebasePort(context.Background(), 3001, 4321)
	if err != nil {
		t.Fatalf("RebasePort error: %v", err)
	}
	if len(result.Manual) != 1 || result.Manual[0] != "codex" {
		t.Fatalf("manual = %v, want [codex]", result.Manual)
	}
	if got := readClientFile(t, s.paths.Codex); got != edited {
		t.Fatal("externally edited config was overwritten")
	}
}

func TestRebasePortIgnoresUnmanagedClient(t *testing.T) {
	s := fixture(t)
	if err := os.MkdirAll(filepath.Dir(s.paths.Codex), 0700); err != nil {
		t.Fatal(err)
	}
	original := "model_provider='openai'\nopenai_base_url='http://127.0.0.1:3001/codex/v1'\n"
	if err := os.WriteFile(s.paths.Codex, []byte(original), 0600); err != nil {
		t.Fatal(err)
	}
	result, err := s.RebasePort(context.Background(), 3001, 4321)
	if err != nil {
		t.Fatalf("RebasePort error: %v", err)
	}
	if len(result.Updated) != 0 || len(result.Manual) != 0 {
		t.Fatalf("result = %+v, want no managed client touched", result)
	}
	if got := readClientFile(t, s.paths.Codex); got != original {
		t.Fatal("unmanaged config was modified")
	}
}

func TestRebasePortRewritesHarnessBundle(t *testing.T) {
	s, _ := gatewayClientFixture(t)
	ctx := context.Background()
	client := "harness-web"
	path, err := s.path(client)
	if err != nil {
		t.Fatal(err)
	}
	writeGatewayFixture(t, path, "# preserve comment\n- id: llm-pi-ai\n  config:\n    providers:\n      existing:\n        displayName: fixture\n")
	before, err := s.snapshotClient(client, path)
	if err != nil {
		t.Fatal(err)
	}
	defer clear(before.body)
	input := Input{Client: client, Revision: before.revision}
	plan, err := s.PreviewPlan(ctx, input)
	if err != nil {
		t.Fatal(err)
	}
	input.CatalogRevision = plan.CatalogRevision
	if _, err := s.Apply(ctx, input); err != nil {
		t.Fatal(err)
	}
	if applied := readClientFile(t, path); !strings.Contains(applied, "127.0.0.1:3001") {
		t.Fatalf("fixture does not contain the old port: %s", applied)
	}
	result, err := s.RebasePort(ctx, 3001, 4321)
	if err != nil {
		t.Fatalf("RebasePort error: %v", err)
	}
	if len(result.Updated) != 1 || result.Updated[0] != client {
		t.Fatalf("updated = %v, want [%s]", result.Updated, client)
	}
	rebased := readClientFile(t, path)
	if !strings.Contains(rebased, "127.0.0.1:4321") || strings.Contains(rebased, "127.0.0.1:3001") {
		t.Fatalf("harness config not rebased: %s", rebased)
	}
	if !strings.Contains(rebased, "preserve comment") {
		t.Fatal("unrelated configuration lost")
	}
}

func TestReplaceLoopbackPortOnlyMatchesWholePort(t *testing.T) {
	body := []byte(`{"a":"http://127.0.0.1:3000/x","b":"http://127.0.0.1:30000/y","c":"http://localhost:3000/z","d":"https://api.invalid:3000/w"}`)
	next, changed := replaceLoopbackPort(body, 3000, 4321)
	if !changed {
		t.Fatal("expected a change")
	}
	text := string(next)
	for _, want := range []string{"127.0.0.1:4321/x", "127.0.0.1:30000/y", "localhost:4321/z", "api.invalid:3000/w"} {
		if !strings.Contains(text, want) {
			t.Fatalf("result %q missing %q", text, want)
		}
	}
}

func TestReplaceLoopbackPortWithoutMatch(t *testing.T) {
	body := []byte(`{"a":"https://api.invalid/v1"}`)
	if next, changed := replaceLoopbackPort(body, 3000, 4321); changed || next != nil {
		t.Fatalf("replaceLoopbackPort = %q, %t; want no change", next, changed)
	}
	if _, changed := replaceLoopbackPort(body, 3000, 3000); changed {
		t.Fatal("identical ports reported a change")
	}
}
