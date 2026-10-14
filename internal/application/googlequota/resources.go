package googlequota

import (
	"context"
	"database/sql"
	"time"
)

type Observation struct {
	Model      string `json:"model"`
	Category   string `json:"category"`
	Metric     string `json:"metric,omitempty"`
	Limit      *int64 `json:"limit,omitempty"`
	RetryAt    string `json:"retryAt,omitempty"`
	ObservedAt string `json:"observedAt"`
	Stale      bool   `json:"stale"`
}

type Resource struct {
	ID                string        `json:"id"`
	ProjectID         string        `json:"projectId"`
	Verified          bool          `json:"verified"`
	AccountIDs        []string      `json:"accountIds"`
	Observations      []Observation `json:"observations"`
	OfficialRemaining *int64        `json:"officialRemaining"`
}

// List 同项目只返回一个资源池，未绑定时保持账号级未知，不推断独立额度。
func (s *Service) List(ctx context.Context) ([]Resource, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT a.id,COALESCE(b.project_id,'') FROM accounts a JOIN providers p ON p.id=a.provider_id
 LEFT JOIN google_project_bindings b ON b.account_id=a.id AND b.credential_version=(a.credential_ref || ':' || a.auth_account_id)
 WHERE p.kind='google' AND a.auth_method='api_key' ORDER BY a.id LIMIT 1001`)
	if err != nil {
		return nil, ErrUnavailable
	}
	resources := []Resource{}
	groups := map[string]int{}
	accounts := map[string]int{}
	for rows.Next() {
		var id, project string
		if rows.Scan(&id, &project) != nil || len(accounts) >= 1000 {
			rows.Close()
			return nil, ErrUnavailable
		}
		scope := "account:" + id
		if project != "" {
			scope = "project:" + project
		}
		index, found := groups[scope]
		if !found {
			index = len(resources)
			groups[scope] = index
			resources = append(resources, Resource{ID: scope, ProjectID: project, AccountIDs: []string{}, Observations: []Observation{}})
		}
		resources[index].AccountIDs = append(resources[index].AccountIDs, id)
		accounts[id] = index
	}
	if rows.Err() != nil {
		rows.Close()
		return nil, ErrUnavailable
	}
	if rows.Close() != nil {
		return nil, ErrUnavailable
	}
	observed, err := s.db.QueryContext(ctx, `SELECT o.account_id,o.model,o.category,o.metric,o.quota_limit,o.retry_at,o.observed_at FROM google_resource_observations o
 JOIN accounts a ON a.id=o.account_id AND o.credential_version=(a.credential_ref || ':' || a.auth_account_id) ORDER BY o.observed_at DESC LIMIT 1000`)
	if err != nil {
		return nil, ErrUnavailable
	}
	defer observed.Close()
	seen := map[string]bool{}
	now := time.Now()
	for observed.Next() {
		var id string
		var item Observation
		var limit sql.NullInt64
		if observed.Scan(&id, &item.Model, &item.Category, &item.Metric, &limit, &item.RetryAt, &item.ObservedAt) != nil {
			return nil, ErrUnavailable
		}
		index, found := accounts[id]
		if !found {
			continue
		}
		identity := resources[index].ID + ":" + item.Model + ":" + item.Category + ":" + item.Metric
		if seen[identity] {
			continue
		}
		seen[identity] = true
		if limit.Valid {
			value := limit.Int64
			item.Limit = &value
		}
		at, err := time.Parse(time.RFC3339Nano, item.ObservedAt)
		if err != nil {
			return nil, ErrUnavailable
		}
		item.Stale = now.Sub(at) > 24*time.Hour
		if item.RetryAt != "" {
			until, err := time.Parse(time.RFC3339Nano, item.RetryAt)
			if err != nil {
				return nil, ErrUnavailable
			}
			item.Stale = item.Stale || !until.After(now)
		}
		resources[index].Observations = append(resources[index].Observations, item)
	}
	if observed.Err() != nil {
		return nil, ErrUnavailable
	}
	return resources, nil
}
