// Package storage provides filesystem helpers for the OmniProxy data
// directory: size measurement and relocation.
package storage

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
)

// DirSize returns the total size of all regular files under path (recursive).
// A missing directory reports 0.
func DirSize(path string) (int64, error) {
	var total int64
	err := filepath.WalkDir(path, func(_ string, d fs.DirEntry, err error) error {
		if err != nil {
			if os.IsNotExist(err) {
				return nil // a missing root simply has no size
			}
			return err
		}
		if d.IsDir() {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		total += info.Size()
		return nil
	})
	if err != nil {
		return 0, fmt.Errorf("storage: size of %s: %w", path, err)
	}
	return total, nil
}

// FileSize returns the size of a single file; a missing file reports 0.
func FileSize(path string) (int64, error) {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, fmt.Errorf("storage: stat %s: %w", path, err)
	}
	if info.IsDir() {
		return 0, fmt.Errorf("storage: %s is a directory", path)
	}
	return info.Size(), nil
}

// MoveDir moves every entry of src into dst, creating dst if needed. It uses
// os.Rename first and falls back to a recursive copy for cross-volume moves
// (os.Rename fails across devices on Windows). Existing files in dst are
// overwritten. A missing src is a no-op.
func MoveDir(src, dst string) error {
	entries, err := os.ReadDir(src)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("storage: read %s: %w", src, err)
	}
	if err := os.MkdirAll(dst, 0o750); err != nil {
		return fmt.Errorf("storage: create %s: %w", dst, err)
	}
	for _, entry := range entries {
		if err := moveEntry(filepath.Join(src, entry.Name()), filepath.Join(dst, entry.Name())); err != nil {
			return err
		}
	}
	return nil
}

func moveEntry(from, to string) error {
	if err := os.Rename(from, to); err == nil {
		return nil
	}
	info, err := os.Lstat(from)
	if err != nil {
		return fmt.Errorf("storage: stat %s: %w", from, err)
	}
	if info.IsDir() {
		if err := copyDir(from, to); err != nil {
			return err
		}
		return os.RemoveAll(from)
	}
	if err := copyFile(from, to, info.Mode()); err != nil {
		return err
	}
	return os.Remove(from)
}

func copyDir(src, dst string) error {
	if err := os.MkdirAll(dst, 0o750); err != nil {
		return fmt.Errorf("storage: create %s: %w", dst, err)
	}
	entries, err := os.ReadDir(src)
	if err != nil {
		return fmt.Errorf("storage: read %s: %w", src, err)
	}
	for _, entry := range entries {
		if err := moveEntry(filepath.Join(src, entry.Name()), filepath.Join(dst, entry.Name())); err != nil {
			return err
		}
	}
	return nil
}

func copyFile(src, dst string, mode fs.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("storage: open %s: %w", src, err)
	}
	defer func() { _ = in.Close() }()

	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, mode)
	if err != nil {
		return fmt.Errorf("storage: create %s: %w", dst, err)
	}
	if _, err := io.Copy(out, in); err != nil {
		_ = out.Close()
		return fmt.Errorf("storage: copy %s -> %s: %w", src, dst, err)
	}
	if err := out.Close(); err != nil {
		return fmt.Errorf("storage: close %s: %w", dst, err)
	}
	return nil
}
