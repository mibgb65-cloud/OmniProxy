package subscription

import (
	"context"
	"testing"
	"time"

	"omniproxy/internal/infrastructure/codex"
)

const resetTestKey = "11111111-2222-4333-8444-555555555555"

func TestSuccessfulResetIsNotReportedAsUnusedWhenRefreshFails(t *testing.T) {
	f, client, id := resetFixture(t)
	client.failQuotaAfterReset = true
	result, err := f.manager.ConsumeReset(context.Background(), ResetInput{AccountID: id, CreditID: "credit-test", OperationID: resetTestKey})
	if err != nil || result.Operation.State != "completed" || result.Operation.Outcome != "reset" || !result.RefreshFailed || result.Quota != nil {
		t.Fatal("refresh failure lost confirmed consumption result")
	}
	status, err := f.manager.ResetStatus(context.Background(), id)
	if err != nil || status.State != "completed" || status.Outcome != "reset" {
		t.Fatal("completed receipt not persisted independently of refresh")
	}
}

func resetFixture(t *testing.T) (*fixture, *testClient, string) {
	f := newFixture(t)
	view, client, _ := f.begin(t, "")
	client.authorize("reset@example.test")
	id := awaitState(t, f.manager, view.ID, "completed").Account.ID
	count, granted, expires := int64(2), time.Now().Unix()-60, time.Now().Unix()+3600
	client.rateLimits = &codex.RateLimits{AccountID: "11111111-2222-3333-4444-555555555555", Snapshot: quotaSnapshot("plus", 25, 50), ResetCredits: &codex.ResetCreditSummary{AvailableCount: &count, Credits: []codex.ResetCredit{{ID: "credit-test", ResetType: "codexRateLimits", Status: "available", GrantedAt: &granted, ExpiresAt: &expires}}}}
	return f, client, id
}

func TestManualResetPersistsOutcomeAndDoesNotRedeemDuplicateRequest(t *testing.T) {
	for _, outcome := range []string{"reset", "alreadyRedeemed", "nothingToReset", "noCredit"} {
		t.Run(outcome, func(t *testing.T) {
			f, client, id := resetFixture(t)
			client.resetOutcome = outcome
			input := ResetInput{AccountID: id, CreditID: "credit-test", OperationID: resetTestKey}
			result, err := f.manager.ConsumeReset(context.Background(), input)
			if err != nil || result.Operation.State != "completed" || result.Operation.Outcome != outcome || result.Quota == nil || result.Quota.FiveHour.RemainingPercent != 75 {
				t.Fatal("reset outcome or official post-query quota incorrect")
			}
			duplicate, err := f.manager.ConsumeReset(context.Background(), input)
			if err != nil || duplicate.Operation.Outcome != outcome || len(client.resetKeys) != 1 {
				t.Fatal("duplicate request consumed another credit")
			}
			status, err := f.manager.ResetStatus(context.Background(), id)
			if err != nil || status.OperationID != resetTestKey || status.State != "completed" {
				t.Fatal("lost IPC completion receipt cannot be recovered")
			}
		})
	}
}

func TestUncertainResetSurvivesServiceRestartAndReusesKeyEvenWithZeroInventory(t *testing.T) {
	f, client, id := resetFixture(t)
	client.failReset = true
	input := ResetInput{AccountID: id, CreditID: "credit-test", OperationID: resetTestKey}
	result, err := f.manager.ConsumeReset(context.Background(), input)
	if err != nil || result.Operation.State != "pending" {
		t.Fatal("uncertain reset did not retain operation")
	}
	resumed := New(f.accounts, f.manager.root, f.manager.factory)
	status, err := resumed.ResetStatus(context.Background(), id)
	if err != nil || status.OperationID != resetTestKey {
		t.Fatal("pending reset lost across service restart")
	}
	other, err := resumed.ConsumeReset(context.Background(), ResetInput{AccountID: id, OperationID: "22222222-2222-4333-8444-555555555555"})
	if err != nil || other.Operation.OperationID != resetTestKey || len(client.resetKeys) != 1 {
		t.Fatal("pending operation overwritten by another request")
	}
	client.failReset, client.resetOutcome = false, "alreadyRedeemed"
	zero := int64(0)
	client.rateLimits.ResetCredits.AvailableCount = &zero
	result, err = resumed.ConsumeReset(context.Background(), input)
	if err != nil || result.Operation.Outcome != "alreadyRedeemed" || result.Operation.State != "completed" || len(client.resetKeys) != 2 || client.resetKeys[0] != client.resetKeys[1] {
		t.Fatal("retry did not reuse original idempotency key")
	}
	item, _ := f.accounts.Metadata(context.Background(), id)
	if item.AuthState != "authenticated" || client.logoutCount != 0 {
		t.Fatal("reset uncertainty damaged authorization")
	}
}

func TestResetValidationPreventsUnusableOrForeignCredits(t *testing.T) {
	for _, invalid := range []string{"unknown", "expired", "redeemed", "unsupported", "different-account"} {
		t.Run(invalid, func(t *testing.T) {
			f, client, id := resetFixture(t)
			card := &client.rateLimits.ResetCredits.Credits[0]
			input := ResetInput{AccountID: id, CreditID: card.ID, OperationID: resetTestKey}
			switch invalid {
			case "unknown":
				input.CreditID = "foreign-card"
			case "expired":
				old := time.Now().Unix() - 1
				card.ExpiresAt = &old
			case "redeemed":
				card.Status = "redeemed"
			case "unsupported":
				card.ResetType = "unknown"
			case "different-account":
				client.rateLimits.AccountID = "other-account"
			}
			if _, err := f.manager.ConsumeReset(context.Background(), input); err == nil || len(client.resetKeys) != 0 {
				t.Fatal("unusable or foreign credit reached consume API")
			}
		})
	}
	f, client, id := resetFixture(t)
	for _, input := range []ResetInput{{AccountID: id, OperationID: "bad"}, {AccountID: id, OperationID: resetTestKey, CreditID: "bad\tcard"}} {
		if _, err := f.manager.ConsumeReset(context.Background(), input); err == nil {
			t.Fatal("invalid reset input accepted")
		}
	}
	client.rateLimits.ResetCredits.Credits = nil
	result, err := f.manager.ConsumeReset(context.Background(), ResetInput{AccountID: id, OperationID: resetTestKey})
	if err != nil || result.Operation.State != "completed" || client.resetCreditIDs[0] != "" {
		t.Fatal("count-only summary cannot use official next available credit")
	}
}
