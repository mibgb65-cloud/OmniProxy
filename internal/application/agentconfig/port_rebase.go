package agentconfig

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"strconv"
)

// PortRebase 汇总网关端口变化后受管客户端配置的处理结果。
type PortRebase struct {
	// Updated 是地址已改写为新端口的客户端。
	Updated []string `json:"updated"`
	// Unchanged 是受管但配置里没有本机旧地址、无需改写的客户端。
	Unchanged []string `json:"unchanged"`
	// Manual 是需要用户重新读取、预览并应用（或手工改地址）的客户端：
	// 配置被外部改动过，或本次改写失败。
	Manual []string `json:"manual"`
}

// SetPort 更新写入客户端配置的网关端口。已写入客户端的地址由 RebasePort 处理。
func (s *Service) SetPort(port int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.port = port
}

// RebasePort 把受管客户端配置中指向本机旧端口的地址改写为新端口。
//
// 只改写 agent_backups.applied_hash 与磁盘内容仍然一致的配置：被外部改动过的
// 文件保持原样并列入 Manual，避免覆盖用户或其它工具的修改。
func (s *Service) RebasePort(ctx context.Context, oldPort, newPort int) (PortRebase, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var result PortRebase
	if oldPort == newPort || oldPort < 1 || newPort < 1 || oldPort > 65535 || newPort > 65535 {
		return result, nil
	}
	var resultErr error
	fail := func(client string, err error) {
		result.Manual = append(result.Manual, client)
		if err != nil {
			resultErr = errors.Join(resultErr, err)
		}
	}
	for _, client := range Clients() {
		var expected string
		err := s.db.QueryRowContext(ctx, `SELECT applied_hash FROM agent_backups WHERE client=?`, client).Scan(&expected)
		if errors.Is(err, sql.ErrNoRows) {
			continue // 没有受管记录：不读取也不改写该客户端
		}
		if err != nil {
			// 备份表不可用时无法判断哪些客户端受管：停止改写，不逐项猜测。
			return result, errors.Join(resultErr, failure("backup_unavailable"))
		}
		path, err := s.path(client)
		if err != nil {
			fail(client, err)
			continue
		}
		before, err := s.snapshotClient(client, path)
		if err != nil {
			fail(client, err)
			continue
		}
		if !before.exists {
			clear(before.body)
			continue
		}
		if before.revision != expected {
			clear(before.body)
			fail(client, nil)
			continue
		}
		body, changed, err := rebaseClientPort(client, before.body, oldPort, newPort)
		if err != nil {
			clear(before.body)
			fail(client, err)
			continue
		}
		if !changed {
			clear(before.body)
			result.Unchanged = append(result.Unchanged, client)
			continue
		}
		err = s.writeClient(client, path, body, before.revision)
		if err == nil {
			if err = s.commitFileVersions(ctx, client, digest(body, true), false, ""); err != nil {
				err = errors.Join(err, s.rollbackClient(client, path, digest(body, true), before))
			}
		}
		clear(body)
		clear(before.body)
		if err != nil {
			fail(client, err)
			continue
		}
		result.Updated = append(result.Updated, client)
	}
	return result, resultErr
}

// rebaseClientPort 只改写受管配置里指向本机旧端口的地址。
//
// harness 与 bundled 客户端的文件内容封装在 JSON 备份里（[]byte 以 base64 编码），
// 因此必须先解码内部字节再改写，不能直接在封装层做文本替换。
func rebaseClientPort(client string, body []byte, oldPort, newPort int) ([]byte, bool, error) {
	if isHarness(client) {
		var bundle harnessBundle
		if json.Unmarshal(body, &bundle) != nil {
			return nil, false, failure("invalid_config")
		}
		next, changed := replaceLoopbackPort(bundle.Config.Body, oldPort, newPort)
		if !changed {
			return nil, false, nil
		}
		bundle.Config.Body = next
		encoded, err := json.Marshal(bundle)
		if err != nil || len(encoded) > maxConfigBytes {
			return nil, false, failure("unsafe_config")
		}
		return encoded, true, nil
	}
	if isBundledClient(client) {
		bundle, err := decodeBundle(body)
		if err != nil {
			return nil, false, err
		}
		changed := false
		for index := range bundle.Files {
			if next, fileChanged := replaceLoopbackPort(bundle.Files[index].Body, oldPort, newPort); fileChanged {
				bundle.Files[index].Body = next
				changed = true
			}
		}
		if !changed {
			return nil, false, nil
		}
		encoded, err := encodeBundle(bundle)
		if err != nil {
			return nil, false, err
		}
		return encoded, true, nil
	}
	next, changed := replaceLoopbackPort(body, oldPort, newPort)
	if !changed {
		return nil, false, nil
	}
	return next, true, nil
}

// replaceLoopbackPort 把文本里指向本机旧端口的地址改写为新端口。
//
// 只匹配完整的端口数字，因此 3000 不会误伤 30000；其余内容（注释、其它提供商、
// 密钥引用）保持原样。
func replaceLoopbackPort(body []byte, oldPort, newPort int) ([]byte, bool) {
	if oldPort == newPort {
		return nil, false
	}
	oldText := strconv.Itoa(oldPort)
	newText := []byte(strconv.Itoa(newPort))
	var out []byte
	changed := false
	// scan 负责向前查找，cursor 记录已写入 out 的位置：未命中的主机保持原样。
	scan, cursor := 0, 0
	for scan < len(body) {
		host, hostLength := nextLoopbackHost(body, scan)
		if host < 0 {
			break
		}
		start := host + hostLength
		end := start
		for end < len(body) && body[end] >= '0' && body[end] <= '9' {
			end++
		}
		if string(body[start:end]) == oldText {
			if out == nil {
				out = make([]byte, 0, len(body)+len(newText))
			}
			out = append(out, body[cursor:start]...)
			out = append(out, newText...)
			changed = true
			cursor = end
		}
		scan = end
	}
	if !changed {
		return nil, false
	}
	return append(out, body[cursor:]...), true
}

// nextLoopbackHost 返回 from 之后最近的回环主机加冒号的位置与长度。
func nextLoopbackHost(body []byte, from int) (int, int) {
	index, length := -1, 0
	for _, host := range []string{"127.0.0.1:", "localhost:"} {
		found := bytes.Index(body[from:], []byte(host))
		if found < 0 {
			continue
		}
		if index < 0 || from+found < index {
			index, length = from+found, len(host)
		}
	}
	return index, length
}
