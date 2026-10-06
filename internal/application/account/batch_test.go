package account

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
)

type batchStore struct {
	values     map[string][]byte
	puts       int
	failPut    int
	failDelete bool
	deleted    []string
}

func (s *batchStore) Put(key []byte) (string, error) {
	s.puts++
	if s.puts == s.failPut {
		return "", errors.New("protected storage unavailable")
	}
	ref := fmt.Sprintf("test-ref-%d", s.puts)
	s.values[ref] = append([]byte(nil), key...)
	return ref, nil
}

func (s *batchStore) Delete(ref string) error {
	s.deleted = append(s.deleted, ref)
	if s.failDelete {
		return errors.New("cleanup failed")
	}
	delete(s.values, ref)
	return nil
}

func TestCreateAutomaticallyNamesUnnamedAccounts(t *testing.T) {
	for _, test := range []struct{ key, name string }{
		{"  12345678-test-only  ", "12345678"},
		{"一二三四五六七八九十-test", "一二三四五六七八"},
		{"short", "short"},
	} {
		t.Run(test.name, func(t *testing.T) {
			service, _ := testService(t)
			item, err := service.Create(context.Background(), CreateInput{ProviderID: "builtin_openai", Name: "  ", APIKey: test.key})
			if err != nil || item.Name != test.name {
				t.Fatalf("automatic name = %q, error = %v", item.Name, err)
			}
		})
	}
}

func TestBatchCreatePreservesKeysOnlyInProtectedStore(t *testing.T) {
	service, _ := testService(t)
	store := &batchStore{values: make(map[string][]byte)}
	service.store = store
	keys := []string{"12345678-first-test", "abcdefgh-second-test"}
	items, err := service.CreateBatch(context.Background(), BatchCreateInput{ProviderID: "builtin_deepseek", APIKeys: keys})
	if err != nil || len(items) != 2 {
		t.Fatalf("batch count = %d, error = %v", len(items), err)
	}
	if items[0].Name != "12345678" || items[1].Name != "abcdefgh" || items[0].ID == items[1].ID {
		t.Fatal("batch names or unique IDs are incorrect")
	}
	metadata, err := json.Marshal(items)
	if err != nil {
		t.Fatal(err)
	}
	for i, item := range items {
		var ref string
		if err := service.db.QueryRow(`SELECT credential_ref FROM accounts WHERE id = ?`, item.ID).Scan(&ref); err != nil {
			t.Fatal(err)
		}
		if string(store.values[ref]) != keys[i] || strings.Contains(string(metadata), keys[i]) {
			t.Fatal("credential protection or metadata contract failed")
		}
	}
}

func TestBatchValidationWritesNothing(t *testing.T) {
	for _, test := range []struct {
		name  string
		input BatchCreateInput
	}{
		{"empty", BatchCreateInput{ProviderID: "builtin_openai"}},
		{"too many", BatchCreateInput{ProviderID: "builtin_openai", APIKeys: make([]string, maxBatchKeys+1)}},
		{"missing provider", BatchCreateInput{ProviderID: "missing", APIKeys: []string{"valid-test-key"}}},
		{"invalid provider", BatchCreateInput{ProviderID: "invalid\x00", APIKeys: []string{"valid-test-key"}}},
		{"empty key", BatchCreateInput{ProviderID: "builtin_openai", APIKeys: []string{"valid-test-key", "  "}}},
		{"control", BatchCreateInput{ProviderID: "builtin_openai", APIKeys: []string{"valid-test-key", "bad\x00key"}}},
		{"invalid UTF8", BatchCreateInput{ProviderID: "builtin_openai", APIKeys: []string{"valid-test-key", "\xff"}}},
		{"too long", BatchCreateInput{ProviderID: "builtin_openai", APIKeys: []string{"valid-test-key", strings.Repeat("k", maxKeyBytes+1)}}},
	} {
		t.Run(test.name, func(t *testing.T) {
			service, _ := testService(t)
			store := &batchStore{values: make(map[string][]byte)}
			service.store = store
			_, err := service.CreateBatch(context.Background(), test.input)
			if err == nil || store.puts != 0 {
				t.Fatal("invalid batch reached credential storage")
			}
		})
	}
}

func TestBatchFailureRollsBackAndCleansAllWrittenCredentials(t *testing.T) {
	for _, failure := range []string{"credential write", "database insert", "cleanup"} {
		t.Run(failure, func(t *testing.T) {
			service, _ := testService(t)
			store := &batchStore{values: make(map[string][]byte)}
			service.store = store
			if failure == "credential write" {
				store.failPut = 2
			} else {
				_, err := service.db.Exec(`CREATE TRIGGER reject_second BEFORE INSERT ON accounts WHEN (SELECT COUNT(*) FROM accounts) = 1 BEGIN SELECT RAISE(FAIL, 'blocked'); END`)
				if err != nil {
					t.Fatal(err)
				}
				store.failDelete = failure == "cleanup"
			}
			items, err := service.CreateBatch(context.Background(), BatchCreateInput{ProviderID: "builtin_openai", APIKeys: []string{"first-test-key", "second-test-key"}})
			if err == nil || items != nil {
				t.Fatal("failed batch reported success")
			}
			listed, listErr := service.List(context.Background())
			if listErr != nil || len(listed) != 0 {
				t.Fatal("batch metadata was not rolled back")
			}
			expectedCleanups := 2
			if failure == "credential write" {
				expectedCleanups = 1
			}
			if len(store.deleted) != expectedCleanups || (!store.failDelete && len(store.values) != 0) {
				t.Fatal("not all written credentials were cleaned")
			}
			if store.failDelete && !strings.Contains(err.Error(), "cleanup failed") {
				t.Fatal("credential cleanup errors were hidden")
			}
		})
	}
}

func TestBatchAcceptsMaximumCountAndRejectsCancelledRequests(t *testing.T) {
	service, _ := testService(t)
	store := &batchStore{values: make(map[string][]byte)}
	service.store = store
	keys := make([]string, maxBatchKeys)
	for i := range keys {
		keys[i] = fmt.Sprintf("test-key-%03d", i)
	}
	items, err := service.CreateBatch(context.Background(), BatchCreateInput{ProviderID: "builtin_openai", APIKeys: keys})
	if err != nil || len(items) != maxBatchKeys || len(store.values) != maxBatchKeys {
		t.Fatal("maximum-size batch failed")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err = service.CreateBatch(ctx, BatchCreateInput{ProviderID: "builtin_openai", APIKeys: []string{"cancelled-test-key"}})
	if !errors.Is(err, context.Canceled) || store.puts != maxBatchKeys {
		t.Fatal("cancelled batch reached credential storage")
	}
}
