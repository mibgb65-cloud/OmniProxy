package workbuddyai

import (
	"bytes"
	"context"
	"encoding/json"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"omniproxy/internal/core/proxy"
)

const (
	// maxCreditsBytes 限制额度正文；资源总览只是少量 JSON。
	maxCreditsBytes = 256 << 10
	// maxQuotaNodes / maxQuotaDepth / maxQuotaArray 限制容错遍历的规模。
	maxQuotaNodes = 20000
	maxQuotaDepth = 8
	maxQuotaArray = 2000
	// maxQuotaAmount 拒绝异常大的额度数字。
	maxQuotaAmount = 1e12
)

// beijingZone 是上游计费接口使用的时间（UTC+8，无时区后缀）。
var beijingZone = time.FixedZone("UTC+8", 8*3600)

// CreditWindow 是一个公开的额度窗口；字段缺失表示未知，不补 0。
type CreditWindow struct {
	Name      string  `json:"name"`
	Used      float64 `json:"used"`
	Amount    float64 `json:"amount,omitempty"`
	Limit     float64 `json:"limit,omitempty"`
	Display   string  `json:"display,omitempty"`
	Unit      string  `json:"unit,omitempty"`
	Aside     bool    `json:"aside,omitempty"`
	ResetsAt  string  `json:"resetsAt,omitempty"`
	RunsOutAt string  `json:"runsOutAt,omitempty"`
	RunsOut   float64 `json:"runsOut,omitempty"`
	Prepaid   bool    `json:"prepaid,omitempty"`
}

// Credits 是账号的 Credits 快照：计划内、额外与合计窗口，另有可选的预付余额。
type Credits struct {
	Plan      string
	Windows   []CreditWindow
	FetchedAt time.Time
}

// ReadCredits 读取账号的 Credits 资源总览。上游结构不固定，因此按字段名容错解析：
// 只有真的读到可解析数字才输出窗口，任何未知都返回错误而不是当成 0。
func (c *Client) ReadCredits(ctx context.Context, cred Credentials) (credits Credits, result error) {
	if !cred.Valid() {
		return Credits{}, Failure(CodeCredentialUnavailable)
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, c.base+creditsPath, nil)
	if err != nil {
		return Credits{}, proxy.UpstreamFailure(err)
	}
	c.applySign(request, cred)
	defer request.Header.Del("Authorization")
	response, err := c.http.Do(request)
	if err != nil {
		return Credits{}, proxy.UpstreamFailure(err)
	}
	defer func() {
		if err := response.Body.Close(); err != nil && result == nil {
			result = Failure(CodeUpstreamUnavailable)
		}
	}()
	body, err := readBounded(response.Body, maxCreditsBytes)
	if err != nil {
		return Credits{}, err
	}
	if failure := businessStatus(response.StatusCode, body); failure != nil {
		return Credits{}, failure
	}
	data, err := decodeBusiness(body)
	if err != nil {
		return Credits{}, err
	}
	return summarizeCredits(data, cred.Plan)
}

// creditTotals 是一类额度的已用与总量。
type creditTotals struct {
	used  float64
	total float64
}

// quotaSummary 是从资源总览里容错收集到的数字。
type quotaSummary struct {
	plan, extra creditTotals
	// left 是无总量包（预付/钱包）的剩余合计，bags 是这类包的数量。
	left float64
	bags int
	// isPaid 用于账号没有套餐名时给出可展示的计划名。
	isPaid bool
	// planResetsAt 是计划包的周期结束时间。
	planResetsAt string
	// planRunsOutAt/extraRunsOutAt 是两类额度各自最早的到期时间与对应数量。
	planRunsOutAt, extraRunsOutAt string
	planRunsOut, extraRunsOut     float64
}

