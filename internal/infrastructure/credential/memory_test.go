package credential

import (
	"errors"
	"testing"
)

func TestMemorySecretRejectsInvalidSizesAndClearedValues(t *testing.T) {
	for _, data := range [][]byte{nil, {}, make([]byte, MaxSecretSize+1)} {
		if _, err := NewMemorySecret(data); !errors.Is(err, ErrInvalidSecret) {
			t.Fatal("invalid memory secret size accepted")
		}
	}
	for _, secret := range []*MemorySecret{nil, {}} {
		secret.Clear()
		if _, err := secret.Open(); !errors.Is(err, ErrNotFound) {
			t.Fatal("empty protected memory was readable")
		}
	}
}
