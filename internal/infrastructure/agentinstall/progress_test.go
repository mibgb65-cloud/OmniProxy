package agentinstall

import (
	"strings"
	"testing"
)

func TestProgressParsesFramesAndNeverReturnsRawOutput(t *testing.T) {
	var updates []Progress
	writer := &progressWriter{report: func(value Progress) { updates = append(updates, value) }}
	for _, chunk := range []string{"Downloading https://private.invalid/token\r", "\x1b[3", "2m████ 25.0 MB / 100 MB\x1b[0m\r", "Starting package install...\r", "▒▒ 42%\n"} {
		if _, err := writer.Write([]byte(chunk)); err != nil {
			t.Fatal(err)
		}
	}
	if len(updates) != 4 || updates[0].Phase != "downloading" || updates[1].Percent == nil || *updates[1].Percent != 25 || updates[1].DownloadedBytes != 25<<20 || updates[2].Percent != nil || updates[3].Phase != "installing" || *updates[3].Percent != 42 {
		t.Fatalf("unexpected structured progress: %+v", updates)
	}
}

func TestProgressRejectsInvalidValuesAndBoundsFrameMemory(t *testing.T) {
	for _, input := range []string{"200%", "-1%", "-1 MB / 4 MB", "token=secret 20 MB / 40 MB", "1000 MB / 1 MB", "1 MB / 0 MB", "99999999999999999 TB / 99999999999999999 TB", "token=secret 20%", "NaN%"} {
		if _, ok := parseProgress(input); ok {
			t.Fatalf("invalid frame accepted: %q", input)
		}
	}
	called := false
	writer := &progressWriter{report: func(Progress) { called = true }}
	if _, err := writer.Write([]byte(strings.Repeat("x", 128<<10) + " 20 MB / 40 MB\r")); err != nil {
		t.Fatal(err)
	}
	if called || len(writer.line) > 8192 {
		t.Fatal("oversized frame retained or parsed")
	}
	if _, err := writer.Write([]byte("10 MB / 20 MB\r")); err != nil {
		t.Fatal(err)
	}
	if !called {
		t.Fatal("parser did not recover after oversized frame")
	}
}

func TestStructuredTerminalProgressWorksAcrossChunkBoundaries(t *testing.T) {
	var updates []Progress
	writer := &progressWriter{report: func(value Progress) { updates = append(updates, value) }}
	for _, chunk := range []string{"Downloading package\r", "\x1b]9;4;1;", "37\x1b", "\\", "\x1b]9;4;3\a", "\x1b]9;4;1;999\a", "\x1b]8;;https://private.invalid/secret\a"} {
		if _, err := writer.Write([]byte(chunk)); err != nil {
			t.Fatal(err)
		}
	}
	if len(updates) != 3 || updates[1].Percent == nil || *updates[1].Percent != 37 || updates[1].Phase != "downloading" || updates[2].Percent != nil {
		t.Fatal("structured terminal progress not handled")
	}
}
