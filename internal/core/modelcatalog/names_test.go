package modelcatalog

import "testing"

// 上游不同 ID 共用同一个显示名时，菜单会出现多行完全相同的选项。
func TestUniqueNamesFallsBackToIDForDuplicates(t *testing.T) {
	entries := []Entry{
		{ID: "gemini-2.5-flash", Name: "Gemini 3.5 Flash Lite"},
		{ID: "gemini-2.5-flash-lite", Name: "Gemini 3.5 Flash Lite"},
		{ID: "gemini-2.5-pro", Name: "Gemini 2.5 Pro"},
	}
	unique := UniqueNames(entries)
	if unique[0].Name != "gemini-2.5-flash" || unique[1].Name != "gemini-2.5-flash-lite" {
		t.Fatalf("duplicate names not replaced by ids: %#v", unique[:2])
	}
	if unique[2].Name != "Gemini 2.5 Pro" {
		t.Fatalf("unique name changed: %q", unique[2].Name)
	}
	// 原切片不被修改，目录构建可以继续复用输入。
	if entries[0].Name != "Gemini 3.5 Flash Lite" {
		t.Fatal("input entries mutated")
	}
}

func TestUniqueNamesKeepsSingleEntries(t *testing.T) {
	entries := []Entry{{ID: "a", Name: "Alpha"}, {ID: "b", Name: "Beta"}}
	unique := UniqueNames(entries)
	if unique[0].Name != "Alpha" || unique[1].Name != "Beta" {
		t.Fatalf("names changed: %#v", unique)
	}
}
