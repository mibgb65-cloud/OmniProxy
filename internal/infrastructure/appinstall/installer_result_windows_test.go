//go:build windows

package appinstall

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unicode/utf16"
)

func TestInstallerDirectoryResultPreservesUnicodeWithoutCodePage(t *testing.T) {
	executable := filepath.Join(t.TempDir(), "installer-helper.exe")
	want := filepath.Join(t.TempDir(), "旧 数据目录")
	if err := writeInstallerDirectoryResult(executable, want); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(filepath.Dir(executable), installerDirectoryResultName)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	units := make([]uint16, len(data)/2)
	for index := range units {
		units[index] = binary.LittleEndian.Uint16(data[index*2:])
	}
	if got := string(utf16.Decode(units)); got != want+"\r\n" {
		t.Fatalf("directory result = %q", got)
	}
	if err := writeInstallerDirectoryResult(executable, "replacement"); err == nil {
		t.Fatal("existing result was overwritten")
	}
	after, err := os.ReadFile(path)
	if err != nil || string(after) != string(data) {
		t.Fatalf("existing result was modified: %v", err)
	}
}

func TestInstallerDirectoryResultRejectsInvalidInput(t *testing.T) {
	for _, directory := range []string{"", "line\r\ninjection", "nul\x00byte", strings.Repeat("x", 1022)} {
		if err := writeInstallerDirectoryResult(filepath.Join(t.TempDir(), "helper.exe"), directory); err == nil {
			t.Fatal("invalid directory result was written")
		}
	}
	if err := writeInstallerDirectoryResult("relative.exe", "data"); err == nil {
		t.Fatal("relative helper path was accepted")
	}
}
