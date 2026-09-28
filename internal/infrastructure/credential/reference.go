package credential

import (
	"crypto/rand"
	"encoding/hex"
	"path/filepath"
)

const (
	refPrefix      = "cred-v1-"
	refRandomBytes = 16
	refAttempts    = 3
)

func (s *Store) pathForRef(ref string) (string, error) {
	if s == nil || s.dir == "" {
		return "", ErrInvalidStore
	}
	if len(ref) != len(refPrefix)+refRandomBytes*2 || ref[:len(refPrefix)] != refPrefix {
		return "", ErrInvalidRef
	}
	for _, c := range ref[len(refPrefix):] {
		if c < '0' || c > '9' {
			if c < 'a' || c > 'f' {
				return "", ErrInvalidRef
			}
		}
	}
	return filepath.Join(s.dir, ref+".bin"), nil
}

func newRef() (string, error) {
	var random [refRandomBytes]byte
	if _, err := rand.Read(random[:]); err != nil {
		return "", err
	}
	return refPrefix + hex.EncodeToString(random[:]), nil
}
