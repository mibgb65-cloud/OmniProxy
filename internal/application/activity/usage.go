package activity

import (
	"context"
	"math/big"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	usageWindowDays      = 30
	usageModelTrendLimit = 6
	usageModelsLimit     = 1000
	usageCalendarDays    = 365
)

func summary(input, output, requests, complete int64) Summary {
	total := new(big.Int).Add(big.NewInt(input), big.NewInt(output))
	return Summary{InputTokens: strconv.FormatInt(input, 10), OutputTokens: strconv.FormatInt(output, 10), TotalTokens: total.String(), Requests: requests, CompleteUsage: complete}
}

// Usage 读取累计用量与近 30 天的分析维度；所有数字仍以十进制字符串返回，避免前端精度损失。
func (s *Service) Usage(ctx context.Context) (UsageInfo, error) {
	now := time.Now()
	currentFrom := now.AddDate(0, 0, -(usageWindowDays - 1)).Format(time.DateOnly)
	previousFrom := now.AddDate(0, 0, -(usageWindowDays*2 - 1)).Format(time.DateOnly)
	result := UsageInfo{
		Days:      []Breakdown{},
		Calendar:  []DayUsage{},
		Hours:     make([]HourUsage, 24),
		Providers: []Breakdown{},
		Accounts:  []Breakdown{},
		Models:    []Breakdown{},
		ModelDays: []ModelDay{},
	}
	for hour := range result.Hours {
		result.Hours[hour].Hour = hour
	}
	var input, output, requests, complete int64
	if s.db.QueryRowContext(ctx, `SELECT input_tokens,output_tokens,requests,completed_usage FROM usage_totals WHERE id=1`).Scan(&input, &output, &requests, &complete) != nil {
		return result, ErrUnavailable
	}
	result.AllTime = summary(input, output, requests, complete)
	families, err := loadModelFamilies(ctx, s.db)
	if err != nil {
		return result, err
	}
	if result.Previous, err = s.previous(ctx, previousFrom, currentFrom); err != nil {
		return result, err
	}
	if result.Days, err = s.breakdown(ctx, `SELECT day,day,'','',SUM(input_tokens),SUM(output_tokens),SUM(requests),SUM(completed_usage) FROM usage_buckets WHERE day>=? GROUP BY day ORDER BY day`, currentFrom); err != nil {
		return result, err
	}
	if result.Providers, err = s.providers(ctx); err != nil {
		return result, err
	}
	if result.Accounts, err = s.breakdown(ctx, `SELECT subject,MAX(account_name),MAX(email),MAX(workspace_id),SUM(input_tokens),SUM(output_tokens),SUM(requests),SUM(completed_usage) FROM usage_buckets GROUP BY subject ORDER BY SUM(input_tokens)+SUM(output_tokens) DESC LIMIT 1000`); err != nil {
		return result, err
	}
	if result.Models, err = s.modelsBreakdown(ctx, families); err != nil {
		return result, err
	}
	if result.ModelDays, err = s.modelDays(ctx, currentFrom, families); err != nil {
		return result, err
	}
	if result.Calendar, err = s.calendarDays(ctx, now); err != nil {
		return result, err
	}
	if result.Hours, result.Reliability, err = s.usageHours(ctx, now); err != nil {
		return result, err
	}
	if result.Cache, err = s.cacheUsage(ctx); err != nil {
		return result, err
	}
	result.ProviderModels, err = s.providerModels(ctx, families)
	return result, err
}

func (s *Service) previous(ctx context.Context, from, to string) (Summary, error) {
	var input, output, requests, complete int64
	if s.db.QueryRowContext(ctx, `SELECT COALESCE(SUM(input_tokens),0),COALESCE(SUM(output_tokens),0),COALESCE(SUM(requests),0),COALESCE(SUM(completed_usage),0) FROM usage_buckets WHERE day>=? AND day<?`, from, to).Scan(&input, &output, &requests, &complete) != nil {
		return Summary{}, ErrUnavailable
	}
	return summary(input, output, requests, complete), nil
}

