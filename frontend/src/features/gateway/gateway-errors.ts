import type { TranslationKey } from "../../i18n/translation"

const errorKeys = new Map<string, TranslationKey>([
  ["duplicate_credential", "failover.duplicate"],
  ["port_unavailable", "gateway.error.port"],
  ["invalid_configuration", "gateway.error.configuration"],
  ["configuration_unavailable", "gateway.error.configuration"],
  ["no_enabled_deepseek_provider", "gateway.error.provider"],
  ["no_enabled_deepseek_account", "gateway.error.account"],
  ["invalid_credential", "gateway.error.credential"],
  ["credential_unavailable", "gateway.error.credential"],
  ["upstream_authentication_failed", "gateway.error.authentication"],
  ["upstream_insufficient_balance", "gateway.error.balance"],
  ["upstream_rate_limited", "gateway.error.rateLimit"],
  ["upstream_quota_exhausted", "failover.quotaExhausted"],
  ["account_pool_cooling", "gateway.error.rateLimit"],
  ["route_configuration_changed", "gateway.error.configuration"],
  ["accounting_unavailable", "gateway.error.configuration"],
  ["request_timeout", "gateway.error.timeout"],
  ["request_cancelled", "gateway.error.cancelled"],
  ["gateway_busy", "gateway.error.busy"],
  ["gateway_not_running", "gateway.error.stopped"],
  ["unsupported_model", "gateway.error.model"],
  ["invalid_upstream_response", "gateway.error.response"],
  ["upstream_rejected_request", "gateway.error.response"],
  ["upstream_unavailable", "gateway.error.upstream"],
])

/** 未知代码和绑定错误只返回固定文案，不将原始错误或上游正文显示在页面。 */
export function gatewayErrorKey(code: string): TranslationKey {
  return errorKeys.get(code) ?? "gateway.error.unknown"
}

export function gatewayOperationErrorKey(error: unknown): TranslationKey {
  const message = error instanceof Error ? error.message : ""
  const code = /gateway: ([a-z_]+)/.exec(message)?.[1] ?? ""
  return gatewayErrorKey(code)
}
