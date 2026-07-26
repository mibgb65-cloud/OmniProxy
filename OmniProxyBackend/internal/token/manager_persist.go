package token

import "time"

func normalizeStoredToken(item Token) Token {
	provider, credentialType, err := NormalizeProviderAndCredential(item.Provider, item.CredentialType)
	if err != nil {
		provider = ProviderOpenAI
		credentialType = CredentialTypeAPIKey
	}
	item.Provider = provider
	item.CredentialType = credentialType
	item.Region = normalizeStoredRegion(provider, credentialType, item.Region)
	item.BaseURL = normalizeStoredBaseURL(provider, item.BaseURL)
	if item.Status == "" {
		item.Status = StatusActive
	}
	if item.Pinned {
		item.Selected = true
		item.Pinned = false
	}
	if item.Disabled {
		item.Selected = false
	}
	return item
}

func normalizeConsumption(consumption TokenConsumption) TokenConsumption {
	if consumption.InputTokens < 0 {
		consumption.InputTokens = 0
	}
	if consumption.OutputTokens < 0 {
		consumption.OutputTokens = 0
	}
	if consumption.TotalTokens < 0 {
		consumption.TotalTokens = 0
	}
	if consumption.CacheCreationTokens < 0 {
		consumption.CacheCreationTokens = 0
	}
	if consumption.CacheReadTokens < 0 {
		consumption.CacheReadTokens = 0
	}
	if consumption.TotalTokens == 0 && (consumption.InputTokens > 0 || consumption.OutputTokens > 0) {
		consumption.TotalTokens = consumption.InputTokens + consumption.OutputTokens
	}
	if consumption.TotalTokens == 0 && (consumption.CacheCreationTokens > 0 || consumption.CacheReadTokens > 0) {
		consumption.TotalTokens = consumption.CacheCreationTokens + consumption.CacheReadTokens
	}
	return consumption
}

func recordDailyUsage(existing []DailyTokenUsage, now time.Time, consumption TokenConsumption) []DailyTokenUsage {
	return recordDailyStats(existing, now, consumption, true, true)
}

func recordDailyRequest(existing []DailyTokenUsage, now time.Time) []DailyTokenUsage {
	return recordDailyStats(existing, now, TokenConsumption{}, true, false)
}

func recordDailyConsumption(existing []DailyTokenUsage, now time.Time, consumption TokenConsumption) []DailyTokenUsage {
	return recordDailyStats(existing, now, consumption, false, true)
}

func recordDailyStats(existing []DailyTokenUsage, now time.Time, consumption TokenConsumption, countRequest bool, countConsumption bool) []DailyTokenUsage {
	day := now.Format("2006-01-02")
	for i := range existing {
		if existing[i].Date != day {
			continue
		}
		// List and Get hand out shallow Token copies that share this backing
		// array, so today's row is updated in a fresh slice rather than in
		// place. Mutating it here would race with anyone reading those copies.
		updated := make([]DailyTokenUsage, len(existing))
		copy(updated, existing)
		if countRequest {
			updated[i].RequestCount++
		}
		if countConsumption {
			updated[i].InputTokens += int64(consumption.InputTokens)
			updated[i].OutputTokens += int64(consumption.OutputTokens)
			updated[i].TotalTokens += int64(consumption.TotalTokens)
			updated[i].CacheCreationTokens += int64(consumption.CacheCreationTokens)
			updated[i].CacheReadTokens += int64(consumption.CacheReadTokens)
		}
		return trimDailyUsage(updated)
	}

	row := DailyTokenUsage{Date: day}
	if countRequest {
		row.RequestCount = 1
	}
	if countConsumption {
		row.InputTokens = int64(consumption.InputTokens)
		row.OutputTokens = int64(consumption.OutputTokens)
		row.TotalTokens = int64(consumption.TotalTokens)
		row.CacheCreationTokens = int64(consumption.CacheCreationTokens)
		row.CacheReadTokens = int64(consumption.CacheReadTokens)
	}
	next := make([]DailyTokenUsage, 0, len(existing)+1)
	next = append(next, existing...)
	next = append(next, row)
	return trimDailyUsage(next)
}

func trimDailyUsage(existing []DailyTokenUsage) []DailyTokenUsage {
	const maxDays = 365
	if len(existing) <= maxDays {
		return existing
	}
	return existing[len(existing)-maxDays:]
}

func (m *Manager) persistLocked() error {
	if m.saveTimer != nil {
		m.saveTimer.Stop()
		m.saveTimer = nil
	}
	snapshot := make([]Token, len(m.tokens))
	copy(snapshot, m.tokens)
	err := m.store.Save(snapshot)
	m.dirty = err != nil
	return err
}

func (m *Manager) schedulePersistLocked() error {
	m.dirty = true
	if m.persistDelay <= 0 {
		return m.persistLocked()
	}
	if m.saveTimer == nil {
		m.saveTimer = time.AfterFunc(m.persistDelay, func() {
			_ = m.Flush()
		})
	}
	return nil
}

func (m *Manager) Flush() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.saveTimer != nil {
		m.saveTimer.Stop()
		m.saveTimer = nil
	}
	if !m.dirty {
		return nil
	}
	return m.persistLocked()
}
