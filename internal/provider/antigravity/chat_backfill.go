package antigravity

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// pendingCall 是一条 assistant 工具调用的名字与原始参数（OpenAI 的 JSON 字符串）。
// tool 结果按 tool_call_id 回查它：客户端把图片换成占位文本后，路径只在这里还能找到。
type pendingCall struct {
	name      string
	arguments string
}

// binaryImagePlaceholder 是客户端把读到的二进制换成的占位文本前缀，后面紧跟 MIME。
const binaryImagePlaceholder = "Showing binary file of type "

// backfillFileLimit 是回读本地文件的上限，base64 之后仍在 chatImageLimit 以内。
const backfillFileLimit = 12 << 20

// backfillPathKeys 是已知客户端放置文件路径的参数名，大小写不敏感。
var backfillPathKeys = []string{"path", "file_path", "filepath", "file", "filename", "target_file", "targetfile"}

// backfillInlineImage 为「工具结果只剩二进制占位文本」补回像素。
//
// Antigravity 在网关模式（wireProtocol: openai）下把 read_file 读到的图片从 tool result
// 里剥掉，只留 `Showing binary file of type image/png`；图片本体仍在同一次调用的参数路径上。
// 网关与客户端同机，按该路径回读并转成 inlineData，模型才能看到像素。
//
// 边界：只认图片占位文本、绝对路径、普通文件、大小上限与内容嗅探；任何一步不满足都返回 nil，
// 保持原有纯文本语义，也不让请求失败。非图片内容不会被读出，因此不构成额外文件访问能力。
func backfillInlineImage(result, arguments string) object {
	if !strings.Contains(strings.TrimSpace(result), binaryImagePlaceholder+"image/") {
		return nil
	}
	for _, path := range callPathCandidates(arguments) {
		info, err := os.Stat(path)
		if err != nil || !info.Mode().IsRegular() || info.Size() <= 0 || info.Size() > backfillFileLimit {
			continue
		}
		data, err := os.ReadFile(path)
		if err != nil || len(data) == 0 || len(data) > backfillFileLimit {
			continue
		}
		media := strings.ToLower(strings.TrimSpace(strings.SplitN(http.DetectContentType(data), ";", 2)[0]))
		if !chatImageMedia(media) {
			continue
		}
		return object{"inlineData": object{"mimeType": media, "data": base64.StdEncoding.EncodeToString(data)}}
	}
	return nil
}

// callPathCandidates 从工具调用参数里取文件路径候选；参数不是 JSON 对象时返回空。
func callPathCandidates(arguments string) []string {
	var args map[string]json.RawMessage
	if json.Unmarshal([]byte(arguments), &args) != nil {
		return nil
	}
	for _, key := range backfillPathKeys {
		for name, raw := range args {
			if strings.ToLower(strings.TrimSpace(name)) != key {
				continue
			}
			return pathCandidates(raw)
		}
	}
	return nil
}

// pathCandidates 把参数值还原成绝对路径候选。
//
// 各客户端的编码层次不一样：有的一层 JSON 字符串，有的把路径再包一层引号，甚至再转义一次
// 反斜杠。这里按「原文 → 解引用一次」两种来源各自展开，并补上双反斜杠还原的变体；
// 具体哪个存在由调用方 os.Stat 决定，这里不做文件系统访问。
func pathCandidates(raw json.RawMessage) []string {
	var value string
	if json.Unmarshal(raw, &value) != nil {
		return nil
	}
	out := []string{}
	seen := map[string]bool{}
	add := func(candidate string) {
		candidate = strings.TrimSpace(candidate)
		// 客户端把路径又包了一层引号时剥掉；这不是 JSON 转义，反斜杠保持原样。
		if len(candidate) >= 2 && strings.HasPrefix(candidate, `"`) && strings.HasSuffix(candidate, `"`) {
			candidate = strings.TrimSpace(candidate[1 : len(candidate)-1])
		}
		if candidate == "" {
			return
		}
		for _, variant := range []string{candidate, strings.ReplaceAll(candidate, `\\`, `\`)} {
			variant = filepath.Clean(variant)
			if !filepath.IsAbs(variant) || seen[variant] {
				continue
			}
			seen[variant] = true
			out = append(out, variant)
		}
	}
	add(value)
	var inner string
	if json.Unmarshal([]byte(strings.TrimSpace(value)), &inner) == nil {
		add(inner)
	}
	return out
}
