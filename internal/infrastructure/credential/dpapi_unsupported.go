//go:build !windows

package credential

func protect([]byte) ([]byte, error)   { return nil, ErrUnsupported }
func unprotect([]byte) ([]byte, error) { return nil, ErrUnsupported }
