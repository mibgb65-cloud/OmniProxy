package wails

import (
	"errors"
	"strings"
	"testing"
)

func TestOpenProjectResourceFixedTargets(t *testing.T) {
	for resource, address := range map[string]string{
		"website":    "https://omniproxy.aicnos.com/",
		"repository": "https://github.com/mibgb65-cloud/OmniProxy",
		"disclaimer": "https://omniproxy.aicnos.com/legal/disclaimer/",
		"terms":      "https://omniproxy.aicnos.com/legal/terms/",
		"privacy":    "https://omniproxy.aicnos.com/legal/privacy/",
	} {
		t.Run(resource, func(t *testing.T) {
			var opened []string
			s := &AppService{openAgentURL: func(url string) error {
				opened = append(opened, url)
				return nil
			}}
			if err := s.OpenProjectResource(resource); err != nil {
				t.Fatal(err)
			}
			if len(opened) != 1 || opened[0] != address {
				t.Fatalf("unexpected browser calls: %v", opened)
			}
		})
	}
}

func TestOpenProjectResourceRejectsUntrustedInput(t *testing.T) {
	calls := 0
	s := &AppService{openAgentURL: func(string) error { calls++; return nil }}
	for _, input := range []string{
		"", "Website", " website", "website\n", "privacy?redirect=https://example.com",
		"https://example.com", "file:///C:/Windows", "javascript:alert(1)",
		"claude://code/new", "../privacy", "privacy\x00", strings.Repeat("x", 4096),
	} {
		if err := s.OpenProjectResource(input); !errors.Is(err, errProjectResourceInvalid) {
			t.Fatalf("invalid resource accepted: %v", err)
		}
	}
	if calls != 0 {
		t.Fatalf("browser called for invalid resource: %d", calls)
	}
}

func TestOpenProjectResourceBrowserFailuresAreSafe(t *testing.T) {
	s := &AppService{}
	if err := s.OpenProjectResource("website"); !errors.Is(err, errProjectResourceUnavailable) {
		t.Fatalf("missing browser: %v", err)
	}
	s.openAgentURL = func(string) error { return errors.New("private browser launch details") }
	if err := s.OpenProjectResource("privacy"); err != errProjectResourceUnavailable {
		t.Fatalf("browser failure was not sanitized: %v", err)
	}
}