// providers 按提供商配置聚合提供商／模型累计；历史归属用 "legacy:" 前缀保留，方便前端标记。
func (s *Service) providers(ctx context.Context) ([]Breakdown, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT u.provider_id,COALESCE(MAX(p.name),MAX(u.provider_name)),SUM(u.requests),SUM(u.completed_usage),SUM(u.input_tokens),SUM(u.output_tokens) FROM usage_provider_models u LEFT JOIN providers p ON p.id=u.provider_id GROUP BY u.provider_id ORDER BY SUM(u.input_tokens)+SUM(u.output_tokens) DESC,COALESCE(MAX(p.name),MAX(u.provider_name)),u.provider_id`)
	if err != nil {
		return nil, ErrUnavailable
	}
	defer rows.Close()
	items := []Breakdown{}
	for rows.Next() {
		var item Breakdown
		var requests, complete, input, output int64
		if rows.Scan(&item.Key, &item.Name, &requests, &complete, &input, &output) != nil {
			return nil, ErrUnavailable
		}
		item.Summary = summary(input, output, requests, complete)
		items = append(items, item)
	}
	if rows.Err() != nil {
		return nil, ErrUnavailable
	}
	return items, nil
}

// modelsBreakdown 汇总累计各模型的用量，并按模型家族归并推理档位变体。
// 归并后才排序与截断，避免同一家族的多个变体挤占列表名额。
func (s *Service) modelsBreakdown(ctx context.Context, families map[string]string) ([]Breakdown, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT model,SUM(input_tokens),SUM(output_tokens),SUM(requests),SUM(completed_usage) FROM usage_buckets GROUP BY model`)
	if err != nil {
		return nil, ErrUnavailable
	}
	defer rows.Close()
	totals := map[string]*modelTotals{}
	order := []string{}
	for rows.Next() {
		var model string
		var input, output, requests, complete int64
		if rows.Scan(&model, &input, &output, &requests, &complete) != nil {
			return nil, ErrUnavailable
		}
		name := modelFamilyName(families, model)
		total := totals[name]
		if total == nil {
			total = &modelTotals{}
			totals[name] = total
			order = append(order, name)
		}
		total.add(input, output, requests, complete)
	}
	if rows.Err() != nil {
		return nil, ErrUnavailable
	}
	sortModelTotals(order, totals)
	items := []Breakdown{}
	for _, name := range order {
		if len(items) >= usageModelsLimit {
			break
		}
		item := Breakdown{Key: name, Name: name}
		item.Summary = totals[name].summary()
		items = append(items, item)
	}
	return items, nil
}

// sortModelTotals 按合计 Token 降序、名称升序排列归并后的模型行。
func sortModelTotals(order []string, totals map[string]*modelTotals) {
	sort.SliceStable(order, func(i, j int) bool {
		left, right := totals[order[i]], totals[order[j]]
		if left.weightsMore(*right) || right.weightsMore(*left) {
			return left.weightsMore(*right)
		}
		return strings.ToLower(order[i]) < strings.ToLower(order[j])
	})
}

