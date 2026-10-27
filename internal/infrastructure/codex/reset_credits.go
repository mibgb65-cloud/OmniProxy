package codex

import "context"

// ResetCreditSummary 与余额 credits 区分，availableCount 是官方可用总数。
type ResetCreditSummary struct {
	AvailableCount *int64        `json:"availableCount"`
	Credits        []ResetCredit `json:"credits"`
}

type ResetCredit struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	ResetType   string `json:"resetType"`
	Status      string `json:"status"`
	GrantedAt   *int64 `json:"grantedAt"`
	ExpiresAt   *int64 `json:"expiresAt"`
}

func validResetCredits(summary *ResetCreditSummary) bool {
	if summary == nil {
		return true
	}
	if summary.AvailableCount == nil || *summary.AvailableCount < 0 || *summary.AvailableCount > 1000000 || len(summary.Credits) > 1000 {
		return false
	}
	seen := map[string]bool{}
	for _, card := range summary.Credits {
		// 官方 HTTP 汇总可能不带获得时间：缺失按未知处理，不否决整份明细。
		if card.ID == "" || seen[card.ID] || !validMetadata(card.ID, 256) || !validMetadata(card.Title, 256) || !validMetadata(card.Description, 4096) || !validMetadata(card.ResetType, 80) || !validMetadata(card.Status, 80) || card.GrantedAt != nil && !validUnixSeconds(*card.GrantedAt) || card.ExpiresAt != nil && !validUnixSeconds(*card.ExpiresAt) {
			return false
		}
		seen[card.ID] = true
	}
	return true
}

func ValidResetOutcome(outcome string) bool {
	return outcome == "reset" || outcome == "alreadyRedeemed" || outcome == "nothingToReset" || outcome == "noCredit"
}

// ConsumeReset 使用固定幂等键兑换一次权益，错误时不由客户端自动重试。
func (c *rpcClient) ConsumeReset(ctx context.Context, key, creditID string) (string, error) {
	params := map[string]string{"idempotencyKey": key}
	if creditID != "" {
		params["creditId"] = creditID
	}
	var result struct {
		Outcome string `json:"outcome"`
	}
	if err := c.call(ctx, "account/rateLimitResetCredit/consume", params, &result); err != nil {
		return "", err
	}
	if !ValidResetOutcome(result.Outcome) {
		return "", ErrProtocol
	}
	return result.Outcome, nil
}
