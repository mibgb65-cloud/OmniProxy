// Package agentprotocol 实现文本与工具调用的 Responses／Messages 协议转换。
package agentprotocol

import (
	"encoding/json"
	"errors"
	"fmt"
)

var ErrUnsupported = errors.New("protocol: unsupported_content")
var ErrUnsupportedImage = fmt.Errorf("%w: image_content", ErrUnsupported)

type Object = map[string]any
type Call struct{ ID, Name, Arguments, Kind string }
type Result struct {
	ID, Model, Text, Finish   string
	Calls                     []Call
	InputTokens, OutputTokens int
}

func text(v any) string   { s, _ := v.(string); return s }
func object(v any) Object { m, _ := v.(map[string]any); return m }
func array(v any) []any   { a, _ := v.([]any); return a }
func encode(v any) []byte { b, _ := json.Marshal(v); return b }
func content(v any) (string, error) {
	if s, ok := v.(string); ok {
		return s, nil
	}
	output := ""
	for _, entry := range array(v) {
		part := object(entry)
		switch text(part["type"]) {
		case "text", "input_text", "output_text":
			output += text(part["text"])
		case "image", "input_image", "image_url":
			// 图片由 chatImage 单独转换；纯文本入口（system 指令、自定义工具输入）
			// 无法承载图片，必须显式失败而不是静默丢弃。
			return "", ErrUnsupportedImage
		default:
			return "", ErrUnsupported
		}
	}
	if v != nil && array(v) == nil {
		return "", ErrUnsupported
	}
	return output, nil
}
