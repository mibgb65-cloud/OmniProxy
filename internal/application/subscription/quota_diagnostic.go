package subscription

import (
	"context"
	"errors"
)

// QuotaDiagnostic 只包含固定阶段及稳定错误码，不保留官方错误正文或账号身份。
type QuotaDiagnostic struct {
	Stage     string
	Code      string
	CauseCode string
}

type quotaReadError struct{ diagnostic QuotaDiagnostic }

func (e *quotaReadError) Error() string { return "subscription: " + e.diagnostic.Code }

func quotaFailure(ctx context.Context, stage, code string, cause error) error {
	causeCode := ErrorCode(cause)
	switch {
	case errors.Is(ctx.Err(), context.DeadlineExceeded) || errors.Is(cause, context.DeadlineExceeded):
		code, causeCode = "quota_timeout", "quota_timeout"
		if stage == "queue" {
			code = "quota_busy"
		}
	case errors.Is(ctx.Err(), context.Canceled) || errors.Is(cause, context.Canceled):
		code, causeCode = "operation_cancelled", "operation_cancelled"
	}
	return &quotaReadError{diagnostic: QuotaDiagnostic{Stage: stage, Code: code, CauseCode: causeCode}}
}

// QuotaErrorDiagnostic 提取额度查询阶段，未知错误只返回安全通用码。
func QuotaErrorDiagnostic(err error) QuotaDiagnostic {
	var quotaErr *quotaReadError
	if errors.As(err, &quotaErr) {
		return quotaErr.diagnostic
	}
	return QuotaDiagnostic{Stage: "prepare", Code: ErrorCode(err), CauseCode: ErrorCode(err)}
}
