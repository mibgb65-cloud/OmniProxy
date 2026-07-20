package history

import (
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"omniproxy/internal/sanitize"
)

const (
	defaultPersistDelay = 500 * time.Millisecond
	defaultMaxEntries   = 50000
)

type Store interface {
	Load() ([]Entry, error)
	Save([]Entry) error
}

type QueryStore interface {
	Store
	Append(Entry) error
	List(Filter, int) ([]Entry, error)
	Prune(int) error
}

type UsageStore interface {
	DailyUsage(string, int) ([]DailyUsage, error)
	DailyUsageDates(int) ([]string, error)
	BillingSummary(int) (BillingSummary, error)
	PruneBeforeDate(string) error
	ClearDailyUsage() error
	ClearRequestHistory() error
	RebuildSummaries() error
}

type SummaryStore interface {
	Summary(Filter, int) (Summary, error)
}

type Entry struct {
	ID                int64          `json:"id"`
	Time              time.Time      `json:"time"`
	Level             string         `json:"level"`
	Method            string         `json:"method,omitempty"`
	Path              string         `json:"path,omitempty"`
	Provider          string         `json:"provider,omitempty"`
	Protocol          string         `json:"protocol,omitempty"`
	ClientKey         string         `json:"clientKey,omitempty"`
	ClientName        string         `json:"clientName,omitempty"`
	Model             string         `json:"model,omitempty"`
	Status            int            `json:"status,omitempty"`
	Duration          int64          `json:"durationMs,omitempty"`
	TokenID           string         `json:"tokenId,omitempty"`
	TokenName         string         `json:"tokenName,omitempty"`
	InputTokens       int            `json:"inputTokens,omitempty"`
	OutputTokens      int            `json:"outputTokens,omitempty"`
	TotalTokens       int            `json:"totalTokens,omitempty"`
	CooldownTriggered bool           `json:"cooldownTriggered,omitempty"`
	RetryChain        []RetryAttempt `json:"retryChain,omitempty"`
	Message           string         `json:"message"`
}

type RetryAttempt struct {
	Attempt           int    `json:"attempt"`
	Provider          string `json:"provider,omitempty"`
	Protocol          string `json:"protocol,omitempty"`
	Model             string `json:"model,omitempty"`
	Status            int    `json:"status,omitempty"`
	Duration          int64  `json:"durationMs,omitempty"`
	TokenID           string `json:"tokenId,omitempty"`
	TokenName         string `json:"tokenName,omitempty"`
	CooldownTriggered bool   `json:"cooldownTriggered,omitempty"`
	Message           string `json:"message,omitempty"`
}

type Filter struct {
	Provider string `json:"provider,omitempty"`
	Client   string `json:"client,omitempty"`
	Level    string `json:"level,omitempty"`
	Status   string `json:"status,omitempty"`
	Model    string `json:"model,omitempty"`
	TokenID  string `json:"tokenId,omitempty"`
	Token    string `json:"token,omitempty"`
	Search   string `json:"search,omitempty"`
	Limit    int    `json:"limit,omitempty"`
}

