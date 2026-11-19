// Package xlsx 用标准库写出单工作表的 XLSX 文件，不引入第三方表格依赖。
package xlsx

import (
	"math"
	"strconv"
	"strings"
)

// MaxCellChars 是 Excel 单元格的字符上限（按 UTF-16 码元计数）。
const MaxCellChars = 32767

// Cell 是一行中的一个单元格；Number 非空时按数值写入，否则按文本写入。
type Cell struct {
	Text   string
	Number string
}

// FitCell 按 Excel 单元格上限截断文本，并报告是否发生截断。
// 代理层用它决定「内容截断」列，避免超长正文写成无效工作簿。
func FitCell(text string) (string, bool) {
	units := 0
	for index, char := range text {
		width := 1
		if char > 0xFFFF {
			width = 2
		}
		if units+width > MaxCellChars {
			return text[:index], true
		}
		units += width
	}
	return text, false
}

// sanitizeText 去掉 XML 1.0 不允许的控制字符；换行、制表符保留。
func sanitizeText(text string) string {
	return strings.Map(func(char rune) rune {
		switch {
		case char == '\t' || char == '\n' || char == '\r':
			return char
		case char < 0x20:
			return -1
		case char >= 0xD800 && char <= 0xDFFF:
			return -1
		default:
			return char
		}
	}, text)
}

// columnName 把 0 基列索引转成 A、B、…、Z、AA 的单元格列名。
func columnName(index int) string {
	name := ""
	for index >= 0 {
		name = string(rune('A'+index%26)) + name
		index = index/26 - 1
	}
	return name
}

// validNumber 只接受有限十进制数值，避免把任意字符串写进 <v>。
func validNumber(value string) bool {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" || strings.ContainsAny(strings.ToLower(trimmed), "xpi") {
		return false
	}
	number, err := strconv.ParseFloat(trimmed, 64)
	if err != nil || math.IsNaN(number) || math.IsInf(number, 0) {
		return false
	}
	return true
}

// sanitizeSheetName 修正工作表名的非法字符与长度，保持可读。
func sanitizeSheetName(name string) string {
	name = strings.TrimSpace(name)
	cleaned := strings.Map(func(char rune) rune {
		if strings.ContainsRune(`[]:*?/\`, char) || char < 0x20 {
			return ' '
		}
		return char
	}, name)
	cleaned = strings.Trim(cleaned, "'")
	if cleaned == "" {
		return "Sheet1"
	}
	runes := []rune(cleaned)
	if len(runes) > 31 {
		cleaned = string(runes[:31])
	}
	return cleaned
}
