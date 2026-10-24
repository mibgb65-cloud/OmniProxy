package agentconfig

import (
	"context"
	"os"
	"path/filepath"
)

const codexAuthBackupSuffix = ".omniproxy.bak"

// codexLoginChange 记录本次登录文件写入前的状态，供失败时回滚。
type codexLoginChange struct {
	path     string
	previous []byte
	existed  bool
	applied  bool
}

// SetCodexLogin 安装"网关拥有登录"的凭据来源（v1 语义）：返回池账号的 auth.json。
// nil 表示保留客户端原有登录，不覆盖也不备份。
func (s *Service) SetCodexLogin(read func(context.Context) ([]byte, error)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.codexLogin = read
}

// codexAuthPath 返回受管 Codex 登录文件，与 config.toml 同目录。
func (s *Service) codexAuthPath() string {
	return filepath.Join(filepath.Dir(s.paths.Codex), "auth.json")
}

// applyCodexLogin 把池账号写进客户端登录：v1 的 writeCodexAccountAuth 语义。
// 首次覆盖前保留 .omniproxy.bak，恢复时用它还原用户原来的登录。
func (s *Service) applyCodexLogin(ctx context.Context) (*codexLoginChange, error) {
	read := s.codexLogin
	if read == nil {
		return nil, nil
	}
	body, err := read(ctx)
	if err != nil || len(body) == 0 {
		return nil, err
	}
	defer clear(body)
	path := s.codexAuthPath()
	current, exists, err := readAuthFile(path)
	if err != nil {
		return nil, err
	}
	defer clear(current)
	change := &codexLoginChange{path: path, previous: append([]byte(nil), current...), existed: exists}
	if !exists {
		return change, nil
	}
	backup := path + codexAuthBackupSuffix
	if _, statErr := os.Lstat(backup); os.IsNotExist(statErr) {
		if err := os.WriteFile(backup, current, 0o600); err != nil {
			return nil, failure("write_failed")
		}
	}
	if err := writeAtomic(path, body, digest(current, exists)); err != nil {
		return nil, err
	}
	change.applied = true
	return change, nil
}

// rollback 撤销本次写入，保持客户端登录仍是调用前的内容。
func (c *codexLoginChange) rollback() error {
	if c == nil || !c.applied {
		return nil
	}
	if !c.existed {
		if err := os.Remove(c.path); err != nil && !os.IsNotExist(err) {
			return failure("write_failed")
		}
		return nil
	}
	current, exists, err := readAuthFile(c.path)
	if err != nil {
		return err
	}
	defer clear(current)
	return writeAtomic(c.path, c.previous, digest(current, exists))
}

func (c *codexLoginChange) clear() {
	if c != nil {
		clear(c.previous)
	}
}

// restoreCodexLogin 恢复客户端原登录，并删除一次性备份。
func (s *Service) restoreCodexLogin() error {
	path := s.codexAuthPath()
	backup := path + codexAuthBackupSuffix
	body, err := os.ReadFile(backup)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return failure("read_failed")
	}
	defer clear(body)
	current, exists, err := readAuthFile(path)
	if err != nil {
		return err
	}
	defer clear(current)
	if err := writeAtomic(path, body, digest(current, exists)); err != nil {
		return err
	}
	if err := os.Remove(backup); err != nil && !os.IsNotExist(err) {
		return failure("write_failed")
	}
	return nil
}

// readAuthFile 只读取登录文件本身，不跟随符号链接、不放宽大小上限。
func readAuthFile(path string) ([]byte, bool, error) {
	info, err := os.Lstat(path)
	if os.IsNotExist(err) {
		return []byte{}, false, nil
	}
	if err != nil || !info.Mode().IsRegular() || info.Size() > maxConfigBytes {
		return nil, false, failure("unsafe_auth")
	}
	body, err := os.ReadFile(path)
	if err != nil {
		return nil, false, failure("read_failed")
	}
	return body, true, nil
}
