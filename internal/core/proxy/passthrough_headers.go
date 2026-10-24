package proxy

import "net/http"

// passthroughHopHeaders 是不跨跳转转发的连接级头部（RFC 7230 §6.1）。
var passthroughHopHeaders = []string{
	"Connection",
	"Proxy-Connection",
	"Keep-Alive",
	"Proxy-Authenticate",
	"Proxy-Authorization",
	"Te",
	"Trailer",
	"Transfer-Encoding",
	"Upgrade",
}

// localOnlyHeaders 是本网关自己的归属标记与接入凭据，绝不透传给上游。
var localOnlyHeaders = []string{
	"X-OmniProxy-Key",
	"X-OmniProxy-Client",
	"X-Client-Name",
	"X-Source-Client",
}

// replacedAuthHeaders 是每次尝试由网关重写的鉴权头，客户端自带的同类头先删除。
var replacedAuthHeaders = []string{"Authorization", "X-Api-Key", "Api-Key", "X-Goog-Api-Key", "ChatGPT-Account-Id"}

// cloneHeader 复制客户端头部，避免转发过程修改原请求。
func cloneHeader(src http.Header) http.Header {
	if src == nil {
		return http.Header{}
	}
	dst := make(http.Header, len(src))
	for key, values := range src {
		copied := make([]string, len(values))
		copy(copied, values)
		dst[key] = copied
	}
	return dst
}

// copyPassthroughHeaders 复刻 v1 的头部透传：除跳转级、网关自有标记外全部保留。
func copyPassthroughHeaders(dst, src http.Header) {
	for key, values := range src {
		if skipPassthroughHeader(key) {
			continue
		}
		for _, value := range values {
			dst.Add(key, value)
		}
	}
}

func skipPassthroughHeader(key string) bool {
	return containsFold(passthroughHopHeaders, key) || containsFold(localOnlyHeaders, key)
}

// applyAttemptHeaders 用本次尝试的鉴权头替换客户端同名头；其余头部保持不变。
func applyAttemptHeaders(dst, attempt http.Header) {
	for _, key := range replacedAuthHeaders {
		dst.Del(key)
	}
	for key, values := range attempt {
		for _, value := range values {
			dst.Add(key, value)
		}
	}
}

// clearAttemptHeaders 在请求结束后抹掉内存中的令牌副本，避免随请求对象滞留。
func clearAttemptHeaders(header http.Header) {
	for key, values := range header {
		for index := range values {
			values[index] = ""
		}
		header[key] = values
	}
	for key := range header {
		header.Del(key)
	}
}

// removePassthroughResponseHeaders 写入客户端前去掉上游连接级头部。
func removePassthroughResponseHeaders(header http.Header) {
	for _, key := range append(append([]string{}, passthroughHopHeaders...), "Content-Length") {
		header.Del(key)
	}
}

func containsFold(values []string, wanted string) bool {
	for _, value := range values {
		if http.CanonicalHeaderKey(value) == http.CanonicalHeaderKey(wanted) {
			return true
		}
	}
	return false
}
