//go:build windows

package credential

import (
	"errors"
	"fmt"
	"runtime"
	"unsafe"

	"golang.org/x/sys/windows"
)

func protect(data []byte) ([]byte, error) {
	in := windows.DataBlob{Size: uint32(len(data)), Data: &data[0]}
	var out windows.DataBlob
	err := windows.CryptProtectData(&in, nil, nil, 0, nil, windows.CRYPTPROTECT_UI_FORBIDDEN, &out)
	runtime.KeepAlive(data)
	if err != nil {
		return nil, errors.Join(err, freeBlob(out))
	}
	return copyAndFree(out, maxEncryptedSize)
}

func unprotect(data []byte) ([]byte, error) {
	in := windows.DataBlob{Size: uint32(len(data)), Data: &data[0]}
	var out windows.DataBlob
	err := windows.CryptUnprotectData(&in, nil, nil, 0, nil, windows.CRYPTPROTECT_UI_FORBIDDEN, &out)
	runtime.KeepAlive(data)
	if err != nil {
		return nil, errors.Join(err, freeBlob(out))
	}
	return copyAndFree(out, MaxSecretSize)
}

func copyAndFree(blob windows.DataBlob, maxSize int) (result []byte, err error) {
	if blob.Data == nil {
		return nil, fmt.Errorf("credential: empty DPAPI output")
	}
	defer func() {
		freeErr := freeBlob(blob)
		if freeErr != nil {
			clear(result)
			result = nil
			err = errors.Join(err, fmt.Errorf("credential: free DPAPI output: %w", freeErr))
		}
	}()
	if blob.Size == 0 || blob.Size > uint32(maxSize) {
		return nil, fmt.Errorf("credential: invalid DPAPI output size")
	}
	native := unsafe.Slice(blob.Data, int(blob.Size))
	result = append([]byte(nil), native...)
	return result, nil
}

func freeBlob(blob windows.DataBlob) error {
	if blob.Data == nil {
		return nil
	}
	// DPAPI 的输出分配在 Go 堆外；释放前清零正常大小的缓冲区。
	if blob.Size > 0 && blob.Size <= maxEncryptedSize {
		clear(unsafe.Slice(blob.Data, int(blob.Size)))
	}
	_, err := windows.LocalFree(windows.Handle(uintptr(unsafe.Pointer(blob.Data))))
	return err
}