// summarizeCredits 把资源总览换算成窗口列表。
func summarizeCredits(data json.RawMessage, plan string) (Credits, error) {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	var root any
	if decoder.Decode(&root) != nil {
		return Credits{}, Failure(CodeInvalidResponse)
	}
	walker := &quotaWalk{nodes: maxQuotaNodes}
	if err := walker.walk(root, 0, false); err != nil {
		return Credits{}, err
	}
	summary, err := walker.summarize()
	if err != nil {
		return Credits{}, err
	}
	windows := summary.windows()
	if len(windows) == 0 {
		return Credits{}, Failure(CodeInvalidResponse)
	}
	return Credits{Plan: planLabel(plan, summary.isPaid), Windows: windows, FetchedAt: time.Now().UTC()}, nil
}

// quotaWalk 有界遍历任意结构的资源总览。
//
// CreditGroups 与 Packages 可能同时出现：前者是上游官方的口径（plan_base 等），
// 后者是同一批包的明细。两者都有总量时只按 CreditGroups 求和，避免重复计数；
// Packages 仍用于预付余额、周期与到期时间。
type quotaWalk struct {
	groups       []map[string]any
	packages     []map[string]any
	subscription string
	isPaid       bool
	nodes        int
}

func (w *quotaWalk) walk(value any, depth int, inGroups bool) error {
	if depth > maxQuotaDepth || w.nodes <= 0 {
		return Failure(CodeInvalidResponse)
	}
	w.nodes--
	switch typed := value.(type) {
	case map[string]any:
		if w.subscription == "" {
			w.subscription = lookupString(typed, "subscriptionpackagecode")
		}
		if paid, found := lookupBool(typed, "ispaiduser"); found {
			w.isPaid = paid
		}
		if isCapacityObject(typed) {
			if inGroups {
				w.groups = append(w.groups, typed)
			} else {
				w.packages = append(w.packages, typed)
			}
			return nil
		}
		for key, nested := range typed {
			if err := w.walk(nested, depth+1, inGroups || normalizeKey(key) == "creditgroups"); err != nil {
				return err
			}
		}
	case []any:
		if len(typed) > maxQuotaArray {
			return Failure(CodeInvalidResponse)
		}
		for _, nested := range typed {
			if err := w.walk(nested, depth+1, inGroups); err != nil {
				return err
			}
		}
	}
	return nil
}

// isCapacityObject 判断一个对象是否是额度包：至少带一个周期容量字段。
func isCapacityObject(object map[string]any) bool {
	for key := range object {
		switch normalizeKey(key) {
		case "cycletotalcapacity", "cycleusedcapacity", "cycleremaincapacity", "cyclecapacityremainprecise", "cyclecapacityremain", "remaincapacity":
			return true
		}
	}
	return false
}

// summarize 汇总包字段；一个数字都读不到时返回无法识别。
func (w *quotaWalk) summarize() (*quotaSummary, error) {
	summary := &quotaSummary{isPaid: w.isPaid}
	parsed := 0
	// 份额：CreditGroups 有总量时以它为准，否则用 Packages。
	counted := w.groups
	if !hasGroupTotals(w.groups) {
		counted = w.packages
	}
	for _, object := range counted {
		total, hasTotal := lookupNumber(object, "cycletotalcapacity")
		used, hasUsed := lookupNumber(object, "cycleusedcapacity", "used", "usedcapacity", "cyclecapacityused")
		if hasTotal {
			parsed++
		}
		if hasUsed {
			parsed++
		}
		if !hasTotal || total <= 0 {
			continue
		}
		if w.isPlan(object) {
			summary.plan.total += total
			summary.plan.used += used
		} else {
			summary.extra.total += total
			summary.extra.used += used
		}
	}
	now := time.Now()
	// 预付余额、周期结束与到期时间只看 Packages：它们是包级明细。
	for _, object := range w.packages {
		total, hasTotal := lookupNumber(object, "cycletotalcapacity")
		used, _ := lookupNumber(object, "cycleusedcapacity", "used", "usedcapacity", "cyclecapacityused")
		remain, hasRemain := lookupNumber(object, "cycleremaincapacity", "cyclecapacityremainprecise", "cyclecapacityremain", "remaincapacity")
		if hasRemain {
			parsed++
		}
		if !hasTotal || total <= 0 {
			// 没有总量的包是预付/钱包余额：不参与百分比，只展示剩余数量。
			if hasRemain {
				summary.left += remain
				summary.bags++
			}
			continue
		}
		left := remainOf(hasRemain, remain, total, used)
		if w.isPlan(object) {
			if reset := cycleEnd(object); reset != "" && (summary.planResetsAt == "" || reset < summary.planResetsAt) {
				summary.planResetsAt = reset
			}
			summary.planRunsOutAt, summary.planRunsOut = earliestExpiry(summary.planRunsOutAt, summary.planRunsOut, object, left, now)
		} else {
			summary.extraRunsOutAt, summary.extraRunsOut = earliestExpiry(summary.extraRunsOutAt, summary.extraRunsOut, object, left, now)
		}
	}
	if parsed == 0 {
		return nil, Failure(CodeInvalidResponse)
	}
	return summary, nil
}

