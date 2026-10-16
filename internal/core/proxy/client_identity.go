package proxy

// ValidClientUserAgent 限制非敏感身份头的长度和字符，禁止注入其他 HTTP 头。
func ValidClientUserAgent(value string) bool {
	if len(value) > 1024 {
		return false
	}
	for i := 0; i < len(value); i++ {
		if value[i] < 32 || value[i] > 126 {
			return false
		}
	}
	return true
}
