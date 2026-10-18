package database

import (
	"context"
	"crypto/sha256"
	"os"
	"path/filepath"
	"testing"
)

func TestResolveLegacyPathPreservesExistingDatabase(t *testing.T) {
	for _, test := range []struct {
		name, schema string
		isolated     bool
		wantError    bool
	}{
		{"legacy", `CREATE TABLE request_records(client,client_model,provider_model,success); INSERT INTO request_records VALUES('old','model','model',1);`, true, false},
		{"partial-beta1", `CREATE TABLE request_records(client,client_model,provider_model,success); CREATE TABLE accounts(id,credential_ref);`, true, false},
		{"v2", `CREATE TABLE request_records(day,protocol,input_model,provider_id); CREATE TABLE accounts(id,credential_ref); INSERT INTO accounts VALUES('saved','ref');`, false, false},
		{"pre-accounting-v2", `CREATE TABLE accounts(id,credential_ref); INSERT INTO accounts VALUES('saved','ref');`, false, false},
		{"mixed-accounts", `CREATE TABLE request_records(client,client_model,provider_model,success); CREATE TABLE accounts(id); INSERT INTO accounts VALUES('saved');`, false, true},
		{"mixed-migrations", `CREATE TABLE request_records(client,client_model,provider_model,success); CREATE TABLE schema_version(version); INSERT INTO schema_version VALUES(8);`, false, true},
		{"unknown-schema", `CREATE TABLE request_records(unknown);`, false, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			// 路径中的 URI 特殊字符不能改变 mode=ro 或实际读取位置。
			path := filepath.Join(t.TempDir(), "data # space", "omniproxy.db")
			db, err := Open(Config{Path: path})
			if err != nil {
				t.Fatal(err)
			}
			if _, err := db.Exec(test.schema); err != nil {
				t.Fatal(err)
			}
			if err := db.Close(); err != nil {
				t.Fatal(err)
			}
			before, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			got, err := ResolveLegacyPath(context.Background(), path)
			if (err != nil) != test.wantError {
				t.Fatalf("ResolveLegacyPath error = %v", err)
			}
			if err == nil {
				want := path
				if test.isolated {
					want = filepath.Join(filepath.Dir(path), "v2", "omniproxy.db")
				}
				if got != want {
					t.Fatalf("path = %q, want %q", got, want)
				}
			}
			after, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if sha256.Sum256(before) != sha256.Sum256(after) {
				t.Fatal("schema inspection modified the source database")
			}
		})
	}
}

func TestResolveLegacyPathRejectsInvalidTargetAndSource(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "omniproxy.db")
	if got, err := ResolveLegacyPath(context.Background(), path); err != nil || got != path {
		t.Fatalf("fresh path = %q, %v", got, err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("read-only inspection created a database")
	}
	for _, target := range []string{path, filepath.Join(root, "v2", "omniproxy.db")} {
		db, err := Open(Config{Path: target})
		if err != nil {
			t.Fatal(err)
		}
		_, err = db.Exec(`CREATE TABLE request_records(client,client_model,provider_model,success)`)
		_ = db.Close()
		if err != nil {
			t.Fatal(err)
		}
	}
	if _, err := ResolveLegacyPath(context.Background(), path); err == nil {
		t.Fatal("must not recursively isolate another legacy target")
	}
	if err := os.WriteFile(path, []byte("not a database"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := ResolveLegacyPath(context.Background(), path); err == nil {
		t.Fatal("corrupt source must not silently select a new database")
	}
}
