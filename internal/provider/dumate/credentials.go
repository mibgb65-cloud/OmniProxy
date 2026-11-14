package dumate

import (
	"encoding/json"
	"strings"
	"unicode"
	"unicode/utf8"
)

const (
	// AuthMethod 是账号表的认证类型；Cookie 登录态不能伪装成 API Key。
	AuthMethod = "dumate_cookie"
	// SourceDesktop 表示只读复用了搭子桌面端登录文件。
	SourceDesktop = "desktop"
	// SourceBrowser 表示用户从浏览器 DevTools 复制了 Cookie 请求头。
	SourceBrowser = "browser"
	// SourceManual 表示用户手动粘贴了 Cookie 字符串。
	SourceManual = "manual"

	maxCredentialBytes = 64 << 10
	maxSubjectBytes    = 128
	maxNameRunes       = 80
)

// Credentials 是受保护存储中的 Cookie 登录态；只允许通过回调短暂解密使用。
type Credentials struct {
	Cookies   []Cookie `json:"cookies"`
	Source    string   `json:"source"`
	AccountID string   `json:"accountId,omitempty"`
	Name      string   `json:"name,omitempty"`
}

// Valid 校验登录态结构与来源；身份字段由远端 user/info 核对后填入。
func (c Credentials) Valid() bool {
	if Validate(c.Cookies) != nil {
		return false
	}
	switch c.Source {
	case SourceDesktop, SourceBrowser, SourceManual:
	default:
		return false
	}
	if len(c.AccountID) > maxSubjectBytes || !utf8.ValidString(c.AccountID) || strings.TrimSpace(c.AccountID) != c.AccountID {
		return false
	}
	if len(c.Name) > maxNameRunes*4 || !utf8.ValidString(c.Name) {
		return false
	}
	for _, char := range c.Name {
		if unicode.IsControl(char) {
			return false
		}
	}
	return true
}

// Header 生成控制台请求使用的 Cookie 头。
func (c Credentials) Header() string {
	return HeaderFor(c.Cookies, hostOnly(ConsoleBase))
}

// Marshal 序列化为受保护存储的正文；不包含任何调试信息。
func (c Credentials) Marshal() ([]byte, error) {
	if !c.Valid() {
		return nil, Failure(CodeCredentialUnavailable)
	}
	data, err := json.Marshal(c)
	if err != nil || len(data) > maxCredentialBytes {
		return nil, Failure(CodeCredentialUnavailable)
	}
	return data, nil
}

// ParseCredentials 解析受保护存储中的正文，拒绝超限或结构不符的数据。
func ParseCredentials(data []byte) (Credentials, error) {
	if len(data) == 0 || len(data) > maxCredentialBytes {
		return Credentials{}, Failure(CodeCredentialUnavailable)
	}
	var credentials Credentials
	if json.Unmarshal(data, &credentials) != nil || !credentials.Valid() {
		return Credentials{}, Failure(CodeCredentialUnavailable)
	}
	credentials.Cookies = CloneCookies(credentials.Cookies)
	return credentials, nil
}

// hostOnly 去掉控制台基址的协议与路径，得到 Cookie 匹配用的主机名。
func hostOnly(base string) string {
	base = strings.TrimPrefix(base, "https://")
	base = strings.TrimPrefix(base, "http://")
	if index := strings.IndexAny(base, "/?#"); index >= 0 {
		base = base[:index]
	}
	return base
}
