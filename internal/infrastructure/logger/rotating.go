package logger

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// Rotation defaults for the application log file.
const (
	defaultMaxSize = 5 << 20 // 5 MiB
	defaultBackups = 3
)

// rotatingFile is an io.Writer that rotates the log file once it exceeds
// maxSize, keeping at most backups older files (path.1 .. path.N).
type rotatingFile struct {
	mu      sync.Mutex
	path    string
	maxSize int64
	backups int
	file    *os.File
	size    int64
}

// NewRotatingFile opens path for appending with the default rotation policy
// (5 MiB per file, 3 backups).
func NewRotatingFile(path string) (*rotatingFile, error) {
	return newRotatingFile(path, defaultMaxSize, defaultBackups)
}

func newRotatingFile(path string, maxSize int64, backups int) (*rotatingFile, error) {
	if maxSize <= 0 {
		return nil, fmt.Errorf("logger: maxSize must be positive, got %d", maxSize)
	}
	w := &rotatingFile{path: path, maxSize: maxSize, backups: backups}
	if err := w.open(); err != nil {
		return nil, err
	}
	return w, nil
}

func (w *rotatingFile) open() error {
	if err := os.MkdirAll(filepath.Dir(w.path), 0o750); err != nil {
		return fmt.Errorf("logger: create log dir: %w", err)
	}
	f, err := os.OpenFile(w.path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return fmt.Errorf("logger: open log file: %w", err)
	}
	info, err := f.Stat()
	if err != nil {
		_ = f.Close()
		return fmt.Errorf("logger: stat log file: %w", err)
	}
	w.file, w.size = f, info.Size()
	return nil
}

// Write implements io.Writer, rotating before the write when the file would
// grow past maxSize.
func (w *rotatingFile) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.size > 0 && w.size+int64(len(p)) > w.maxSize {
		if err := w.rotate(); err != nil {
			return 0, err
		}
	}
	n, err := w.file.Write(p)
	w.size += int64(n)
	return n, err
}

func (w *rotatingFile) rotate() error {
	if err := w.file.Close(); err != nil {
		return fmt.Errorf("logger: close during rotate: %w", err)
	}
	for i := w.backups - 1; i >= 1; i-- {
		src := fmt.Sprintf("%s.%d", w.path, i)
		dst := fmt.Sprintf("%s.%d", w.path, i+1)
		if _, err := os.Stat(src); err != nil {
			continue
		}
		if err := os.Rename(src, dst); err != nil {
			return fmt.Errorf("logger: rotate %s -> %s: %w", src, dst, err)
		}
	}
	if err := os.Rename(w.path, w.path+".1"); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("logger: rotate current file: %w", err)
	}
	return w.open()
}

// Path returns the path of the active log file.
func (w *rotatingFile) Path() string {
	return w.path
}

// Truncate empties the file and resets the size counter. It returns the
// number of bytes freed.
func (w *rotatingFile) Truncate() (int64, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.file == nil {
		return 0, nil
	}
	previous := w.size
	// On Windows, SetEndOfFile is denied on handles opened with O_APPEND
	// (FILE_APPEND_DATA only), so close, truncate by path, and reopen.
	if err := w.file.Close(); err != nil {
		return 0, fmt.Errorf("logger: close before truncate: %w", err)
	}
	if err := os.Truncate(w.path, 0); err != nil {
		_ = w.open() // best effort: keep logging alive
		return 0, fmt.Errorf("logger: truncate log file: %w", err)
	}
	if err := w.open(); err != nil {
		return 0, err
	}
	return previous, nil
}

// Close closes the underlying file. It is safe to call more than once.
func (w *rotatingFile) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.file == nil {
		return nil
	}
	err := w.file.Close()
	w.file = nil
	return err
}
