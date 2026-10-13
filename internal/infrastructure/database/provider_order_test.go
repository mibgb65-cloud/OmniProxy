package database

import (
	"context"
	"os"
	"testing"
	"testing/fstest"
)

func TestProviderOrderMigrationPreservesExistingPreference(t *testing.T) {
	db := openTestDB(t)
	original, err := os.ReadFile("../../../migrations/012_model_provider_preferences.sql")
	if err != nil {
		t.Fatal(err)
	}
	upgrade, err := os.ReadFile("../../../migrations/015_model_provider_order.sql")
	if err != nil {
		t.Fatal(err)
	}
	migrations := fstest.MapFS{"012_model_provider_preferences.sql": {Data: original}}
	ctx := context.Background()
	if err := Migrate(ctx, db, migrations); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO model_provider_preferences VALUES('shared-model','saved-provider','original-time')`); err != nil {
		t.Fatal(err)
	}
	migrations["015_model_provider_order.sql"] = &fstest.MapFile{Data: upgrade}
	for range 2 {
		if err := Migrate(ctx, db, migrations); err != nil {
			t.Fatal(err)
		}
		var providerID, timestamp, order string
		if err := db.QueryRow(`SELECT provider_id,updated_at,provider_order FROM model_provider_preferences WHERE model='shared-model'`).Scan(&providerID, &timestamp, &order); err != nil {
			t.Fatal(err)
		}
		if providerID != "saved-provider" || timestamp != "original-time" || order != `["saved-provider"]` {
			t.Fatal("existing preference changed during migration")
		}
	}
}
