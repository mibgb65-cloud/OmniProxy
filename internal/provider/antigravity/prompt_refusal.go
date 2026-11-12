package antigravity

import (
	"sort"
	"strings"
)

// zeroWidth 是插入被拒行内部的 U+200B：模型按词读取，Google 的行匹配看不到它。
const zeroWidth = "\u200b"

// refusals 是实测会让 Antigravity 以「Resource has been exhausted」拒绝整个请求的
// system instruction 行（与剩余额度无关，任意模型）：Claude Code 的计费行、
// Claude Agent SDK 的身份行、Codex GPT-5 的两条身份行、Hermes Agent 的旧人设。
var refusals = []string{
	"x-anthropic-billing-header:",
	"built on anthropic's claude agent sdk",
	"you are codex, an agent based on gpt-5",
	"you are codex, a coding agent based on gpt-5",
	"hermes agent, an intelligent ai assistant created by nous research",
}

// turnsAway 判断 system 是否携带上述行；判断前去掉已有的零宽空格，
// 因此被旧做法（每词只插一个）部分打散的 prompt 仍能识别。
func turnsAway(system string) bool {
	if system == "" {
		return false
	}
	low := asciiLower(stripMarks(system))
	for _, line := range refusals {
		if strings.Contains(low, line) {
			return true
		}
	}
	return false
}

// cloaked 把命中的行按词打散：每个长度 ≥3 的 ASCII 字母/数字连段，在第 1、第 2 个
// 字符后各插入一个零宽空格。未命中时原样返回；已打散或部分打散的输入重建为规范形态，
// 因此同一输入重复调用结果不变。
func cloaked(system string) string {
	if !turnsAway(system) {
		return system
	}
	stripped, at := withoutMarks(system)
	low := asciiLower(stripped)
	var spans [][2]int
	for _, line := range refusals {
		for next := 0; ; {
			index := strings.Index(low[next:], line)
			if index < 0 {
				break
			}
			from := next + index
			spans = append(spans, [2]int{from, from + len(line)})
			next = from + len(line)
		}
	}
	if len(spans) == 0 {
		return system
	}
	sort.Slice(spans, func(i, j int) bool { return spans[i][0] < spans[j][0] })
	merged := spans[:1]
	for _, span := range spans[1:] {
		last := &merged[len(merged)-1]
		if span[0] <= last[1] {
			if span[1] > last[1] {
				last[1] = span[1]
			}
			continue
		}
		merged = append(merged, span)
	}
	var out strings.Builder
	out.Grow(len(system) + len(system)/8)
	previous := 0
	for _, span := range merged {
		start, end := at[span[0]], at[span[1]]
		out.WriteString(system[previous:start])
		out.WriteString(brokenText(stripped, span[0], span[1]))
		previous = end
	}
	out.WriteString(system[previous:])
	return out.String()
}

// brokenText 是 text[from:to] 的规范形态：每个 ≥3 字符的 ASCII 字母/数字连段，
// 在第一、第二个字符之后各插入一个零宽空格；其余字符原样复制。
func brokenText(text string, from, to int) string {
	var out strings.Builder
	for index := from; index < to; {
		end := index
		for end < to && isAlnum(text[end]) {
			end++
		}
		switch {
		case end-index >= 3:
			out.WriteString(text[index : index+1])
			out.WriteString(zeroWidth)
			out.WriteString(text[index+1 : index+2])
			out.WriteString(zeroWidth)
			out.WriteString(text[index+2 : end])
		case end > index:
			out.WriteString(text[index:end])
		default:
			out.WriteString(text[index : index+1])
			end = index + 1
		}
		index = end
	}
	return out.String()
}

// withoutMarks 去掉零宽空格，并给出每个剩余字节在原串中的下标（末尾多一项 len(s)）。
func withoutMarks(s string) (string, []int) {
	if !strings.Contains(s, zeroWidth) {
		at := make([]int, len(s)+1)
		for i := range at {
			at[i] = i
		}
		return s, at
	}
	var out strings.Builder
	out.Grow(len(s))
	at := make([]int, 0, len(s)+1)
	for i := 0; i < len(s); {
		if strings.HasPrefix(s[i:], zeroWidth) {
			i += len(zeroWidth)
			continue
		}
		at = append(at, i)
		out.WriteByte(s[i])
		i++
	}
	at = append(at, len(s))
	return out.String(), at
}

// stripMarks 仅去掉零宽空格，供判定使用。
func stripMarks(s string) string {
	if !strings.Contains(s, zeroWidth) {
		return s
	}
	return strings.ReplaceAll(s, zeroWidth, "")
}

// asciiLower 只把小写化 ASCII 字母，保持每个字节的位置不变。
func asciiLower(s string) string {
	needs := false
	for i := 0; i < len(s); i++ {
		if s[i] >= 'A' && s[i] <= 'Z' {
			needs = true
			break
		}
	}
	if !needs {
		return s
	}
	buffer := []byte(s)
	for i, c := range buffer {
		if c >= 'A' && c <= 'Z' {
			buffer[i] = c + 'a' - 'A'
		}
	}
	return string(buffer)
}

func isAlnum(c byte) bool {
	return c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9'
}
