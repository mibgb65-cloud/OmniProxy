import { AppService } from "../../bindings/omniproxy/internal/transport/wails"
import type {
  Input,
  State,
  Quota,
  QuotaWindow,
  ResetCredit,
  ResetOperation,
  ResetResult,
  ResetInput,
} from "../../bindings/omniproxy/internal/application/subscription"

export type SubscriptionInput = Input
export type SubscriptionSession = State
export type SubscriptionQuota = Quota
export type SubscriptionQuotaWindow = QuotaWindow
export const readSubscriptionQuota = async (id: string) => AppService.SubscriptionQuota(id)
export type SubscriptionResetCredit = ResetCredit
export type SubscriptionResetOperation = ResetOperation
export type SubscriptionResetResult = ResetResult
export type SubscriptionResetInput = ResetInput
export const readSubscriptionResetStatus = async (id: string) =>
  AppService.SubscriptionResetStatus(id)
export const consumeSubscriptionReset = async (input: ResetInput) =>
  AppService.SubscriptionConsumeReset(input)
export const beginSubscription = async (input: Input) => AppService.SubscriptionBegin(input)
export const readSubscription = async (id: string) => AppService.SubscriptionRead(id)
export const currentSubscription = async (providerId: string, accountId: string) =>
  AppService.SubscriptionCurrent(providerId, accountId)
export const cancelSubscription = async (id: string) => AppService.SubscriptionCancel(id)
export const openSubscription = async (id: string) => AppService.SubscriptionOpen(id)
export const refreshSubscription = async (id: string) => AppService.SubscriptionRefresh(id)
export const logoutSubscription = async (id: string) => AppService.SubscriptionLogout(id)

// 只识别稳定错误码，原始错误可能包含提供商信息，不渲染到界面。
export function subscriptionError(error: unknown) {
  const text = error instanceof Error ? error.message : String(error)
  const known = [
    "account_unavailable",
    "initialization_failed",
    "request_failed",
    "protocol_failed",
    "login_failed",
    "operation_cancelled",
    "invalid_account",
    "invalid_provider",
    "quota_changed",
    "agy_unavailable",
    "client_incompatible",
    "reauth_required",
    "access_denied",
    "rate_limited",
    "project_unavailable",
    "invalid_response",
    "upstream_unavailable",
    "credential_unavailable",
    "invalid_input",
    "cancelled",
    "reset_record_failed",
    "reset_unavailable",
    "reset_no_credit",
    "reset_not_usable",
    "quota_timeout",
    "quota_busy",
    "quota_refresh_failed",
    "quota_unavailable",
    "quota_not_authenticated",
    "account_id_pending",
    "codex_unavailable",
    "login_expired",
    "busy",
    "identity_changed",
    "cleanup_failed",
    "browser_failed",
  ] as const
  return known.find((code) => text.includes(`subscription: ${code}`)) ?? "operation_failed"
}