// hasGroupTotals 报告 CreditGroups 里是否有可用的总量。
func hasGroupTotals(groups []map[string]any) bool {
	for _, object := range groups {
		if total, found := lookupNumber(object, "cycletotalcapacity"); found && total > 0 {
			return true
		}
	}
	return false
}

// isPlan 判定额度是否属于计划内：Kind 为 plan_base，或包编码等于订阅包编码。
// 其余（plan_bonus、addon、benefit、活动）都算额外额度。
func (w *quotaWalk) isPlan(object map[string]any) bool {
	if normalizeKey(lookupString(object, "kind", "creditgroupkind", "groupkind")) == "planbase" {
		return true
	}
	code := lookupString(object, "packagecode")
	return code != "" && w.subscription != "" && code == w.subscription
}

// remainOf 优先用上游剩余量，否则用 total-used（不小于 0）。
func remainOf(hasRemain bool, remain, total, used float64) float64 {
	if hasRemain {
		return remain
	}
	return math.Max(0, total-used)
}

// earliestExpiry 记录有剩余、且尚未过期的包中最早到期的时间与数量；同一时刻累加。
func earliestExpiry(at string, amount float64, object map[string]any, remain float64, now time.Time) (string, float64) {
	if remain <= 0 {
		return at, amount
	}
	end, found := packageExpiry(object)
	if !found || !end.After(now) {
		return at, amount
	}
	text := end.UTC().Format(time.RFC3339)
	switch {
	case at == "" || text < at:
		return text, remain
	case text == at:
		return at, amount + remain
	}
	return at, amount
}

// windows 按插件语义组装窗口：两类都有总量时给三个窗口，否则一个合计窗口；
// 有预付余额时追加 Prepaid credits。
func (s *quotaSummary) windows() []CreditWindow {
	var windows []CreditWindow
	switch {
	case s.plan.total > 0 && s.extra.total > 0:
		plan := percentageWindow("Credits", s.plan)
		plan.ResetsAt = s.planResetsAt
		plan.RunsOutAt, plan.RunsOut = s.planRunsOutAt, roundCredit(s.planRunsOut)
		// 两个分项都不单独停止账号，只有合计窗口代表账号整体。
		plan.Aside = true
		extra := percentageWindow("Extra credits", s.extra)
		extra.RunsOutAt, extra.RunsOut = s.extraRunsOutAt, roundCredit(s.extraRunsOut)
		extra.Aside = true
		windows = append(windows, plan, extra, totalWindow(plan, extra))
	case s.plan.total+s.extra.total > 0:
		combined := creditTotals{used: s.plan.used + s.extra.used, total: s.plan.total + s.extra.total}
		window := percentageWindow("Credits", combined)
		window.ResetsAt = s.planResetsAt
		window.RunsOutAt, window.RunsOut = s.planRunsOutAt, roundCredit(s.planRunsOut)
		if window.RunsOutAt == "" || s.extraRunsOutAt != "" && s.extraRunsOutAt < window.RunsOutAt {
			window.RunsOutAt, window.RunsOut = s.extraRunsOutAt, roundCredit(s.extraRunsOut)
		}
		windows = append(windows, window)
	}
	if s.bags > 0 {
		left := roundCredit(s.left)
		windows = append(windows, CreditWindow{
			Name:    "Prepaid credits",
			Amount:  left,
			Display: compactAmount(left) + " credits",
			Unit:    "credits",
			Prepaid: true,
		})
	}
	return windows
}

