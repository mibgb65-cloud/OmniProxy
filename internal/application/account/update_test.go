package account

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

type updateInterleaveStore struct {
	*batchStore
	afterPut func()
}

func (s *updateInterleaveStore) Put(key []byte) (string, error) {
	ref, err := s.batchStore.Put(key)
	if err == nil {
		s.afterPut()
	}
	return ref, err
}

func testUpdateAccount(t *testing.T) (*Service, *batchStore, Item, string) {
	t.Helper()
	service, _ := testService(t)
	store := &batchStore{values: make(map[string][]byte)}
	service.store = store
	item, err := service.Create(context.Background(), CreateInput{ProviderID: "builtin_deepseek", Name: "original", APIKey: "old-test-protected-key"})
	if err != nil {
		t.Fatal(err)
	}
	_, ref, err := service.get(context.Background(), item.ID)
	if err != nil {
		t.Fatal(err)
	}
	return service, store, item, ref
}

func TestUpdateNamePreservesKeyAndAccountIdentity(t *testing.T) {
	service, store, item, ref := testUpdateAccount(t)
	result, err := service.Update(context.Background(), UpdateInput{ID: item.ID, Name: " renamed "})
	if err != nil {
		t.Fatal(err)
	}
	if result.Account.Name != "renamed" || result.Account.ID != item.ID || result.Account.ProviderID != item.ProviderID || result.Account.CreatedAt != item.CreatedAt || !result.Account.Enabled {
		t.Fatal("name update changed account identity")
	}
	if store.puts != 1 || len(store.deleted) != 0 || string(store.values[ref]) != "old-test-protected-key" {
		t.Fatal("name-only update touched the protected key")
	}
}

func TestUpdateReplacesKeyAndSupportsAutomaticNames(t *testing.T) {
	service, store, item, oldRef := testUpdateAccount(t)
	result, err := service.Update(context.Background(), UpdateInput{ID: item.ID, APIKey: "new-test-protected-key"})
	if err != nil || result.CleanupWarning || result.Account.Name != "new-test" {
		t.Fatal("key replacement or automatic name failed")
	}
	_, newRef, err := service.get(context.Background(), item.ID)
	if err != nil || newRef == oldRef || string(store.values[newRef]) != "new-test-protected-key" {
		t.Fatal("new key was not referenced")
	}
	if _, found := store.values[oldRef]; found {
		t.Fatal("old credential was not cleaned")
	}
	result, err = service.Update(context.Background(), UpdateInput{ID: item.ID, Name: " "})
	if err != nil || result.Account.Name != "new-test" || store.puts != 2 {
		t.Fatal("existing key automatic name failed")
	}
}

func TestFailedUpdatePreservesOriginalKeyAndMetadata(t *testing.T) {
	for _, failure := range []string{"validation", "key write", "database", "conflict"} {
		t.Run(failure, func(t *testing.T) {
			service, store, item, oldRef := testUpdateAccount(t)
			input := UpdateInput{ID: item.ID, Name: "changed", APIKey: "new-test-protected-key"}
			if failure == "validation" {
				input.APIKey = "bad\x00key"
			}
			if failure == "key write" {
				store.failPut = 2
			}
			if failure == "database" {
				if _, err := service.db.Exec(`CREATE TRIGGER reject_update BEFORE UPDATE ON accounts BEGIN SELECT RAISE(FAIL, 'blocked'); END`); err != nil {
					t.Fatal(err)
				}
			}
			if failure == "conflict" {
				if _, err := service.db.Exec(`CREATE TRIGGER ignore_update BEFORE UPDATE ON accounts BEGIN SELECT RAISE(IGNORE); END`); err != nil {
					t.Fatal(err)
				}
			}
			_, err := service.Update(context.Background(), input)
			if err == nil || (failure == "conflict" && !errors.Is(err, ErrConflict)) {
				t.Fatal("failed update reported success")
			}
			current, ref, err := service.get(context.Background(), item.ID)
			if err != nil || current != item || ref != oldRef || string(store.values[oldRef]) != "old-test-protected-key" || len(store.values) != 1 {
				t.Fatal("failed update lost original account data or leaked the replacement")
			}
		})
	}
}

func TestUpdateReportsCleanupWarningAfterSuccessfulCommit(t *testing.T) {
	service, store, item, oldRef := testUpdateAccount(t)
	store.failDelete = true
	result, err := service.Update(context.Background(), UpdateInput{ID: item.ID, Name: "changed", APIKey: "new-test-protected-key"})
	if err != nil || !result.CleanupWarning || result.Account.Name != "changed" {
		t.Fatal("committed update cleanup warning was lost")
	}
	_, newRef, err := service.get(context.Background(), item.ID)
	if err != nil || newRef == oldRef || len(store.values) != 2 {
		t.Fatal("cleanup failure corrupted committed update")
	}
}

func TestUpdateValidatesNamesIDsAndKeyLength(t *testing.T) {
	service, store, item, _ := testUpdateAccount(t)
	for _, input := range []UpdateInput{
		{ID: ""}, {ID: "missing", Name: "valid"}, {ID: item.ID, Name: "bad\x00name"},
		{ID: item.ID, Name: strings.Repeat("n", 81)}, {ID: item.ID, Name: "valid", APIKey: strings.Repeat("k", maxKeyBytes+1)},
	} {
		if _, err := service.Update(context.Background(), input); err == nil {
			t.Fatal("invalid update accepted")
		}
	}
	if store.puts != 1 {
		t.Fatal("invalid update reached protected storage")
	}
}

func TestConcurrentReplacementKeepsOtherCommittedCredential(t *testing.T) {
	service, store, item, oldRef := testUpdateAccount(t)
	otherRef := ""
	service.store = &updateInterleaveStore{batchStore: store, afterPut: func() {
		var err error
		otherRef, err = store.Put([]byte("other-test-protected-key"))
		if err != nil {
			t.Fatal(err)
		}
		if _, err := service.db.Exec(`UPDATE accounts SET name = ?, credential_ref = ?, updated_at = ? WHERE id = ?`, "other edit", otherRef, "concurrent-change", item.ID); err != nil {
			t.Fatal(err)
		}
		if err := store.Delete(oldRef); err != nil {
			t.Fatal(err)
		}
	}}
	_, err := service.Update(context.Background(), UpdateInput{ID: item.ID, Name: "stale edit", APIKey: "new-test-protected-key"})
	if !errors.Is(err, ErrConflict) {
		t.Fatal("concurrent replacement was overwritten")
	}
	current, ref, err := service.get(context.Background(), item.ID)
	if err != nil || ref != otherRef || current.Name != "other edit" || string(store.values[otherRef]) != "other-test-protected-key" || len(store.values) != 1 {
		t.Fatal("update removed the other committed credential")
	}
}

func TestUpdateResultDoesNotReturnCompleteKey(t *testing.T) {
	service, _, item, _ := testUpdateAccount(t)
	result, err := service.Update(context.Background(), UpdateInput{ID: item.ID, Name: "named", APIKey: "new-test-protected-key"})
	if err != nil {
		t.Fatal(err)
	}
	data, err := json.Marshal(result)
	if err != nil || strings.Contains(string(data), "new-test-protected-key") {
		t.Fatal("update result leaked the complete key")
	}
}
