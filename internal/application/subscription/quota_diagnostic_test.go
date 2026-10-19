package subscription

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

func TestQuotaDiagnosticDoesNotRetainRawErrors(t *testing.T) {
	for _, cause := range []error{errors.New("private-marker"), errors.New("private-marker: subscription: request_failed")} {
		err := quotaFailure(context.Background(), "rate_limits", "quota_unavailable", cause)
		info := QuotaErrorDiagnostic(err)
		data, marshalErr := json.Marshal(info)
		if marshalErr != nil || strings.Contains(string(data)+err.Error(), "private-marker") || info.CauseCode != "operation_failed" {
			t.Fatal("quota diagnostic retained untrusted response data")
		}
	}
	if info := QuotaErrorDiagnostic(errors.New("private-marker")); info.Stage != "prepare" || info.Code != "operation_failed" || info.CauseCode != "operation_failed" {
		t.Fatal("unclassified quota failure exposed a raw message")
	}
}
