package token

import (
	"math"
	"time"
)

func (m *Manager) RecordUsage(id string, remaining int) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for i := range m.tokens {
		if m.tokens[i].ID != id {
			continue
		}

		now := time.Now()
		m.tokens[i].LastUsedAt = &now
		m.tokens[i].UpdatedAt = now
		m.tokens[i].LastError = ""
		m.tokens[i].CooldownUntil = nil

		if remaining >= 0 {
			m.tokens[i].Remaining = remaining
			m.tokens[i].Usage.APIRemaining = remaining
			m.tokens[i].Usage.UpdatedAt = &now
			switch {
			case remaining <= 0:
				m.tokens[i].Status = StatusExhausted
			case remaining <= m.threshold:
				m.tokens[i].Status = StatusLow
			default:
				m.tokens[i].Status = StatusActive
			}
		}

		return m.schedulePersistLocked()
	}

	return ErrTokenNotFound
}

func (m *Manager) RecordUsageInfo(id string, usage UsageInfo) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for i := range m.tokens {
		if m.tokens[i].ID != id {
			continue
		}

		now := time.Now()
		usage.UpdatedAt = &now
		m.tokens[i].Usage = usage
		m.tokens[i].UpdatedAt = now
		m.tokens[i].LastError = ""
		m.tokens[i].CooldownUntil = nil

		if usage.SubscriptionQuotaAvailable {
			remaining := usage.EffectiveRemainingPercent()
			m.tokens[i].Remaining = remaining
			switch {
			case usage.LimitReached || remaining <= 0:
				m.tokens[i].Status = StatusExhausted
			case remaining <= m.threshold:
				m.tokens[i].Status = StatusLow
			default:
				m.tokens[i].Status = StatusActive
			}
		} else if usage.BalanceUnit != "" {
			remaining := balanceRemainingPercent(usage)
			m.tokens[i].Remaining = remaining
			switch {
			case usage.BalanceUnlimited:
				m.tokens[i].Status = StatusActive
			case usage.BalanceRemaining <= 0:
				m.tokens[i].Status = StatusExhausted
			case remaining <= m.threshold:
				m.tokens[i].Status = StatusLow
			default:
				m.tokens[i].Status = StatusActive
			}
		}

		return m.schedulePersistLocked()
	}

	return ErrTokenNotFound
}

func (m *Manager) InvalidateCodexResetCredits(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for i := range m.tokens {
		if m.tokens[i].ID != id {
			continue
		}
		m.tokens[i].Usage.CodexResetCreditsCheckedAt = 0
		m.tokens[i].Usage.CodexResetCreditsError = ""
		return nil
	}

	return ErrTokenNotFound
}

func (m *Manager) RecordCodexWeeklyEstimateReset(id string, resetAt time.Time, baseline TokenStats) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for i := range m.tokens {
		if m.tokens[i].ID != id {
			continue
		}
		m.tokens[i].Stats.CodexWeeklyEstimateResetAt = resetAt.Unix()
		m.tokens[i].Stats.CodexWeeklyEstimateBaseline = &CodexWeeklyEstimateBaseline{
			InputTokens:         baseline.InputTokens,
			OutputTokens:        baseline.OutputTokens,
			TotalTokens:         baseline.TotalTokens,
			CacheCreationTokens: baseline.CacheCreationTokens,
			CacheReadTokens:     baseline.CacheReadTokens,
		}
		return m.schedulePersistLocked()
	}

	return ErrTokenNotFound
}

func balanceRemainingPercent(usage UsageInfo) int {
	if usage.BalanceUnlimited {
		return 100
	}
	if usage.BalanceTotal > 0 {
		value := int(math.Round((usage.BalanceRemaining / usage.BalanceTotal) * 100))
		if value < 0 {
			return 0
		}
		if value > 100 {
			return 100
		}
		return value
	}
	if usage.BalanceRemaining <= 0 {
		return 0
	}
	return 100
}

func (m *Manager) RecordProxyUsage(id string, consumption TokenConsumption) error {
	return m.recordProxyStats(id, consumption, true, true)
}

func (m *Manager) RecordProxyRequest(id string) error {
	return m.recordProxyStats(id, TokenConsumption{}, true, false)
}

func (m *Manager) RecordProxyConsumption(id string, consumption TokenConsumption) error {
	return m.recordProxyStats(id, consumption, false, true)
}

func (m *Manager) recordProxyStats(id string, consumption TokenConsumption, countRequest bool, countConsumption bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for i := range m.tokens {
		if m.tokens[i].ID != id {
			continue
		}

		now := time.Now()
		consumption = normalizeConsumption(consumption)
		m.tokens[i].LastUsedAt = &now
		m.tokens[i].UpdatedAt = now
		m.tokens[i].LastError = ""
		if countRequest {
			m.tokens[i].Stats.RequestCount++
		}
		if countConsumption {
			m.tokens[i].Stats.InputTokens += int64(consumption.InputTokens)
			m.tokens[i].Stats.OutputTokens += int64(consumption.OutputTokens)
			m.tokens[i].Stats.TotalTokens += int64(consumption.TotalTokens)
			m.tokens[i].Stats.CacheCreationTokens += int64(consumption.CacheCreationTokens)
			m.tokens[i].Stats.CacheReadTokens += int64(consumption.CacheReadTokens)
			m.tokens[i].Stats.LastInputTokens = consumption.InputTokens
			m.tokens[i].Stats.LastOutputTokens = consumption.OutputTokens
			m.tokens[i].Stats.LastTotalTokens = consumption.TotalTokens
			m.tokens[i].Stats.LastCacheCreationTokens = consumption.CacheCreationTokens
			m.tokens[i].Stats.LastCacheReadTokens = consumption.CacheReadTokens
		}
		m.tokens[i].Stats.UpdatedAt = &now
		switch {
		case countRequest && countConsumption:
			m.tokens[i].Stats.Daily = recordDailyUsage(m.tokens[i].Stats.Daily, now, consumption)
		case countRequest:
			m.tokens[i].Stats.Daily = recordDailyRequest(m.tokens[i].Stats.Daily, now)
		case countConsumption:
			m.tokens[i].Stats.Daily = recordDailyConsumption(m.tokens[i].Stats.Daily, now, consumption)
		}

		return m.schedulePersistLocked()
	}

	return ErrTokenNotFound
}

func (m *Manager) MarkExhausted(id string, reason string) error {
	return m.MarkExhaustedUntil(id, reason, nil)
}

func (m *Manager) MarkExhaustedUntil(id string, reason string, until *time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	_, err := m.mutateTokenLocked(id, func(item *Token) {
		item.Status = StatusExhausted
		item.LastError = reason
		item.CooldownUntil = until
		if until != nil {
			item.Health.NextCheckAt = until
		}
		item.UpdatedAt = time.Now()
	})
	if err != nil {
		return err
	}
	return m.persistLocked()
}

func (m *Manager) MarkInvalid(id string, reason string) error {
	return m.setStatus(id, StatusInvalid, reason)
}

func (m *Manager) MarkActive(id string) error {
	return m.setStatus(id, StatusActive, "")
}

func (m *Manager) setStatus(id string, status Status, reason string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	_, err := m.mutateTokenLocked(id, func(item *Token) {
		item.Status = status
		item.LastError = reason
		if status != StatusExhausted {
			item.CooldownUntil = nil
		}
		item.UpdatedAt = time.Now()
	})
	if err != nil {
		return err
	}
	return m.persistLocked()
}