// percentageWindow 生成「已用 / 总量」窗口；比例限制在 0..100。
func percentageWindow(name string, totals creditTotals) CreditWindow {
	percent := 0.0
	if totals.total > 0 {
		percent = 100 * totals.used / totals.total
	}
	return CreditWindow{
		Name:    name,
		Used:    clampPercent(percent),
		Amount:  roundCredit(totals.used),
		Limit:   roundCredit(totals.total),
		Display: compactAmount(totals.used) + " / " + compactAmount(totals.total),
		Unit:    "credits",
	}
}

// totalWindow 是账号整体额度：续期跟随计划，到期取两者最早。
func totalWindow(plan, extra CreditWindow) CreditWindow {
	window := percentageWindow("Total credits", creditTotals{used: plan.Amount + extra.Amount, total: plan.Limit + extra.Limit})
	window.ResetsAt = plan.ResetsAt
	switch {
	case plan.RunsOutAt != "" && plan.RunsOutAt == extra.RunsOutAt:
		window.RunsOutAt = plan.RunsOutAt
		window.RunsOut = roundCredit(plan.RunsOut + extra.RunsOut)
	case plan.RunsOutAt != "" && (extra.RunsOutAt == "" || plan.RunsOutAt < extra.RunsOutAt):
		window.RunsOutAt, window.RunsOut = plan.RunsOutAt, plan.RunsOut
	case extra.RunsOutAt != "":
		window.RunsOutAt, window.RunsOut = extra.RunsOutAt, extra.RunsOut
	}
	return window
}

// planLabel 只用账号名或上游布尔值给出可展示的计划名，不推断套餐等级。
func planLabel(plan string, isPaid bool) string {
	if trimmed := strings.TrimSpace(plan); trimmed != "" {
		return trimmed
	}
	if isPaid {
		return "Pro"
	}
	return "Free plan"
}

func clampPercent(value float64) float64 {
	if math.IsNaN(value) || math.IsInf(value, 0) || value < 0 {
		return 0
	}
	if value > 100 {
		return 100
	}
	return value
}

func roundCredit(value float64) float64 {
	if math.IsNaN(value) || math.IsInf(value, 0) || value < 0 {
		return 0
	}
	return math.Round(value*100) / 100
}

// compactAmount 用千分位展示数量：整数直接展示，小数最多两位。
func compactAmount(value float64) string {
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return "0"
	}
	text := strconv.FormatFloat(value, 'f', 2, 64)
	if strings.Contains(text, ".") {
		text = strings.TrimRight(strings.TrimRight(text, "0"), ".")
	}
	return groupDigits(text)
}

// groupDigits 只给整数部分插入千分位，保留符号与小数。
func groupDigits(text string) string {
	sign := ""
	if strings.HasPrefix(text, "-") {
		sign, text = "-", text[1:]
	}
	integer, fraction, hasFraction := strings.Cut(text, ".")
	var builder strings.Builder
	builder.WriteString(sign)
	for index, char := range integer {
		if index > 0 && (len(integer)-index)%3 == 0 {
			builder.WriteByte(',')
		}
		builder.WriteRune(char)
	}
	if hasFraction && fraction != "" {
		builder.WriteString(".")
		builder.WriteString(fraction)
	}
	return builder.String()
}

// normalizeKey 把字段名归一成小写去分隔符形式，便于容错匹配。
func normalizeKey(key string) string {
	var builder strings.Builder
	for _, char := range strings.ToLower(key) {
		if char == '_' || char == '-' {
			continue
		}
		builder.WriteRune(char)
	}
	return builder.String()
}