type DailyUsage struct {
	Date         string    `json:"date"`
	Provider     string    `json:"provider,omitempty"`
	Protocol     string    `json:"protocol,omitempty"`
	ClientKey    string    `json:"clientKey,omitempty"`
	ClientName   string    `json:"clientName,omitempty"`
	TokenID      string    `json:"tokenId,omitempty"`
	TokenName    string    `json:"tokenName,omitempty"`
	Model        string    `json:"model,omitempty"`
	RequestCount int       `json:"requestCount"`
	InputTokens  int       `json:"inputTokens"`
	OutputTokens int       `json:"outputTokens"`
	TotalTokens  int       `json:"totalTokens"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

type DailySummary struct {
	Date         string `json:"date"`
	RequestCount int    `json:"requestCount"`
	FailedCount  int    `json:"failedCount"`
	TotalTokens  int64  `json:"totalTokens"`
}

type BillingDailySummary struct {
	Date         string `json:"date"`
	RequestCount int    `json:"requestCount"`
	InputTokens  int64  `json:"inputTokens"`
	OutputTokens int64  `json:"outputTokens"`
	TotalTokens  int64  `json:"totalTokens"`
}

type BillingSummary struct {
	RequestCount int                   `json:"requestCount"`
	InputTokens  int64                 `json:"inputTokens"`
	OutputTokens int64                 `json:"outputTokens"`
	TotalTokens  int64                 `json:"totalTokens"`
	DailyRows    []BillingDailySummary `json:"dailyRows"`
}

type Rank struct {
	Label       string `json:"label"`
	Count       int    `json:"count"`
	TotalTokens int64  `json:"totalTokens"`
	FailedCount int    `json:"failedCount"`
}

type Summary struct {
	Total              int            `json:"total"`
	Failed             int            `json:"failed"`
	FailureRate        int            `json:"failureRate"`
	TotalTokens        int64          `json:"totalTokens"`
	AverageDuration    int64          `json:"averageDuration"`
	DailyRows          []DailySummary `json:"dailyRows"`
	ProviderRanks      []Rank         `json:"providerRanks"`
	ClientRanks        []Rank         `json:"clientRanks"`
	TokenRanks         []Rank         `json:"tokenRanks"`
	ModelRanks         []Rank         `json:"modelRanks"`
	TokenFailureRanks  []Rank         `json:"tokenFailureRanks"`
	FailureReasonRanks []Rank         `json:"failureReasonRanks"`
}

type Recorder struct {
	mu            sync.RWMutex
	store         Store
	queryStore    QueryStore
	usageStore    UsageStore
	max           int
	retentionDays int
	nextID        int64
	entries       []Entry
	persistDelay  time.Duration
	saveTimer     *time.Timer
	dirty         bool
}

func NewRecorder(store Store, max int) (*Recorder, error) {
	if max <= 0 {
		max = defaultMaxEntries
	}

	entries, err := store.Load()
	if err != nil {
		return nil, err
	}
	if len(entries) > max {
		entries = entries[len(entries)-max:]
	}

	var nextID int64
	for _, entry := range entries {
		if entry.ID > nextID {
			nextID = entry.ID
		}
	}

	return &Recorder{
		store:        store,
		queryStore:   queryStore(store),
		usageStore:   usageStore(store),
		max:          max,
		nextID:       nextID,
		entries:      entries,
		persistDelay: defaultPersistDelay,
	}, nil
}

func (r *Recorder) Add(entry Entry) Entry {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.nextID++
	entry.ID = r.nextID
	if entry.Time.IsZero() {
		entry.Time = time.Now()
	}
	if entry.Level == "" {
		entry.Level = "info"
	}
	entry.Path = sanitize.Text(entry.Path)
	entry.Message = sanitize.Text(entry.Message)
	for i := range entry.RetryChain {
		entry.RetryChain[i].Message = sanitize.Text(entry.RetryChain[i].Message)
	}
	r.entries = append(r.entries, entry)
	if len(r.entries) > r.max {
		r.entries = r.entries[len(r.entries)-r.max:]
	}
	if r.queryStore != nil {
		if err := r.queryStore.Append(entry); err == nil {
			_ = r.queryStore.Prune(r.max)
			if r.retentionDays > 0 && r.usageStore != nil {
				_ = r.usageStore.PruneBeforeDate(retentionCutoffDate(time.Now(), r.retentionDays))
			}
			return entry
		}
	}
	if r.retentionDays > 0 {
		r.pruneEntriesBeforeLocked(retentionCutoffDate(time.Now(), r.retentionDays))
	}
	_ = r.schedulePersistLocked()
	return entry
}

func (r *Recorder) List(filter Filter) []Entry {
	limit := r.limit(filter)
	if r.queryStore != nil {
		if out, err := r.queryStore.List(filter, limit); err == nil {
			return out
		}
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	out := make([]Entry, 0, len(r.entries))
	for _, entry := range r.entries {
		if !matches(entry, filter) {
			continue
		}
		out = append(out, entry)
	}
	sort.SliceStable(out, func(i, j int) bool {
		return out[i].ID > out[j].ID
	})
	if len(out) > limit {
		out = out[:limit]
	}
	return out
}

func (r *Recorder) DailyUsage(date string) []DailyUsage {
	if r.usageStore == nil {
		return []DailyUsage{}
	}
	date = strings.TrimSpace(date)
	if date == "" {
		date = time.Now().Format("2006-01-02")
	}
	out, err := r.usageStore.DailyUsage(date, r.max)
	if err != nil {
		return []DailyUsage{}
	}
	return out
}

func (r *Recorder) DailyUsageDates(limit int) []string {
	if r.usageStore == nil {
		return []string{}
	}
	if limit <= 0 || limit > 366 {
		limit = 30
	}
	out, err := r.usageStore.DailyUsageDates(limit)
	if err != nil {
		return []string{}
	}
	return out
}

func (r *Recorder) BillingSummary(days int) BillingSummary {
	days = normalizeSummaryDays(days)
	if r.usageStore != nil {
		if out, err := r.usageStore.BillingSummary(days); err == nil {
			return out
		}
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	out := BillingSummary{DailyRows: []BillingDailySummary{}}
	daily := map[string]*BillingDailySummary{}
	for _, entry := range r.entries {
		if !dailyUsageCandidate(entry) {
			continue
		}
		input, output, total := tokenCounts(entry)
		out.RequestCount++
		out.InputTokens += int64(input)
		out.OutputTokens += int64(output)
		out.TotalTokens += int64(total)
		date := entry.Time.Local().Format("2006-01-02")
		current := daily[date]
		if current == nil {
			current = &BillingDailySummary{Date: date}
			daily[date] = current
		}
		current.RequestCount++
		current.InputTokens += int64(input)
		current.OutputTokens += int64(output)
		current.TotalTokens += int64(total)
	}
	out.DailyRows = billingDailySummaryWindow(daily, days)
	return out
}

func (r *Recorder) Summary(filter Filter, days int) Summary {
	days = normalizeSummaryDays(days)
	if store, ok := r.store.(SummaryStore); ok {
		if out, err := store.Summary(filter, days); err == nil {
			return out
		}
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	out := Summary{DailyRows: []DailySummary{}, ProviderRanks: []Rank{}, ClientRanks: []Rank{}, TokenRanks: []Rank{}, ModelRanks: []Rank{}, TokenFailureRanks: []Rank{}, FailureReasonRanks: []Rank{}}
	daily := map[string]*DailySummary{}
	var durationSum int64
	for _, entry := range r.entries {
		if !matches(entry, filter) {
			continue
		}
		out.Total++
		if failedEntry(entry) {
			out.Failed++
		}
		out.TotalTokens += int64(maxInt(entry.TotalTokens, 0))
		durationSum += maxInt64(entry.Duration, 0)
		day := entry.Time.Local().Format("2006-01-02")
		current := daily[day]
		if current == nil {
			current = &DailySummary{Date: day}
			daily[day] = current
		}
		current.RequestCount++
		if failedEntry(entry) {
			current.FailedCount++
		}
		current.TotalTokens += int64(maxInt(entry.TotalTokens, 0))
	}
	if out.Total > 0 {
		out.FailureRate = int((int64(out.Failed)*100 + int64(out.Total)/2) / int64(out.Total))
		out.AverageDuration = (durationSum + int64(out.Total)/2) / int64(out.Total)
	}
	out.DailyRows = dailySummaryWindow(daily, days)
	return out
}

func (r *Recorder) SetRetentionDays(days int) error {
	r.mu.Lock()
	cutoffDate := retentionCutoffDate(time.Now(), days)
	if days > 0 && r.usageStore != nil {
		if err := r.usageStore.PruneBeforeDate(cutoffDate); err != nil {
			r.mu.Unlock()
			return err
		}
	}
	r.retentionDays = days
	if days > 0 {
		r.pruneEntriesBeforeLocked(cutoffDate)
	}
	r.mu.Unlock()
	return nil
}

func (r *Recorder) ClearDailyUsage() error {
	if r.usageStore == nil {
		return nil
	}
	return r.usageStore.ClearDailyUsage()
}

func (r *Recorder) RebuildSummaries() error {
	if r.usageStore == nil {
		return nil
	}
	return r.usageStore.RebuildSummaries()
}

func (r *Recorder) ClearRequestHistory() error {
	r.mu.Lock()
	r.entries = []Entry{}
	r.nextID = 0
	r.dirty = false
	if r.saveTimer != nil {
		r.saveTimer.Stop()
		r.saveTimer = nil
	}
	r.mu.Unlock()

	if r.usageStore != nil {
		return r.usageStore.ClearRequestHistory()
	}
	return r.store.Save([]Entry{})
}

func (r *Recorder) Close() error {
	if err := r.Flush(); err != nil {
		return err
	}
	if closer, ok := r.store.(interface{ Close() error }); ok {
		return closer.Close()
	}
	return nil
}

func (r *Recorder) Flush() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.saveTimer != nil {
		r.saveTimer.Stop()
		r.saveTimer = nil
	}
	if !r.dirty {
		return nil
	}
	return r.persistLocked()
}

func (r *Recorder) limit(filter Filter) int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	limit := filter.Limit
	if limit <= 0 || limit > r.max {
		limit = r.max
	}
	return limit
}

func (r *Recorder) schedulePersistLocked() error {
	r.dirty = true
	if r.persistDelay <= 0 {
		return r.persistLocked()
	}
	if r.saveTimer == nil {
		r.saveTimer = time.AfterFunc(r.persistDelay, func() {
			_ = r.Flush()
		})
	}
	return nil
}

func queryStore(store Store) QueryStore {
	query, ok := store.(QueryStore)
	if !ok {
		return nil
	}
	return query
}

func usageStore(store Store) UsageStore {
	usage, ok := store.(UsageStore)
	if !ok {
		return nil
	}
	return usage
}

func (r *Recorder) persistLocked() error {
	snapshot := make([]Entry, len(r.entries))
	copy(snapshot, r.entries)
	err := r.store.Save(snapshot)
	r.dirty = err != nil
	return err
}

func (r *Recorder) pruneEntriesBeforeLocked(cutoffDate string) {
	if cutoffDate == "" {
		return
	}
	out := r.entries[:0]
	for _, entry := range r.entries {
		if entry.Time.IsZero() || entry.Time.Local().Format("2006-01-02") >= cutoffDate {
			out = append(out, entry)
		}
	}
	r.entries = out
}

func retentionCutoffDate(now time.Time, days int) string {
	if days <= 0 {
		return ""
	}
	return now.Local().AddDate(0, 0, -days+1).Format("2006-01-02")
}

func normalizeSummaryDays(days int) int {
	if days <= 0 || days > 366 {
		return 14
	}
	return days
}

func dailySummaryWindow(rows map[string]*DailySummary, days int) []DailySummary {
	if len(rows) == 0 {
		return []DailySummary{}
	}
	keys := make([]string, 0, len(rows))
	for key := range rows {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	endDate, err := time.ParseInLocation("2006-01-02", keys[len(keys)-1], time.Local)
	if err != nil {
		return []DailySummary{}
	}
	startDate := endDate.AddDate(0, 0, -days+1)
	out := make([]DailySummary, 0, days)
	for i := 0; i < days; i++ {
		date := startDate.AddDate(0, 0, i).Format("2006-01-02")
		if row := rows[date]; row != nil {
			out = append(out, *row)
			continue
		}
		out = append(out, DailySummary{Date: date})
	}
	return out
}

func billingDailySummaryWindow(rows map[string]*BillingDailySummary, days int) []BillingDailySummary {
	if len(rows) == 0 {
		return []BillingDailySummary{}
	}
	keys := make([]string, 0, len(rows))
	for key := range rows {
		keys = append(keys, key)
	}
	sort.Sort(sort.Reverse(sort.StringSlice(keys)))
	if len(keys) > days {
		keys = keys[:days]
	}
	out := make([]BillingDailySummary, 0, len(keys))
	for _, key := range keys {
		out = append(out, *rows[key])
	}
	return out
}

func failedEntry(entry Entry) bool {
	return strings.EqualFold(entry.Level, "error") || strings.EqualFold(entry.Level, "warn") || entry.Status >= 400
}

func maxInt64(value int64, minimum int64) int64 {
	if value < minimum {
		return minimum
	}
	return value
}

func matches(entry Entry, filter Filter) bool {
	if filter.Provider != "" && filter.Provider != "all" && !strings.EqualFold(entry.Provider, filter.Provider) {
		return false
	}
	if filter.Client != "" && filter.Client != "all" && !strings.EqualFold(entry.ClientKey, filter.Client) {
		return false
	}
	if filter.Level != "" && filter.Level != "all" && !strings.EqualFold(entry.Level, filter.Level) {
		return false
	}
	if filter.Status != "" && filter.Status != "all" && !matchesStatus(entry.Status, filter.Status) {
		return false
	}
	if filter.Model != "" && !strings.Contains(strings.ToLower(entry.Model), strings.ToLower(filter.Model)) {
		return false
	}
	if filter.TokenID != "" && filter.TokenID != "all" && !strings.EqualFold(entry.TokenID, filter.TokenID) {
		return false
	}
	if filter.Token != "" && !strings.Contains(strings.ToLower(entry.TokenName+" "+entry.TokenID), strings.ToLower(filter.Token)) {
		return false
	}
	if filter.Search != "" {
		needle := strings.ToLower(filter.Search)
		haystack := strings.ToLower(strings.Join([]string{
			entry.Method,
			entry.Path,
			entry.Provider,
			entry.Protocol,
			entry.ClientKey,
			entry.ClientName,
			entry.Model,
			entry.TokenID,
			entry.TokenName,
			entry.Message,
			strconv.Itoa(entry.Status),
		}, " "))
		if !strings.Contains(haystack, needle) {
			return false
		}
	}
	return true
}

func matchesStatus(status int, filter string) bool {
	switch strings.ToLower(strings.TrimSpace(filter)) {
	case "success":
		return status >= 200 && status < 400
	case "error":
		return status == 0 || status >= 400
	default:
		parsed, err := strconv.Atoi(filter)
		return err == nil && status == parsed
	}
}
