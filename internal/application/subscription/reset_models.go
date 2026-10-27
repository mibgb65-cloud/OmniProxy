package subscription

import (
	"time"
	"unicode"
	"unicode/utf8"

	"omniproxy/internal/infrastructure/codex"
)

type ResetCreditSummary struct {
	AvailableCount int64         `json:"availableCount"`
	DetailsKnown   bool          `json:"detailsKnown"`
	Credits        []ResetCredit `json:"credits"`
}

type ResetCredit struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	ResetType   string `json:"resetType"`
	Status      string `json:"status"`
	GrantedAt   string `json:"grantedAt"`
	ExpiresAt   string `json:"expiresAt,omitempty"`
}

type ResetInput struct {
	AccountID   string `json:"accountId"`
	CreditID    string `json:"creditId"`
	OperationID string `json:"operationId"`
}

type ResetOperation struct {
	OperationID string `json:"operationId"`
	CreditID    string `json:"creditId"`
	State       string `json:"state"`
	Outcome     string `json:"outcome"`
}

type ResetResult struct {
	Operation     ResetOperation `json:"operation"`
	Quota         *Quota         `json:"quota,omitempty"`
	RefreshFailed bool           `json:"refreshFailed"`
}

func normalizeResetCredits(raw *codex.ResetCreditSummary) *ResetCreditSummary {
	if raw == nil {
		return nil
	}
	result := &ResetCreditSummary{AvailableCount: *raw.AvailableCount, DetailsKnown: raw.Credits != nil, Credits: []ResetCredit{}}
	for _, value := range raw.Credits {
		card := ResetCredit{ID: value.ID, Title: value.Title, Description: value.Description, ResetType: value.ResetType, Status: value.Status}
		// 官方可能不返回获得时间：留空由界面显示未知，不伪造时间。
		if value.GrantedAt != nil {
			card.GrantedAt = time.Unix(*value.GrantedAt, 0).UTC().Format(time.RFC3339)
		}
		if value.ExpiresAt != nil {
			card.ExpiresAt = time.Unix(*value.ExpiresAt, 0).UTC().Format(time.RFC3339)
		}
		result.Credits = append(result.Credits, card)
	}
	return result
}

func validOperationID(id string) bool {
	if len(id) != 36 {
		return false
	}
	for i, char := range id {
		if i == 8 || i == 13 || i == 18 || i == 23 {
			if char != '-' {
				return false
			}
			continue
		}
		if !(char >= '0' && char <= '9' || char >= 'a' && char <= 'f') {
			return false
		}
	}
	return true
}

func validCreditID(id string) bool {
	if len(id) > 256 || !utf8.ValidString(id) {
		return false
	}
	for _, char := range id {
		if unicode.IsControl(char) {
			return false
		}
	}
	return true
}