// lookupValue 按归一化字段名取第一个存在的值。
func lookupValue(object map[string]any, names ...string) (any, bool) {
	for _, name := range names {
		for key, value := range object {
			if normalizeKey(key) == name {
				return value, true
			}
		}
	}
	return nil, false
}

func lookupString(object map[string]any, names ...string) string {
	value, found := lookupValue(object, names...)
	if !found {
		return ""
	}
	text, ok := value.(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(text)
}

func lookupBool(object map[string]any, names ...string) (bool, bool) {
	value, found := lookupValue(object, names...)
	if !found {
		return false, false
	}
	flag, ok := value.(bool)
	return flag, ok
}

// lookupNumber 解析容量数字（数字或数字字符串）；非法值按未知处理。
func lookupNumber(object map[string]any, names ...string) (float64, bool) {
	value, found := lookupValue(object, names...)
	if !found {
		return 0, false
	}
	return creditNumber(value)
}

func creditNumber(value any) (float64, bool) {
	var number float64
	switch typed := value.(type) {
	case json.Number:
		parsed, err := typed.Float64()
		if err != nil {
			return 0, false
		}
		number = parsed
	case float64:
		number = typed
	case string:
		text := strings.ReplaceAll(strings.TrimSpace(typed), ",", "")
		if text == "" {
			return 0, false
		}
		parsed, err := strconv.ParseFloat(text, 64)
		if err != nil {
			return 0, false
		}
		number = parsed
	default:
		return 0, false
	}
	if math.IsNaN(number) || math.IsInf(number, 0) || number < 0 || number > maxQuotaAmount {
		return 0, false
	}
	return number, true
}

// packageExpiry 是包内积分失去的时刻：DeductionEndTime > ExpiredTime > CycleEndTime。
func packageExpiry(object map[string]any) (time.Time, bool) {
	for _, name := range []string{"deductionendtime", "expiredtime", "cycleendtime"} {
		value, found := lookupValue(object, name)
		if !found {
			continue
		}
		if at, ok := quotaTime(value); ok {
			return at, true
		}
	}
	return time.Time{}, false
}

// cycleEnd 取计划包的周期结束时间（UTC RFC3339）；已过去或缺失返回空串。
func cycleEnd(object map[string]any) string {
	value, found := lookupValue(object, "cycleendtime")
	if !found {
		return ""
	}
	at, ok := quotaTime(value)
	if !ok || !at.After(time.Now()) {
		return ""
	}
	return at.UTC().Format(time.RFC3339)
}

// quotaTime 兼容毫秒/秒时间戳、数字字符串、RFC3339 与北京时间 "2006-01-02 15:04:05"。
func quotaTime(value any) (time.Time, bool) {
	var number float64
	switch typed := value.(type) {
	case json.Number:
		parsed, err := typed.Float64()
		if err != nil {
			return time.Time{}, false
		}
		number = parsed
	case float64:
		number = typed
	case string:
		text := strings.TrimSpace(typed)
		if text == "" {
			return time.Time{}, false
		}
		if parsed, err := strconv.ParseFloat(text, 64); err == nil {
			number = parsed
		} else {
			if at, err := time.Parse(time.RFC3339Nano, text); err == nil {
				return at.UTC(), true
			}
			for _, layout := range []string{"2006-01-02 15:04:05", "2006-01-02 15:04:05.000"} {
				if at, err := time.ParseInLocation(layout, text, beijingZone); err == nil {
					return at.UTC(), true
				}
			}
			return time.Time{}, false
		}
	default:
		return time.Time{}, false
	}
	if math.IsNaN(number) || math.IsInf(number, 0) || number <= 0 {
		return time.Time{}, false
	}
	switch {
	case number >= 1e12:
		return time.UnixMilli(int64(number)).UTC(), true
	case number >= 1e9:
		return time.Unix(int64(number), 0).UTC(), true
	}
	return time.Time{}, false
}
