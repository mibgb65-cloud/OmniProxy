package antigravity

import (
	"crypto/sha256"
	"encoding/json"
	"sync"
	"time"
)

const signatureLimit = 256
const signatureLifetime = time.Hour

type signatureEntry struct {
	value   string
	expires time.Time
}
type signatureCache struct {
	mu      sync.Mutex
	entries map[[32]byte]signatureEntry
}

// Responses／Messages 不携带 Google 扩展字段；只缓存不透明签名，不保存消息或工具正文。
// 授权、项目、型号及完整工具内容参与摘要，禁止跨账号或修改参数后复用签名。
func signatureKey(cred Credentials, model, id, name string, args any) [32]byte {
	encoded, _ := json.Marshal([]any{cred.Subject, cred.Project, cred.RefreshToken, model, id, name, args})
	defer clear(encoded)
	return sha256.Sum256(encoded)
}

func (c *Client) rememberSignatures(cred Credentials, model string, calls []object) {
	c.signatures.mu.Lock()
	defer c.signatures.mu.Unlock()
	now := time.Now()
	if c.signatures.entries == nil {
		c.signatures.entries = map[[32]byte]signatureEntry{}
	}
	for key, entry := range c.signatures.entries {
		if !entry.expires.After(now) {
			delete(c.signatures.entries, key)
		}
	}
	for _, call := range calls {
		extra, _ := call["extra_content"].(object)
		google, _ := extra["google"].(object)
		signature, _ := google["thought_signature"].(string)
		if signature == "" || len(signature) > 65536 {
			continue
		}
		function := call["function"].(object)
		var args object
		if json.Unmarshal([]byte(function["arguments"].(string)), &args) != nil {
			continue
		}
		key := signatureKey(cred, model, call["id"].(string), function["name"].(string), args)
		if len(c.signatures.entries) >= signatureLimit {
			var oldest [32]byte
			var expiry time.Time
			for candidate, entry := range c.signatures.entries {
				if expiry.IsZero() || entry.expires.Before(expiry) {
					oldest, expiry = candidate, entry.expires
				}
			}
			delete(c.signatures.entries, oldest)
		}
		c.signatures.entries[key] = signatureEntry{signature, now.Add(signatureLifetime)}
	}
}

func (c *Client) restoreSignatures(body []byte, cred Credentials, model string) ([]byte, error) {
	var payload struct {
		Request object `json:"request"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, err
	}
	contents, _ := payload.Request["contents"].([]any)
	c.signatures.mu.Lock()
	defer c.signatures.mu.Unlock()
	for _, content := range contents {
		parts, _ := content.(object)["parts"].([]any)
		for _, value := range parts {
			part := value.(object)
			call, ok := part["functionCall"].(object)
			if !ok || part["thoughtSignature"] != nil {
				continue
			}
			id, _ := call["id"].(string)
			name, _ := call["name"].(string)
			key := signatureKey(cred, model, id, name, call["args"])
			if entry, found := c.signatures.entries[key]; found {
				if entry.expires.After(time.Now()) {
					part["thoughtSignature"] = entry.value
				} else {
					delete(c.signatures.entries, key)
				}
			}
		}
	}
	var envelope object
	if err := json.Unmarshal(body, &envelope); err != nil {
		return nil, err
	}
	envelope["request"] = payload.Request
	return json.Marshal(envelope)
}
