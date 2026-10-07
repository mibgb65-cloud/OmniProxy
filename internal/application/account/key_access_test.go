package account

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"
)

func TestKeyPreviewUsesRealCredentialAndMasksShortKeys(t *testing.T) {
	for _, test := range []struct{ key, preview string }{
		{"sk-test-only-credential-1234", "sk-t••••••••1234"},
		{"一二三四五六七八九十甲乙丙丁戊己庚辛壬癸", "一二三四••••••••庚辛壬癸"},
		{"short", keyMask},
		{"123456789012", keyMask},
		{"1234567890123456", keyMask},
	} {
		t.Run(test.preview, func(t *testing.T) {
			service, _ := testService(t)
			item, err := service.Create(context.Background(), CreateInput{ProviderID: "builtin_openai", Name: "自定义名称", APIKey: test.key})
			if err != nil {
				t.Fatal(err)
			}
			preview, err := service.KeyPreview(context.Background(), item.ID)
			if err != nil || preview != test.preview || strings.Contains(preview, test.key) {
				t.Fatal("key preview was incorrect or revealed the full key")
			}
		})
	}
}

func TestUseAPIKeyClearsCallbackBufferOnSuccessAndFailure(t *testing.T) {
	service, _ := testService(t)
	item, err := service.Create(context.Background(), CreateInput{ProviderID: "builtin_openai", Name: "test", APIKey: "test-only-protected-key"})
	if err != nil {
		t.Fatal(err)
	}
	for _, failure := range []error{nil, errors.New("clipboard failed")} {
		var received []byte
		err := service.UseAPIKey(context.Background(), item.ID, func(key []byte) error {
			if string(key) != "test-only-protected-key" {
				t.Fatal("wrong key delivered to consumer")
			}
			received = key
			return failure
		})
		if len(received) == 0 || !errors.Is(err, failure) || !slices.Equal(received, make([]byte, len(received))) {
			t.Fatal("key use did not propagate failure or clear its buffer")
		}
	}
}

func TestKeyAccessRejectsInvalidIDsAndMissingCredentials(t *testing.T) {
	service, store := testService(t)
	for _, id := range []string{"", "invalid\x00", "missing"} {
		if _, err := service.KeyPreview(context.Background(), id); err == nil {
			t.Fatal("invalid key access succeeded")
		}
	}
	item, err := service.Create(context.Background(), CreateInput{ProviderID: "builtin_openai", Name: "test", APIKey: "test-only-protected-key"})
	if err != nil {
		t.Fatal(err)
	}
	store.deleted = true
	called := false
	if err := service.UseAPIKey(context.Background(), item.ID, func([]byte) error { called = true; return nil }); err == nil || called {
		t.Fatal("unreadable credential reached the consumer")
	}
}
