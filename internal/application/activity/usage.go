package activity

import (
	"context"
	"math/big"
	"strconv"
	"time"
)

func summary(input, output, requests, complete int64) Summary {
	total := new(big.Int).Add(big.NewInt(input), big.NewInt(output))
	return Summary{InputTokens: strconv.FormatInt(input, 10), OutputTokens: strconv.FormatInt(output, 10), TotalTokens: total.String(), Requests: requests, CompleteUsage: complete}
}
func (s *Service) Usage(ctx context.Context) (UsageInfo, error) {
	result := UsageInfo{Days: []Breakdown{}, Accounts: []Breakdown{}, Models: []Breakdown{}}
	var input, output, requests, complete int64
	if s.db.QueryRowContext(ctx, `SELECT input_tokens,output_tokens,requests,completed_usage FROM usage_totals WHERE id=1`).Scan(&input, &output, &requests, &complete) != nil {
		return result, ErrUnavailable
	}
	result.AllTime = summary(input, output, requests, complete)
	var err error
	result.Days, err = s.breakdown(ctx, `SELECT day,day,'','',SUM(input_tokens),SUM(output_tokens),SUM(requests),SUM(completed_usage) FROM usage_buckets WHERE day>=? GROUP BY day ORDER BY day`, time.Now().AddDate(0, 0, -29).Format(time.DateOnly))
	if err != nil {
		return result, err
	}
	result.Accounts, err = s.breakdown(ctx, `SELECT subject,MAX(account_name),MAX(email),MAX(workspace_id),SUM(input_tokens),SUM(output_tokens),SUM(requests),SUM(completed_usage) FROM usage_buckets GROUP BY subject ORDER BY SUM(input_tokens)+SUM(output_tokens) DESC LIMIT 1000`)
	if err != nil {
		return result, err
	}
	result.Models, err = s.breakdown(ctx, `SELECT model,model,'','',SUM(input_tokens),SUM(output_tokens),SUM(requests),SUM(completed_usage) FROM usage_buckets GROUP BY model ORDER BY SUM(input_tokens)+SUM(output_tokens) DESC LIMIT 1000`)
	if err != nil {
		return result, err
	}
	result.ProviderModels, err = s.providerModels(ctx)
	return result, err
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
