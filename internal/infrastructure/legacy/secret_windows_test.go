//go:build windows

package legacy

import (
	"encoding/base64"
	"strings"
	"testing"
	"unsafe"

	"golang.org/x/sys/windows"
)

func TestDecodeMatchesRealV1DPAPIEntropy(t *testing.T) {
	plain := []byte("sk-fake-legacy-only-test")
	entropy := []byte("OmniProxy token store v1")
	in := windows.DataBlob{Size: uint32(len(plain)), Data: &plain[0]}
	extra := windows.DataBlob{Size: uint32(len(entropy)), Data: &entropy[0]}
	var out windows.DataBlob
	if err := windows.CryptProtectData(&in, nil, &extra, 0, nil, windows.CRYPTPROTECT_UI_FORBIDDEN, &out); err != nil {
		t.Fatal(err)
	}
	defer windows.LocalFree(windows.Handle(uintptr(unsafe.Pointer(out.Data))))
	encoded := "omniproxy-secret:v1:" + base64.RawURLEncoding.EncodeToString(unsafe.Slice(out.Data, int(out.Size)))
	got, err := Unprotect(encoded)
	if err != nil || string(got) != string(plain) {
		t.Fatal("v1 credential could not be decoded")
	}
	for _, bad := range []string{"", "omniproxy-secret:v1:invalid", strings.Repeat("x", 32769)} {
		if _, err := Unprotect(bad); err == nil {
			t.Fatal("invalid credential accepted")
		}
	}
}
