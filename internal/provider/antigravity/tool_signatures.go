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

// fillUnsignedSignatures 给「整轮都没有 Google 签名」的模型回合补上跳过校验的标记：
// 跨协议客户端（Claude Code、Codex）不带 Google 签名，缺标记时上游会拒绝该轮；
// Claude 模型的工具调用尤其如此。缓存回填过的回合保持原样。
func fillUnsignedSignatures(body []byte) ([]byte, error) {
	var envelope object
	if err := json.Unmarshal(body, &envelope); err != nil {
		return nil, err
	}
	request, _ := envelope["request"].(map[string]any)
	data, _ := request["contents"].([]any)
	for _, value := range data {
		content, _ := value.(map[string]any)
		if content["role"] != "model" {
			continue
		}
		parts, _ := content["parts"].([]any)
		signed := false
		for _, part := range parts {
			if entry, ok := part.(map[string]any); ok && entry["thoughtSignature"] != nil {
				signed = true
				break
			}
		}
		if signed {
			continue
		}
		for _, part := range parts {
			entry, _ := part.(map[string]any)
			if entry["functionCall"] != nil {
				entry["thoughtSignature"] = skipSignature
			}
		}
	}
	return json.Marshal(envelope)
}

// restoreSignatures 回填跨协议客户端没有携带的 Google 签名（按授权、项目、
// 模型与完整工具内容摘要隔离），不保存消息正文。
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
