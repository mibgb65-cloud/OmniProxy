package activity

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"omniproxy/internal/core/proxy"
	"omniproxy/internal/infrastructure/database"
)

func TestCompressedContentMigrationPreservesLegacyRows(t *testing.T) {
	db, err := database.Open(database.Config{Path: filepath.Join(t.TempDir(), "content-migration.db")})
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	entries, err := os.ReadDir("../../../migrations")
	if err != nil {
		t.Fatal(err)
	}
	files := fstest.MapFS{}
	for _, entry := range entries {
		if entry.Name() >= "054_compressed_request_contents.sql" {
			continue
		}
		raw, err := os.ReadFile("../../../migrations/" + entry.Name())
		if err != nil {
			t.Fatal(err)
		}
		files[entry.Name()] = &fstest.MapFile{Data: raw}
	}
	ctx := context.Background()
	if err := database.Migrate(ctx, db, files); err != nil {
		t.Fatal(err)
	}
	legacy := `{"messages":[{"role":"user","content":"迁移前的正文"}]}`
	if _, err := db.ExecContext(ctx, `INSERT INTO request_contents(request_id,content,original_bytes,stored_bytes,truncated,captured_at) VALUES(?,?,?,?,1,?)`,
		"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", legacy, len(legacy), len(legacy), "2026-11-19T00:00:00.000000000Z"); err != nil {
		t.Fatal(err)
	}
	if err := database.Migrate(ctx, db, os.DirFS("../../../migrations")); err != nil {
		t.Fatal(err)
	}
	var data []byte
	var format, preview string
	var truncated int
	if err := db.QueryRowContext(ctx, `SELECT data,format,preview,truncated FROM request_contents WHERE request_id='aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa'`).
		Scan(&data, &format, &preview, &truncated); err != nil {
		t.Fatal(err)
	}
	if string(data) != legacy || format != "text" || preview != "" || truncated != 1 {
		t.Fatalf("legacy row changed: data=%q format=%q preview=%q truncated=%d", data, format, preview, truncated)
	}
	service := New(db)
	got, err := service.RequestContent(ctx, "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	if err != nil || !got.Available || got.Content != legacy || !got.Truncated {
		t.Fatalf("legacy content unreadable: %+v %v", got, err)
	}
	// 迁移后新写入的正文使用 gzip 并带预览。
	newContent := `{"messages":[{"role":"user","content":"` + strings.Repeat("新正文", 2000) + `"}]}`
	if err := service.RecordRequestContent(ctx, proxy.RequestContentEvent{
		RequestID: "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", Content: newContent, OriginalBytes: len(newContent),
	}); err != nil {
		t.Fatal(err)
	}
	var newFormat string
	var newPreview string
	if err := db.QueryRowContext(ctx, `SELECT format,preview FROM request_contents WHERE request_id='bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb'`).
		Scan(&newFormat, &newPreview); err != nil {
		t.Fatal(err)
	}
	if newFormat != "gzip" || !strings.Contains(newPreview, "新正文") {
		t.Fatalf("new row not compressed: format=%q preview=%q", newFormat, newPreview)
	}
}
