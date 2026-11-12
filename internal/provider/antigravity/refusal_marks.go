package antigravity

import (
	"crypto/sha256"
	"sync"
	"time"
)

const (
	// refusalLifetime 与参考实现一致：同一 system prompt 被拒后 6 小时内不再先撞一次。
	refusalLifetime = 6 * time.Hour
	// refusalLimit 是本机内存表上限，避免长时间运行后无界增长。
	refusalLimit = 256
)

type refusalMarks struct {
	mutex sync.Mutex
	marks map[[32]byte]time.Time
}

// refusalKey 按账号身份、项目与原文 system 计算指纹：不同账号或不同 prompt 不互相跳过。
func refusalKey(cred Credentials, system string) [32]byte {
	hash := sha256.New()
	for _, part := range []string{cred.Subject, cred.Project, system} {
		hash.Write([]byte(part))
		hash.Write([]byte{0})
	}
	var key [32]byte
	copy(key[:], hash.Sum(nil))
	return key
}

// noteRefusal 记录一次内容拒绝，供后续同 prompt 的请求跳过该账号。
func (c *Client) noteRefusal(cred Credentials, system string) {
	key := refusalKey(cred, system)
	now := time.Now()
	c.refusals.mutex.Lock()
	defer c.refusals.mutex.Unlock()
	if c.refusals.marks == nil {
		c.refusals.marks = make(map[[32]byte]time.Time, refusalLimit)
	}
	if len(c.refusals.marks) >= refusalLimit {
		for candidate, until := range c.refusals.marks {
			if !until.After(now) {
				delete(c.refusals.marks, candidate)
			}
		}
		if len(c.refusals.marks) >= refusalLimit {
			// 仍有上限条数：淘汰最早到期的一条，保证新记录一定写下。
			var oldest [32]byte
			var earliest time.Time
			for candidate, until := range c.refusals.marks {
				if earliest.IsZero() || until.Before(earliest) {
					oldest, earliest = candidate, until
				}
			}
			delete(c.refusals.marks, oldest)
		}
	}
	c.refusals.marks[key] = now.Add(refusalLifetime)
}

// refusedBefore 报告同一账号是否在 6 小时内拒绝过这段 system prompt。
func (c *Client) refusedBefore(cred Credentials, system string) bool {
	key := refusalKey(cred, system)
	now := time.Now()
	c.refusals.mutex.Lock()
	defer c.refusals.mutex.Unlock()
	until, ok := c.refusals.marks[key]
	if !ok {
		return false
	}
	if !until.After(now) {
		delete(c.refusals.marks, key)
		return false
	}
	return true
}
