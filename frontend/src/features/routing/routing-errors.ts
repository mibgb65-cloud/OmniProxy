import type { TranslationKey } from "../../i18n/translation"

const codes = new Map<string, TranslationKey>([
  ["ambiguous_minimax_pool", "routing.error.ambiguous_minimax_pool"],
  ["ambiguous_kimi_platform", "routing.error.ambiguous_kimi_platform"],
  ["ambiguous_stepfun_pool", "routing.error.ambiguous_stepfun_pool"],
  ["provider_missing", "routing.error.provider_missing"],
  ["provider_disabled", "routing.error.provider_disabled"],
  ["adapter_unavailable", "routing.error.adapter_unavailable"],
  ["unsupported_model", "routing.error.unsupported_model"],
  ["account_missing", "routing.error.account_missing"],
  ["account_mismatch", "routing.error.account_mismatch"],
  ["account_disabled", "routing.error.account_disabled"],
  ["no_enabled_account", "routing.error.no_enabled_account"],
  ["no_enabled_deepseek_provider", "routing.error.no_enabled_deepseek_provider"],
  ["configuration_unavailable", "routing.error.configuration_unavailable"],
])
export function routingErrorKey(code: string): TranslationKey {
  return codes.get(code) ?? "routing.actionFailed"
}
export function routingSaveErrorKey(error: unknown): TranslationKey {
  const message = error instanceof Error ? error.message : ""
  if (message.includes("routing: duplicate alias")) return "routing.aliasDuplicate"
  if (message.includes("routing: limit reached")) return "routing.error.limit"
  if (message.includes("routing: invalid target")) return "routing.targetInvalid"
  if (message.includes("routing: invalid account")) return "routing.accountInvalid"
  return "routing.saveFailed"
}
