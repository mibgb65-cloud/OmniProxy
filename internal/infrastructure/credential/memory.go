package credential

import "sync"

// MemorySecret 仅在进程内保留系统保护的密文，不创建凭据文件。
type MemorySecret struct {
	mu        sync.Mutex
	protected []byte
}

// NewMemorySecret 使用与凭据库相同的系统保护，调用方仍负责清零输入。
func NewMemorySecret(secret []byte) (*MemorySecret, error) {
	if len(secret) == 0 || len(secret) > MaxSecretSize {
		return nil, ErrInvalidSecret
	}
	data, err := protect(secret)
	if err != nil {
		clear(data)
		return nil, err
	}
	if len(data) == 0 || len(data) > maxEncryptedSize {
		clear(data)
		return nil, ErrInvalidSecret
	}
	return &MemorySecret{protected: data}, nil
}

// Open 返回独立明文副本，调用方必须在使用后清零。
func (s *MemorySecret) Open() ([]byte, error) {
	if s == nil {
		return nil, ErrNotFound
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.protected) == 0 {
		return nil, ErrNotFound
	}
	return unprotect(s.protected)
}

// Clear 清零并释放密文，使后续读取失败；可与 Open 并发调用。
func (s *MemorySecret) Clear() {
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	clear(s.protected)
	s.protected = nil
}
