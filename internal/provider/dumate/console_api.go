package dumate

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const (
	userInfoPath    = "/api/dumate/user/info"
	tokenPath       = "/api/dumate/app/token/get"
	overviewPath    = "/api/dumate/points/quota_overview?timezone=Asia%2FShanghai&clientType=desktop&ignoreLoginBonus=true"
	loginBonusPath  = "/api/dumate/points/loginBonusInfo"
	claimBonusPath  = "/api/dumate/points/loginBonus"
	claimBody       = `{}`
	maxCheckinTimes = 100000
)

// Identity 是远端核对后的账号身份，不包含 Cookie。
type Identity struct {
	AccountID   string
	DisplayName string
}

// Token 是 gateway token 及其过期时间。
type Token struct {
	Value     string
	ExpiresAt time.Time
}

// Pack 是积分包；ExpireDate 为空表示不过期。
type Pack struct {
	Status      string
	TotalPoints float64
	ExpireDate  *int64
}

// Overview 是积分总览的公开数值。
type Overview struct {
	TotalPoints float64
	UsedPoints  float64
	Packs       []Pack
}

// LoginBonus 是签到状态；HasIssued 表示今天已经领取。
type LoginBonus struct {
	HasIssued   bool
	TotalTimes  int
	TotalPoints float64
}

// UserInfo 校验 Cookie 是否仍然有效，并返回远端身份。
func (c *Console) UserInfo(ctx context.Context, credentials Credentials) (Identity, error) {
	result, err := c.do(ctx, http.MethodGet, userInfoPath, credentials.Cookies, nil)
	if err != nil {
		return Identity{}, err
	}
	var payload struct {
		BCEAccountID textValue `json:"bceAccountId"`
		DisplayName  textValue `json:"displayName"`
	}
	if json.Unmarshal(result, &payload) != nil {
		return Identity{}, Failure(CodeInvalidResponse)
	}
	identity := Identity{AccountID: payload.BCEAccountID.string(), DisplayName: payload.DisplayName.string()}
	if identity.AccountID == "" {
		return Identity{}, Failure(CodeInvalidResponse)
	}
	return identity, nil
}

// GatewayToken 用 Cookie 换取 gateway token。
func (c *Console) GatewayToken(ctx context.Context, credentials Credentials) (Token, error) {
	result, err := c.do(ctx, http.MethodGet, tokenPath, credentials.Cookies, nil)
	if err != nil {
		return Token{}, err
	}
	var payload struct {
		Token     textValue `json:"token"`
		ExpiredAt textValue `json:"expired_at"`
	}
	if json.Unmarshal(result, &payload) != nil {
		return Token{}, Failure(CodeInvalidResponse)
	}
	value := payload.Token.string()
	if value == "" || len(value) > 8192 || !printable(value) {
		return Token{}, Failure(CodeInvalidResponse)
	}
	expires := parseExpiry(payload.ExpiredAt.string())
	return Token{Value: value, ExpiresAt: expires}, nil
}

// printable 拒绝空白与控制字符，防止令牌被拼进非法请求头。
func printable(value string) bool {
	for _, char := range value {
		if char < 0x21 || char > 0x7e {
			return false
		}
	}
	return true
}

// parseExpiry 接受 RFC3339、日期时间或 unix 秒／毫秒；无法解析时回退 1 小时。
func parseExpiry(raw string) time.Time {
	now := time.Now()
	if raw == "" {
		return now.Add(TokenTTL)
	}
	for _, layout := range []string{time.RFC3339Nano, time.RFC3339, "2006-01-02 15:04:05", "2006-01-02T15:04:05"} {
		if parsed, err := time.Parse(layout, raw); err == nil && parsed.After(now) {
			return parsed
		}
	}
	if value, ok := parseUnix(raw); ok {
		seconds := value
		if value > 1e12 {
			seconds = value / 1000
		}
		if seconds > now.Unix() {
			return time.Unix(seconds, 0)
		}
	}
	return now.Add(TokenTTL)
}

// parseUnix 解析十进制 unix 时间；非数字或越界时返回 false。
func parseUnix(raw string) (int64, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0, false
	}
	value, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || value <= 0 || value > 1e15 {
		return 0, false
	}
	return value, true
}

// Overview 读取积分总览；订阅与增量积分包合并处理。
func (c *Console) Overview(ctx context.Context, credentials Credentials) (Overview, error) {
	result, err := c.do(ctx, http.MethodGet, overviewPath, credentials.Cookies, nil)
	if err != nil {
		return Overview{}, err
	}
	var payload struct {
		TotalPoints  amount `json:"totalPoints"`
		UsedPoints   amount `json:"usedPoints"`
		Subscription []struct {
			Status      textValue `json:"status"`
			TotalPoints amount    `json:"totalPoints"`
			ExpireDate  amount    `json:"expireDate"`
		} `json:"subscription"`
		Incremental []struct {
			Status      textValue `json:"status"`
			TotalPoints amount    `json:"totalPoints"`
			ExpireDate  amount    `json:"expireDate"`
		} `json:"incremental"`
	}
	if json.Unmarshal(result, &payload) != nil {
		return Overview{}, Failure(CodeInvalidResponse)
	}
	overview := Overview{TotalPoints: payload.TotalPoints.float(), UsedPoints: payload.UsedPoints.float()}
	if len(payload.Subscription)+len(payload.Incremental) > maxPackEntries {
		return Overview{}, Failure(CodeInvalidResponse)
	}
	for _, pack := range payload.Subscription {
		overview.Packs = append(overview.Packs, convertPack(pack.Status, pack.TotalPoints, pack.ExpireDate))
	}
	for _, pack := range payload.Incremental {
		overview.Packs = append(overview.Packs, convertPack(pack.Status, pack.TotalPoints, pack.ExpireDate))
	}
	return overview, nil
}

func convertPack(status textValue, total amount, expire amount) Pack {
	pack := Pack{Status: strings.ToLower(status.string()), TotalPoints: total.float()}
	if expire.float() > 0 {
		seconds := int64(expire.float())
		pack.ExpireDate = &seconds
	}
	return pack
}

// LoginBonus 读取今日签到状态与累计天数。
func (c *Console) LoginBonus(ctx context.Context, credentials Credentials) (LoginBonus, error) {
	result, err := c.do(ctx, http.MethodGet, loginBonusPath, credentials.Cookies, nil)
	if err != nil {
		return LoginBonus{}, err
	}
	var payload struct {
		HasIssued   *bool  `json:"hasIssued"`
		TotalTimes  amount `json:"totalTimes"`
		TotalPoints amount `json:"totalPoints"`
	}
	if json.Unmarshal(result, &payload) != nil {
		return LoginBonus{}, Failure(CodeInvalidResponse)
	}
	bonus := LoginBonus{TotalPoints: payload.TotalPoints.float()}
	if payload.HasIssued != nil {
		bonus.HasIssued = *payload.HasIssued
	}
	if times, ok := payload.TotalTimes.intValue(maxCheckinTimes); ok {
		bonus.TotalTimes = times
	}
	return bonus, nil
}

// ClaimLoginBonus 提交签到；成功与否由调用方再用 LoginBonus 核对。
func (c *Console) ClaimLoginBonus(ctx context.Context, credentials Credentials) error {
	_, err := c.do(ctx, http.MethodPost, claimBonusPath, credentials.Cookies, []byte(claimBody))
	return err
}
