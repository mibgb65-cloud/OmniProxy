package wails

import "testing"

func TestGoogleConsoleOnlyOpensFixedOfficialURLs(t *testing.T) {
	s := &AppService{}
	if s.OpenGoogleConsole("usage") == nil {
		t.Fatal("missing browser accepted")
	}
	opened := ""
	s.openAgentURL = func(address string) error { opened = address; return nil }
	if s.OpenGoogleConsole("https://example.com") == nil || opened != "" {
		t.Fatal("arbitrary URL opened")
	}
	if err := s.OpenGoogleConsole("usage"); err != nil || opened != "https://aistudio.google.com/usage" {
		t.Fatal("official usage not opened")
	}
	if err := s.OpenGoogleConsole("billing"); err != nil || opened != "https://aistudio.google.com/billing" {
		t.Fatal("official billing not opened")
	}
}
