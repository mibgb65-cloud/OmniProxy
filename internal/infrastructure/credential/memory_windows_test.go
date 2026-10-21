//go:build windows

package credential

import (
	"bytes"
	"errors"
	"sync"
	"testing"
)

func TestMemorySecretRoundTripIndependentCopiesAndClear(t *testing.T) {
	want := []byte("synthetic-memory-credential-only")
	input := append([]byte(nil), want...)
	secret, err := NewMemorySecret(input)
	if err != nil {
		t.Fatal("memory protection failed", err)
	}
	defer secret.Clear()
	if bytes.Contains(secret.protected, want) {
		t.Fatal("protected memory contains plaintext")
	}
	clear(input)
	for range 2 {
		got, err := secret.Open()
		if err != nil || !bytes.Equal(got, want) {
			t.Fatal("protected memory did not return an independent copy")
		}
		clear(got)
	}
	retained := secret.protected
	secret.Clear()
	for _, value := range retained {
		if value != 0 {
			t.Fatal("cleared ciphertext still retained bytes")
		}
	}
	if _, err := secret.Open(); !errors.Is(err, ErrNotFound) {
		t.Fatal("cleared memory secret was readable")
	}
}

func TestMemorySecretConcurrentReadClearAndTampering(t *testing.T) {
	secret, err := NewMemorySecret([]byte("synthetic-concurrent-value"))
	if err != nil {
		t.Fatal(err)
	}
	var group sync.WaitGroup
	for range 8 {
		group.Go(func() {
			for range 10 {
				value, err := secret.Open()
				clear(value)
				if err != nil && !errors.Is(err, ErrNotFound) {
					t.Error("concurrent protected read failed")
				}
			}
		})
	}
	group.Go(secret.Clear)
	group.Wait()
	tampered, err := NewMemorySecret([]byte("synthetic-tamper-value"))
	if err != nil {
		t.Fatal(err)
	}
	defer tampered.Clear()
	tampered.protected[len(tampered.protected)/2] ^= 1
	value, err := tampered.Open()
	defer clear(value)
	if err == nil {
		t.Fatal("tampered protected memory was accepted")
	}
}
