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
export const readCachedSubscriptionQuota = async (id: string) =>
  AppService.SubscriptionCachedQuota(id)
export const readSubscriptionQuotaEstimate = async (id: string) =>
  AppService.SubscriptionQuotaEstimate(id)
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

export function subscriptionInProgress(state?: string) {
  return !!state && ["starting", "waiting", "exchanging", "verifying", "saving"].includes(state)
}

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
    "operation_timeout",
    "refresh_timeout",
    "refresh_busy",
    "refresh_unavailable",
    "refresh_cancelled",
    "account_update_failed",
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
    "quota_cache_unavailable",
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

// 普通刷新失败不是取消登录；兼容旧后端错误时也不引导用户无谓地重新授权。
export function subscriptionRefreshError(error: unknown) {
  const code = subscriptionError(error)
  switch (code) {
    case "operation_timeout":
      return "refresh_timeout"
    case "operation_cancelled":
    case "cancelled":
      return "refresh_cancelled"
    case "operation_failed":
    case "request_failed":
    case "protocol_failed":
    case "upstream_unavailable":
      return "refresh_unavailable"
    default:
      return code
  }
}

export function transientSubscriptionRefreshError(
  code: ReturnType<typeof subscriptionError> | ReturnType<typeof subscriptionRefreshError>,
) {
  return [
    "refresh_timeout",
    "refresh_busy",
    "refresh_unavailable",
    "refresh_cancelled",
    "account_id_pending",
    "busy",
  ].includes(code)
}
