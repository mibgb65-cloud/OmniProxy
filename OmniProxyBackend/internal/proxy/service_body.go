package proxy

import (
	"compress/gzip"
	"compress/zlib"
	"github.com/klauspost/compress/zstd"
	"io"
	"net/http"
	"strings"
)

func copyHeader(dst, src http.Header) {
	for key, values := range src {
		for _, value := range values {
			dst.Add(key, value)
		}
	}
}

func removeHopHeaders(header http.Header) {
	for _, key := range []string{
		"Connection",
		"Proxy-Connection",
		"Keep-Alive",
		"Proxy-Authenticate",
		"Proxy-Authorization",
		"Te",
		"Trailer",
		"Transfer-Encoding",
		"Upgrade",
	} {
		header.Del(key)
	}
}

func removeClientIdentificationHeaders(header http.Header) {
	for _, key := range []string{
		"X-OmniProxy-Client",
		"X-Client-Name",
		"X-Source-Client",
	} {
		header.Del(key)
	}
}

func upstreamRespBody(resp *http.Response) io.Closer {
	if resp == nil {
		return nil
	}
	return resp.Body
}

func responseHeaders(resp *http.Response) http.Header {
	if resp == nil {
		return nil
	}
	return resp.Header
}

func readProxyRequestBody(body io.ReadCloser, contentEncoding string) ([]byte, bool, error) {
	if body == nil {
		return nil, false, nil
	}
	defer closeBody(body)

	reader, decoded, err := decodedRequestBodyReader(body, contentEncoding)
	if err != nil {
		return nil, false, err
	}
	if decodedCloser, ok := reader.(io.Closer); ok {
		defer closeBody(decodedCloser)
	}

	limited := io.LimitReader(reader, maxProxyRequestBodyBytes+1)
	data, err := io.ReadAll(limited)
	if err != nil {
		return nil, decoded, err
	}
	if len(data) > maxProxyRequestBodyBytes {
		return nil, decoded, errRequestBodyTooLarge
	}
	return data, decoded, nil
}

func readProxyResponseBody(body io.Reader) ([]byte, error) {
	limited := io.LimitReader(body, maxProxyResponseBodyBytes+1)
	data, err := io.ReadAll(limited)
	if err != nil {
		return nil, err
	}
	if len(data) > maxProxyResponseBodyBytes {
		return nil, errResponseBodyTooLarge
	}
	return data, nil
}

func decodedRequestBodyReader(body io.Reader, contentEncoding string) (io.Reader, bool, error) {
	encoding := strings.ToLower(strings.TrimSpace(contentEncoding))
	if encoding == "" || encoding == "identity" {
		return body, false, nil
	}
	parts := strings.Split(encoding, ",")
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}
	if len(parts) != 1 {
		return body, false, nil
	}
	switch parts[0] {
	case "zstd":
		reader, err := zstd.NewReader(body)
		if err != nil {
			return nil, false, err
		}
		return reader.IOReadCloser(), true, nil
	case "gzip":
		reader, err := gzip.NewReader(body)
		if err != nil {
			return nil, false, err
		}
		return reader, true, nil
	case "deflate":
		reader, err := zlib.NewReader(body)
		if err != nil {
			return nil, false, err
		}
		return reader, true, nil
	default:
		return body, false, nil
	}
}

func closeBody(body io.Closer) {
	if body != nil {
		_ = body.Close()
	}
}

type flushWriter struct {
	writer  io.Writer
	flusher http.Flusher
}

func (w flushWriter) Write(p []byte) (int, error) {
	n, err := w.writer.Write(p)
	w.flusher.Flush()
	return n, err
}
