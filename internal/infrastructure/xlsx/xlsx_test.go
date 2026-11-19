package xlsx

import (
	"archive/zip"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func readParts(t *testing.T, path string) map[string]string {
	t.Helper()
	reader, err := zip.OpenReader(path)
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	parts := map[string]string{}
	for _, file := range reader.File {
		handle, err := file.Open()
		if err != nil {
			t.Fatal(err)
		}
		data, err := io.ReadAll(handle)
		handle.Close()
		if err != nil {
			t.Fatal(err)
		}
		parts[file.Name] = string(data)
	}
	return parts
}

func TestWriteCreatesSingleSheetWorkbook(t *testing.T) {
	path := filepath.Join(t.TempDir(), "requests.xlsx")
	long := strings.Repeat("中", MaxCellChars+10)
	count, err := Write(path, "请求记录", []string{"时间", "数值"}, func(yield func([]Cell) error) error {
		if err := yield([]Cell{{Text: "第一行"}, {Number: "42"}}); err != nil {
			return err
		}
		return yield([]Cell{{Text: "control\x00char"}, {Text: long}})
	})
	if err != nil || count != 2 {
		t.Fatalf("write count=%d err=%v", count, err)
	}
	parts := readParts(t, path)
	for _, name := range []string{"[Content_Types].xml", "_rels/.rels", "xl/workbook.xml", "xl/_rels/workbook.xml.rels", "xl/worksheets/sheet1.xml"} {
		if parts[name] == "" {
			t.Fatalf("missing part %s", name)
		}
	}
	if !strings.Contains(parts["xl/workbook.xml"], "请求记录") {
		t.Fatal("sheet name missing")
	}
	sheet := parts["xl/worksheets/sheet1.xml"]
	if !strings.Contains(sheet, "<v>42</v>") || !strings.Contains(sheet, "第一行") || !strings.Contains(sheet, "controlchar") || strings.Contains(sheet, "\x00") {
		t.Fatalf("unexpected sheet: %s", sheet)
	}
	// 超过 Excel 上限的文本被截到 32767 个码元，不会写出超长单元格。
	if !strings.Contains(sheet, strings.Repeat("中", MaxCellChars)) || strings.Contains(sheet, strings.Repeat("中", MaxCellChars+1)) {
		t.Fatal("long cell was not truncated exactly at the Excel limit")
	}
}

func TestFitCellCountsUTF16Units(t *testing.T) {
	if text, truncated := FitCell("hello"); text != "hello" || truncated {
		t.Fatal("short text truncated")
	}
	text, truncated := FitCell(strings.Repeat("😀", MaxCellChars))
	if !truncated || len([]rune(text)) != MaxCellChars/2 {
		t.Fatalf("astral text truncated at %d runes (%v)", len([]rune(text)), truncated)
	}
}

func TestWriteRejectsBadInputAndKeepsExistingFile(t *testing.T) {
	dir := t.TempDir()
	if _, err := Write(filepath.Join(dir, "out.txt"), "Sheet", nil, func(func([]Cell) error) error { return nil }); err == nil {
		t.Fatal("unsafe extension accepted")
	}
	path := filepath.Join(dir, "existing.xlsx")
	if err := os.WriteFile(path, []byte("keep"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Write(path, "Sheet", []string{"A"}, func(yield func([]Cell) error) error {
		return yield([]Cell{{Number: "not-a-number"}})
	}); err == nil {
		t.Fatal("invalid number accepted")
	}
	data, err := os.ReadFile(path)
	if err != nil || string(data) != "keep" {
		t.Fatalf("existing file changed: %q %v", data, err)
	}
	if matches, _ := filepath.Glob(filepath.Join(dir, ".omniproxy-xlsx-*.tmp")); len(matches) != 0 {
		t.Fatal("temporary files leaked")
	}
}

func TestSanitizeSheetNameAndColumnNames(t *testing.T) {
	if got := sanitizeSheetName("a/b:c*d?e[f]g'h"); strings.ContainsAny(got, `[]:*?/`) {
		t.Fatalf("invalid sheet name: %q", got)
	}
	if got := sanitizeSheetName(strings.Repeat("x", 40)); len([]rune(got)) != 31 {
		t.Fatalf("sheet name length %d", len([]rune(got)))
	}
	if columnName(0) != "A" || columnName(25) != "Z" || columnName(26) != "AA" || columnName(701) != "ZZ" {
		t.Fatal("column names wrong")
	}
}
