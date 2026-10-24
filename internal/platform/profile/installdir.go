package profile

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf16"
)

// installDataDirFileName is the file the Windows installer writes next to the
// executable when the user picks a data location during setup. It holds a
// single absolute path; an empty or missing file means "use the profile
// default".
const installDataDirFileName = "data-dir.txt"

// executablePath is indirected so tests can simulate an installation layout
// without replacing the running test binary.
var executablePath = os.Executable

// installDataDir resolves the data directory recorded by the installer, if
// any. It never returns a relative path: an unreadable or malformed choice
// falls back to the caller's default.
func installDataDir() string {
	exe, err := executablePath()
	if err != nil || exe == "" {
		return ""
	}
	return readInstallDataDir(filepath.Dir(exe))
}

// readInstallDataDir reads the installer choice from an installation
// directory.
func readInstallDataDir(dir string) string {
	if dir == "" {
		return ""
	}
	data, err := os.ReadFile(filepath.Join(dir, installDataDirFileName))
	if err != nil {
		return ""
	}
	return parseInstallDataDir(decodeInstallText(data))
}

// decodeInstallText converts the installer file to text. NSIS built with
// `Unicode true` writes UTF-16LE with a BOM; a hand-edited file is normally
// UTF-8.
func decodeInstallText(data []byte) string {
	switch {
	case len(data) >= 2 && data[0] == 0xFF && data[1] == 0xFE:
		return decodeUTF16(data[2:], binary.LittleEndian)
	case len(data) >= 2 && data[0] == 0xFE && data[1] == 0xFF:
		return decodeUTF16(data[2:], binary.BigEndian)
	}
	return strings.TrimPrefix(string(data), "\ufeff")
}

func decodeUTF16(data []byte, order binary.ByteOrder) string {
	units := make([]uint16, 0, len(data)/2)
	for i := 0; i+1 < len(data); i += 2 {
		units = append(units, order.Uint16(data[i:]))
	}
	return string(utf16.Decode(units))
}

// parseInstallDataDir normalises the installer file content. The installer
// writes one path per file; only the first line is used and surrounding quotes
// are tolerated so a hand-edited file still works.
func parseInstallDataDir(raw string) string {
	value := strings.TrimPrefix(raw, "\ufeff")
	if i := strings.IndexAny(value, "\r\n"); i >= 0 {
		value = value[:i]
	}
	value = strings.TrimSpace(value)
	value = strings.TrimSpace(strings.Trim(value, `"`))
	if value == "" || !filepath.IsAbs(value) {
		return ""
	}
	return filepath.Clean(value)
}
