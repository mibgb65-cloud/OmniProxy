package agentinstall

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCodingDesktopDetectionChecksProductAndPublisher(t *testing.T) {
	for _, item := range []struct{ id, name, publisher string }{
		{"harness-desktop", "DeepSeek Harness", "DeepSeek"},
		{"kilocode-desktop", "Kilo Desktop", "Kilo"},
		{"qwen-code-desktop", "Qwen Code", "Alibaba"},
		{"mimo-desktop-cn", "Xiaomi MiMo", "Xiaomi"},
		{"mimo-desktop-global", "Xiaomi MiMo AI", "Xiaomi"},
	} {
		if !codingDesktopMatches(item.id, item.name, item.publisher) || codingDesktopMatches(item.id, item.name, "Unknown") || codingDesktopMatches(item.id, item.name+" Plugin", item.publisher) {
			t.Fatal("desktop detection accepted a different product or publisher", item.id)
		}
	}
	if codingDesktopMatches("../kilocode-desktop", "Kilo Desktop", "Kilo") {
		t.Fatal("unknown product matched")
	}
}

func TestMiMoDesktopRegionsAndNSISMetadata(t *testing.T) {
	root := t.TempDir()
	for _, id := range []string{"mimo-desktop-cn", "mimo-desktop-global"} {
		name := nsisDesktopName(id)
		path := filepath.Join(root, name+".exe")
		if err := os.WriteFile(path, []byte("fixture"), 0600); err != nil {
			t.Fatal(err)
		}
		location, ok := codingDesktopLocation(id, name+" 26.922.222056", "", "26.922.222056", "", path+",0")
		if !ok || location != root {
			t.Fatal("valid NSIS metadata not detected", id)
		}
		for _, bad := range []string{path + " --command bad", path + ",1", filepath.Join(root, "unknown.exe"), "https://example.invalid/client.exe"} {
			if _, ok := codingDesktopLocation(id, name, "", "26.922.222056", "", bad); ok {
				t.Fatal("unsafe icon target accepted", id)
			}
		}
		other := "mimo-desktop-global"
		if id == other {
			other = "mimo-desktop-cn"
		}
		if _, ok := codingDesktopLocation(other, name, "", "26.922.222056", "", path+",0"); ok {
			t.Fatal("wrong region detected", id)
		}
	}
}
