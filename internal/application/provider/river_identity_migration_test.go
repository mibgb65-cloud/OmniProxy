package provider

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"

	"omniproxy/internal/infrastructure/database"
)

func TestRiverIdentityMigrationPreservesLogicalPreferenceAndMovesLegacyRows(t *testing.T) {
	ctx := context.Background()
	db, err := database.Open(database.Config{Path: filepath.Join(t.TempDir(), "upgrade.db")})
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	files, err := os.ReadDir("../../../migrations")
	if err != nil {
		t.Fatal(err)
	}
	before := fstest.MapFS{}
	for _, file := range files {
		if file.Name() == "030_river_logical_models.sql" {
			continue
		}
		body, err := os.ReadFile("../../../migrations/" + file.Name())
		if err != nil {
			t.Fatal(err)
		}
		before[file.Name()] = &fstest.MapFile{Data: body}
	}
	if err := database.Migrate(ctx, db, before); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO model_provider_preferences(model,provider_id,updated_at,provider_order) VALUES
('deepseek-flash','builtin_deepseek','old','["builtin_deepseek"]'),
('deepseek-ai/DeepSeek-V4.1-Flash','builtin_river','old','["builtin_river"]'),
('zai-org/GLM-5.3-Flash','builtin_river','old','["builtin_river"]'),
('nvidia/NVIDIA-Nemotron-3.5-Super-VL-120B-A12B-NVFP4','builtin_river','old','["builtin_river"]')`); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		if err := database.Migrate(ctx, db, os.DirFS("../../../migrations")); err != nil {
			t.Fatal(err)
		}
		var count int
		if err := db.QueryRow(`SELECT count(*) FROM model_provider_preferences`).Scan(&count); err != nil || count != 3 {
			t.Fatal("legacy preference duplicates remain", count, err)
		}
		for model, expected := range map[string]string{"deepseek-flash": "builtin_deepseek", "GLM-5.3-Flash": "builtin_river", "NVIDIA-Nemotron-3.5": "builtin_river"} {
			var actual string
			if err := db.QueryRow(`SELECT provider_id FROM model_provider_preferences WHERE model=?`, model).Scan(&actual); err != nil || actual != expected {
				t.Fatal("saved priority changed", model, actual, err)
			}
		}
	}
}
