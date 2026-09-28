// Package credential stores opaque secrets as per-user protected files.
// The database keeps only the random reference returned by Store.Put.
package credential

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"omniproxy/internal/platform/profile"
)

const (
	// MaxSecretSize bounds a single credential bundle, including future token sets.
	MaxSecretSize    = 1 << 20
	maxEncryptedSize = 2*MaxSecretSize + 4096
)

var (
	// ErrInvalidStore means the store has not been initialized by New.
	ErrInvalidStore = errors.New("credential: invalid store")
	// ErrInvalidRef means a credential reference is malformed.
	ErrInvalidRef = errors.New("credential: invalid reference")
	// ErrInvalidSecret means a secret is empty or exceeds MaxSecretSize.
	ErrInvalidSecret = errors.New("credential: invalid secret size")
	// ErrNotFound means a valid reference has no stored credential in this profile.
	ErrNotFound = errors.New("credential: not found")
	// ErrUnsupported means the current platform has no credential protector.
	ErrUnsupported = errors.New("credential: unsupported platform")
)

// Store keeps encrypted credential files separate for each runtime profile.
type Store struct {
	dir string
}

// New creates a store for a resolved profile. It does not create files until Put.
func New(p *profile.Profile) (*Store, error) {
	if p == nil || !filepath.IsAbs(p.DataDir) {
		return nil, fmt.Errorf("credential: an absolute profile data directory is required")
	}
	switch p.Kind {
	case profile.Production, profile.Development, profile.Test:
	default:
		return nil, fmt.Errorf("credential: invalid profile kind")
	}
	return &Store{dir: filepath.Join(filepath.Clean(p.DataDir), "credentials", string(p.Kind))}, nil
}

// Put protects secret and returns a new opaque reference. The caller retains
// ownership of secret and should clear its own copy when it is no longer needed.
func (s *Store) Put(secret []byte) (string, error) {
	if s == nil || s.dir == "" {
		return "", ErrInvalidStore
	}
	if len(secret) == 0 || len(secret) > MaxSecretSize {
		return "", ErrInvalidSecret
	}
	protected, err := protect(secret)
	if err != nil {
		return "", fmt.Errorf("credential: protect: %w", err)
	}
	defer clear(protected)
	if len(protected) == 0 || len(protected) > maxEncryptedSize {
		return "", fmt.Errorf("credential: invalid protected data size")
	}
	if err := os.MkdirAll(s.dir, 0o700); err != nil {
		return "", fmt.Errorf("credential: create store: %w", err)
	}

	for range refAttempts {
		ref, err := newRef()
		if err != nil {
			return "", fmt.Errorf("credential: generate reference: %w", err)
		}
		err = writeNewFile(filepath.Join(s.dir, ref+".bin"), protected)
		if errors.Is(err, os.ErrExist) {
			continue
		}
		if err != nil {
			return "", fmt.Errorf("credential: write protected data: %w", err)
		}
		return ref, nil
	}
	return "", fmt.Errorf("credential: could not allocate a unique reference")
}

// Get returns a copy of the secret for ref. The caller owns the returned bytes.
func (s *Store) Get(ref string) ([]byte, error) {
	path, err := s.pathForRef(ref)
	if err != nil {
		return nil, err
	}
	protected, err := readBoundedFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, ErrNotFound
	}
	if err != nil {
		clear(protected)
		return nil, fmt.Errorf("credential: read protected data: %w", err)
	}
	defer clear(protected)
	secret, err := unprotect(protected)
	if err != nil {
		return nil, fmt.Errorf("credential: unprotect: %w", err)
	}
	if len(secret) == 0 || len(secret) > MaxSecretSize {
		clear(secret)
		return nil, fmt.Errorf("credential: invalid decrypted data size")
	}
	return secret, nil
}

// Delete removes the encrypted file for ref. Missing credentials return ErrNotFound.
func (s *Store) Delete(ref string) error {
	path, err := s.pathForRef(ref)
	if err != nil {
		return err
	}
	if err := os.Remove(path); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return ErrNotFound
		}
		return fmt.Errorf("credential: delete protected data: %w", err)
	}
	return nil
}

func writeNewFile(path string, data []byte) error {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return err
	}
	n, writeErr := f.Write(data)
	if writeErr == nil && n != len(data) {
		writeErr = io.ErrShortWrite
	}
	if writeErr == nil {
		writeErr = f.Sync()
	}
	closeErr := f.Close()
	if err := errors.Join(writeErr, closeErr); err != nil {
		return errors.Join(err, os.Remove(path))
	}
	return nil
}

func readBoundedFile(path string) (data []byte, err error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() { err = errors.Join(err, f.Close()) }()
	info, err := f.Stat()
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() || info.Size() <= 0 || info.Size() > maxEncryptedSize {
		return nil, fmt.Errorf("credential: invalid protected file")
	}
	data, err = io.ReadAll(io.LimitReader(f, maxEncryptedSize+1))
	if err != nil {
		clear(data)
		return nil, err
	}
	if len(data) == 0 || len(data) > maxEncryptedSize {
		clear(data)
		return nil, fmt.Errorf("credential: invalid protected file")
	}
	return data, nil
}