// modelDays 只返回近 30 天用量最高的若干模型家族，其余由前端按每日总量合并为「其他」。
// 先按家族归并再取前六：否则同一模型的多个档位变体会占满名额，其它模型被挤进「其他」。
func (s *Service) modelDays(ctx context.Context, from string, families map[string]string) ([]ModelDay, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT day,model,SUM(input_tokens),SUM(output_tokens),SUM(requests),SUM(completed_usage) FROM usage_buckets WHERE day>=? GROUP BY day,model ORDER BY day,model`, from)
	if err != nil {
		return nil, ErrUnavailable
	}
	defer rows.Close()
	type dayModel struct{ day, model string }
	daily := map[dayModel]*modelTotals{}
	totals := map[string]*modelTotals{}
	order := []string{}
	days := map[string]bool{}
	for rows.Next() {
		var day, model string
		var input, output, requests, complete int64
		if rows.Scan(&day, &model, &input, &output, &requests, &complete) != nil {
			return nil, ErrUnavailable
		}
		name := modelFamilyName(families, model)
		key := dayModel{day: day, model: name}
		entry := daily[key]
		if entry == nil {
			entry = &modelTotals{}
			daily[key] = entry
		}
		entry.add(input, output, requests, complete)
		if totals[name] == nil {
			totals[name] = &modelTotals{}
			order = append(order, name)
		}
		totals[name].add(input, output, requests, complete)
		days[day] = true
	}
	if rows.Err() != nil {
		return nil, ErrUnavailable
	}
	sortModelTotals(order, totals)
	selected := map[string]bool{}
	for index, name := range order {
		if index >= usageModelTrendLimit {
			break
		}
		selected[name] = true
	}
	names := make([]string, 0, len(selected))
	for name := range selected {
		names = append(names, name)
	}
	sort.SliceStable(names, func(i, j int) bool {
		return strings.ToLower(names[i]) < strings.ToLower(names[j])
	})
	dayOrder := make([]string, 0, len(days))
	for day := range days {
		dayOrder = append(dayOrder, day)
	}
	sort.Strings(dayOrder)
	items := []ModelDay{}
	for _, day := range dayOrder {
		for _, name := range names {
			entry, ok := daily[dayModel{day: day, model: name}]
			if !ok {
				continue
			}
			items = append(items, ModelDay{Day: day, Model: name, Summary: entry.summary()})
		}
	}
	return items, nil
}

// calendarDays 读取近一年按本地日期归属的用量，供年度热图。没有记录的日子不返回，
// 由前端补零——「没有请求」是真实的 0，与「未报告 Token」是两件事。
func (s *Service) calendarDays(ctx context.Context, now time.Time) ([]DayUsage, error) {
	local := now.In(time.Local)
	from := local.AddDate(0, 0, -(usageCalendarDays - 1)).Format(time.DateOnly)
	rows, err := s.db.QueryContext(ctx, `SELECT day,SUM(input_tokens),SUM(output_tokens),SUM(requests) FROM usage_buckets WHERE day>=? AND day<=? GROUP BY day ORDER BY day`, from, local.Format(time.DateOnly))
	if err != nil {
		return nil, ErrUnavailable
	}
	defer rows.Close()
	items := []DayUsage{}
	for rows.Next() {
		var item DayUsage
		var input, output int64
		if rows.Scan(&item.Day, &input, &output, &item.Requests) != nil {
			return nil, ErrUnavailable
		}
		item.InputTokens, item.OutputTokens = strconv.FormatInt(input, 10), strconv.FormatInt(output, 10)
		item.TotalTokens = new(big.Int).Add(big.NewInt(input), big.NewInt(output)).String()
		items = append(items, item)
	}
	if rows.Err() != nil {
		return nil, ErrUnavailable
	}
	return items, nil
}

// usageHours 用未随明细淘汰的 15 分钟桶还原本地时段分布，并顺带汇总近 30 天请求质量。
func (s *Service) usageHours(ctx context.Context, now time.Time) ([]HourUsage, Reliability, error) {
	hours := make([]HourUsage, 24)
	for hour := range hours {
		hours[hour].Hour = hour
	}
	local := now.In(time.Local)
	start := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, time.Local).AddDate(0, 0, -(usageWindowDays - 1))
	rows, err := s.db.QueryContext(ctx, `SELECT bucket,requests,success,failed,cancelled,latency_ms,input_tokens,output_tokens,complete_usage,first_token_ms,first_token_samples FROM dashboard_buckets WHERE bucket>=? AND bucket<? ORDER BY bucket`, start.Unix(), now.Unix()+1)
	if err != nil {
		return nil, Reliability{}, ErrUnavailable
	}
	defer rows.Close()
	var total dashboardCounts
	var requests [24]int64
	var input [24]int64
	var output [24]int64
	var complete [24]int64
	for rows.Next() {
		var bucket int64
		var counts dashboardCounts
		if rows.Scan(&bucket, &counts.requests, &counts.success, &counts.failed, &counts.cancelled, &counts.latency, &counts.input, &counts.output, &counts.complete, &counts.firstToken, &counts.firstTokenSamples) != nil {
			return nil, Reliability{}, ErrUnavailable
		}
		if err := total.add(counts); err != nil {
			return nil, Reliability{}, err
		}
		hour := time.Unix(bucket, 0).In(time.Local).Hour()
		requests[hour] += counts.requests
		input[hour] += counts.input
		output[hour] += counts.output
		complete[hour] += counts.complete
	}
	if rows.Err() != nil {
		return nil, Reliability{}, ErrUnavailable
	}
	for hour := range hours {
		hours[hour].Summary = summary(input[hour], output[hour], requests[hour], complete[hour])
	}
	quality := total.summary()
	return hours, Reliability{Requests: quality.Requests, Success: quality.Success, Failed: quality.Failed, Cancelled: quality.Cancelled, SuccessRate: quality.SuccessRate, AverageLatencyMS: quality.AverageLatencyMS, AverageFirstTokenMS: quality.AverageFirstTokenMS, FirstTokenSamples: quality.FirstTokenSamples}, nil
}

// cacheUsage 汇总身份维度上已确认的缓存命中，未报告缓存的历史请求不计入分母。
func (s *Service) cacheUsage(ctx context.Context) (CacheUsage, error) {
	var cached, known int64
	if s.db.QueryRowContext(ctx, `SELECT COALESCE(SUM(cached_input_tokens),0),COALESCE(SUM(cached_known),0) FROM usage_accounts`).Scan(&cached, &known) != nil {
		return CacheUsage{}, ErrUnavailable
	}
	return CacheUsage{CachedInputTokens: strconv.FormatInt(cached, 10), KnownRequests: known}, nil
}

func (s *Service) breakdown(ctx context.Context, query string, args ...any) ([]Breakdown, error) {
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, ErrUnavailable
	}
	defer rows.Close()
	items := []Breakdown{}
	for rows.Next() {
		var item Breakdown
		var input, output, requests, complete int64
		if rows.Scan(&item.Key, &item.Name, &item.Email, &item.WorkspaceID, &input, &output, &requests, &complete) != nil {
			return nil, ErrUnavailable
		}
		item.Summary = summary(input, output, requests, complete)
		items = append(items, item)
	}
	if rows.Err() != nil {
		return nil, ErrUnavailable
	}
	return items, nil
}
