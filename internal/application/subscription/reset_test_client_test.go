package subscription

import (
	"context"
	"errors"
)

func (c *testClient) ConsumeReset(_ context.Context, key, creditID string) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.resetKeys = append(c.resetKeys, key)
	c.resetCreditIDs = append(c.resetCreditIDs, creditID)
	if c.failReset {
		return "", errors.New("private-reset-error")
	}
	if c.failQuotaAfterReset {
		c.failQuota = true
	}
	if c.resetOutcome != "" {
		return c.resetOutcome, nil
	}
	return "reset", nil
}
