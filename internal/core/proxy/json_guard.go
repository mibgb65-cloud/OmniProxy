package proxy

import (
	"bytes"
	"encoding/json"
	"io"
	"unicode/utf8"
)

// validateJSON 拒绝重复字段和过深嵌套，避免代理与上游对同一正文产生不同解释。
func validateJSON(body []byte) error {
	return validateJSONSize(body, MaxRequestBytes)
}

func validateJSONSize(body []byte, limit int) error {
	if len(body) == 0 || len(body) > limit || !utf8.Valid(body) {
		return Failure("invalid_request", 400)
	}
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.UseNumber()
	if err := scanJSON(decoder, 0); err != nil {
		return err
	}
	if _, err := decoder.Token(); err != io.EOF {
		return Failure("invalid_request", 400)
	}
	return nil
}

func scanJSON(decoder *json.Decoder, depth int) error {
	if depth > 32 {
		return Failure("invalid_request", 400)
	}
	token, err := decoder.Token()
	if err != nil {
		return Failure("invalid_request", 400)
	}
	delim, nested := token.(json.Delim)
	if !nested {
		return nil
	}
	if delim != '{' && delim != '[' {
		return Failure("invalid_request", 400)
	}
	seen := make(map[string]bool)
	for decoder.More() {
		if delim == '{' {
			key, err := decoder.Token()
			name, ok := key.(string)
			if err != nil || !ok || seen[name] {
				return Failure("invalid_request", 400)
			}
			seen[name] = true
		}
		if err := scanJSON(decoder, depth+1); err != nil {
			return err
		}
	}
	end, err := decoder.Token()
	if err != nil || (delim == '{' && end != json.Delim('}')) || (delim == '[' && end != json.Delim(']')) {
		return Failure("invalid_request", 400)
	}
	return nil
}
