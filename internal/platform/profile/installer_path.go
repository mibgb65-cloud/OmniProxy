package profile

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"unicode"
)

func validInstallerPath(dir string) error {
	if dir == "" || len(dir) > 4096 || !filepath.IsAbs(dir) ||
		strings.IndexFunc(dir, unicode.IsControl) >= 0 ||
		strings.HasPrefix(dir, `\\?\`) || strings.HasPrefix(dir, `\\.\`) ||
		strings.ContainsAny(strings.TrimPrefix(dir, filepath.VolumeName(dir)), `:*?"<>|`) ||
		filepath.Clean(dir) == filepath.VolumeName(dir)+string(filepath.Separator) {
		return errors.New("profile: invalid installer directory")
	}
	return nil
}

func installerLegacyDataDir(installDir string) (string, error) {
	file, err := os.Open(filepath.Join(installDir, installDataDirFileName))
	if os.IsNotExist(err) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("profile: read installer data marker: %w", err)
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, 16385))
	if err != nil {
		return "", err
	}
	if len(data) > 16384 {
		return "", errors.New("profile: installer data marker too large")
	}
	dir := parseInstallDataDir(decodeInstallText(data))
	if err := validInstallerPath(dir); err != nil {
		return "", err
	}
	return dir, nil
}

func installerContainsDatabase(dir string) (bool, error) {
	for _, path := range []string{filepath.Join(dir, databaseFileName), filepath.Join(dir, "v2", databaseFileName)} {
		info, err := os.Stat(path)
		if err == nil {
			if !info.Mode().IsRegular() {
				return false, errors.New("profile: invalid database entry")
			}
			return true, nil
		}
		if !os.IsNotExist(err) {
			return false, fmt.Errorf("profile: inspect installer data: %w", err)
		}
	}
	return false, nil
}
