package dumate

import (
	"encoding/json"
	"errors"
	"math"
	"strconv"
	"strings"
	"unicode"
)

// errMalformed 表示控制台返回了不能安全解析的字段。
var errMalformed = errors.New("dumate: malformed field")

const (
	maxAmount      = 1e12
	maxTextBytes   = 512
	maxPackEntries = 200
)

// amount 接受 JSON 数字或十进制字符串，拒绝 NaN、无穷与超限值。
type amount float64

func (a *amount) UnmarshalJSON(data []byte) error {
	text := strings.TrimSpace(string(data))
	if text == "" || text == "null" {
		*a = 0
		return nil
	}
	if text[0] == '"' {
		var decoded string
		if json.Unmarshal(data, &decoded) != nil {
			return errMalformed
		}
		text = strings.TrimSpace(decoded)
	}
	if text == "" {
		*a = 0
		return nil
	}
	value, err := strconv.ParseFloat(text, 64)
	if err != nil || math.IsNaN(value) || math.IsInf(value, 0) || math.Abs(value) > maxAmount {
		return errMalformed
	}
	*a = amount(value)
	return nil
}

// float 返回有限浮点值。
func (a amount) float() float64 { return float64(a) }

// textValue 接受字符串或数字，用于账号标识与时间字段。
type textValue string

func (t *textValue) UnmarshalJSON(data []byte) error {
	text := strings.TrimSpace(string(data))
	if text == "" || text == "null" {
		*t = ""
		return nil
	}
	if text[0] == '"' {
		var decoded string
		if json.Unmarshal(data, &decoded) != nil || len(decoded) > maxTextBytes {
			return errMalformed
		}
		for _, char := range decoded {
			if unicode.IsControl(char) {
				return errMalformed
			}
		}
		*t = textValue(decoded)
		return nil
	}
	var number json.Number
	if json.Unmarshal(data, &number) != nil || len(number.String()) > maxTextBytes {
		return errMalformed
	}
	*t = textValue(number.String())
	return nil
}

func (t textValue) string() string { return strings.TrimSpace(string(t)) }

// intValue 把数量字段转换为有界整数；非法或超限时返回 false。
func (a amount) intValue(limit int) (int, bool) {
	value := a.float()
	if value < 0 || value > float64(limit) || value != math.Trunc(value) {
		return 0, false
	}
	return int(value), true
}
