package antigravity

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"strconv"
	"strings"

	"omniproxy/internal/core/proxy"
)

// textContent 只接受字符串或纯文本 part：图片、语音等由上游能力边界明确拒绝。
func textContent(raw json.RawMessage) (string, error) {
	if string(raw) == "null" || len(raw) == 0 {
		return "", nil
	}
	var value string
	if json.Unmarshal(raw, &value) == nil {
		return value, nil
	}
	var parts []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}
	if json.Unmarshal(raw, &parts) != nil || len(parts) > 1000 {
		return "", proxy.Failure("unsupported_content", 400)
	}
	for _, part := range parts {
		if part.Type != "text" {
			return "", proxy.Failure("unsupported_content", 400)
		}
		value += part.Text
	}
	return value, nil
}

// assistantParts 是模型回合的工具调用：保留客户端回传的不透明签名，
// 缺失的签名由 fillUnsignedSignatures 在缓存回填之后统一补齐。
func assistantParts(message chatMessage) ([]object, error) {
	parts := []object{}
	for _, call := range message.Calls {
		if !safeString(call.ID, 256) || !safeString(call.Function.Name, 128) || call.Type != "function" {
			return nil, proxy.Failure("invalid_request", 400)
		}
		var args object
		if json.Unmarshal([]byte(call.Function.Arguments), &args) != nil || args == nil {
			return nil, proxy.Failure("invalid_request", 400)
		}
		part := object{"functionCall": object{"id": call.ID, "name": call.Function.Name, "args": args}}
		// 客户端带回的签名原样回传；缺失的签名由 fillUnsignedSignatures 在
		// 缓存回填之后统一补上，避免提前写入标记而挡住缓存里的真实签名。
		if signature := toolSignature(call.Extra); signature != "" {
			part["thoughtSignature"] = signature
		}
		parts = append(parts, part)
	}
	return parts, nil
}

// toolSignature 读取 Chat 客户端回传的 Google 扩展签名。
func toolSignature(raw json.RawMessage) string {
	var extra struct {
		Google struct {
			Signature string `json:"thought_signature"`
		} `json:"google"`
	}
	if json.Unmarshal(raw, &extra) != nil || len(extra.Google.Signature) > 65536 {
		return ""
	}
	return extra.Google.Signature
}

// sessionID 用首条用户文本的哈希作为会话标识：同一会话每一轮保持同一个，
// 与参考实现一致；没有文本时回退为随机值。
func sessionID(firstUser string) string {
	if firstUser == "" {
		return "-" + strconv.FormatUint(randomUint64(), 10)
	}
	sum := sha256.Sum256([]byte(firstUser))
	value := binary.BigEndian.Uint64(sum[:8]) & 0x7fffffffffffffff
	return "-" + strconv.FormatUint(value, 10)
}

// randomHex 生成指定字节数的随机十六进制串。
func randomHex(size int) string {
	buffer := make([]byte, size)
	if _, err := rand.Read(buffer); err != nil {
		return strings.Repeat("0", size*2)
	}
	return hex.EncodeToString(buffer)
}

// randomUint64 生成一个非负随机数，用作无文本会话的标识。
func randomUint64() uint64 {
	var buffer [8]byte
	if _, err := rand.Read(buffer[:]); err != nil {
		return 0
	}
	return binary.BigEndian.Uint64(buffer[:]) & 0x7fffffffffffffff
}
